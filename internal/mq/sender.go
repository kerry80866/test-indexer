package mq

import (
	"context"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/pb"
	"errors"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/zeromicro/go-zero/core/logx"
)

// SendEventsAndWait 将 events 按 token 分区后并发发送，所有发送完成后才返回，失败会记录日志
// 参数说明：
//   - ctx: 上下文，用于控制超时和取消
//   - producer: Kafka 生产者实例
//   - topic: Kafka 主题
//   - events: 待发送的事件列表
//   - numPartitions: 分区数量，用于事件分区
func SendEventsAndWait(ctx context.Context, producer *kafka.Producer, topic string, events []*core.Event, numPartitions int) error {
	// 空事件列表直接返回
	if len(events) == 0 {
		logx.Infof("no events to send, skipped")
		return nil
	}
	// 参数验证
	if producer == nil || topic == "" {
		return errors.New("invalid Kafka producer or topic")
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		logx.Infof("send aborted before dispatch: %v", ctx.Err())
		return ctx.Err()
	default:
	}

	// 单分区情况直接发送
	if numPartitions <= 1 {
		err := sendPartitionEventsAndWait(ctx, producer, topic, 0, events)
		if err != nil {
			logx.Errorf("Kafka send failed: partition=0 error=%v", err)
		}
		return err
	}

	// 初始化分区桶，预分配容量以减少内存分配和扩容开销
	buckets := make([][]*core.Event, numPartitions)
	// 容量计算策略：
	// 1. 最小容量为 10，避免频繁扩容
	// 2. 最大容量为事件总数的一半，避免过度分配
	// 3. 理想容量为平均值的 4 倍，以应对负载不均衡的情况
	avg := len(events) / numPartitions
	capacity := max(10, min(avg*4, len(events)/2))
	for i := range buckets {
		buckets[i] = make([]*core.Event, 0, capacity)
	}

	// 按 token 的最后一个字节对事件进行分区
	for _, ev := range events {
		if len(ev.Token) == 0 {
			logx.Infof("skip event with empty token: slot=%d eventId=%d", ev.Tx.TxCtx.Slot, ev.EventId)
			continue
		}
		partition := int(ev.Token[len(ev.Token)-1]) % numPartitions
		buckets[partition] = append(buckets[partition], ev)
	}

	// 并发发送到各个分区
	var wg sync.WaitGroup
	// 错误通道容量设为事件总数，保守估计最大错误数
	errCh := make(chan error, len(events))

	// 为每个非空分区启动一个 goroutine 进行发送
	for i, evts := range buckets {
		if len(evts) == 0 {
			continue
		}
		wg.Add(1)
		go func(partitionID int, events []*core.Event) {
			defer wg.Done()
			if err := sendPartitionEventsAndWait(ctx, producer, topic, partitionID, events); err != nil {
				logx.Errorf("Kafka send failed: partition=%d error=%v", partitionID, err)
				errCh <- err
			}
		}(i, evts)
	}

	// 等待所有发送完成并关闭通道
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errCh)
	}()

	// 等待发送完成或上下文取消
	var firstErr error
	select {
	case <-done:
		// 收集所有错误，但只返回第一个错误
		for err := range errCh {
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// sendPartitionEventsAndWait 将一批事件发送到指定 Kafka 分区，并等待全部 ack
// 参数说明：
//   - ctx: 上下文，用于控制超时和取消
//   - producer: Kafka 生产者实例
//   - topic: Kafka 主题
//   - partitionID: 目标分区 ID
//   - events: 待发送的事件列表
func sendPartitionEventsAndWait(ctx context.Context, producer *kafka.Producer, topic string, partitionID int, events []*core.Event) error {
	if len(events) == 0 {
		return nil
	}

	logx.Infof("sending %d events to topic %s", len(events), topic)

	// 发送所有消息
	for _, ev := range events {
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: int32(partitionID),
			},
			Value: ev.Data,
		}

		if err := producer.Produce(msg, nil); err != nil {
			logx.Errorf("produce failed: slot=%d eventId=%d error=%v", ev.Tx.TxCtx.Slot, ev.EventId, err)
			return err
		}
	}

	// 等待所有消息发送完成
	s := time.Now()
	producer.Flush(10000) // 等待最多10秒
	logx.Infof("all events sent, %v elapsed", time.Since(s))
	return nil
}

// isCriticalEvent 判断是否是需要强关注的关键事件
// 目前只有 Buy/Sell 类型的事件被认为是关键事件
// 关键事件发送失败会被记录到错误通道，非关键事件只记录日志
func isCriticalEvent(eventType uint32) bool {
	return (eventType == uint32(pb.EventType_TRADE_BUY) || eventType == uint32(pb.EventType_TRADE_SELL))
}

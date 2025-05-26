package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaJob 表示一条需要发送的 Kafka 消息
type KafkaJob struct {
	Topic     string
	Partition int32
	Value     []byte
	Msg       proto.Message
	SendTime  time.Duration // 实际发送耗时
	AckTime   time.Duration // 等待确认耗时
}

// KafkaSendResult 表示每条消息的发送结果
type KafkaSendResult struct {
	Job      *KafkaJob
	Err      error
	SendTime time.Duration // 实际发送耗时
	AckTime  time.Duration // 等待确认耗时
}

// SendKafkaJobs 并发发送多条 Kafka 消息，支持外部 context 控制超时/取消
func SendKafkaJobs(
	ctx context.Context,
	producer *kafka.Producer,
	jobs []*KafkaJob,
	perMessageTimeout time.Duration,
) (ok []*KafkaJob, failed []KafkaSendResult) {
	if len(jobs) == 0 {
		return nil, nil
	}
	var wg sync.WaitGroup
	resultCh := make(chan KafkaSendResult, len(jobs)) // 缓冲避免阻塞

	for _, job := range jobs {
		wg.Add(1)
		go func(job *KafkaJob) {
			defer wg.Done()

			deliveryChan := make(chan kafka.Event, 1)
			if len(job.Value) == 0 {
				if job.Msg == nil {
					resultCh <- KafkaSendResult{Job: job, Err: fmt.Errorf("no value or message provided")}
					return
				}
				var err error
				job.Value, err = proto.Marshal(job.Msg)
				if err != nil {
					resultCh <- KafkaSendResult{Job: job, Err: fmt.Errorf("marshal error: %w", err)}
					return
				}
			}

			// 记录发送开始时间
			sendStartTime := time.Now()
			err := producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &job.Topic,
					Partition: job.Partition,
				},
				Value: job.Value,
			}, deliveryChan)
			sendTime := time.Since(sendStartTime)

			if err != nil {
				resultCh <- KafkaSendResult{Job: job, Err: fmt.Errorf("produce error: %w", err)}
				return
			}

			// 记录确认开始时间
			ackStartTime := time.Now()
			select {
			case e, ok := <-deliveryChan:
				ackTime := time.Since(ackStartTime)
				if !ok {
					resultCh <- KafkaSendResult{Job: job, Err: fmt.Errorf("delivery channel closed unexpectedly")}
					return
				}
				msg, ok := e.(*kafka.Message)
				if !ok {
					resultCh <- KafkaSendResult{Job: job, Err: fmt.Errorf("invalid message type: %T", e)}
					return
				}
				if msg.TopicPartition.Error != nil {
					job.SendTime = sendTime
					job.AckTime = ackTime
					resultCh <- KafkaSendResult{Job: job, Err: msg.TopicPartition.Error}
				} else {
					job.SendTime = sendTime
					job.AckTime = ackTime
					resultCh <- KafkaSendResult{Job: job, Err: nil}
				}
			case <-time.After(perMessageTimeout):
				go safeDrain(deliveryChan, job.Topic)
				job.SendTime = sendTime
				job.AckTime = time.Since(ackStartTime)
				resultCh <- KafkaSendResult{Job: job, Err: fmt.Errorf("delivery timeout (>%v)", perMessageTimeout)}
			case <-ctx.Done():
				go safeDrain(deliveryChan, job.Topic)
				job.SendTime = sendTime
				job.AckTime = time.Since(ackStartTime)
				resultCh <- KafkaSendResult{Job: job, Err: fmt.Errorf("ctx cancelled: %w", ctx.Err())}
			}
		}(job)
	}

	// 等待所有发送完成再关闭结果通道
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 聚合结果
	for res := range resultCh {
		if res.Err != nil {
			failed = append(failed, res)
		} else {
			ok = append(ok, res.Job)
		}
	}

	return ok, failed
}

// safeDrain 用于确保 deliveryChan 被 drain 避免 Kafka 回调阻塞
func safeDrain(ch <-chan kafka.Event, topic string) {
	defer func() {
		_ = recover() // 如果 deliveryChan 已被 Kafka 回收导致 panic（极少见），吞掉
	}()
	select {
	case <-ch:
	case <-time.After(2 * time.Second): // 最多等 2 秒
	}
}

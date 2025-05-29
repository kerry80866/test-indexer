package mq

import (
	"context"
	"dex-indexer-sol/pb"
	"fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// deleteConsumerGroup 删除指定的消费组
func deleteConsumerGroup(brokers string, groupID string) error {
	// 创建管理员客户端
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return fmt.Errorf("创建管理员客户端失败: %w", err)
	}
	defer adminClient.Close()

	// 创建删除消费组的请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 删除消费组
	_, err = adminClient.DeleteConsumerGroups(ctx, []string{groupID})
	if err != nil {
		return fmt.Errorf("删除消费组失败: %w", err)
	}

	return nil
}

func TestKafkaMessageDeserialization(t *testing.T) {
	// 生成唯一的消费组ID
	groupID := fmt.Sprintf("test-consumer-group-%d", time.Now().UnixNano())

	// 配置Kafka消费者
	config := &kafka.ConfigMap{
		"bootstrap.servers":       "172.19.3.15:9092",
		"group.id":                groupID,
		"auto.offset.reset":       "latest", // 从最新的消息开始消费
		"enable.auto.commit":      true,     // 自动提交 offset
		"auto.commit.interval.ms": 5000,     // 每 5 秒自动提交一次 offset
		"session.timeout.ms":      10000,    // 会话超时时间 10 秒
		"heartbeat.interval.ms":   3000,     // 心跳间隔 3 秒
	}

	// 创建消费者
	consumer, err := kafka.NewConsumer(config)
	assert.NoError(t, err)
	defer consumer.Close()

	// 订阅主题
	topics := []string{"dex_indexer_sol_event", "dex_indexer_sol_balance"}
	err = consumer.SubscribeTopics(topics, nil)
	assert.NoError(t, err)

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 接收消息
	for {
		select {
		case <-ctx.Done():
			t.Log("接收消息超时")
			// 测试结束后删除消费组
			err := deleteConsumerGroup("172.19.3.15:9092", groupID)
			if err != nil {
				t.Logf("删除消费组失败: %v", err)
			} else {
				t.Logf("成功删除消费组: %s", groupID)
			}
			return
		default:
			msg, err := consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				t.Fatalf("接收消息错误: %v", err)
			}

			// 反序列化消息
			events := &pb.Events{}
			err = proto.Unmarshal(msg.Value, events)
			assert.NoError(t, err)

			// 遍历所有事件并只打印 Trade 事件
			for i, event := range events.Events {
				if trade, ok := event.Event.(*pb.Event_Trade); ok {
					t.Logf("Trade事件[%d]: slot=%d, type=%s, priceUsd=%.8f", i, events.Slot, trade.Trade.Type, trade.Trade.PriceUsd)
				}
			}
		}
	}
}

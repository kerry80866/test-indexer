package mq

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/zeromicro/go-zero/core/logx"
)

// 测试用的简单事件结构
type testEvent struct {
	ID   int
	Data []byte
}

func TestSendToFixedPartition(t *testing.T) {
	// 创建 Kafka 生产者配置
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Timeout = 5 * time.Second

	// 创建生产者
	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// 创建测试事件
	events := []testEvent{
		{ID: 1, Data: []byte("test-data-1")},
		{ID: 2, Data: []byte("test-data-2")},
	}

	// 发送到固定分区（分区0）
	topic := "test-topic"

	// 发送所有消息
	for _, ev := range events {
		msg := &sarama.ProducerMessage{
			Topic:     topic,
			Partition: 0, // 固定分区0
			Value:     sarama.ByteEncoder(ev.Data),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}
		logx.Infof("Message sent: partition=%d offset=%d", partition, offset)
	}

	logx.Info("Test completed successfully")
}

// 测试发送大量事件
func TestSendBulkEvents(t *testing.T) {
	// 创建 Kafka 生产者配置
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Timeout = 5 * time.Second
	config.Producer.Flush.Frequency = 100 * time.Millisecond
	config.Producer.Flush.MaxMessages = 100

	// 创建生产者
	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// 创建100个测试事件
	events := make([]testEvent, 100)
	for i := 0; i < 100; i++ {
		events[i] = testEvent{
			ID:   i + 1,
			Data: []byte("test-data"),
		}
	}

	// 发送到固定分区（分区0）
	topic := "test-topic"

	// 发送所有消息
	for _, ev := range events {
		msg := &sarama.ProducerMessage{
			Topic:     topic,
			Partition: 0, // 固定分区0
			Value:     sarama.ByteEncoder(ev.Data),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}
		logx.Infof("Message sent: partition=%d offset=%d", partition, offset)
	}

	logx.Info("Bulk test completed successfully")
}

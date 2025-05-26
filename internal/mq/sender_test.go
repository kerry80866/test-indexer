package mq

import (
	"context"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
)

const (
	testBatchSize = 32 * 1024 // 32KB
	testLingerMs  = 5         // 5ms
	testTopic     = "test-topic"
)

// 创建测试用的 Kafka 配置
func createTestConfig(clientID string) *kafka.ConfigMap {
	return &kafka.ConfigMap{
		"bootstrap.servers": "127.0.0.1:9092",
		"client.id":         clientID,

		// 可靠性保障
		"acks":               "all",
		"enable.idempotence": false,

		// 超时与重试
		"delivery.timeout.ms":      30000,
		"request.timeout.ms":       30000,
		"message.send.max.retries": 3,
		"retry.backoff.ms":         100,

		// 性能优化
		"batch.size":       testBatchSize,
		"linger.ms":        testLingerMs,
		"compression.type": "none",

		// 消息大小
		"message.max.bytes": 2 * 1024 * 1024, // 2MB

		// 允许自动创建 topic
		"allow.auto.create.topics": true,
	}
}

// 创建测试用的生产者
func createTestProducer(t *testing.T, clientID string) *kafka.Producer {
	producer, err := kafka.NewProducer(createTestConfig(clientID))
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	return producer
}

// 测试正常发送消息
func TestSendKafkaJobs_RealKafka(t *testing.T) {
	producer := createTestProducer(t, "test-producer")
	defer producer.Close()

	// 创建消费者
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "127.0.0.1:9092",
		"group.id":          "test-group-" + time.Now().Format("20060102150405"), // 动态生成消费者组
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// 订阅 topic
	err = consumer.Subscribe(testTopic, nil)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// 测试数据
	jobs := []*KafkaJob{
		{
			Topic: testTopic,
			Value: []byte("test message 1"),
		},
		{
			Topic: testTopic,
			Value: []byte("test message 2"),
		},
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 发送消息
	ok, failed := SendKafkaJobs(ctx, producer, jobs, 2*time.Second)

	// 验证结果
	assert.Equal(t, 2, len(ok), "应该成功发送 2 条消息")
	assert.Equal(t, 0, len(failed), "不应该有失败的消息")

	// 等待消息发送完成
	producer.Flush(1000)

	// 验证消息接收
	receivedMessages := make(map[string]bool)
	for i := 0; i < 2; i++ {
		msg, err := consumer.ReadMessage(5 * time.Second)
		if err != nil {
			t.Fatalf("Failed to read message: %v", err)
		}
		receivedMessages[string(msg.Value)] = true
	}

	// 验证消息内容
	assert.True(t, receivedMessages["test message 1"], "未收到第一条消息")
	assert.True(t, receivedMessages["test message 2"], "未收到第二条消息")
}

// 测试超时场景
func TestSendKafkaJobs_RealKafka_Timeout(t *testing.T) {
	producer := createTestProducer(t, "test-producer-timeout")
	defer func() {
		// 确保所有消息都发送完成
		producer.Flush(1000)
		producer.Close()
	}()

	// 测试数据
	jobs := []*KafkaJob{
		{
			Topic: testTopic,
			Value: []byte("test message"),
		},
	}

	// 创建带超时的上下文，设置很短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// 发送消息，设置更短的超时时间
	ok, failed := SendKafkaJobs(ctx, producer, jobs, 5*time.Millisecond)

	// 验证结果
	assert.Equal(t, 0, len(ok), "由于超时，不应该有成功的消息")
	assert.Equal(t, 1, len(failed), "应该有 1 条失败的消息")
}

// 测试并发发送
func TestSendKafkaJobs_RealKafka_Concurrent(t *testing.T) {
	producer := createTestProducer(t, "test-producer-concurrent")
	defer producer.Close()

	// 创建 10 条测试消息
	jobs := make([]*KafkaJob, 10)
	for i := 0; i < 10; i++ {
		jobs[i] = &KafkaJob{
			Topic: testTopic,
			Value: []byte("test message " + string(rune('0'+i))),
		}
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 发送消息
	ok, failed := SendKafkaJobs(ctx, producer, jobs, 2*time.Second)

	// 验证结果
	assert.Equal(t, 10, len(ok), "应该成功发送 10 条消息")
	assert.Equal(t, 0, len(failed), "不应该有失败的消息")

	// 等待消息发送完成
	producer.Flush(1000)
}

// 测试空消息列表
func TestSendKafkaJobs_RealKafka_Empty(t *testing.T) {
	producer := createTestProducer(t, "test-producer-empty")
	defer producer.Close()

	// 空消息列表
	jobs := []*KafkaJob{}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 发送消息
	ok, failed := SendKafkaJobs(ctx, producer, jobs, 2*time.Second)

	// 验证结果
	assert.Equal(t, 0, len(ok), "空消息列表应该返回空成功列表")
	assert.Equal(t, 0, len(failed), "空消息列表应该返回空失败列表")

	// 等待消息发送完成
	producer.Flush(1000)
}

// 测试大消息
func TestSendKafkaJobs_RealKafka_LargeMessage(t *testing.T) {
	producer := createTestProducer(t, "test-producer-large")
	defer producer.Close()

	// 创建 900KB 的消息
	largeMessage := make([]byte, 900*1024)
	for i := range largeMessage {
		largeMessage[i] = byte(i % 256)
	}

	// 测试数据
	jobs := []*KafkaJob{
		{
			Topic: testTopic,
			Value: largeMessage,
		},
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 发送消息
	ok, failed := SendKafkaJobs(ctx, producer, jobs, 2*time.Second)

	// 验证结果
	assert.Equal(t, 1, len(ok), "应该成功发送 1 条大消息")
	assert.Equal(t, 0, len(failed), "不应该有失败的消息")

	// 等待消息发送完成
	producer.Flush(1000)
}

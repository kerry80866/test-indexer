package mq

import (
	"dex-indexer-sol/internal/config"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewKafkaProducer 创建 Kafka 生产者
func NewKafkaProducer(cfg config.KafkaProducerConfig) (*kafka.Producer, error) {
	conf := &kafka.ConfigMap{
		// 基本配置
		"bootstrap.servers":  cfg.Brokers,
		"client.id":          "solana-grpc-indexer",
		"acks":               "all", // 所有副本确认（可靠性）
		"enable.idempotence": false, // 幂等性（防止重复）

		// 超时与重试
		"delivery.timeout.ms":      30000, // 投递 + ack 最大等待时间
		"message.send.max.retries": 3,     // 最大重试次数
		"retry.backoff.ms":         100,   // 重试间隔

		// 批处理性能优化（吞吐优先）
		"batch.size": cfg.BatchSize, // 批处理大小
		"linger.ms":  cfg.LingerMs,  // 批处理延迟（聚合更多消息）

		// 压缩
		"compression.type": "none", // 可选：snappy / gzip / lz4, 不开

		// 队列控制
		"queue.buffering.max.messages": 100000, // 队列最大消息数
		"queue.buffering.max.ms":       100,    // 队列最大等待时间
		"queue.buffering.max.kbytes":   1024,   // 队列最大大小（KB）

		// 调试
		"debug": "all", // 开启所有调试信息

		// 连接配置
		"socket.timeout.ms":        30000, // Socket 超时
		"socket.keepalive.enable":  true,  // 保持连接
		"reconnect.backoff.ms":     1000,  // 重连间隔
		"reconnect.backoff.max.ms": 10000, // 最大重连间隔

		// 消息发送配置
		"message.max.bytes":     1000000, // 消息最大大小
		"request.timeout.ms":    30000,   // 请求超时
		"request.required.acks": -1,      // 等待所有副本确认
	}

	logx.Infof("creating Kafka producer with config: %+v", conf)

	producer, err := kafka.NewProducer(conf)
	if err != nil {
		logx.Errorf("failed to create Kafka producer: %v", err)
		return nil, err
	}

	// 启动事件处理 goroutine
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logx.Errorf("delivery failed: %v", ev.TopicPartition.Error)
				} else {
					logx.Infof("delivered message to %v", ev.TopicPartition)
				}
			case kafka.Error:
				logx.Errorf("producer error: %v", ev)
			default:
				logx.Infof("producer event: %v", ev)
			}
		}
	}()

	logx.Infof("Kafka producer created successfully")
	return producer, nil
}

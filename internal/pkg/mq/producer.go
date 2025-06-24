package mq

import (
	"context"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/pkg/utils"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	defaultBatchSize = 32 * 1024
	defaultLingerMs  = 5
)

type KafkaProducerOption struct {
	Brokers   string // Kafka broker 地址，多个用英文逗号分隔（如 "localhost:9092,localhost:9093"）
	BatchSize int    // 批处理大小（单位字节），如 32768 = 32KB
	LingerMs  int    // 批处理最大延迟（毫秒），建议 5~20ms 之间

	Topics []struct {
		Topic      string // topic名称
		Partitions int    // 分区数
	}
}

// NewKafkaProducer 创建 Kafka 生产者
func NewKafkaProducer(cfg KafkaProducerOption) (*kafka.Producer, error) {
	// 创建管理员客户端来管理 topic
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Brokers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	// 检查 topic 是否存在
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	meta, err := adminClient.GetMetadata(nil, true, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	brokerCount := len(meta.Brokers)

	// replicationFactor 是 Kafka 主题（Topic）中每个分区（Partition）副本的数量
	replicationFactor := 1
	if brokerCount > 1 {
		replicationFactor = 2
	}
	logger.Infof("[mq] Kafka broker count = %d, using replication factor = %d", brokerCount, replicationFactor)

	// 检查需要创建的 topic
	var topicsToCreate []kafka.TopicSpecification
	existingTopics := make(map[string]bool)
	for _, topic := range meta.Topics {
		existingTopics[topic.Topic] = true
	}

	// 如果 topic 不存在，则添加 topic 到创建列表
	for _, topic := range cfg.Topics {
		if !existingTopics[topic.Topic] {
			topicsToCreate = append(topicsToCreate, kafka.TopicSpecification{
				Topic:             topic.Topic,
				NumPartitions:     topic.Partitions,
				ReplicationFactor: replicationFactor,
			})
		}
	}

	// 如果有需要创建的 topic，则创建
	if len(topicsToCreate) > 0 {
		results, err := adminClient.CreateTopics(ctx, topicsToCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create topics: %w", err)
		}

		// 检查创建结果
		for _, result := range results {
			if result.Error.Code() != kafka.ErrNoError {
				return nil, fmt.Errorf("failed to create topic %s: %w", result.Topic, result.Error)
			}
		}
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	lingerMs := cfg.LingerMs
	if lingerMs < 0 {
		lingerMs = defaultLingerMs
	}

	localIP, _ := utils.GetLocalIP()
	if localIP == "" {
		localIP = "unknown"
	}

	// 创建生产者
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		// 基础连接
		"bootstrap.servers": cfg.Brokers,
		"client.id":         fmt.Sprintf("solana-grpc-indexer-%s", localIP),

		// PLAINTEXT: 不加密(明文传输), 不认证
		// SSL: 只加密，不认证
		// SASL_PLAINTEXT: 只认证，不加密
		// SASL_SSL: 加密 + 认证（
		//"security.protocol":  "SASL_SSL", // 生成环境建议: SASL_SSL

		// 1. PLAIN: 明文传输;
		// 2. SCRAM-SHA-256: 用户名 + 密码 + 哈希认证;
		// 3. SCRAM-SHA-512: 用户名 + 密码 + 哈希认证(更强);
		// 4. GSSAPI: Kerberos 身份认证;
		// 5. OAUTHBEARER: OAuth 令牌认证
		//"sasl.mechanisms":    "SCRAM-SHA-256",

		//"sasl.username":      "user",
		//"sasl.password":      "password",
		//"ssl.ca.location":    "/etc/ssl/certs/ca-certificates.crt",
		//"sasl.oauthbearer.token.endpoint.url": "https://your-auth.com/oauth2/token", // 可选

		// 可靠性保障
		"acks":                                  "all", // 必须
		"enable.idempotence":                    true,  // 幂等开启
		"max.in.flight.requests.per.connection": 5,     // 幂等场景下最大值为 5

		// 超时与重试
		"delivery.timeout.ms": 30000,
		"request.timeout.ms":  30000,
		"retries":             5,   // 重试次数必须 > 0
		"retry.backoff.ms":    100, // 重试间隔

		// 性能优化
		"batch.size":       batchSize,
		"linger.ms":        lingerMs,
		"compression.type": "none",

		// 消息大小
		"message.max.bytes": 2 * 1024 * 1024, // 2MB
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return producer, nil
}

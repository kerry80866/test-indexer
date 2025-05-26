package mq

import (
	"dex-indexer-sol/internal/config"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultBatchSize = 32 * 1024
	defaultLingerMs  = 5
)

// NewKafkaProducer 创建 Kafka 生产者
func NewKafkaProducer(cfg config.KafkaProducerConfig) (*kafka.Producer, error) {
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	lingerMs := cfg.LingerMs
	if lingerMs < 0 {
		lingerMs = defaultLingerMs
	}

	conf := &kafka.ConfigMap{
		// 基础连接
		"bootstrap.servers": cfg.Brokers,
		"client.id":         "solana-grpc-indexer",

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
		"acks":               "all",
		"enable.idempotence": false,

		// 超时与重试
		"delivery.timeout.ms":      30000,
		"request.timeout.ms":       30000,
		"message.send.max.retries": 3,
		"retry.backoff.ms":         100,

		// 性能优化
		"batch.size":       batchSize,
		"linger.ms":        lingerMs,
		"compression.type": "none",

		// 消息大小
		"message.max.bytes": 2 * 1024 * 1024, // 2MB
	}

	logx.Infof("creating Kafka producer with config: batchSize=%d, lingerMs=%d, brokers=%s", batchSize, lingerMs, cfg.Brokers)

	producer, err := kafka.NewProducer(conf)
	if err != nil {
		logx.Errorf("failed to create Kafka producer: %v", err)
		return nil, err
	}
	return producer, nil
}

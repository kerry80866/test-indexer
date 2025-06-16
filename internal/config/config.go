package config

import (
	"dex-indexer-sol/pkg/logger"
	"dex-indexer-sol/pkg/mq"
)

type LogConfig struct {
	Format   string `yaml:"format"`   // 日志格式，支持 "console" 或 "json"
	LogDir   string `yaml:"log_dir"`  // 日志目录（可为相对路径或绝对路径）
	Level    string `yaml:"level"`    // 日志级别：debug / info / warn / error
	Compress bool   `yaml:"compress"` // 是否压缩旧日志文件
}

func (c *LogConfig) ToLogOption() logger.LogOption {
	return logger.LogOption{
		Format:   c.Format,
		LogDir:   c.LogDir,
		Level:    c.Level,
		Compress: c.Compress,
	}
}

// PriceServiceConfig 表示价格服务配置
type PriceServiceConfig struct {
	Endpoint      string  `yaml:"endpoint"`        // 价格服务地址，例如 http://price.service.local
	SyncIntervalS int     `yaml:"sync_interval_s"` // 同步价格的时间间隔（秒）
	WSolPrice     float64 `yaml:"wsol_price"`      // 初始 WSOL 价格配置
}

// KafkaProducerConfig 表示 Kafka 生产者相关配置
type KafkaProducerConfig struct {
	Brokers   string `yaml:"brokers"`    // Kafka broker 地址，多个用英文逗号分隔
	BatchSize int    `yaml:"batch_size"` // 批处理大小（单位字节）
	LingerMs  int    `yaml:"linger_ms"`  // 批处理最大延迟（毫秒）

	Topics struct {
		Balance string `yaml:"balance"` // 余额变更事件的 Kafka topic
		Event   string `yaml:"event"`   // 综合事件的 Kafka topic
	} `yaml:"topics"`

	Partitions struct {
		Balance int `yaml:"balance"` // balance topic 的分区数
		Event   int `yaml:"event"`   // event topic 的分区数
	} `yaml:"partitions"`
}

func (c *KafkaProducerConfig) ToKafkaOption() mq.KafkaProducerOption {
	return mq.KafkaProducerOption{
		Brokers:   c.Brokers,
		BatchSize: c.BatchSize,
		LingerMs:  c.LingerMs,
		Topics: []struct {
			Topic      string
			Partitions int
		}{
			{Topic: c.Topics.Balance, Partitions: c.Partitions.Balance},
			{Topic: c.Topics.Event, Partitions: c.Partitions.Event},
		},
	}
}

// TimeConfig 表示各种超时配置（单位：毫秒）
type TimeConfig struct {
	SlotDispatchTimeoutMs int `yaml:"slot_dispatch_timeout_ms"` // 每个 slot 的处理最大耗时（Kafka + Redis + DB）
	EventSendTimeoutMs    int `yaml:"event_send_timeout_ms"`    // 单条事件发送到 Kafka 并等待 ack 的超时时间
}

// GrpcConfig 是主配置结构体，用于驱动索引器服务
type GrpcConfig struct {
	LogConf           LogConfig           `yaml:"logger"`         // 日志配置
	PriceServiceConf  PriceServiceConfig  `yaml:"price_service"`  // 价格服务配置
	KafkaProducerConf KafkaProducerConfig `yaml:"kafka_producer"` // Kafka 生产者配置
	TimeConf          TimeConfig          `yaml:"time_conf"`      // 时间相关配置

	RedisAddr    string `yaml:"redis_addr"`   // Redis 地址
	PostgresDSN  string `yaml:"postgres_dsn"` // PostgreSQL 数据源
	ProgressConf struct {
		RecentThresholdSec int `yaml:"recent_threshold_sec"` // 判定为“近期 block”的时间阈值（秒）
	} `yaml:"progress"` // 表示索引器中的进度管理配置

	// gRPC 客户端连接相关配置
	Grpc struct {
		Endpoint string `yaml:"endpoint"` // gRPC 服务端地址
		XToken   string `yaml:"x_token"`  // x-token 认证

		// 应用级逻辑心跳（ping）配置
		StreamPingIntervalSec int `yaml:"stream_ping_interval_sec"` // 应用层 ping 心跳间隔（秒）

		// gRPC Keepalive 底层连接检测配置
		KeepalivePingIntervalSec int `yaml:"keepalive_ping_interval_sec"` // 底层 keepalive 间隔（秒）
		KeepalivePingTimeoutSec  int `yaml:"keepalive_ping_timeout_sec"`  // 底层 keepalive 超时（秒）

		// gRPC 窗口大小调优（用于大数据流推送）
		InitialWindowSize     int `yaml:"initial_window_size"`      // 单流窗口大小（字节）
		InitialConnWindowSize int `yaml:"initial_conn_window_size"` // 整体连接窗口大小（字节）

		// 消息体大小限制
		MaxCallSendMsgSize int `yaml:"max_call_send_msg_size"` // 单条消息最大发送字节数
		MaxCallRecvMsgSize int `yaml:"max_call_recv_msg_size"` // 单条消息最大接收字节数

		// 超时与重连策略
		ReconnectIntervalSec int `yaml:"reconnect_interval_sec"` // 重连最小间隔（秒）
		ConnectTimeoutSec    int `yaml:"connect_timeout_sec"`    // 连接建立超时（秒）
		SendTimeoutSec       int `yaml:"send_timeout_sec"`       // 发送超时（秒）
		RecvTimeoutSec       int `yaml:"recv_timeout_sec"`       // 接收超时（秒）
		MaxLatencyWarnMs     int `yaml:"max_latency_warn_ms"`    // 延迟告警阈值（毫秒）
		MaxLatencyDropMs     int `yaml:"max_latency_drop_ms"`    // 延迟断连阈值（毫秒）
	} `yaml:"grpc"`
}

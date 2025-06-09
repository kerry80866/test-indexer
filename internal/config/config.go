package config

type LogConfig struct {
	Format   string `json:",default=console"` // 日志格式，console/json
	LogDir   string `json:",default=logs"`    // 日志目录
	Level    string `json:",default=info"`    // 日志级别 debug/info/warn/error
	Compress bool   `json:",default=false"`   // 是否压缩旧日志
}

// PriceServiceConfig 表示价格服务配置
type PriceServiceConfig struct {
	Endpoint      string // 价格服务地址，例如 http://price.service.local
	SyncIntervalS int    // 同步价格的时间间隔（秒）
	WSolPrice     float64
}

// KafkaProducerConfig 表示 Kafka 生产者相关配置
type KafkaProducerConfig struct {
	Brokers   string // Kafka broker 地址，多个用英文逗号分隔（如 "localhost:9092,localhost:9093"）
	BatchSize int    // 批处理大小（单位字节），如 32768 = 32KB
	LingerMs  int    // 批处理最大延迟（毫秒），建议 5~20ms 之间

	Topics struct {
		Balance string // 余额变更事件的 Kafka topic
		Event   string // 综合事件（如交易、流动性等）的 Kafka topic
	}

	Partitions struct {
		Balance int // 余额变更 topic 的分区数（如 4）
		Event   int // 综合事件 topic 的分区数（如 3）
	}
}

// TimeConfig 表示各种超时配置（单位：毫秒）
type TimeConfig struct {
	SlotDispatchTimeoutMs int `json:",default=2000"` // 每个 slot 的处理最大耗时（Kafka + Redis + DB）
	EventSendTimeoutMs    int `json:",default=1000"` // 单条事件发送到 Kafka 并等待 ack 的超时时间
}

// GrpcConfig 是主配置结构体，用于驱动索引器服务
type GrpcConfig struct {
	// 日志配置
	LogConf LogConfig

	// Price Service 价格服务配置（可选）
	PriceServiceConf PriceServiceConfig

	// Kafka 生产者配置
	KafkaProducerConf KafkaProducerConfig

	// 时间相关配置
	TimeConf TimeConfig

	// Redis 地址，用于 RedisProgressStore，例如 "127.0.0.1:6379"
	RedisAddr string

	// PostgreSQL 数据源，例如 "postgres://user:pass@host:5432/dbname?sslmode=disable"
	PostgresDSN string

	// ProgressConf 表示索引器中的进度管理配置
	ProgressConf struct {
		RecentThresholdSec int // 判定为“近期 block”的时间阈值（秒），默认建议为 60
	}

	// gRPC 客户端连接相关配置
	Grpc struct {
		Endpoint string // gRPC 服务端地址（例如 "localhost:50051"）
		XToken   string // 用于认证的 x-token（支持 JWT / 固定 Token）

		// 应用级逻辑心跳（ping）配置
		StreamPingIntervalSec int // 上层心跳 ping 间隔（秒）

		// gRPC Keepalive 底层连接检测配置
		KeepalivePingIntervalSec int // 底层 keepalive ping 间隔（秒）
		KeepalivePingTimeoutSec  int // 底层 keepalive 超时时间（秒）

		// gRPC 窗口大小调优（用于大数据流推送）
		InitialWindowSize     int // 单流初始窗口大小（字节），如 1GB = 1073741824
		InitialConnWindowSize int // 整个连接初始窗口大小（字节）

		// 消息体大小限制
		MaxCallSendMsgSize int // 发送单条消息最大值（字节），如 32MB = 33554432
		MaxCallRecvMsgSize int // 接收单条消息最大值（字节），如 128MB = 134217728

		// 超时与重连策略
		ReconnectIntervalSec int `json:",default=2"`    // 每次重连之间的最小间隔（秒）
		ConnectTimeoutSec    int `json:",default=12"`   // gRPC 连接建立超时（秒）
		SendTimeoutSec       int `json:",default=3"`    // gRPC Send 超时时间（秒）
		RecvTimeoutSec       int `json:",default=5"`    // gRPC Recv 超时时间（秒）
		MaxLatencyWarnMs     int `json:",default=3000"` // 区块延迟超 3 秒打 warning
		MaxLatencyDropMs     int `json:",default=5000"` // 区块延迟超 5 秒断流重连
	}
}

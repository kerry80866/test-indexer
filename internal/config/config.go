package config

// 价格服务配置
type PriceServiceConfig struct {
	Endpoint      string // 价格服务地址
	SyncIntervalS int    // 同步间隔(秒)
}

// Kafka 生产者配置
type KafkaProducerConfig struct {
	Brokers       string // Kafka 服务器地址，多个地址用逗号分隔
	BatchSize     int    // 批处理大小（字节）
	LingerMs      int    // 批处理延迟（毫秒）
	NumPartitions int    // 分区数量
	Topics        struct {
		Balance string // 余额变更事件主题
		Event   string // 普通事件主题
	}
}

// GRPC服务配置
type GrpcConfig struct {
	// Price Service 价格服务配置（可选）
	PriceServiceConf PriceServiceConfig

	// Kafka 生产者配置
	KafkaProducerConf KafkaProducerConfig

	// Redis 地址（用于 RedisProgressStore）
	RedisAddr string // eg: "127.0.0.1:6379"

	// PostgreSQL 数据源（用于 DBProgressStore）
	PostgresDSN string // eg: "postgres://user:pass@host:5432/dbname?sslmode=disable"

	// 进度管理配置
	ProgressConf struct {
		RecentThresholdSec int `yaml:"recent_threshold_sec"` // 判定“是否为近期 block”的时间阈值（单位秒），建议默认 60
	}

	// gRPC 客户端连接配置
	Grpc struct {
		Endpoint string // gRPC 服务端地址（含端口）
		XToken   string // 鉴权用的 x-token（如需要 JWT 可替换）

		// 应用级 ping 配置（用于逻辑心跳检测）
		StreamPingIntervalSec int // 应用逻辑层 ping 间隔（秒）

		// gRPC Keepalive 低层连接管理配置
		KeepalivePingIntervalSec int // gRPC 底层 ping 间隔（秒）
		KeepalivePingTimeoutSec  int // gRPC 底层 ping 超时时间（秒）

		// gRPC 窗口大小调优
		InitialWindowSize     int // 每个流的初始窗口大小（字节），如 1GB = 1073741824
		InitialConnWindowSize int // 整个连接的窗口大小（字节）

		// 消息体大小限制
		MaxCallSendMsgSize int // 单条消息最大发送大小（字节），如 32MB = 33554432
		MaxCallRecvMsgSize int // 单条消息最大接收大小（字节），如 128MB = 134217728

		// 重连 & 超时策略
		ReconnectIntervalSec int // gRPC 自动重连基础间隔（秒）
		RecvTimeoutSec       int // gRPC 接收超时时间（秒）
		ConnectTimeoutSec    int // gRPC 首次连接超时时间（秒）
		SendTimeoutSec       int // gRPC 发送消息超时时间（秒）
	}
}

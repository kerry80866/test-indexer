package config

// 价格服务配置
type PriceServiceConfig struct {
	Endpoint      string // 价格服务地址
	SyncIntervalS int    // 同步间隔(秒)
}

// GRPC服务配置
type GrpcConfig struct {
	// GRPC基础配置
	PriceServiceConf PriceServiceConfig

	// GRPC特有配置
	Grpc struct {
		Endpoint                 string // gRPC 服务端地址
		XToken                   string // 认证用的 x-token
		StreamPingIntervalSec    int    // Stream心跳包发送间隔（秒）
		KeepalivePingIntervalSec int    // gRPC底层keepalive间隔（秒）
		KeepalivePingTimeoutSec  int    // gRPC底层keepalive超时（秒）
		InitialWindowSize        int    // 单个流窗口大小（字节）
		InitialConnWindowSize    int    // 整个连接窗口大小（字节）
		MaxCallSendMsgSize       int    // 单条消息最大发送字节数
		MaxCallRecvMsgSize       int    // 单条消息最大接收字节数
		ReconnectIntervalSec     int    // 重连基础间隔（秒）
		RecvTimeoutSec           int    // Recv 超时时间（秒）
		ConnectTimeoutSec        int    // gRPC连接超时时间（秒）
		SendTimeoutSec           int    // gRPC发送超时时间（秒）
	}
}

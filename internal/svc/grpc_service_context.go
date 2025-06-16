package svc

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/dex-indexer-sol/internal/cache"
	"github.com/dex-indexer-sol/internal/config"
	"github.com/dex-indexer-sol/internal/logic/progress"
	"github.com/dex-indexer-sol/pkg/logger"
	"github.com/dex-indexer-sol/pkg/mq"
)

// GrpcServiceContext 包含GRPC服务资源
type GrpcServiceContext struct {
	Config          config.GrpcConfig
	PriceCache      *cache.PriceCache
	Producer        *kafka.Producer
	ProgressManager *progress.ProgressManager
}

// NewGrpcServiceContext 创建一个新的 GRPC 服务上下文
func NewGrpcServiceContext(c config.GrpcConfig) (*GrpcServiceContext, error) {
	// 1. 初始化 Kafka 生产者
	producer, err := mq.NewKafkaProducer(c.KafkaProducerConf.ToKafkaOption())
	if err != nil {
		logger.Errorf("Kafka producer 初始化失败: %v", err)
		return nil, err
	}

	//// 2. 初始化 Redis 客户端（用于 slot 状态缓存）
	//rdb := redis.NewClient(&redis.Options{
	//	Addr: c.RedisAddr, // eg: "127.0.0.1:6379"
	//	// 可按需添加密码/数据库等参数
	//})
	//
	//// 3. 初始化 PostgreSQL 数据库连接（用于 slot 落库）
	//db, err := sql.Open("postgres", c.PostgresDSN)
	//if err != nil {
	//	logger.Errorf("PostgreSQL 连接失败: %v", err)
	//	return nil, err
	//}

	// 4. 判定"近期 block"的时间阈值（默认 60 秒）
	threshold := c.ProgressConf.RecentThresholdSec
	if threshold <= 0 {
		threshold = 60
	}

	// 5. 初始化进度管理器（Redis + DB + 缓冲）
	//redisStore := progress.NewRedisProgressStore(rdb)
	//dbStore := progress.NewDBProgressStore(db)
	//pm := progress.NewProgressManager(nil, nil, threshold)

	// 6. 构造上下文
	ctx := &GrpcServiceContext{
		Config:          c,
		PriceCache:      cache.NewPriceCache(),
		Producer:        producer,
		ProgressManager: nil,
	}

	logger.Infof("GRPC 服务上下文初始化完成")
	return ctx, nil
}

// Close 关闭服务上下文中的资源
func (ctx *GrpcServiceContext) Close() {
	if ctx.Producer != nil {
		ctx.Producer.Close()
	}
}

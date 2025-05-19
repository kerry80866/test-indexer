package svc

import (
	"dex-indexer-sol/internal/cache"
	"dex-indexer-sol/internal/config"
	"log"
)

// GrpcServiceContext 包含GRPC服务资源
type GrpcServiceContext struct {
	Config     config.GrpcConfig
	PriceCache *cache.PriceCache
}

// NewGrpcServiceContext 创建一个新的GRPC服务上下文
func NewGrpcServiceContext(c config.GrpcConfig) *GrpcServiceContext {
	ctx := &GrpcServiceContext{
		Config:     c,
		PriceCache: cache.NewPriceCache(),
	}

	log.Println("GRPC服务上下文初始化完成")
	return ctx
}

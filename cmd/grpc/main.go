package main

import (
	"dex-indexer-sol/internal/cache"
	"dex-indexer-sol/internal/config"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/eventparser"
	"dex-indexer-sol/internal/logic/grpc"
	"dex-indexer-sol/internal/svc"
	"flag"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	zerosvc "github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/grpc.yaml", "the config file")

func main() {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("panic: %+v\nstack: %s", r, debug.Stack())
		}
	}()

	flag.Parse()

	var c config.GrpcConfig
	conf.MustLoad(*configFile, &c)

	serviceContext := svc.NewGrpcServiceContext(c)

	// 初始化事件解析器模块：注册各协议的指令解析handler
	eventparser.Init()

	// 初始化价格同步服务
	//priceSyncService := service.NewPriceSyncService(&c.PriceServiceConf, serviceContext.PriceCache)
	serviceContext.PriceCache.UpdateFrom(map[string][]cache.TokenPricePoint{
		consts.WSOLMintStr: {
			{PriceUsd: 180.0, Timestamp: time.Now().Unix()},
		},
		consts.USDCMintStr: {
			{PriceUsd: 1.0, Timestamp: time.Now().Unix()},
		},
		consts.USDTMintStr: {
			{PriceUsd: 1.0, Timestamp: time.Now().Unix()},
		},
	})

	sg := zerosvc.NewServiceGroup()
	//sg.Add(priceSyncService)

	blockChan := make(chan *pb.SubscribeUpdateBlock, 200)
	defer close(blockChan)

	grpcService, err := grpc.NewGrpcStreamManager(serviceContext, blockChan)
	if err != nil {
		panic(err)
	}
	sg.Add(grpcService)

	blockProcessor := grpc.NewBlockProcessor(serviceContext, blockChan)
	sg.Add(blockProcessor)

	logx.Infof("Starting grpc stream service")

	// 启动服务
	sg.Start()

	// 等待退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logx.Info("Shutting down services...")
	sg.Stop()
}

package main

import (
	"dex-indexer-sol/internal/cache"
	"dex-indexer-sol/internal/config"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/eventparser"
	"dex-indexer-sol/internal/logic/grpc"
	"dex-indexer-sol/internal/svc"
	"dex-indexer-sol/pkg/logger"
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
	defer logger.Sync() // 保证日志一定会刷盘
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic: %+v\nstack: %s", r, debug.Stack())
		}
	}()

	flag.Parse()

	var c config.GrpcConfig
	conf.MustLoad(*configFile, &c)

	// === 初始化 zap logger 并接管 logx 输出 ===
	logger.InitLogger(c.LogConf.ToLogOption())
	logx.SetWriter(logger.ZapWriter{})

	serviceContext, err := svc.NewGrpcServiceContext(c)
	if err != nil {
		panic(err)
	}

	// 初始化事件解析器模块：注册各协议的指令解析handler
	eventparser.Init()

	// 初始化价格同步服务
	//priceSyncService, err := service.NewPriceSyncService(&c.PriceServiceConf, serviceContext.PriceCache)
	//if err != nil {
	//	panic(err)
	//}
	serviceContext.PriceCache.UpdateFrom(map[string][]cache.TokenPricePoint{
		consts.WSOLMintStr: {
			{PriceUsd: serviceContext.Config.PriceServiceConf.WSolPrice, Timestamp: time.Now().Unix()},
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

	logger.Info("Starting grpc stream service")

	// 启动服务
	sg.Start()

	// 等待退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Info("Shutting down services...")
	sg.Stop()
}

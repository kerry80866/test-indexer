package main

import (
	"dex-indexer-sol/internal/config"
	"dex-indexer-sol/internal/logic/eventparser"
	"dex-indexer-sol/internal/logic/grpc"
	"dex-indexer-sol/internal/pkg/configloader"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/pkg/monitor"
	"dex-indexer-sol/internal/service"
	"dex-indexer-sol/internal/svc"
	"flag"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"github.com/zeromicro/go-zero/core/logx"
	zerosvc "github.com/zeromicro/go-zero/core/service"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

var configFile = flag.String("f", "etc/grpc.yaml", "the config file")

func main() {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic: %+v\nstack: %s", r, debug.Stack())
		}
	}()
	defer logger.Sync()

	flag.Parse()

	var c config.GrpcConfig
	if err := configloader.LoadConfig(*configFile, &c); err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// === 初始化 zap logger 并接管 logx 输出 ===
	logger.InitLogger(c.LogConf.ToLogOption())
	logx.SetWriter(logger.ZapWriter{})

	serviceContext, err := svc.NewGrpcServiceContext(c)
	if err != nil {
		panic(err)
	}

	// 初始化事件解析器模块：注册各协议的指令解析handler
	eventparser.Init()

	sg := zerosvc.NewServiceGroup()

	// 初始化价格同步服务
	priceSyncService, err := service.NewRpcPriceSyncService(&c.PriceServiceConf, serviceContext.PriceCache)
	if err != nil {
		panic(err)
	}

	sg.Add(priceSyncService)

	blockChan := make(chan *pb.SubscribeUpdateBlock, 200)
	defer close(blockChan)

	grpcService, err := grpc.NewGrpcStreamManager(serviceContext, blockChan)
	if err != nil {
		panic(err)
	}
	sg.Add(grpcService)

	blockProcessor := grpc.NewBlockProcessor(serviceContext, blockChan)
	sg.Add(blockProcessor)

	if c.Monitor.Port > 0 {
		monitorServer := monitor.NewMonitorServer(c.Monitor.Port)
		sg.Add(monitorServer)
	}

	// 启动服务
	logger.Infof("索引器服务启动成功")
	sg.Start()

	// 等待退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down services...")
	sg.Stop()
}

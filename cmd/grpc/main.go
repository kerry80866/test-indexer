package main

import (
	"dex-indexer-sol/internal/config"
	"dex-indexer-sol/internal/logic/grpc"
	"dex-indexer-sol/internal/svc"
	"flag"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

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

	// 初始化价格同步服务
	//priceSyncService := service.NewPriceSyncService(&c.PriceServiceConf, serviceContext.PriceCache)

	sg := zerosvc.NewServiceGroup()
	//sg.Add(priceSyncService)

	blockChan := make(chan *pb.SubscribeUpdateBlock, 200)
	defer close(blockChan)

	grpcService, err := grpc.NewGrpcStreamManager(serviceContext, blockChan)
	if err != nil {
		panic(err)
	}
	sg.Add(grpcService)

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

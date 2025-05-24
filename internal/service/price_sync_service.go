package service

import (
	"context"
	"dex-indexer-sol/internal/config"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"dex-indexer-sol/pb"
	"fmt"
	"log"
	"time"

	"dex-indexer-sol/internal/cache"
	"dex-indexer-sol/internal/consts"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PriceSyncService struct {
	priceCache *cache.PriceCache
	client     pb.PriceServiceClient
	tokens     []string // 定时拉取价格的币种
	interval   time.Duration
	stopChan   chan struct{}
	conn       *grpc.ClientConn // 保存连接以便在停止服务时关闭
}

func NewPriceSyncService(config *config.PriceServiceConfig, priceCache *cache.PriceCache) *PriceSyncService {
	// 必须连接成功
	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	conn, err := grpc.DialContext(dialCtx, config.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	cancel() // 显式取消超时上下文
	if err != nil {
		log.Fatalf("连接价格服务失败: %v", err)
	}

	// 创建 gRPC 客户端
	client := pb.NewPriceServiceClient(conn)

	// 创建服务实例
	p := &PriceSyncService{
		priceCache: priceCache,
		client:     client,
		interval:   time.Duration(config.SyncIntervalS) * time.Second,
		tokens:     utils.USDQuoteMintStrs,
		stopChan:   make(chan struct{}),
		conn:       conn,
	}

	// 初始化同步价格数据
	const retryCount = 2
	var lastErr error

	pubkeys := types.PubkeysFromBase58(p.tokens)
	for i := 0; i <= retryCount; i++ {
		if err := p.update(); err != nil {
			log.Printf("第 %d 次 update() 失败（忽略）: %v", i+1, err)
			lastErr = err
		}

		if _, ok := priceCache.GetQuotePricesAt(pubkeys, 0); ok {
			log.Println("初始价格同步成功（缓存已就绪）")
			return p
		}

		log.Printf("第 %d 次尝试后仍无法获取完整价格", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("初始价格同步失败（%d 次尝试）: %v", retryCount+1, lastErr)
	return nil // 满足编译器，永不执行
}

func (ps *PriceSyncService) Start() {
	ticker := time.NewTicker(ps.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ps.update(); err != nil {
				log.Printf("周期性价格更新失败: %v", err)
			}
		case <-ps.stopChan:
			return
		}
	}
}

func (ps *PriceSyncService) Stop() {
	close(ps.stopChan)

	// 关闭gRPC连接
	if ps.conn != nil {
		ps.conn.Close()
	}
}

func (ps *PriceSyncService) update() error {
	resp, err := ps.fetchPriceHistory()
	if err != nil {
		return err
	}
	ps.priceCache.UpdateFrom(resp)
	return nil
}

func (ps *PriceSyncService) fetchPriceHistory() (map[string][]cache.TokenPricePoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	resp, err := ps.client.GetPriceHistory(ctx, &pb.GetPriceHistoryRequest{
		ChainId:        int32(consts.ChainIDSolana),
		TokenAddresses: ps.tokens,
		FromTimestamp:  time.Now().Add(-ps.interval).Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("请求价格历史数据失败: %w", err)
	}

	result := make(map[string][]cache.TokenPricePoint)
	for tokenAddr, history := range resp.Prices {
		var points []cache.TokenPricePoint
		for _, p := range history.Points {
			points = append(points, cache.TokenPricePoint{
				Timestamp: p.Timestamp,
				PriceUsd:  p.PriceUsd,
			})
		}
		result[tokenAddr] = points
	}

	return result, nil
}

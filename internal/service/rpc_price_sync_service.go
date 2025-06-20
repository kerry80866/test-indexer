package service

import (
	"context"
	"dex-indexer-sol/internal/cache"
	"dex-indexer-sol/internal/config"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/pkg/types"
	"dex-indexer-sol/internal/tools"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/blocto/solana-go-sdk/client"
	"math"
	"runtime/debug"
	"time"
)

type RpcPriceSyncService struct {
	priceCache *cache.PriceCache
	interval   time.Duration
	stopChan   chan struct{}
	client     *client.Client // Solana RPC客户端
	ctx        context.Context
	cancel     func(err error)
	accounts   []string
	tokens     []types.Pubkey
}

func NewRpcPriceSyncService(cfg *config.PriceServiceConfig, priceCache *cache.PriceCache) (*RpcPriceSyncService, error) {
	ctx, cancel := context.WithCancelCause(context.Background())
	s := &RpcPriceSyncService{
		priceCache: priceCache,
		interval:   time.Duration(cfg.SyncIntervalS) * time.Second,
		stopChan:   make(chan struct{}),
		accounts: []string{
			consts.PythSOLAccount,
			consts.PythUSDCAccount,
			consts.PythUSDTAccount,
		},
		tokens: []types.Pubkey{
			consts.WSOLMint,
			consts.USDCMint,
			consts.USDTMint,
		},
		client: client.NewClient(cfg.Endpoint),
		ctx:    ctx,
		cancel: cancel,
	}
	if s.client == nil {
		return nil, errors.New("rpc client init failed")
	}

	// 初始化
	const retryCount = 3
	for i := 0; i <= retryCount; i++ {
		if err := s.update(); err != nil {
			logger.Warnf("[RpcPriceSyncService] 第 %d 次 update() 失败: %v", i+1, err)
		} else {
			if _, ok := priceCache.GetQuotePricesAt(tools.USDQuoteMints, 0); ok {
				logger.Infof("[RpcPriceSyncService] 初始价格同步成功")
				return s, nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("[RpcPriceSyncService] 初始同步失败")
}

func (s *RpcPriceSyncService) Start() {
	s.scheduleNext()
	<-s.stopChan
}

func (s *RpcPriceSyncService) scheduleNext() {
	time.AfterFunc(s.interval, func() {
		if err := s.update(); err != nil {
			logger.Warnf("[RpcPriceSyncService] 周期性更新失败: %v", err)
		}
		// 如果没有被 Stop，就继续调度
		select {
		case <-s.ctx.Done():
			return
		default:
			s.scheduleNext()
		}
	})
}

func (s *RpcPriceSyncService) Stop() {
	s.cancel(errors.New("RpcPriceSyncService stop"))
	select {
	case <-s.stopChan:
		// 已关闭，无需重复关闭
	default:
		close(s.stopChan)
	}
}

func (s *RpcPriceSyncService) update() (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[RpcPriceSyncService] update panic: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("update panic: %v", r)
		}
	}()

	resp, err := s.fetchPriceAccounts()
	if err != nil {
		return err
	}
	s.priceCache.Insert(resp)
	return nil
}

func (s *RpcPriceSyncService) fetchPriceAccounts() (map[types.Pubkey]cache.TokenPricePoint, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	infos, err := s.client.GetMultipleAccounts(ctx, s.accounts)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("GetMultipleAccounts failed: %w", err)
	}
	logger.Infof("[RpcPriceSyncService] GetMultipleAccounts 成功, 账户数: %d, 耗时: %v", len(s.accounts), duration)

	if len(infos) != len(s.accounts) {
		return nil, fmt.Errorf("返回账户数与请求不一致: got=%d want=%d", len(infos), len(s.accounts))
	}

	result := make(map[types.Pubkey]cache.TokenPricePoint)

	for i, info := range infos {
		account := s.accounts[i]
		if len(info.Data) == 0 {
			logger.Warnf("[RpcPriceSyncService] 账户数据为空: index=%d token=%s account=%s", i, s.tokens[i], account)
			continue
		}

		token := s.tokens[i]
		priceInfo, err := parsePythPriceAccount(token, info.Data)
		if err != nil {
			logger.Warnf("[RpcPriceSyncService] 解析失败: token=%s account=%s err=%v", s.tokens[i], account, err)
			continue
		}
		logger.Infof("[RpcPriceSyncService] %s: %.6f (ts=%s)", token, priceInfo.PriceUsd, time.Unix(priceInfo.Timestamp, 0).Format("2006-01-02 15:04:05"))

		// 反查 token mint（如 USDC）对应的 account 地址
		result[token] = *priceInfo
	}

	return result, nil
}

// 参考: https://github.com/pyth-network/pyth-client-js/blob/main/src/index.ts - parsePriceData
func parsePythPriceAccount(token types.Pubkey, data []byte) (*cache.TokenPricePoint, error) {
	if len(data) < 240 {
		return nil, errors.New("price account data too short")
	}

	exponent := int32(binary.LittleEndian.Uint32(data[20:24]))
	publishTimestamp := binary.LittleEndian.Uint64(data[96:104])

	// 取 aggregate 区块（偏移 208 起）
	agg := parsePriceInfo(data[208:240], int(exponent))
	if agg.Status != 1 {
		return nil, fmt.Errorf("price status not trading: token=%s", token)
	}
	if isConfidenceTooLow(token, agg.Price, agg.Confidence) {
		return nil, fmt.Errorf("confidence too low: token=%s, price=%.6f, conf=%.6f", token, agg.Price, agg.Confidence)
	}
	if time.Now().Unix()-int64(publishTimestamp) > 120 {
		return nil, fmt.Errorf("price too old: token=%s, ts=%d", token, publishTimestamp)
	}
	return &cache.TokenPricePoint{
		PriceUsd:  agg.Price,
		Timestamp: int64(publishTimestamp),
	}, nil
}

type PriceInfo struct {
	PriceComponent      int64
	ConfidenceComponent uint64
	Status              uint32
	CorporateAction     uint32
	PublishSlot         uint64
	Price               float64
	Confidence          float64
}

// 参考: https://github.com/pyth-network/pyth-client-js/blob/main/src/index.ts - parsePriceInfo
func parsePriceInfo(data []byte, exponent int) PriceInfo {
	// [0:8]   -> priceComponent (int64)
	// [8:16]  -> confidenceComponent (uint64)
	// [16:20] -> status (1 = trading)
	// [20:24] -> corporateAction (忽略)
	// [24:32] -> publishSlot
	priceComponent := int64(binary.LittleEndian.Uint64(data[0:8]))
	confidenceComponent := binary.LittleEndian.Uint64(data[8:16])
	status := binary.LittleEndian.Uint32(data[16:20])
	corporateAction := binary.LittleEndian.Uint32(data[20:24])
	publishSlot := binary.LittleEndian.Uint64(data[24:32])

	price := float64(priceComponent) * math.Pow10(exponent)
	confidence := float64(confidenceComponent) * math.Pow10(exponent)

	return PriceInfo{
		PriceComponent:      priceComponent,
		ConfidenceComponent: confidenceComponent,
		Status:              status,
		CorporateAction:     corporateAction,
		PublishSlot:         publishSlot,
		Price:               price,
		Confidence:          confidence,
	}
}

func isConfidenceTooLow(token types.Pubkey, price, conf float64) bool {
	switch token {
	case consts.USDTMint, consts.USDCMint:
		return conf > 0.005*price // 稳定币允许最大 0.5% 的置信误差
	case consts.WSOLMint, consts.NativeSOLMint:
		return conf > 0.02*price // SOL 允许最大 2% 的误差
	default:
		return conf > 0.05*price // 其他未知 token，允许最大 5%
	}
}

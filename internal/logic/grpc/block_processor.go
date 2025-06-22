package grpc

import (
	"bytes"
	"context"
	"dex-indexer-sol/internal/cache"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser"
	"dex-indexer-sol/internal/logic/jobbuilder"
	"dex-indexer-sol/internal/logic/progress"
	"dex-indexer-sol/internal/logic/txadapter"
	"dex-indexer-sol/internal/pkg/mq"
	"dex-indexer-sol/internal/pkg/types"
	"dex-indexer-sol/internal/pkg/utils"
	"dex-indexer-sol/internal/svc"
	"dex-indexer-sol/internal/tools"
	pb2 "dex-indexer-sol/pb"

	"errors"
	"sync/atomic"
	"time"

	"dex-indexer-sol/internal/pkg/logger"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

const (
	sourceGrpc      = 1
	maxSlotDispatch = 200
)

var workerCount = consts.CpuCount + 2

type BlockProcessor struct {
	sc        *svc.GrpcServiceContext
	blockChan chan *pb.SubscribeUpdateBlock // 接收 block 的 channel
	ctx       context.Context
	cancel    func(err error)

	activeSlotDispatch    int64 // 当前活跃的 slot dispatch goroutine 数（用于限流发事件 + 同步进度）
	lastBlockChanWarnTime int64
}

func NewBlockProcessor(sc *svc.GrpcServiceContext, blockChan chan *pb.SubscribeUpdateBlock) *BlockProcessor {
	ctx, cancel := context.WithCancelCause(context.Background())
	return &BlockProcessor{
		sc:        sc,
		blockChan: blockChan,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (p *BlockProcessor) Start() {
	for {
		select {
		case <-p.ctx.Done():
			return // 退出
		case block := <-p.blockChan:
			p.procBlock(block)
			if len(p.blockChan) > 10 {
				now := time.Now().Unix()
				if now-p.lastBlockChanWarnTime >= 10 {
					p.lastBlockChanWarnTime = now
					logger.Warnf("[BlockProcessor] block chan len: %v", len(p.blockChan))
				}
			}
		}
	}
}

func (p *BlockProcessor) Stop() {
	p.cancel(errors.New("service stop"))
}

func (p *BlockProcessor) procBlock(block *pb.SubscribeUpdateBlock) {
	startTime := time.Now()
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[BlockProcessor] procBlock panic: %v (slot: %d)", r, block.Slot)
		}
		logger.Infof("[BlockProcessor] 区块处理总耗时: %v, slot: %d", time.Since(startTime), block.Slot)
	}()

	// 1. 过滤合法交易
	filterStart := time.Now()
	validTxs := make([]*pb.SubscribeUpdateTransactionInfo, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		if IsValidGrpcTx(tx) {
			validTxs = append(validTxs, tx)
		}
	}
	logger.Infof("[BlockProcessor] 交易过滤耗时: %v, 总交易数: %d, 有效交易数: %d",
		time.Since(filterStart), len(block.Transactions), len(validTxs))

	// 2. 构造上下文
	ctxStart := time.Now()
	txCtx := p.buildTxContext(block)
	if txCtx == nil {
		logger.Errorf("[BlockProcessor] 构造 txCtx 失败, slot: %d", block.Slot)
		return
	}
	logger.Infof("[BlockProcessor] 上下文构造耗时: %v", time.Since(ctxStart))

	// 3. 并发解析所有交易，构造 ParsedTxResult 列表。
	parseStart := time.Now()
	results := utils.ParallelMap(
		validTxs,
		workerCount,
		func(workerID int) map[string]types.Pubkey {
			// 每个 goroutine 共享独立 owner cache map，提高解析效率。
			return make(map[string]types.Pubkey, len(validTxs)*4/consts.CpuCount)
		},
		func(ownerCache map[string]types.Pubkey, tx *pb.SubscribeUpdateTransactionInfo) core.ParsedTxResult {
			return p.parseTx(txCtx, ownerCache, tx)
		})
	logger.Infof("[BlockProcessor] 事件解析耗时: %v", time.Since(parseStart))

	// 4. 更新价格缓存，并补全 USD 估值
	usdStart := time.Now()
	p.updatePriceCacheFromEvents(results)                      // 更新 token 最新价格至 PriceCache
	quotePrices := p.loadQuotePricesFromCache(txCtx.BlockTime) // 从 PriceCache 读取 quote token 价格（SOL/USDC/USDT）
	if quotePrices == nil {
		logger.Errorf("[BlockProcessor] 获取 quotePrices 失败, slot: %d", block.Slot)
		return
	}
	fillUsdAmountForEvents(results, quotePrices) // 用 quotePrices 填充所有 TradeEvent 的 USD 金额
	logger.Infof("[BlockProcessor] 补全 USD 估值完成, 耗时: %v", time.Since(usdStart))

	// 5. 构建事件类 Kafka 任务
	eventStart := time.Now()
	eventJobs, eventCount, tradeCount, validTradeCount, transferCount := jobbuilder.BuildEventKafkaJobs(
		txCtx,
		quotePrices,
		sourceGrpc,
		p.sc.Config.KafkaProducerConf.Topics.Event,
		p.sc.Config.KafkaProducerConf.Partitions.Event,
		results,
	)
	eventDuration := time.Since(eventStart)
	logger.Infof("[BlockProcessor] Kafka事件：事件 %d 条（trade %d，有效trade %d，transfer %d）, 耗时 %s",
		eventCount, tradeCount, validTradeCount, transferCount, eventDuration)

	// 6. 构建余额类 Kafka 任务
	balanceStart := time.Now()
	balanceJobs, balanceCount := jobbuilder.BuildBalanceKafkaJobs(
		txCtx,
		quotePrices,
		sourceGrpc,
		p.sc.Config.KafkaProducerConf.Topics.Balance,
		p.sc.Config.KafkaProducerConf.Partitions.Balance,
		results,
	)
	balanceDuration := time.Since(balanceStart)
	logger.Infof("[BlockProcessor] Kafka余额事件：事件 %d 条, 耗时 %s", balanceCount, balanceDuration)

	// 7. 合并 Kafka 任务
	mqJobs := make([]*mq.KafkaJob, 0, len(eventJobs)+len(balanceJobs))
	mqJobs = append(mqJobs, eventJobs...)
	mqJobs = append(mqJobs, balanceJobs...)

	// 8. 分发任务（Kafka 推送 + 写进度）
	dispatchStart := time.Now()
	p.dispatchSlot(txCtx.Slot, txCtx.BlockTime, mqJobs)
	logger.Infof("[BlockProcessor] 任务分发耗时: %v", time.Since(dispatchStart))
}

func IsValidGrpcTx(tx *pb.SubscribeUpdateTransactionInfo) bool {
	if tx == nil || // - nil transaction info
		tx.Transaction == nil || // - missing Transaction field
		tx.Transaction.Message == nil || // - missing Message field in transaction
		len(tx.Transaction.Signatures) == 0 || // - missing transaction signature
		len(tx.Transaction.Signatures[0]) != 64 || // - invalid transaction signature length
		tx.IsVote || // - vote transaction skipped
		tx.Meta == nil || // - missing transaction meta data
		tx.Meta.Err != nil { // - transaction execution failed
		return false
	}
	return true
}

func (p *BlockProcessor) buildTxContext(block *pb.SubscribeUpdateBlock) *core.TxContext {
	// 尝试解析 blockHash，如果失败只打日志但继续执行
	blockHash, err := types.HashFromBase58(block.Blockhash)
	if err != nil {
		logger.Errorf("[BlockProcessor] BlockHash 无法解析，将使用零值：slot=%d, blockhash=%s, err=%v",
			block.Slot, block.Blockhash, err)
	}

	return &core.TxContext{
		BlockTime:  block.BlockTime.Timestamp,
		Slot:       block.Slot,
		BlockHash:  blockHash, // 若解析失败为零值
		ParentSlot: block.ParentSlot,
	}
}

func (p *BlockProcessor) parseTx(txCtx *core.TxContext, ownerCache map[string]types.Pubkey, tx *pb.SubscribeUpdateTransactionInfo) core.ParsedTxResult {
	adaptedTx, err := txadapter.AdaptGrpcTx(txCtx, ownerCache, tx)
	if err != nil {
		return core.ParsedTxResult{}
	}

	events, priceEvents := eventparser.ExtractEventsFromTx(adaptedTx)
	return core.ParsedTxResult{
		Balances:    adaptedTx.Balances,
		Events:      events,
		PriceEvents: priceEvents,
	}
}

// updatePriceCacheFromEvents 从事件中提取Token的价格，并写入 PriceCache。
func (p *BlockProcessor) updatePriceCacheFromEvents(results []core.ParsedTxResult) {
	latest := make(map[types.Pubkey]*core.PriceEvent)

	// 遍历所有交易结果，保留每个 token 的最新价格事件（取 PublishTime 最大）
	for _, result := range results {
		for _, e := range result.PriceEvents {
			old, ok := latest[e.TokenMint]
			if !ok || e.PublishTime > old.PublishTime {
				latest[e.TokenMint] = e
			}
		}
	}

	if len(latest) == 0 {
		return
	}

	// 构建写入缓存的数据结构（Token → PricePoint）
	points := make(map[types.Pubkey]cache.TokenPricePoint, len(latest))
	for token, ev := range latest {
		points[token] = cache.TokenPricePoint{
			Timestamp: ev.PublishTime,
			PriceUsd:  ev.PriceUsd,
		}
	}
	p.sc.PriceCache.Insert(points)
}

// loadQuotePricesFromCache 从 PriceCache 拉取 quote token 的价格（含 SOL/WSOL/USDC/USDT）
func (p *BlockProcessor) loadQuotePricesFromCache(blockTime int64) []*pb2.TokenPrice {
	type quoteDef struct {
		Mint     types.Pubkey
		Decimals uint32
	}

	// 定义查询顺序和映射信息
	defs := []quoteDef{
		{Mint: consts.WSOLMint, Decimals: tools.WSOLDecimals},
		{Mint: consts.USDCMint, Decimals: tools.USDCDecimals},
		{Mint: consts.USDTMint, Decimals: tools.USDTDecimals},
	}

	// 提取价格
	mints := make([]types.Pubkey, 0, len(defs))
	for _, def := range defs {
		mints = append(mints, def.Mint)
	}

	priceVals, ok := p.sc.PriceCache.GetQuotePricesAt(mints, blockTime)
	if !ok {
		return nil
	}

	result := make([]*pb2.TokenPrice, 0, len(defs)+1)
	// NativeSOL 用 WSOL 价格
	result = append(result, &pb2.TokenPrice{
		Token:    consts.NativeSOLMint[:],
		Decimals: tools.WSOLDecimals,
		Price:    priceVals[0],
	})
	// 其余 quote token
	for i, def := range defs {
		result = append(result, &pb2.TokenPrice{
			Token:    def.Mint[:],
			Decimals: def.Decimals,
			Price:    priceVals[i],
		})
	}
	return result
}

// fillUsdAmountForEvents 补全每个 TradeEvent 的 USD 金额与单价信息（AmountUsd / PriceUsd）
func fillUsdAmountForEvents(results []core.ParsedTxResult, quotePrices []*pb2.TokenPrice) {
	for _, result := range results {
		for _, e := range result.Events {
			if e.EventType != uint32(pb2.EventType_TRADE_BUY) &&
				e.EventType != uint32(pb2.EventType_TRADE_SELL) {
				continue
			}
			trade := e.Event.GetTrade()
			if trade == nil {
				continue
			}

			var quoteUsd float64
			for _, val := range quotePrices {
				if bytes.Equal(trade.QuoteToken, val.Token) {
					quoteUsd = val.Price
					break
				}
			}
			if quoteUsd == 0 {
				continue
			}

			baseAmount := float64(trade.TokenAmount) / utils.Pow10(trade.TokenDecimals)
			quoteAmount := float64(trade.QuoteTokenAmount) / utils.Pow10(trade.QuoteDecimals)

			trade.AmountUsd = quoteAmount * quoteUsd
			if baseAmount > 0 {
				trade.PriceUsd = trade.AmountUsd / baseAmount
			}
		}
	}
}

func (p *BlockProcessor) dispatchSlot(slotID uint64, blockTime int64, jobs []*mq.KafkaJob) {
	if p.sc.ProgressManager != nil {
		should, _ := p.sc.ProgressManager.ShouldProcessSlot(p.ctx, slotID, blockTime)
		if !should {
			return
		}
	}

	if atomic.AddInt64(&p.activeSlotDispatch, 1) > maxSlotDispatch {
		atomic.AddInt64(&p.activeSlotDispatch, -1)
		logger.Errorf("[BlockProcessor] slot %d 被丢弃：活跃 dispatch 数超过上限", slotID)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("[BlockProcessor] dispatchSlot panic: %v", r)
			}
			atomic.AddInt64(&p.activeSlotDispatch, -1)
		}()

		timeout := time.Duration(p.sc.Config.TimeConf.SlotDispatchTimeoutMs) * time.Millisecond
		ctx, cancel := context.WithTimeout(p.ctx, timeout)
		defer cancel()

		startTime := time.Now()
		// 1. 发送 Kafka
		sendTimeout := time.Duration(p.sc.Config.TimeConf.EventSendTimeoutMs) * time.Millisecond
		okJobs, failedJobs := mq.SendKafkaJobs(ctx, p.sc.Producer, jobs, sendTimeout)

		// 计算最大发送和确认时间
		var maxSendTime, maxAckTime, maxMarshalTime time.Duration
		successCount := len(okJobs)
		for _, job := range okJobs {
			if job.SendTime > maxSendTime {
				maxSendTime = job.SendTime
			}
			if job.AckTime > maxAckTime {
				maxAckTime = job.AckTime
			}
			if job.MarshalTime > maxMarshalTime {
				maxMarshalTime = job.MarshalTime
			}
		}

		if len(failedJobs) == 0 {
			// Kafka 发送成功，写入进度
			progressStartTime := time.Now()
			if p.sc.ProgressManager != nil {
				err := p.sc.ProgressManager.MarkSlotStatus(ctx, progress.SourceGrpc, slotID, blockTime, progress.SlotProcessed)
				if err != nil {
					logger.Errorf("[BlockProcessor] slot %d 写入进度失败: %v", slotID, err)
				}
			}
			progressTime := time.Since(progressStartTime)
			totalTime := time.Since(startTime)

			logger.Infof("[BlockProcessor] slot %d 处理完成 - 最大序列化耗时: %v, Kafka最大发送耗时: %v, 最大确认耗时: %v, 进度写入耗时: %v, 总耗时: %v（成功=%d）",
				slotID, maxMarshalTime, maxSendTime, maxAckTime, progressTime, totalTime, successCount)
		} else {
			// Kafka 有失败，不写进度
			totalTime := time.Since(startTime)
			logger.Errorf("[BlockProcessor] slot %d 处理失败 - 最大序列化耗时: %v, Kafka最大发送耗时: %v, 最大确认耗时: %v, 总耗时: %v（成功=%d 失败=%d）",
				slotID, maxMarshalTime, maxSendTime, maxAckTime, totalTime, successCount, len(failedJobs))
		}
	}()
}

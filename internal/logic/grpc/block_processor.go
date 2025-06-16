package grpc

import (
	"context"
	"errors"
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/internal/logic/dispatcher"
	"github.com/dex-indexer-sol/internal/logic/eventparser"
	"github.com/dex-indexer-sol/internal/logic/progress"
	"github.com/dex-indexer-sol/internal/logic/txadapter"
	"github.com/dex-indexer-sol/internal/svc"
	"github.com/dex-indexer-sol/internal/tools"
	"github.com/dex-indexer-sol/pkg/mq"
	"github.com/dex-indexer-sol/pkg/types"
	"github.com/dex-indexer-sol/pkg/utils"
	"sync/atomic"
	"time"

	"github.com/dex-indexer-sol/pkg/logger"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

const maxSlotDispatch = 200
const sourceGrpc = 1

type BlockProcessor struct {
	sc                 *svc.GrpcServiceContext
	blockChan          chan *pb.SubscribeUpdateBlock // 接收 block 的 channel
	activeSlotDispatch int64                         // 当前活跃的 slot dispatch goroutine 数（用于限流发事件 + 同步进度）
	ctx                context.Context
	cancel             func(err error)
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
				logger.Warnf("[BlockProcessor] block chan len:%v", len(p.blockChan))
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
		consts.CpuCount+2,
		func() map[string]types.Pubkey {
			// 每个 goroutine 共享独立 owner cache map，提高解析效率。
			return make(map[string]types.Pubkey, len(validTxs)*3/consts.CpuCount)
		},
		func(owners map[string]types.Pubkey, tx *pb.SubscribeUpdateTransactionInfo) core.ParsedTxResult {
			return p.parseTx(txCtx, owners, tx)
		})
	logger.Infof("[BlockProcessor] 事件解析耗时: %v", time.Since(parseStart))

	// 4. 构建事件类 Kafka 任务
	eventStart := time.Now()
	eventJobs, eventCount, tradeCount, validTradeCount, transferCount := dispatcher.BuildEventKafkaJobs(
		txCtx,
		sourceGrpc,
		p.sc.Config.KafkaProducerConf.Topics.Event,
		p.sc.Config.KafkaProducerConf.Partitions.Event,
		results,
	)
	eventDuration := time.Since(eventStart)
	logger.Infof("[BlockProcessor] Kafka事件：事件 %d 条（trade %d，有效trade %d，transfer %d）, 耗时 %s",
		eventCount, tradeCount, validTradeCount, transferCount, eventDuration)

	// 5. 构建余额类 Kafka 任务
	balanceStart := time.Now()
	balanceJobs, balanceCount := dispatcher.BuildBalanceKafkaJobs(
		txCtx,
		sourceGrpc,
		p.sc.Config.KafkaProducerConf.Topics.Balance,
		p.sc.Config.KafkaProducerConf.Partitions.Balance,
		results,
	)
	balanceDuration := time.Since(balanceStart)
	logger.Infof("[BlockProcessor] Kafka余额事件：事件 %d 条, 耗时 %s", balanceCount, balanceDuration)

	// 6. 合并 Kafka 任务
	mqJobs := make([]*mq.KafkaJob, 0, len(eventJobs)+len(balanceJobs))
	mqJobs = append(mqJobs, eventJobs...)
	mqJobs = append(mqJobs, balanceJobs...)

	// 7. 分发任务（Kafka 推送 + 写进度）
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

	// 从价格缓存中获取 quote token 价格，如果失败则跳过该 block
	blockTime := block.BlockTime.Timestamp
	prices, ok := p.sc.PriceCache.GetQuotePricesAt(tools.USDQuoteMints, blockTime)
	if !ok {
		logger.Errorf("[BlockProcessor] 获取 QuoteToken 价格失败，跳过该区块：slot=%d, blockTime=%d",
			block.Slot, blockTime)
		return nil
	}

	// 构建 quotesPrice 数组
	quotesPrice := make([]core.QuotePrice, 0, len(prices))
	for i, mint := range tools.USDQuoteMints {
		quotesPrice = append(quotesPrice, core.QuotePrice{
			Token:    mint,
			PriceUsd: prices[i],
		})
	}

	return &core.TxContext{
		BlockTime:   blockTime,
		Slot:        block.Slot,
		BlockHash:   blockHash, // 若解析失败为零值
		ParentSlot:  block.ParentSlot,
		QuotesPrice: quotesPrice,
	}
}

func (p *BlockProcessor) parseTx(txCtx *core.TxContext, owners map[string]types.Pubkey, tx *pb.SubscribeUpdateTransactionInfo) core.ParsedTxResult {
	adaptedTx, err := txadapter.AdaptGrpcTx(txCtx, owners, tx)
	if err != nil {
		return core.ParsedTxResult{}
	}

	events := eventparser.ExtractEventsFromTx(adaptedTx)
	return core.ParsedTxResult{
		Balances: adaptedTx.Balances,
		Events:   events,
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
		defer atomic.AddInt64(&p.activeSlotDispatch, -1)

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
			// ✅ Kafka 发送成功，写入进度
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
			// ❌ Kafka 有失败，不写进度
			totalTime := time.Since(startTime)
			logger.Errorf("[BlockProcessor] slot %d 处理失败 - 最大序列化耗时: %v, Kafka最大发送耗时: %v, 最大确认耗时: %v, 总耗时: %v（成功=%d 失败=%d）",
				slotID, maxMarshalTime, maxSendTime, maxAckTime, totalTime, successCount, len(failedJobs))
		}
	}()
}

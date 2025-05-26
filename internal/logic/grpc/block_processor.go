package grpc

import (
	"context"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/dispatcher"
	"dex-indexer-sol/internal/logic/eventparser"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/logic/progress"
	"dex-indexer-sol/internal/logic/txadapter"
	"dex-indexer-sol/internal/mq"
	"dex-indexer-sol/internal/svc"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"errors"
	"sync/atomic"
	"time"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"github.com/zeromicro/go-zero/core/logx"
)

const maxSlotDispatch = 200

type BlockProcessor struct {
	sc                 *svc.GrpcServiceContext
	blockChan          chan *pb.SubscribeUpdateBlock // 接收 block 的 channel
	activeSlotDispatch int64                         // 当前活跃的 slot dispatch goroutine 数（用于限流发事件 + 同步进度）
	ctx                context.Context
	cancel             func(err error)
	logx.Logger
}

func NewBlockProcessor(sc *svc.GrpcServiceContext, blockChan chan *pb.SubscribeUpdateBlock) *BlockProcessor {
	ctx, cancel := context.WithCancelCause(context.Background())
	return &BlockProcessor{
		sc:        sc,
		blockChan: blockChan,
		Logger:    logx.WithContext(ctx).WithFields(logx.Field("service", "block_processor")),
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
				p.Debugf("block chan len:%v", len(p.blockChan))
			}
		}
	}
}

func (p *BlockProcessor) Stop() {
	p.cancel(errors.New("service stop"))
}

func (p *BlockProcessor) procBlock(block *pb.SubscribeUpdateBlock) {
	defer func() {
		if r := recover(); r != nil {
			p.Errorf("procBlock panic: %v (slot: %d)", r, block.Slot)
		}
	}()

	startTime := time.Now()
	defer func() {
		p.Infof("区块处理总耗时: %v, slot: %d", time.Since(startTime), block.Slot)
	}()

	// 1. 过滤合法交易
	filterStart := time.Now()
	validTxs := make([]*pb.SubscribeUpdateTransactionInfo, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		if IsValidGrpcTx(tx) {
			validTxs = append(validTxs, tx)
		}
	}
	p.Infof("交易过滤耗时: %v, 总交易数: %d, 有效交易数: %d",
		time.Since(filterStart), len(block.Transactions), len(validTxs))

	// 2. 构造上下文
	ctxStart := time.Now()
	txCtx := p.buildTxContext(block)
	if txCtx == nil {
		return
	}
	p.Infof("上下文构造耗时: %v", time.Since(ctxStart))

	// 3. 并发解析出所有事件
	parseStart := time.Now()
	results := utils.ParallelMap(validTxs, consts.CpuCount+2,
		func(tx *pb.SubscribeUpdateTransactionInfo) core.ParsedTxResult {
			return p.parseTx(txCtx, tx)
		})

	// 统计普通事件数量
	var totalEvents int
	for _, result := range results {
		totalEvents += len(result.Events)
	}
	p.Infof("事件解析耗时: %v, 普通事件数: %d", time.Since(parseStart), totalEvents)

	// 4. 生成余额事件
	balanceStart := time.Now()
	balanceEvents := common.BuildBalanceUpdateEvents(results, txCtx.Slot, txCtx.BlockTime)
	p.Infof("余额事件生成耗时: %v, 余额事件数: %d", time.Since(balanceStart), len(balanceEvents))

	// 5. 生成mq任务
	mqStart := time.Now()
	mqjobs := dispatcher.BuildAllKafkaJobs(txCtx.Slot, results, balanceEvents, p.sc.Config.KafkaProducerConf)
	p.Infof("MQ任务生成耗时: %v, 总任务数: %d (普通事件: %d, 余额事件: %d)",
		time.Since(mqStart), len(mqjobs), totalEvents, len(balanceEvents))

	// 6. 分发任务（Kafka 推送 + 写进度）
	dispatchStart := time.Now()
	p.dispatchSlot(txCtx.Slot, txCtx.BlockTime, mqjobs)
	p.Infof("任务分发耗时: %v", time.Since(dispatchStart))
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
		logx.Errorf("[严重] BlockHash 无法解析，将使用零值：slot=%d, blockhash=%s, err=%v",
			block.Slot, block.Blockhash, err)
	}

	// 从价格缓存中获取 quote token 价格，如果失败则跳过该 block
	blockTime := block.BlockTime.Timestamp
	quotesPrice, ok := p.sc.PriceCache.GetQuotePricesAt(utils.USDQuoteMints, blockTime)
	if !ok {
		logx.Errorf("[严重] 获取 QuoteToken 价格失败，跳过该区块：slot=%d, blockTime=%d",
			block.Slot, blockTime)
		return nil
	}

	return &core.TxContext{
		BlockTime:   blockTime,
		Slot:        block.Slot,
		BlockHash:   blockHash, // 若解析失败为零值
		ParentSlot:  block.ParentSlot,
		QuotesPrice: quotesPrice,
	}
}

func (p *BlockProcessor) parseTx(txCtx *core.TxContext, tx *pb.SubscribeUpdateTransactionInfo) core.ParsedTxResult {
	adaptedTx, err := txadapter.AdaptGrpcTx(txCtx, tx)
	if err != nil {
		return core.ParsedTxResult{TxIndex: int(tx.Index)}
	}

	events := eventparser.ExtractEventsFromTx(adaptedTx)
	return core.ParsedTxResult{
		TxIndex:  int(tx.Index),
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
		p.Errorf("❌ [Dispatch] slot %d 被丢弃：活跃 dispatch 数超过上限", slotID)
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
		var maxSendTime, maxAckTime time.Duration
		successCount := len(okJobs)
		for _, job := range okJobs {
			if job.SendTime > maxSendTime {
				maxSendTime = job.SendTime
			}
			if job.AckTime > maxAckTime {
				maxAckTime = job.AckTime
			}
		}

		if len(failedJobs) == 0 {
			// ✅ Kafka 发送成功，写入进度
			progressStartTime := time.Now()
			if p.sc.ProgressManager != nil {
				err := p.sc.ProgressManager.MarkSlotStatus(ctx, progress.SourceGrpc, slotID, blockTime, progress.SlotProcessed)
				if err != nil {
					p.Errorf("⚠️ [Dispatch] slot %d 写入进度失败: %v", slotID, err)
				}
			}
			progressTime := time.Since(progressStartTime)
			totalTime := time.Since(startTime)

			p.Infof("✅ [Dispatch] slot %d 处理完成 - Kafka最大发送耗时: %v, 最大确认耗时: %v, 进度写入耗时: %v, 总耗时: %v（成功=%d）",
				slotID, maxSendTime, maxAckTime, progressTime, totalTime, successCount)
		} else {
			// ❌ Kafka 有失败，不写进度
			totalTime := time.Since(startTime)
			p.Errorf("❌ [Dispatch] slot %d 处理失败 - Kafka最大发送耗时: %v, 最大确认耗时: %v, 总耗时: %v（成功=%d 失败=%d）",
				slotID, maxSendTime, maxAckTime, totalTime, successCount, len(failedJobs))
		}
	}()
}

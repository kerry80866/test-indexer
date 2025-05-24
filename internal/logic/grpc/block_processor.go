package grpc

import (
	"context"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser"
	"dex-indexer-sol/internal/logic/txadapter"
	"dex-indexer-sol/internal/svc"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"errors"
	"time"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"github.com/zeromicro/go-zero/core/logx"
)

type BlockProcessor struct {
	sc        *svc.GrpcServiceContext
	blockChan chan *pb.SubscribeUpdateBlock // 接收 block 的 channel
	ctx       context.Context
	cancel    func(err error)
	logx.Logger
}

type ParsedTxResult struct {
	txIndex  int
	balances map[types.Pubkey]*core.TokenBalance
	events   []*core.Event
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
	startTime := time.Now()
	defer func() {
		p.Infof("区块处理总耗时: %v, slot: %d", time.Since(startTime), block.Slot)
	}()

	// 1. 过滤合法交易
	validTxs := make([]*pb.SubscribeUpdateTransactionInfo, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		if IsValidGrpcTx(tx) {
			validTxs = append(validTxs, tx)
		}
	}

	// 2. 构造上下文
	txCtx := p.buildTxContext(block)
	if txCtx == nil {
		return
	}

	// 3. 并发解析出所有事件
	parseStart := time.Now()
	results := utils.ParallelMap(validTxs, consts.CpuCount+2,
		func(tx *pb.SubscribeUpdateTransactionInfo) ParsedTxResult {
			return p.parseTx(txCtx, tx)
		})
	p.Infof("事件解析耗时: %v", time.Since(parseStart))

	totalLen := 0
	for _, result := range results {
		totalLen += len(result.events)
	}
	p.Infof("总tx数量: %v, 有效tx数量: %v, 总事件数量: %v, ", len(block.Transactions), len(validTxs), totalLen)
	//events := make([]*core.Event, 0, totalLen)
	//for _, result := range results {
	//	events = append(events, result.events...)
	//}
	// 4. 处理解析结果
	//mq.SendEventsAndWait(p.ctx, p.sc.Producer, p.sc.Config.KafkaProducerConf.Topics.Event, events, p.sc.Config.KafkaProducerConf.NumPartitions)

}

func (p *BlockProcessor) parseTx(txCtx *core.TxContext, tx *pb.SubscribeUpdateTransactionInfo) ParsedTxResult {
	adaptedTx, err := txadapter.AdaptGrpcTx(txCtx, tx)
	if err != nil {
		return ParsedTxResult{txIndex: int(tx.Index)}
	}

	events := eventparser.ExtractEventsFromTx(adaptedTx)
	return ParsedTxResult{
		txIndex:  int(tx.Index),
		balances: adaptedTx.Balances,
		events:   events,
	}
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

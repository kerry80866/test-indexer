package grpc

import (
	"context"
	"dex-indexer-sol/internal/logic/domain"
	"dex-indexer-sol/internal/svc"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

type BlockProcessor struct {
	sc        *svc.GrpcServiceContext
	blockChan chan *pb.SubscribeUpdateBlock // 接收 block 的 channel
	ctx       context.Context
	cancel    func(err error)
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
	//txCtx := p.buildTxContext(block)

	//p.decodeBlock(block)
}

//func (p *BlockProcessor) decodeBlock(block *pb.SubscribeUpdateBlock) {
//}

func (p *BlockProcessor) buildTxContext(block *pb.SubscribeUpdateBlock) *domain.TxContext {
	// 尝试解析 blockHash，如果失败只打日志但继续执行
	blockHash, err := types.HashFromBase58(block.Blockhash)
	if err != nil {
		logx.Errorf("[严重] BlockHash 无法解析，将使用零值：slot=%d, blockhash=%s, err=%v",
			block.Slot, block.Blockhash, err)
	}

	// 从价格缓存中获取 quote token 价格，如果失败则跳过该 block
	blockTime := block.BlockTime.Timestamp
	quotesPrice, ok := p.sc.PriceCache.GetQuotePricesAt(utils.QuoteTokens, blockTime)
	if !ok {
		logx.Errorf("[严重] 获取 QuoteToken 价格失败，跳过该区块：slot=%d, blockTime=%d",
			block.Slot, blockTime)
		return nil
	}

	return &domain.TxContext{
		BlockTime:   blockTime,
		Slot:        block.Slot,
		BlockHash:   blockHash, // 若解析失败为零值
		ParentSlot:  block.ParentSlot,
		QuotesPrice: quotesPrice,
	}
}

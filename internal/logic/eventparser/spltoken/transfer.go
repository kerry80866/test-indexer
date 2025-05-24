package spltoken

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/utils"
	"dex-indexer-sol/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

// extractTokenTransferEvent 负责识别并解析 Token Transfer 类型的指令。
// 若当前指令为符合条件的 Transfer 或 TransferChecked，则解析为 TransferEvent 并编码。
// 返回生成的事件（若有）和下一条待处理的指令索引。
func extractTokenTransferEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]
	parsedTransfer, ok := common.ParseTransferInstruction(ctx, ix)
	if !ok {
		return nil, current + 1
	}

	// 原地修改 ctx.BaseEvent 是安全的（tx级别是顺序解析,不会并发）
	eventType := pb.EventType_TRANSFER
	ctx.BaseEvent.Type = eventType
	ctx.BaseEvent.EventIndex = core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex)

	event := pb.TransferEvent{
		BaseEvent:        ctx.BaseEvent,
		Token:            parsedTransfer.Token[:],
		SrcAccount:       parsedTransfer.SrcAccount[:],
		DestAccount:      parsedTransfer.DestAccount[:],
		SrcWallet:        parsedTransfer.SrcWallet[:],
		DestWallet:       parsedTransfer.DestWallet[:],
		Amount:           parsedTransfer.Amount,
		Decimals:         uint32(parsedTransfer.Decimals),
		SrcTokenBalance:  parsedTransfer.SrcPostBalance,
		DestTokenBalance: parsedTransfer.DestPostBalance,
	}

	data, err := utils.EncodeEvent(uint32(eventType), &event)
	if err != nil {
		logx.Errorf("encode TransferEvent failed: %v", err)
		return nil, current + 1
	}

	return &core.Event{
		Tx:        ctx.Tx,
		EventId:   ctx.BaseEvent.EventIndex,
		EventType: uint32(eventType),
		Token:     event.Token,
		Data:      data,
	}, current + 1
}

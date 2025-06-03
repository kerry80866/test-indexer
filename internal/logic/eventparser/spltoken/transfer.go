package spltoken

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pb"
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

	event := pb.TransferEvent{
		Type:             pb.EventType_TRANSFER,
		EventId:          core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:             ctx.Slot,
		BlockTime:        ctx.BlockTime,
		TxHash:           ctx.TxHash,
		Signers:          ctx.Signers,
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

	return &core.Event{
		ID:        event.EventId,
		EventType: uint32(event.Type), // EventType = TRANSFER
		Key:       event.Token,        // 分区 Key，可用 Token / From / Owner
		Event: &pb.Event{ // protobuf 封装
			Event: &pb.Event_Transfer{
				Transfer: &event,
			},
		},
	}, current + 1
}

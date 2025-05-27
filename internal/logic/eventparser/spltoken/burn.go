package spltoken

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pb"
)

// extractTokenBurnEvent 提取一条 SPL Token Burn 或 BurnChecked 指令生成的事件
func extractTokenBurnEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]
	parsedBurn, ok := common.ParseBurnInstruction(ctx, ix)
	if !ok {
		return nil, current + 1
	}

	event := pb.BurnEvent{
		Type:             pb.EventType_BURN,
		EventIndex:       core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:             ctx.Slot,
		BlockTime:        ctx.BlockTime,
		TxHash:           ctx.TxHash,
		TxFrom:           ctx.TxFrom,
		Token:            parsedBurn.Token[:],
		FromTokenAccount: parsedBurn.SrcAccount[:],
		FromAddress:      parsedBurn.SrcWallet[:], // TokenAccount 的 owner
		Amount:           parsedBurn.Amount,
		Decimals:         uint32(parsedBurn.Decimals),
		FromTokenBalance: parsedBurn.SrcPostBalance,
	}

	return &core.Event{
		ID:        event.EventIndex,   // 唯一事件 ID（txIndex + ixIndex + innerIndex）
		EventType: uint32(event.Type), // EventType = BURN
		Key:       event.Token,        // 分区 key，可按 token 拆分
		Event: &pb.Event{
			Event: &pb.Event_Burn{
				Burn: &event,
			},
		},
	}, current + 1
}

package spltoken

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pb"
)

// extractTokenMintToEvent 提取一条 SPL Token MintTo 或 MintToChecked 指令生成的事件
func extractTokenMintToEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]
	parsedMintTo, ok := common.ParseMintToInstruction(ctx, ix)
	if !ok {
		return nil, current + 1
	}

	event := pb.MintToEvent{
		Type:           pb.EventType_MINT_TO,
		EventIndex:     core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:           ctx.Slot,
		BlockTime:      ctx.BlockTime,
		TxHash:         ctx.TxHash,
		TxFrom:         ctx.TxFrom,
		Token:          parsedMintTo.Token[:],
		ToTokenAccount: parsedMintTo.DestAccount[:],
		ToAddress:      parsedMintTo.DestWallet[:], // TokenAccount 的 owner
		Amount:         parsedMintTo.Amount,
		Decimals:       uint32(parsedMintTo.Decimals),
		ToTokenBalance: parsedMintTo.DestPostBalance,
	}

	return &core.Event{
		ID:        event.EventIndex,   // 唯一事件 ID（txIndex + ixIndex + innerIndex）
		EventType: uint32(event.Type), // EventType = MINT_TO
		Key:       event.Token,        // 分区 key，可用 Token 拆分
		Event: &pb.Event{
			Event: &pb.Event_Mint{
				Mint: &event,
			},
		},
	}, current + 1
}

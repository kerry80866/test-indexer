package common

import (
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/pb"
)

// BuildTransferEvent 构造标准的 TransferEvent 并封装为 core.Event。
func BuildTransferEvent(
	ctx *ParserContext,
	transfer *ParsedTransfer,
) *core.Event {
	event := pb.TransferEvent{
		Type:             pb.EventType_TRANSFER,
		EventId:          core.BuildEventID(ctx.Slot, ctx.TxIndex, transfer.IxIndex, transfer.InnerIndex),
		Slot:             ctx.Slot,
		BlockTime:        ctx.BlockTime,
		TxHash:           ctx.TxHash,
		Signers:          ctx.Signers,
		Token:            transfer.Token[:],
		SrcAccount:       transfer.SrcAccount[:],
		DestAccount:      transfer.DestAccount[:],
		SrcWallet:        transfer.SrcWallet[:],
		DestWallet:       transfer.DestWallet[:],
		Amount:           transfer.Amount,
		Decimals:         uint32(transfer.Decimals),
		SrcTokenBalance:  transfer.SrcPostBalance,
		DestTokenBalance: transfer.DestPostBalance,
	}

	return &core.Event{
		ID:        event.EventId,
		EventType: uint32(event.Type),
		Key:       event.Token,
		Event: &pb.Event{
			Event: &pb.Event_Transfer{Transfer: &event},
		},
	}
}

// BuildMintToEvent 构造 MintToEvent 并封装为 core.Event。
func BuildMintToEvent(
	ctx *ParserContext,
	mintTo *ParsedMintTo,
) *core.Event {
	event := pb.MintToEvent{
		Type:           pb.EventType_MINT_TO,
		EventId:        core.BuildEventID(ctx.Slot, ctx.TxIndex, mintTo.IxIndex, mintTo.InnerIndex),
		Slot:           ctx.Slot,
		BlockTime:      ctx.BlockTime,
		TxHash:         ctx.TxHash,
		Signers:        ctx.Signers,
		Token:          mintTo.Token[:],
		ToTokenAccount: mintTo.DestAccount[:],
		ToAddress:      mintTo.DestWallet[:],
		Amount:         mintTo.Amount,
		Decimals:       uint32(mintTo.Decimals),
		ToTokenBalance: mintTo.DestPostBalance,
	}

	return &core.Event{
		ID:        event.EventId,
		EventType: uint32(event.Type),
		Key:       event.Token,
		Event: &pb.Event{
			Event: &pb.Event_Mint{Mint: &event},
		},
	}
}

// BuildBurnEvent 构造 BurnEvent 并封装为 core.Event。
func BuildBurnEvent(
	ctx *ParserContext,
	burn *ParsedBurn,
) *core.Event {
	event := pb.BurnEvent{
		Type:             pb.EventType_BURN,
		EventId:          core.BuildEventID(ctx.Slot, ctx.TxIndex, burn.IxIndex, burn.InnerIndex),
		Slot:             ctx.Slot,
		BlockTime:        ctx.BlockTime,
		TxHash:           ctx.TxHash,
		Signers:          ctx.Signers,
		Token:            burn.Token[:],
		FromTokenAccount: burn.SrcAccount[:],
		FromAddress:      burn.SrcWallet[:],
		Amount:           burn.Amount,
		Decimals:         uint32(burn.Decimals),
		FromTokenBalance: burn.SrcPostBalance,
	}

	return &core.Event{
		ID:        event.EventId,
		EventType: uint32(event.Type),
		Key:       event.Token,
		Event: &pb.Event{
			Event: &pb.Event_Burn{Burn: &event},
		},
	}
}

package common

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/pb"
)

func BuildAddLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	lpMintTo *ParsedMintTo,
	pairAddress types.Pubkey,
	dex int,
) *core.Event {
	if baseTransfer == nil || quoteTransfer == nil {
		return nil
	}

	eventIndex := core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex)

	liquidity := &pb.LiquidityEvent{
		Type:              pb.EventType_ADD_LIQUIDITY,
		EventIndex:        eventIndex,
		Slot:              ctx.Slot,
		BlockTime:         ctx.BlockTime,
		TxHash:            ctx.TxHash[:],
		TxFrom:            ctx.TxFrom[:],
		Dex:               uint32(dex),
		UserWallet:        baseTransfer.SrcWallet[:],
		PairAddress:       pairAddress[:],
		TokenDecimals:     uint32(baseTransfer.Decimals),
		QuoteDecimals:     uint32(quoteTransfer.Decimals),
		TokenAmount:       baseTransfer.Amount,
		QuoteTokenAmount:  quoteTransfer.Amount,
		Token:             baseTransfer.Token[:],
		QuoteToken:        quoteTransfer.Token[:],
		TokenAccount:      baseTransfer.DestAccount[:],
		QuoteTokenAccount: quoteTransfer.DestAccount[:],
		PairTokenBalance:  baseTransfer.DestPostBalance,
		PairQuoteBalance:  quoteTransfer.DestPostBalance,
		UserTokenBalance:  baseTransfer.SrcPostBalance,
		UserQuoteBalance:  quoteTransfer.SrcPostBalance,
		LpToken:           consts.InvalidAddress[:],
		LpAmount:          0,
		LpDecimals:        0,
	}

	if lpMintTo != nil {
		liquidity.LpToken = lpMintTo.Token[:]
		liquidity.LpAmount = lpMintTo.Amount
		liquidity.LpDecimals = uint32(lpMintTo.Decimals)
	}

	return &core.Event{
		ID:        liquidity.EventIndex,
		EventType: uint32(liquidity.Type),
		Key:       liquidity.Token,
		Event: &pb.Event{
			Event: &pb.Event_Liquidity{
				Liquidity: liquidity,
			},
		},
	}
}

func BuildRemoveLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	lpBurn *ParsedBurn,
	pairAddress types.Pubkey,
	dex int,
) *core.Event {
	if baseTransfer == nil || quoteTransfer == nil {
		return nil
	}

	eventIndex := core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex)

	liquidity := &pb.LiquidityEvent{
		Type:              pb.EventType_REMOVE_LIQUIDITY,
		EventIndex:        eventIndex,
		Slot:              ctx.Slot,
		BlockTime:         ctx.BlockTime,
		TxHash:            ctx.TxHash[:],
		TxFrom:            ctx.TxFrom[:],
		Dex:               uint32(dex),
		UserWallet:        baseTransfer.SrcWallet[:],
		PairAddress:       pairAddress[:],
		TokenDecimals:     uint32(baseTransfer.Decimals),
		QuoteDecimals:     uint32(quoteTransfer.Decimals),
		TokenAmount:       baseTransfer.Amount,
		QuoteTokenAmount:  quoteTransfer.Amount,
		Token:             baseTransfer.Token[:],
		QuoteToken:        quoteTransfer.Token[:],
		TokenAccount:      baseTransfer.DestAccount[:],
		QuoteTokenAccount: quoteTransfer.DestAccount[:],
		PairTokenBalance:  baseTransfer.DestPostBalance,
		PairQuoteBalance:  quoteTransfer.DestPostBalance,
		UserTokenBalance:  baseTransfer.SrcPostBalance,
		UserQuoteBalance:  quoteTransfer.SrcPostBalance,
		LpToken:           consts.InvalidAddress[:], // // 部分协议不返回 LP mint/burn 事件，此处容忍缺失。
		LpAmount:          0,
		LpDecimals:        0,
	}

	if lpBurn != nil {
		liquidity.LpToken = lpBurn.Token[:]
		liquidity.LpAmount = lpBurn.Amount
		liquidity.LpDecimals = uint32(lpBurn.Decimals)
	}

	return &core.Event{
		ID:        liquidity.EventIndex,
		EventType: uint32(liquidity.Type),
		Key:       liquidity.Token,
		Event: &pb.Event{
			Event: &pb.Event_Liquidity{
				Liquidity: liquidity,
			},
		},
	}
}

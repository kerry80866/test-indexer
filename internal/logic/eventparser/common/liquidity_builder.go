package common

import (
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/pb"
)

func BuildAddLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	pairAddress types.Pubkey,
	dex int,
) *core.Event {
	return buildBaseLiquidityEvent(ctx, ix, baseTransfer, quoteTransfer, pairAddress, dex, pb.EventType_ADD_LIQUIDITY)
}

func BuildRemoveLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	pairAddress types.Pubkey,
	dex int,
) *core.Event {
	return buildBaseLiquidityEvent(ctx, ix, baseTransfer, quoteTransfer, pairAddress, dex, pb.EventType_REMOVE_LIQUIDITY)
}

// buildBaseLiquidityEvent 构造 Add / Remove 类型的流动性事件（统一表示为 base token / quote token）
func buildBaseLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	pairAddress types.Pubkey,
	dex int,
	eventType pb.EventType,
) *core.Event {
	if baseTransfer == nil || quoteTransfer == nil {
		return nil
	}
	if baseTransfer.Token == quoteTransfer.Token {
		logger.Errorf("[BuildLiquidityEvent] tx=%s: base and quote token mint are the same: mint=%s",
			ctx.TxHashString(), baseTransfer.Token)
		return nil
	}

	liquidity := &pb.LiquidityEvent{
		Type:                   eventType,
		EventId:                core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:                   ctx.Slot,
		BlockTime:              ctx.BlockTime,
		TxHash:                 ctx.TxHash[:],
		Signers:                ctx.Signers,
		Dex:                    uint32(dex),
		UserWallet:             baseTransfer.SrcWallet[:],
		PairAddress:            pairAddress[:],
		TokenDecimals:          uint32(baseTransfer.Decimals),
		QuoteDecimals:          uint32(quoteTransfer.Decimals),
		TokenAmount:            baseTransfer.Amount,
		QuoteTokenAmount:       quoteTransfer.Amount,
		Token:                  baseTransfer.Token[:],
		QuoteToken:             quoteTransfer.Token[:],
		TokenAccount:           baseTransfer.DestAccount[:],
		QuoteTokenAccount:      quoteTransfer.DestAccount[:],
		TokenAccountOwner:      baseTransfer.DestWallet[:],
		QuoteTokenAccountOwner: quoteTransfer.DestWallet[:],
		PairTokenBalance:       baseTransfer.DestPostBalance,
		PairQuoteBalance:       quoteTransfer.DestPostBalance,
		UserTokenBalance:       baseTransfer.SrcPostBalance,
		UserQuoteBalance:       quoteTransfer.SrcPostBalance,
	}

	return &core.Event{
		ID:        liquidity.EventId,
		EventType: uint32(liquidity.Type),
		Key:       liquidity.PairAddress,
		Event: &pb.Event{
			Event: &pb.Event_Liquidity{
				Liquidity: liquidity,
			},
		},
	}
}

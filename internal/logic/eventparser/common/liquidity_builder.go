package common

import (
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/pb"
	"github.com/dex-indexer-sol/pkg/logger"
	"github.com/dex-indexer-sol/pkg/types"
)

func BuildAddLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	poolAddress types.Pubkey,
	dex int,
) *core.Event {
	if baseTransfer == nil || quoteTransfer == nil {
		return nil
	}
	return buildBaseLiquidityEvent(
		ctx,
		ix,
		baseTransfer,
		quoteTransfer,
		baseTransfer.SrcAccount,
		poolAddress,
		dex,
		pb.EventType_ADD_LIQUIDITY,
	)
}

func BuildRemoveLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	poolAddress types.Pubkey,
	dex int,
) *core.Event {
	if baseTransfer == nil || quoteTransfer == nil {
		return nil
	}
	return buildBaseLiquidityEvent(
		ctx,
		ix,
		baseTransfer,
		quoteTransfer,
		baseTransfer.DestWallet,
		poolAddress,
		dex,
		pb.EventType_REMOVE_LIQUIDITY,
	)
}

// buildBaseLiquidityEvent 构造 Add / Remove 类型的流动性事件（统一表示为 base token / quote token）
func buildBaseLiquidityEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	baseTransfer, quoteTransfer *ParsedTransfer,
	userWallet types.Pubkey,
	poolAddress types.Pubkey,
	dex int,
	eventType pb.EventType,
) *core.Event {
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
		UserWallet:             userWallet[:],
		PairAddress:            poolAddress[:],
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

	// 若 Quote 为 WSOL 且为临时账户（余额为 0），用 SOL 余额补充 Quote 余额。
	if liquidity.UserQuoteBalance == 0 && quoteTransfer.Token == consts.WSOLMint {
		patchWSOLBalanceIfNeeded(ctx, quoteTransfer.SrcAccount, quoteTransfer.SrcWallet, func(val uint64) {
			liquidity.UserQuoteBalance = val
		})
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

func CloneLiquidityEvent(orig *core.Event) *core.Event {
	if orig == nil {
		return nil
	}

	src := orig.Event.GetLiquidity()
	if src == nil {
		return nil
	}

	clone := &pb.LiquidityEvent{
		Type:                   src.Type,
		EventId:                src.EventId,
		Slot:                   src.Slot,
		BlockTime:              src.BlockTime,
		TxHash:                 src.TxHash,
		Signers:                src.Signers,
		Dex:                    src.Dex,
		UserWallet:             src.UserWallet,
		PairAddress:            src.PairAddress,
		TokenDecimals:          src.TokenDecimals,
		QuoteDecimals:          src.QuoteDecimals,
		TokenAmount:            src.TokenAmount,
		QuoteTokenAmount:       src.QuoteTokenAmount,
		Token:                  src.Token,
		QuoteToken:             src.QuoteToken,
		TokenAccount:           src.TokenAccount,
		QuoteTokenAccount:      src.QuoteTokenAccount,
		TokenAccountOwner:      src.TokenAccountOwner,
		QuoteTokenAccountOwner: src.QuoteTokenAccountOwner,
		PairTokenBalance:       src.PairTokenBalance,
		PairQuoteBalance:       src.PairQuoteBalance,
		UserTokenBalance:       src.UserTokenBalance,
		UserQuoteBalance:       src.UserQuoteBalance,
	}

	return &core.Event{
		ID:        clone.EventId,
		EventType: uint32(clone.Type),
		Key:       clone.PairAddress,
		Event: &pb.Event{
			Event: &pb.Event_Liquidity{
				Liquidity: clone,
			},
		},
	}
}

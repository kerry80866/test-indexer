package common

import (
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/pb"
	"github.com/dex-indexer-sol/pkg/logger"
	"github.com/dex-indexer-sol/pkg/types"
	"github.com/dex-indexer-sol/pkg/utils"
)

// BuildTradeEvent 根据 token 转账方向构建 BUY 或 SELL 类型的交易事件。
func BuildTradeEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	userToPool *ParsedTransfer,
	poolToUser *ParsedTransfer,
	pairAddress types.Pubkey,
	quote types.Pubkey,
	isQuoteConfirmed bool,
	dex int,
) *core.Event {
	var trade *pb.TradeEvent

	if userToPool.Token == poolToUser.Token {
		logger.Errorf("[BuildTradeEvent] tx=%s: swap token mismatch — both transfer tokens are the same: mint=%s",
			ctx.TxHashString(), userToPool.Token)
		return nil
	}

	if userToPool.Token == quote {
		// 用户支付 quote，获得 base → BUY
		trade = buildBuyEvent(ctx, ix, userToPool, poolToUser, quote, pairAddress, uint32(dex))
	} else {
		// 用户支付 base，获得 quote → SELL
		trade = buildSellEvent(ctx, ix, userToPool, poolToUser, quote, pairAddress, uint32(dex))
	}

	if !isQuoteConfirmed {
		trade.Type = pb.EventType_TRADE_UNKNOWN
	}

	return &core.Event{
		ID:        trade.EventId,
		EventType: uint32(trade.Type),
		Key:       trade.PairAddress,
		Event: &pb.Event{
			Event: &pb.Event_Trade{
				Trade: trade,
			},
		},
	}
}

// buildBuyEvent 构造 BUY 类型交易事件（支付 quote，获得 base）。
func buildBuyEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	userToPool *ParsedTransfer, // 用户转入资金（quote token 转账）
	poolToUser *ParsedTransfer, // 用户获得资金（base token 转账）
	quote types.Pubkey,
	pairAddress types.Pubkey,
	dex uint32,
) *pb.TradeEvent {
	event := &pb.TradeEvent{
		Type:              pb.EventType_TRADE_BUY,
		EventId:           core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:              ctx.Slot,
		BlockTime:         ctx.BlockTime,
		TxHash:            ctx.TxHash,
		Signers:           ctx.Signers,
		Dex:               dex,
		TokenDecimals:     uint32(poolToUser.Decimals), // base 精度
		QuoteDecimals:     uint32(userToPool.Decimals), // quote 精度
		TokenAmount:       poolToUser.Amount,           // 获得 base
		QuoteTokenAmount:  userToPool.Amount,           // 支付 quote
		Token:             poolToUser.Token[:],
		QuoteToken:        quote[:],
		PairAddress:       pairAddress[:],
		TokenAccount:      poolToUser.SrcAccount[:],
		QuoteTokenAccount: userToPool.DestAccount[:],
		UserWallet:        poolToUser.DestWallet[:],
		PairTokenBalance:  poolToUser.SrcPostBalance,
		PairQuoteBalance:  userToPool.DestPostBalance,
		UserTokenBalance:  poolToUser.DestPostBalance,
		UserQuoteBalance:  userToPool.SrcPostBalance,
	}

	// 处理 SOL -> WSOL 的临时账户转账：若 WSOL 账户余额为 0 且为临时创建，则使用用户钱包中的 SOL 补充 quote 余额
	if event.UserQuoteBalance == 0 && quote == consts.WSOLMint && userToPool.SrcWallet == poolToUser.DestWallet {
		patchWSOLBalanceIfNeeded(ctx, userToPool.SrcAccount, userToPool.SrcWallet, func(val uint64) {
			event.UserQuoteBalance = val
		})
	}

	fillUsdEstimate(event, quote, ctx)
	return event
}

// buildSellEvent 构建 DEX Swap 中的 SELL 类型交易事件。
func buildSellEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	userToPool *ParsedTransfer, // 用户转入资金（quote token 转账）
	poolToUser *ParsedTransfer, // 用户获得资金（base token 转账）
	quote types.Pubkey,
	pairAddress types.Pubkey,
	dex uint32,
) *pb.TradeEvent {
	event := &pb.TradeEvent{
		Type:              pb.EventType_TRADE_SELL,
		EventId:           core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:              ctx.Slot,
		BlockTime:         ctx.BlockTime,
		TxHash:            ctx.TxHash,
		Signers:           ctx.Signers,
		Dex:               dex,
		TokenDecimals:     uint32(userToPool.Decimals),
		QuoteDecimals:     uint32(poolToUser.Decimals),
		TokenAmount:       userToPool.Amount, // 卖出的 base
		QuoteTokenAmount:  poolToUser.Amount, // 获得的 quote
		Token:             userToPool.Token[:],
		QuoteToken:        quote[:],
		PairAddress:       pairAddress[:],
		TokenAccount:      userToPool.DestAccount[:],
		QuoteTokenAccount: poolToUser.SrcAccount[:],
		UserWallet:        poolToUser.DestWallet[:],
		PairTokenBalance:  userToPool.DestPostBalance,
		PairQuoteBalance:  poolToUser.SrcPostBalance,
		UserTokenBalance:  userToPool.SrcPostBalance,
		UserQuoteBalance:  poolToUser.DestPostBalance,
	}

	// 若 Quote 为 WSOL 且为临时账户（余额为 0），用 SOL 余额补充 Quote 余额。
	if event.UserQuoteBalance == 0 && quote == consts.WSOLMint {
		patchWSOLBalanceIfNeeded(ctx, poolToUser.DestAccount, poolToUser.DestWallet, func(val uint64) {
			event.UserQuoteBalance = val
		})
	}

	fillUsdEstimate(event, quote, ctx)
	return event
}

// fillUsdEstimate 用 quote token 价格补全交易估值。
func fillUsdEstimate(event *pb.TradeEvent, quote types.Pubkey, ctx *ParserContext) {
	if quoteUsd, ok := ctx.Tx.TxCtx.GetQuoteUsd(quote); ok {
		baseAmount := float64(event.TokenAmount) / utils.Pow10(event.TokenDecimals)
		quoteAmount := float64(event.QuoteTokenAmount) / utils.Pow10(event.QuoteDecimals)

		event.AmountUsd = quoteAmount * quoteUsd
		if baseAmount > 0 {
			event.PriceUsd = event.AmountUsd / baseAmount
		}
	}
}

package common

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"dex-indexer-sol/pb"
)

// BuildTradeEvent 根据 token 转账方向构建 BUY 或 SELL 类型的交易事件。
func BuildTradeEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	userToPool *ParsedTransfer,
	poolToUser *ParsedTransfer,
	pairAddress types.Pubkey,
	quote types.Pubkey,
	dex int,
) *core.Event {
	var trade *pb.TradeEvent

	if userToPool.Token == quote {
		// 用户支付 quote，获得 base → BUY
		trade = buildBuyEvent(ctx, ix, userToPool, poolToUser, quote, pairAddress, uint32(dex))
	} else {
		// 用户支付 base，获得 quote → SELL
		trade = buildSellEvent(ctx, ix, userToPool, poolToUser, quote, pairAddress, uint32(dex))
	}

	return &core.Event{
		ID:        trade.EventIndex,
		EventType: uint32(trade.Type),
		Key:       trade.Token, // base token 分区
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
		EventIndex:        core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:              ctx.Slot,
		BlockTime:         ctx.BlockTime,
		TxHash:            ctx.TxHash,
		TxFrom:            ctx.TxFrom,
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
		EventIndex:        core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:              ctx.Slot,
		BlockTime:         ctx.BlockTime,
		TxHash:            ctx.TxHash,
		TxFrom:            ctx.TxFrom,
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

	fillUsdEstimate(event, quote, ctx)
	return event
}

// fillUsdEstimate 用 quote token 价格补全交易估值。
func fillUsdEstimate(event *pb.TradeEvent, quote types.Pubkey, ctx *ParserContext) {
	if quoteUsd, ok := ctx.Tx.TxCtx.QuotesPrice[quote]; ok {
		baseAmount := float64(event.TokenAmount) / utils.Pow10(event.TokenDecimals)
		quoteAmount := float64(event.QuoteTokenAmount) / utils.Pow10(event.QuoteDecimals)

		event.AmountUsd = quoteAmount * quoteUsd
		if baseAmount > 0 {
			event.PriceUsd = event.AmountUsd / baseAmount
		}
	}
}

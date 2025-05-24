package common

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"dex-indexer-sol/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

// BuildTradeEvent 根据转账方向（base / quote）构建标准的交易事件 Event
func BuildTradeEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	userToPool *ParsedTransfer,
	poolToUser *ParsedTransfer,
	pairAddress types.Pubkey,
	quote types.Pubkey,
	dex int,
) *core.Event {
	var event *pb.TradeEvent

	if userToPool.Token == quote {
		// 用户支付 quote，获得 base，对应 BUY 类型
		event = buildBuyEvent(ctx, userToPool, poolToUser, quote, pairAddress, uint32(dex))
	} else {
		// 用户支付 base，获得 quote，对应 SELL 类型
		event = buildSellEvent(ctx, userToPool, poolToUser, quote, pairAddress, uint32(dex))
	}

	ctx.BaseEvent.EventIndex = core.BuildEventID(ctx.TxIndex, ix.IxIndex, ix.InnerIndex)
	data, err := utils.EncodeEvent(uint32(event.BaseEvent.Type), event)
	if err != nil {
		logx.Errorf("encode TradeEvent failed: %v", err)
		return nil
	}

	return &core.Event{
		Tx:        ctx.Tx,
		EventId:   event.BaseEvent.EventIndex,
		EventType: uint32(event.BaseEvent.Type),
		Token:     event.Token,
		Data:      data,
	}
}

// buildBuyEvent 构建 Raydium Swap 中的 BUY 类型交易事件：
// - 用户支付 quote token，获得 base token；
// - userToPool 表示用户支付的 quote 转账；
// - poolToUser 表示池子发给用户的 base 转账；
// - quote 为确定的 quote token mint；
// - accountOffset 为账户偏移（支持 17 / 18 账户结构）。
func buildBuyEvent(
	ctx *ParserContext,
	userToPool *ParsedTransfer,
	poolToUser *ParsedTransfer,
	quote types.Pubkey,
	pairAddress types.Pubkey,
	dex uint32,
) *pb.TradeEvent {
	ctx.BaseEvent.Type = pb.EventType_TRADE_BUY

	event := &pb.TradeEvent{
		BaseEvent:     ctx.BaseEvent,
		Dex:           dex,
		TokenDecimals: uint32(poolToUser.Decimals), // base token 精度
		QuoteDecimals: uint32(userToPool.Decimals), // quote token 精度

		TokenAmount:      poolToUser.Amount, // 获得的 base 数量
		QuoteTokenAmount: userToPool.Amount, // 支付的 quote 数量

		Token:       poolToUser.Token[:], // base token mint
		QuoteToken:  quote[:],            // quote token mint
		PairAddress: pairAddress[:],      // 交易对地址

		TokenAccount:      poolToUser.SrcAccount[:],  // base token 池子账户
		QuoteTokenAccount: userToPool.DestAccount[:], // quote token 池子账户
		UserWallet:        poolToUser.DestWallet[:],  // 用户钱包地址

		// Swap 后池子余额（PostBalance）
		PairTokenBalance: poolToUser.SrcPostBalance,
		PairQuoteBalance: userToPool.DestPostBalance,

		// Swap 后用户余额
		UserTokenBalance: poolToUser.DestPostBalance,
		UserQuoteBalance: userToPool.SrcPostBalance,
	}

	// 金额估值逻辑（用于后续展示 / K线聚合）
	if quoteUsd, ok := ctx.Tx.TxCtx.QuotesPrice[quote]; ok {
		baseAmount := float64(event.TokenAmount) / utils.Pow10(event.TokenDecimals)
		quoteAmount := float64(event.QuoteTokenAmount) / utils.Pow10(event.QuoteDecimals)

		event.AmountUsd = quoteAmount * quoteUsd
		if baseAmount > 0 {
			event.PriceUsd = event.AmountUsd / baseAmount
		}
	}

	return event
}

// buildSellEvent 构建 Raydium Swap 中的 SELL 类型交易事件：
// - 用户支付 base token，获得 quote token；
// - userToPool 表示用户支付的 base 转账；
// - poolToUser 表示池子转出 quote 给用户；
// - quote 为已确定的 quote token mint；
// - accountOffset 为账户偏移（支持 17 / 18 个账户结构）。
func buildSellEvent(
	ctx *ParserContext,
	userToPool *ParsedTransfer,
	poolToUser *ParsedTransfer,
	quote types.Pubkey,
	pairAddress types.Pubkey,
	dex uint32,
) *pb.TradeEvent {
	ctx.BaseEvent.Type = pb.EventType_TRADE_SELL

	event := &pb.TradeEvent{
		BaseEvent:     ctx.BaseEvent,
		Dex:           dex,
		TokenDecimals: uint32(userToPool.Decimals), // base token 精度
		QuoteDecimals: uint32(poolToUser.Decimals), // quote token 精度

		TokenAmount:      userToPool.Amount, // 卖出的 base 数量
		QuoteTokenAmount: poolToUser.Amount, // 获得的 quote 数量

		Token:       userToPool.Token[:], // base token mint
		QuoteToken:  quote[:],            // quote token mint
		PairAddress: pairAddress[:],      // 交易对地址

		TokenAccount:      userToPool.DestAccount[:], // base token 池子账户
		QuoteTokenAccount: poolToUser.SrcAccount[:],  // quote token 池子账户
		UserWallet:        poolToUser.DestWallet[:],  // 用户钱包地址

		// Swap 后池子余额（PostBalance）
		PairTokenBalance: userToPool.DestPostBalance,
		PairQuoteBalance: poolToUser.SrcPostBalance,

		// Swap 后用户余额（PostBalance）
		UserTokenBalance: userToPool.SrcPostBalance,
		UserQuoteBalance: poolToUser.DestPostBalance,
	}

	// 金额估值逻辑（用于展示 / K 线估算）
	if quoteUsd, ok := ctx.Tx.TxCtx.QuotesPrice[quote]; ok {
		baseAmount := float64(event.TokenAmount) / utils.Pow10(event.TokenDecimals)
		quoteAmount := float64(event.QuoteTokenAmount) / utils.Pow10(event.QuoteDecimals)

		event.AmountUsd = quoteAmount * quoteUsd
		if baseAmount > 0 {
			event.PriceUsd = event.AmountUsd / baseAmount
		}
	}

	return event
}

package orcawhirlpool

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/utils"
)

// Orca Whirlpool Swap 交易中账户结构:
//
// 0 - Token Program
// 1 - Token Authority
// 2 - Whirlpool (Orca Whirlpool 市场池地址)
// 3 - Token Owner Account A（用户的 Token A 账户）
// 4 - Token Vault A（池子的 Token A 账户）
// 5 - Token Owner Account B（用户的 Token B 账户）
// 6 - Token Vault B（池子的 Token B 账户）
//
// 交易示例：
// swap: https://solscan.io/tx/62dJLjdMdhY9HwpHjXmTFqpEidxvyMTxsX4eoLzVH9yXmdoBmUYMBgphXnqNmQGG7GJJLtHWxi5dkWkFMKLoxezG
// swap: https://solscan.io/tx/jv1aR39sDvhh3iqXWr5mENAtiNzvhFGMXJNWahTLnhHjvZFzcsD6srwkW1wzN6j9WazdEftCdZe9uAswhtLNkHA
func extractSwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 校验账户数量是否满足预期
	if len(ix.Accounts) < 7 {
		logger.Errorf("[OrcaWhirlpool:extractSwapEvent] 账户数量不足: tx=%s", ctx.TxHashString())
		return -1
	}

	// 查找转账记录，匹配用户与池子 Token 账户
	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: 3,
		UserToken2AccountIndex: 5,
		PoolToken1AccountIndex: 4,
		PoolToken2AccountIndex: 6,
	}, 0)
	if result == nil {
		logger.Errorf("[OrcaWhirlpool:extractSwapEvent] 转账结构缺失: tx=%s, ix=%d, inner=%d",
			ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return -1
	}

	// 优先尝试使用自定义优先级的quote token（WSOL、USDC、USDT等）
	quote, ok := utils.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		// 默认规则，token B 账户视为 quote token
		if result.UserToPool.SrcAccount == ix.Accounts[5] {
			quote = result.UserToPool.Token
		} else {
			quote = result.PoolToUser.Token
		}
	}

	// 交易对主池地址
	pairAddress := ix.Accounts[2]

	// 构建交易事件
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, true, consts.DexOrcaWhirlpool)
	if event == nil {
		return -1
	}

	ctx.AddEvent(event)
	return result.MaxIndex + 1
}

// Orca Whirlpool Swap2 交易中账户结构:
//
// 0 - Token Program A（Token Program，程序账户）
// 1 - Token Program B（Token Program，程序账户）
// 2 - Memo Program（Memo Program v2）
// 3 - Token Authority（权限账户）
// 4 - Whirlpool（Orca Whirlpool 市场池地址）
// 5 - Token Mint A（Token A 的 Mint 地址）
// 6 - Token Mint B（Token B 的 Mint 地址）
// 7 - Token Owner Account A（用户的Token A Account）
// 8 - Token Vault A（池子的Token A Account）
// 9 - Token Owner Account B（用户的Token B Account）
// 10 - Token Vault B（池子的Token B Account）
//
// 交易示例：
// swap2: https://solscan.io/tx/ZP5kKJdy5oQ9AkMqW2tEMKobNdbiXcVmiX4zhhYpk3R1v8P1vv8nQqaDFWMg9upWHj3g3sYGQevw9Jeht4H3hx6
func extractSwap2Event(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 校验账户数量是否满足预期
	if len(ix.Accounts) < 11 {
		logger.Errorf("[OrcaWhirlpool:extractSwap2Event] 账户数量不足: tx=%s", ctx.TxHashString())
		return -1
	}

	// 查找转账记录，匹配用户与池子 Token 账户
	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: 7,
		UserToken2AccountIndex: 9,
		PoolToken1AccountIndex: 8,
		PoolToken2AccountIndex: 10,
	}, 0)
	if result == nil {
		logger.Errorf("[OrcaWhirlpool:extractSwap2Event] 转账结构缺失: tx=%s, ix=%d, inner=%d",
			ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return -1
	}

	// 严格校验 mint 地址匹配
	if !((result.UserToPool.Token == ix.Accounts[5] && result.PoolToUser.Token == ix.Accounts[6]) ||
		(result.UserToPool.Token == ix.Accounts[6] && result.PoolToUser.Token == ix.Accounts[5])) {
		logger.Errorf("[OrcaWhirlpool:extractSwap2Event] mint 不匹配: tx=%s, userToPool=%s, poolToUser=%s, tokenA=%s, tokenB=%s",
			ctx.TxHashString(), result.UserToPool.Token, result.PoolToUser.Token, ix.Accounts[5], ix.Accounts[6],
		)
		return -1
	}

	// 优先尝试使用自定义优先级的quote token（WSOL、USDC、USDT等）
	quote, ok := utils.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		quote = ix.Accounts[6] // 默认取 Token Mint B 作为 quote token
		if result.UserToPool.Token != quote && result.PoolToUser.Token != quote {
			logger.Warnf("[OrcaWhirlpool:extractSwap2Event] 无法识别 quote token，跳过: tx=%s", ctx.TxHashString())
			return -1
		}
	}

	// 交易对主池地址
	pairAddress := ix.Accounts[4]

	// 构建交易事件
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, true, consts.DexOrcaWhirlpool)
	if event == nil {
		return -1
	}

	ctx.AddEvent(event)
	return result.MaxIndex + 1
}

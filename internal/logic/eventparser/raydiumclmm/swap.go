package raydiumclmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/utils"
)

// extractSwapEvent 解析 Raydium CLMM (Concentrated Liquidity Market Maker) 的 swap 事件，构造标准 TradeEvent（BUY / SELL）。
// 示例交易：https://solscan.io/tx/2ABmxyKMK32gRpTkdNPMgqZNTZGsUP1WftxsFjFYrLSywcpxVHuMgUGqHV6Y21hvdcV77YnnEszjcXoXvRHojQXB
//
// Raydium CLMM Swap 指令账户布局：
//  0. `[signer]`   用户钱包（payer）
//  1. `[]`        AMM 配置账户（pair 唯一标识）
//  2. `[]`        池子账户
//  3. `[writable]` 用户 token1 账户（支出）
//  4. `[writable]` 用户 token2 账户（接收）
//  5. `[writable]` 池子 token1 vault
//  6. `[writable]` 池子 token2 vault
func extractSwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]

	// 至少应有 7 个账户才能覆盖必要的 Swap 参数
	if len(ix.Accounts) < 7 {
		logger.Errorf("[RaydiumCLMM:extractSwapEvent] 账户数量不足: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}

	// 查找 Swap 过程中出现的两个方向的 Transfer（用户 -> 池子、池子 -> 用户）
	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: 3, // 用户支出 token 的账户
		UserToken2AccountIndex: 4, // 用户接收 token 的账户
		PoolToken1AccountIndex: 5, // 池子 token1 vault
		PoolToken2AccountIndex: 6, // 池子 token2 vault
	}, 2)
	if result == nil {
		logger.Errorf("[RaydiumCLMM:extractSwapEvent] 转账结构缺失: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}

	// 推断 quote token（通常为 USDC/USDT 等稳定币），用于区分 BUY / SELL
	quote, ok := utils.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		logger.Warnf("[RaydiumCLMM:extractSwapEvent] 无法识别 quote token，跳过: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}

	// 使用池子账户（Accounts[2]）作为交易对标识
	pairAddress := ix.Accounts[2]

	// 构建标准交易事件（TradeEvent）
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, consts.DexRaydiumCLMM)
	if event == nil {
		return nil, current + 1
	}

	return event, result.MaxIndex + 1
}

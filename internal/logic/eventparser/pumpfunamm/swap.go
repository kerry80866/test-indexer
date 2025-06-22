package pumpfunamm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/tools"
)

// extractSwapEvent 解析 Pump.fun AMM 的 swap 事件，构造标准 TradeEvent（BUY / SELL）。
// 示例交易：
// https://solscan.io/tx/3feQ5jvR1ryaCVNwYCRGVhuisk6YCoWpTuCc5vQHgsTLq3sVTErq6Y8np5kAZsJBMnZJfqNBsdjskkCptgNNWLU9
// https://solscan.io/tx/63AWZvhhienFMG7G8BQxt5MEdWR1TNd9t415CkpfV6WHBBtLoYXdzErrmMTqcJ2TrWgmcY5cezhkjgm2otRmfHLG
//
// Pump.fun AMM Swap 指令账户布局：
//  0. Pool
//  1. User
//  2. Global Config
//  3. Token1 (Mint) // base mint
//  4. Token2 (Mint) // quote mint
//  5. UserToken1Account
//  6. UserToken2Account
//  7. PoolToken1Account
//  8. PoolToken2Account
func extractSwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 基本账户数量校验
	if len(ix.Accounts) < 9 {
		logger.Errorf("[PumpfunAMM:extractSwapEvent] 账户数量不足: tx=%s", ctx.TxHashString())
		return -1
	}

	// 提取 Swap 中的转账记录（用户 -> 池子、池子 -> 用户）
	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: 5,
		UserToken2AccountIndex: 6,
		PoolToken1AccountIndex: 7,
		PoolToken2AccountIndex: 8,
	}, 0)
	if result == nil {
		logger.Infof("[PumpfunAMM:extractSwapEvent] 转账结构缺失: tx=%s, ix=%d, inner=%d",
			ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return -1
	}

	// 合约未开源，严格校验 mint 是否匹配
	if !(result.UserToPool.Token == ix.Accounts[3] && result.PoolToUser.Token == ix.Accounts[4] ||
		result.UserToPool.Token == ix.Accounts[4] && result.PoolToUser.Token == ix.Accounts[3]) {
		logger.Errorf("[PumpfunAMM:extractSwapEvent] mint 不匹配: tx=%s, userToPool=%s, poolToUser=%s, token1=%s, token2=%s",
			ctx.TxHashString(), result.UserToPool.Token, result.PoolToUser.Token, ix.Accounts[3], ix.Accounts[4],
		)
		return -1
	}

	// 优先尝试使用自定义优先级的quote token（WSOL、USDC、USDT等）
	quote, ok := tools.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		quote = ix.Accounts[4] // 使用池子默认 quote token
		if result.UserToPool.Token != quote && result.PoolToUser.Token != quote {
			logger.Warnf("[PumpfunAMM:extractSwapEvent] 无法识别 quote token，跳过: tx=%s", ctx.TxHashString())
			return -1
		}
	}

	// ix.Accounts[0] 为该交易对的主池地址
	pairAddress := ix.Accounts[0]

	// 构建标准 TradeEvent
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, true, consts.DexPumpfunAMM)
	if event == nil {
		return -1
	}

	ctx.AddEvent(event)
	return result.MaxIndex + 1
}

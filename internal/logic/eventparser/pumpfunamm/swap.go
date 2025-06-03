package pumpfunamm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
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
) (*core.Event, int) {
	ix := instrs[current]

	// 基本账户数量校验
	if len(ix.Accounts) < 9 {
		logger.Errorf("[PumpfunAMM:extractSwapEvent] 账户数量不足: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}

	// 提取 Swap 中的转账记录（用户 -> 池子、池子 -> 用户）
	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: 5,
		UserToken2AccountIndex: 6,
		PoolToken1AccountIndex: 7,
		PoolToken2AccountIndex: 8,
	}, 0)
	if result == nil {
		logger.Errorf("[PumpfunAMM:extractSwapEvent] 转账结构缺失: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}

	// 合约未开源，严格校验 mint 是否匹配
	if !(result.UserToPool.Token == ix.Accounts[3] && result.PoolToUser.Token == ix.Accounts[4] ||
		result.UserToPool.Token == ix.Accounts[4] && result.PoolToUser.Token == ix.Accounts[3]) {
		logger.Errorf("[PumpfunAMM:extractSwapEvent] mint 不匹配: tx=%s, userToPool=%s, poolToUser=%s, token1=%s, token2=%s",
			ctx.TxHashString(), result.UserToPool.Token, result.PoolToUser.Token, ix.Accounts[3], ix.Accounts[4],
		)
		return nil, current + 1
	}

	// 用户账户判断（5 / 6）
	isUserAccount := func(p types.Pubkey) bool {
		return p == ix.Accounts[5] || p == ix.Accounts[6]
	}
	// 池子账户判断（7 / 8）
	isPoolAccount := func(p types.Pubkey) bool {
		return p == ix.Accounts[7] || p == ix.Accounts[8]
	}
	// 合约未开源，严格校验账户角色是否正确
	if !isUserAccount(result.UserToPool.SrcAccount) || !isPoolAccount(result.UserToPool.DestAccount) ||
		!isPoolAccount(result.PoolToUser.SrcAccount) || !isUserAccount(result.PoolToUser.DestAccount) {
		logger.Errorf("[PumpfunAMM:extractSwapEvent] 用户和池子账户不匹配: tx=%s, userToPool=%s→%s, poolToUser=%s→%s, 用户账户=[%s,%s], 池子账户=[%s,%s]",
			ctx.TxHashString(),
			result.UserToPool.SrcAccount, result.UserToPool.DestAccount,
			result.PoolToUser.SrcAccount, result.PoolToUser.DestAccount,
			ix.Accounts[5], ix.Accounts[6], ix.Accounts[7], ix.Accounts[8],
		)
		return nil, current + 1
	}

	// 推断 quote token（通常为 USDC/USDT 等稳定币）
	quote, ok := utils.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		quote = ix.Accounts[4]
		if result.UserToPool.Token != quote && result.PoolToUser.Token != quote {
			logger.Warnf("[PumpfunAMM:extractSwapEvent] 无法识别 quote token，跳过: tx=%s", ctx.TxHashString())
			return nil, current + 1
		}
	}

	// 在已确认 mint、用户账户和池子账户结构正确的前提下，
	// ix.Accounts[0] 可安全视为该交易对的主池地址（通常是 Pool PDA 或代表账户）
	pairAddress := ix.Accounts[0]

	// 构建标准 TradeEvent
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, consts.DexPumpfunAMM)
	if event == nil {
		return nil, current + 1
	}

	// 返回事件和实际处理的最大指令索引
	return event, result.MaxIndex + 1
}

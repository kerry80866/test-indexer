package raydiumv4

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/utils"
)

// 来源：https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs
// 示例交易：https://solscan.io/tx/48AjDjnqimjaxSPjB2ALDGgFwRvs7iotjnKRyZmiA2z4g7yGgkyxU4eJFdoJgGG3oo9k8M1928zXfedEz8nbMoJV
//
// Raydium V4 Swap 指令账户布局：
//   0. `[]`  SPL Token Program
//   1. `[]`  AMM 主账户（池子地址）
//   2. `[]`  权限 PDA（Program Derived Address）
//   3. `[]`  AMM open_orders 账户
//   4. `[]`  AMM target orders（已废弃，可选）
//   5. `[writable]` 池子 token1（coin）vault
//   6. `[writable]` 池子 token2（pc）vault
//   7. `[]`  市场程序 ID（Serum）
//   8. `[writable]` 市场账户（由 Serum 控制）
//   9. `[writable]` 市场 bids 账户
//  10. `[writable]` 市场 asks 账户
//  11. `[writable]` 市场 event queue
//  12. `[writable]` 市场 coin vault
//  13. `[writable]` 市场 pc vault
//  14. `[]`  市场 vault signer
//  15. `[writable]` 用户 source token 账户（实际支出 token）
//  16. `[writable]` 用户 destination token 账户（实际获得 token）
//  17. `[signer]`   用户钱包账户（仅在账户数为 18 时存在）

// extractSwapEvent 解析 Raydium V4 swap 事件，构造 TradeEvent（BUY / SELL）
func extractSwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]

	// Raydium V4 固定结构为 17 或 18 个账户
	accountCount := len(ix.Accounts)
	if accountCount != 17 && accountCount != 18 {
		logger.Errorf("[RaydiumV4:extractSwapEvent] 账户数量非法: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}
	accountOffset := accountCount - 17

	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: accountOffset + 14,
		UserToken2AccountIndex: accountOffset + 15,
		PoolToken1AccountIndex: accountOffset + 4,
		PoolToken2AccountIndex: accountOffset + 5,
	}, 4)
	if result == nil {
		logger.Errorf("[RaydiumV4:extractSwapEvent] 转账结构缺失: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}

	// 优先尝试使用自定义优先级的quote token（WSOL、USDC、USDT等）
	quote, ok := utils.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		// 使用池子默认quote token
		if result.UserToPool.DestAccount == ix.Accounts[accountOffset+5] {
			quote = result.UserToPool.Token
		} else {
			quote = result.PoolToUser.Token
		}
	}

	pairAddress := ix.Accounts[1]
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, true, consts.DexRaydiumV4)
	if event == nil {
		return nil, current + 1
	}

	return event, result.MaxIndex + 1
}

package raydiumv4

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
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

// extractRaydiumV4SwapEvent 解析 Raydium V4 swap 事件，构造 TradeEvent（BUY / SELL）
func extractRaydiumV4SwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix0 := instrs[current]

	// Raydium V4 固定结构为 17 或 18 个账户
	accountCount := len(ix0.Accounts)
	if accountCount != 17 && accountCount != 18 {
		logx.Infof("RaydiumV4 swap instruction wrong accounts, got=%d, slot=%d txIndex=%d", accountCount, ctx.Tx.TxCtx.Slot, ctx.TxIndex)
		return nil, current + 1
	}
	accountOffset := accountCount - 17

	// 后续必须存在两个 Transfer 指令
	if current+2 >= len(instrs) {
		return nil, current + 1
	}
	ix1, ix2 := instrs[current+1], instrs[current+2]
	if ix1.IxIndex != ix0.IxIndex || ix2.IxIndex != ix0.IxIndex {
		return nil, current + 1
	}

	// 提取并匹配 transfer 对
	userToPool, poolToUser, ok := extractSwapTransfer(ctx, ix0, ix1, ix2, accountOffset)
	if !ok {
		return nil, current + 1
	}

	// 推断 quote token
	quote, ok := determineQuoteToken(ix0, userToPool, poolToUser, accountOffset)
	if !ok {
		logx.Infof("RaydiumV4 determineQuoteToken failed, slot=%d, txIndex=%d", ctx.Tx.TxCtx.Slot, ctx.TxIndex)
		return nil, current + 1
	}

	pairAddress := ix0.Accounts[1]
	event := common.BuildTradeEvent(ctx, ix0, userToPool, poolToUser, pairAddress, quote, consts.DexRaydiumV4)
	if event == nil {
		return nil, current + 1
	}

	return event, current + 3
}

// extractSwapTransfer 从 Raydium Swap 指令后的两条 Transfer 中提取用户与池子的转账方向。
// 返回值：userToPoolTransfer, poolToUserTransfer, 是否匹配成功
func extractSwapTransfer(
	ctx *common.ParserContext,
	ix0 *core.AdaptedInstruction, // Swap 指令（用于获取账户布局）
	ix1 *core.AdaptedInstruction, // 第 1 条 Transfer
	ix2 *core.AdaptedInstruction, // 第 2 条 Transfer
	accountOffset int, // Swap 指令中账户数量的偏移（17 或 18）
) (*common.ParsedTransfer, *common.ParsedTransfer, bool) {
	// 解析第 1 条 Transfer 指令
	transfer1, ok := common.ParseTransferInstruction(ctx, ix1)
	if !ok {
		return nil, nil, false
	}

	// 解析第 2 条 Transfer 指令
	transfer2, ok := common.ParseTransferInstruction(ctx, ix2)
	if !ok {
		return nil, nil, false
	}

	// 获取用户参与的两个 token 账户地址（账户 14 / 15，含偏移）
	user1TokenAccount := ix0.Accounts[accountOffset+14]
	user2TokenAccount := ix0.Accounts[accountOffset+15]

	switch {
	case transfer1.SrcAccount == user1TokenAccount &&
		transfer2.DestAccount == user2TokenAccount:
		// transfer1: user ➝ pool（发出）
		// transfer2: pool ➝ user（收到）
		return transfer1, transfer2, true

	case transfer2.SrcAccount == user2TokenAccount &&
		transfer1.DestAccount == user1TokenAccount:
		// 方向相反
		return transfer2, transfer1, true

	default:
		// 两个 transfer 与用户账户不匹配
		return nil, nil, false
	}
}

// determineQuoteToken 综合判断 quote token：
// - 优先使用 utils.ChooseQuote(tokenA, tokenB) 自定义逻辑；
// - 若无法确定，则根据池子账户结构推断；
// - 若结构不匹配（即非标准 Raydium Swap 格式），则返回 false。
func determineQuoteToken(
	ix *core.AdaptedInstruction,
	userToPool *common.ParsedTransfer,
	poolToUser *common.ParsedTransfer,
	accountOffset int,
) (types.Pubkey, bool) {
	// Step 1: 先用 utils.ChooseQuote 自定义逻辑尝试选择 quote token
	quote, ok := utils.ChooseQuote(userToPool.Token, poolToUser.Token)

	// Step 2: 获取池子两个标准 token 账户（index = 4 和 5）
	poolAccountA := ix.Accounts[accountOffset+4]
	poolAccountB := ix.Accounts[accountOffset+5]

	// Step 3: 验证结构，并兜底处理 ChooseQuote 失败情况
	switch {
	case userToPool.DestAccount == poolAccountA &&
		poolToUser.SrcAccount == poolAccountB:
		// user ➝ poolA，poolB ➝ user：userToPool 是 quote
		if !ok {
			quote = userToPool.Token
		}
		return quote, true

	case userToPool.DestAccount == poolAccountB &&
		poolToUser.SrcAccount == poolAccountA:
		// user ➝ poolB，poolA ➝ user：poolToUser 是 quote
		if !ok {
			quote = poolToUser.Token
		}
		return quote, true

	default:
		// 非法结构，结构校验失败
		return types.Pubkey{}, false
	}
}

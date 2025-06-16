package raydiumcpmm

import (
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/internal/logic/eventparser/common"
	"github.com/dex-indexer-sol/internal/tools"
	"github.com/dex-indexer-sol/pkg/logger"
)

// Raydium CPMM Swap 账户结构（固定顺序）:
//
// 0 - Payer（交易发起人，Writable、Signer、Fee Payer）
// 1 - Authority（Raydium Vault 授权账户）
// 2 - Amm Config（AMM 配置账户）
// 3 - Pool (AMM 池子地址）
// 4 - Input Token Account（用户输入TokenAccount）
// 5 - Output Token Account（用户输出TokenAccount）
// 6 - Input Vault（池子输入TokenVault）
// 7 - Output Vault（池子输出TokenVault）
// 8 - Input Token Program（Token Program，Program 类型）
// 9 - Output Token Program（Token Program，Program 类型）
// 10 - Input Token Mint（输入 Token Mint 地址）
// 11 - Output Token Mint（输出 Token Mint 地址）
//
// 典型示例交易链接：
// swapBaseInput: https://solscan.io/tx/318RwCgKihTL1CtGv2WnSxzVKvMWqJaZAPQE6ZUNA3dSnqtBr8BpLvcjZzf1MvUM71GaRQynKL6EqVFAYhEbKQho
// swapBaseOutput: https://solscan.io/tx/2oYhut5RS46rJqZNcCmzNpnDmfYrWjDbFyskWigWJ1zAuHJ8dRHgiEWPFQb6yzzX1eg3wjuf1WpRe2BbqtpK1YiV
func extractSwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
	methodID uint64,
) int {
	ix := instrs[current]

	// 校验账户数量是否满足预期
	if len(ix.Accounts) < 12 {
		logger.Errorf("[RaydiumCPMM:extractSwapEvent] 账户数量不足: tx=%s", ctx.TxHashString())
		return -1
	}

	// 查找转账记录，匹配用户与池子 Token 账户
	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: 4,
		UserToken2AccountIndex: 5,
		PoolToken1AccountIndex: 6,
		PoolToken2AccountIndex: 7,
	}, 0)
	if result == nil {
		logger.Errorf("[RaydiumCPMM:extractSwapEvent] 转账结构缺失: tx=%s, ix=%d, inner=%d",
			ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return -1
	}

	// 严格校验 mint 地址匹配
	if !((result.UserToPool.Token == ix.Accounts[10] && result.PoolToUser.Token == ix.Accounts[11]) ||
		(result.UserToPool.Token == ix.Accounts[11] && result.PoolToUser.Token == ix.Accounts[10])) {
		logger.Errorf("[RaydiumCPMM:extractSwapEvent] mint 不匹配: tx=%s, userToPool=%s, poolToUser=%s, tokenX=%s, tokenY=%s",
			ctx.TxHashString(), result.UserToPool.Token, result.PoolToUser.Token, ix.Accounts[10], ix.Accounts[11],
		)
		return -1
	}

	// 优先尝试使用自定义优先级的quote token（WSOL、USDC、USDT等）
	quote, ok := tools.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		// fallback：根据方法区分 base 与 quote
		if methodID == SwapBaseOut {
			quote = ix.Accounts[10] // base 出，input 是 quote
		} else {
			quote = ix.Accounts[11] // base 入，output 是 quote
		}
	}

	// 交易对主池地址
	pairAddress := ix.Accounts[3]

	// 构建交易事件
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, true, consts.DexRaydiumCPMM)
	if event == nil {
		return -1
	}

	ctx.AddEvent(event)
	return result.MaxIndex + 1
}

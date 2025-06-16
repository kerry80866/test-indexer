package meteoradlmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/tools"
)

// extractSwapEvent 解析 Meteora DLMM Swap 交易事件。
// 以下几种 Swap 交易的前 8 个账户结构及顺序基本一致，方便统一解析：
//
// 0 - Lp Pair（池子地址）
// 1 - Bin Array Bitmap Extension（Meteora DLMM Program）
// 2 - Reserve X（池子 Token X 的 TokenAccount）
// 3 - Reserve Y（池子 Token Y 的 TokenAccount）
// 4 - User Token In（用户输入的 TokenAccount）
// 5 - User Token Out（用户输出的 TokenAccount）
// 6 - Token X Mint（Token X 的 Mint 地址, BaseToken）
// 7 - Token Y Mint（Token Y 的 Mint 地址, QuoteToken）
//
// 典型示例交易链接：
// swap: https://solscan.io/tx/3CYS44DoNAtXpeA3LvUW12oXr7KpjjsCXypnhaCQaJLXf13JSjhKi9txvRReQ4pyNJT4Cn4QUH5U4K84Q8vGM5Te
// swap2: https://solscan.io/tx/553sphiE347zoBzYfzFrDq99UBYYvE3sFpP2mCcsDBR8V8pSQSZD67UvYUTNra6eCkq74aVsUagfZQumzgTvTQkn
// swapExactOut: https://solscan.io/tx/4drsEfduEhpSX7g34STypsnSFXzySM4vXT3FhA8SEJpohpZEbanuFfCp7nJotK7mZqPBaVKzFMLyLCgVQeuFU9xS
// swapExactOut2: https://solscan.io/tx/3ERMbdJsGXHxwArPoszX3hZr3NiFz3upyK3SqktaKdEoWv6nS9pKGxfhaVn6k9WPe1qfgL9FQBezS3iWd5QHf84F
// swapWithPriceImpact2: https://solscan.io/tx/4852PmY5jz4XmqJVAEeF1kHMvkXYgGZqScMCCFDkH4tJAXt2cwh4k1rnvd5nGJFHyXToCMfyYFRc5C59zecewAdu
func extractSwapEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 校验账户数量是否满足预期
	if len(ix.Accounts) < 8 {
		logger.Errorf("[MeteoraDLMM:extractSwapEvent] 账户数量不足: tx=%s, accounts=%d", ctx.TxHashString(), len(ix.Accounts))
		return -1
	}

	// 查找转账记录，匹配用户与池子 Token 账户
	result := common.FindSwapTransfersByIndex(ctx, instrs, current, &common.SwapInstructionIndex{
		UserToken1AccountIndex: 4,
		UserToken2AccountIndex: 5,
		PoolToken1AccountIndex: 2,
		PoolToken2AccountIndex: 3,
	}, 0)
	if result == nil {
		logger.Errorf("[MeteoraDLMM:extractSwapEvent] 转账结构缺失: tx=%s, ix=%d, inner=%d",
			ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return -1
	}

	// 严格校验 mint 地址匹配（池子TokenX/TokenY mint 地址）
	if !((result.UserToPool.Token == ix.Accounts[6] && result.PoolToUser.Token == ix.Accounts[7]) ||
		(result.UserToPool.Token == ix.Accounts[7] && result.PoolToUser.Token == ix.Accounts[6])) {
		logger.Errorf("[MeteoraDLMM:extractSwapEvent] mint 不匹配: tx=%s, userToPool=%s, poolToUser=%s, tokenX=%s, tokenY=%s",
			ctx.TxHashString(), result.UserToPool.Token, result.PoolToUser.Token, ix.Accounts[6], ix.Accounts[7],
		)
		return -1
	}

	// 优先尝试使用自定义优先级的quote token（WSOL、USDC、USDT等）
	quote, ok := tools.ChooseQuote(result.UserToPool.Token, result.PoolToUser.Token)
	if !ok {
		// fallback 使用池子默认的 Quote Token (Token Y Mint)
		quote = ix.Accounts[7]
		if result.UserToPool.Token != quote && result.PoolToUser.Token != quote {
			logger.Warnf("[MeteoraDLMM:extractSwapEvent] 无法识别 quote token，跳过: tx=%s", ctx.TxHashString())
			return -1
		}
	}

	// 交易对主池地址
	pairAddress := ix.Accounts[0]

	// 构建交易事件
	event := common.BuildTradeEvent(ctx, ix, result.UserToPool, result.PoolToUser, pairAddress, quote, true, consts.DexMeteoraDLMM)
	if event == nil {
		return -1
	}

	ctx.AddEvent(event)
	return result.MaxIndex + 1
}

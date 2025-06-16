package meteoradlmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
)

// Method1
// 示例交易：https://solscan.io/tx/uzMqzyXc1XXcxC7efSGjJFpBxVKEPAqjqmm1P2kyXgi184QjAqhbAHvJDJggBntYTPQvM3aqDLmDY1ibLydZTZZ
//
// Meteora DLMM - RemoveLiquidity 指令账户布局：
//
// #0  - Position                      // 用户头寸账户（记录 LP 头寸）
// #1  - Lb Pair                       // 流动性池账户（如 JUP-WSOL）
// #2  - Bin Array Bitmap Extension    // Bin 位图扩展账户（流动性分布）
// #3  - User Token X Account          // 用户 Token X 的账户
// #4  - User Token Y Account          // 用户 Token Y 的账户
// #5  - Reserve X                     // 池子 Token X 储备账户
// #6  - Reserve Y                     // 池子 Token Y 储备账户
// #7  - Token X Mint                  // Token X 的 Mint（如 JUP）
// #8  - Token Y Mint                  // Token Y 的 Mint（如 WSOL）
// #9  - Bin Array Lower               // Bin 下界数组账户
// #10 - Bin Array Upper               // Bin 上界数组账户
// #11 - Sender                        // 用户主账户（手续费支付者）
// #12 - Token X Program               // SPL Token 程序（Token X）
// #13 - Token Y Program               // SPL Token 程序（Token Y）
// #14 - Event Authority               // 事件权限 PDA（事件记录校验）
// #15 - Program                       // Meteora DLMM 主程序地址
func extractEventForRemoveLiquidity(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 12
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:RemoveLiquidity] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "RemoveLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // Lb Pair
		TokenMint1Index:        7,  // Token X Mint
		TokenMint2Index:        8,  // Token Y Mint
		UserWalletIndex:        11, // Sender
		UserToken1AccountIndex: 3,  // 用户 Token X Account
		UserToken2AccountIndex: 4,  // 用户 Token Y Account
		UserLpAccountIndex:     -1, // 无 LP Token Account
		PoolToken1AccountIndex: 5,  // Reserve X
		PoolToken2AccountIndex: 6,  // Reserve Y
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// Method2
// 示例交易：https://solscan.io/tx/Sw8AZPDdqyCQWefVmJg3xW12o4d49avRvKv7DmPVw1TTjMhXFxfTpCRjDQpyCTXVkRUCA9jCy4uSyGwafoZsNGX
//
// Meteora DLMM - RemoveLiquidity2 指令账户布局：
//
// #0  - Position                      // 用户头寸账户（记录流动性头寸）
// #1  - Lb Pair                       // 流动性池账户（如 SONIC-USDC）
// #2  - Bin Array Bitmap Extension    // Bin 位图扩展账户
// #3  - User Token X Account          // 用户的 Token X SPL Token 账户
// #4  - User Token Y Account          // 用户的 Token Y SPL Token 账户
// #5  - Reserve X                     // 池子的 Token X 储备账户
// #6  - Reserve Y                     // 池子的 Token Y 储备账户
// #7  - Token X Mint                  // Token X 的 Mint（如 SONIC）
// #8  - Token Y Mint                  // Token Y 的 Mint（如 USDC）
// #9  - Sender                        // 用户主账户（签名者、手续费支付者）
// #10 - Token X Program               // SPL Token 程序（Token X）
// #11 - Token Y Program               // SPL Token 程序（Token Y）
// #12 - Memo Program                  // Memo 程序（可忽略）
// #13 - Event Authority               // 事件权限 PDA（事件记录校验）
// #14 - Program                       // Meteora DLMM 主程序地址
// #15 - Account (Unidentified)        // 暂未知用途（额外账户）
// #16 - Account (Unidentified)        // 暂未知用途（额外账户）
func extractEventForRemoveLiquidity2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 10
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:RemoveLiquidity2] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "RemoveLiquidity2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // Lb Pair
		TokenMint1Index:        7,  // Token X Mint
		TokenMint2Index:        8,  // Token Y Mint
		UserWalletIndex:        9,  // Sender
		UserToken1AccountIndex: 3,  // 用户 Token X 账户
		UserToken2AccountIndex: 4,  // 用户 Token Y 账户
		UserLpAccountIndex:     -1, // 无 LP Token 账户
		PoolToken1AccountIndex: 5,  // Reserve X
		PoolToken2AccountIndex: 6,  // Reserve Y
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// Method3
// 示例交易：https://solscan.io/tx/3K5xGqts4Qx2h21HWprMNRJcS5cuyE1wsWr9mzKVKtP5eEeyJWpbdkz7DatTMGaQp3oJibPqLtEE2wDnGNRUMJK6
//
// Meteora DLMM - RemoveLiquidityByRange 指令账户布局：
//
// #0  - Position                    // 用户头寸账户（记录 LP 持仓）
// #1  - Lb Pair                     // 流动性池账户（如 MASK-WSOL）
// #2  - Bin Array Bitmap Extension // Bin 位图扩展账户（流动性分布）
// #3  - User Token X Account       // 用户的 Token X 账户（如 MASK）
// #4  - User Token Y Account       // 用户的 Token Y 账户（如 WSOL）
// #5  - Reserve X                  // 池子的 Token X 储备账户
// #6  - Reserve Y                  // 池子的 Token Y 储备账户
// #7  - Token X Mint               // Token X 的 Mint（MASK）
// #8  - Token Y Mint               // Token Y 的 Mint（WSOL）
// #9  - Bin Array Lower            // Bin 下界数组账户（起始价格）
// #10 - Bin Array Upper            // Bin 上界数组账户（终止价格）
// #11 - Sender                     // 用户主账户（签名者、手续费支付者）
// #12 - Token X Program            // SPL Token 程序地址（Token X）
// #13 - Token Y Program            // SPL Token 程序地址（Token Y）
// #14 - Event Authority            // 事件权限 PDA（事件认证）
// #15 - Program                    // Meteora DLMM 主程序地址
func extractEventForRemoveLiquidityByRange(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 12
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:RemoveLiquidityByRange] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "RemoveLiquidityByRange", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // Lb Pair
		TokenMint1Index:        7,  // Token X Mint
		TokenMint2Index:        8,  // Token Y Mint
		UserWalletIndex:        11, // Sender
		UserToken1AccountIndex: 3,  // User Token X
		UserToken2AccountIndex: 4,  // User Token Y
		UserLpAccountIndex:     -1, // 无 LP Token
		PoolToken1AccountIndex: 5,  // Reserve X
		PoolToken2AccountIndex: 6,  // Reserve Y
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// Method4
// 示例交易：https://solscan.io/tx/5pV6eRWNgf9BbuWDfhiAZsz51pXoW7KkJqmLGatsuXARPWzP8kwtcPAuz8zHed5GstAW9jjRoKfb3GazpboHiqYL
//
// Meteora DLMM - RemoveLiquidityByRange2 指令账户布局：
//
// #0  - Position                    // 用户头寸账户（记录 LP 持仓）
// #1  - Lb Pair                     // 流动性池账户（如 AP-WSOL）
// #2  - Bin Array Bitmap Extension // Bin 位图扩展账户
// #3  - User Token X Account       // 用户的 Token X 账户（如 AP）
// #4  - User Token Y Account       // 用户的 Token Y 账户（如 WSOL）
// #5  - Reserve X                  // 池子的 Token X 储备账户
// #6  - Reserve Y                  // 池子的 Token Y 储备账户
// #7  - Token X Mint               // Token X 的 Mint（AP）
// #8  - Token Y Mint               // Token Y 的 Mint（WSOL）
// #9  - Sender                     // 用户主账户（交易签名者 / 手续费支付者）
// #10 - Token X Program            // SPL Token 程序地址（Token X）
// #11 - Token Y Program            // SPL Token 程序地址（Token Y）
// #12 - Memo Program               // Memo 程序（可选）
// #13 - Event Authority            // 事件权限 PDA（事件验证标志）
// #14 - Program                    // Meteora DLMM 主程序地址
// #15 - Extra Account              // 额外账户（可能为未来扩展预留）
func extractEventForRemoveLiquidityByRange2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 10
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:RemoveLiquidityByRange2] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "RemoveLiquidityByRange2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // #1 - Lb Pair
		TokenMint1Index:        7,  // #7 - Token X Mint
		TokenMint2Index:        8,  // #8 - Token Y Mint
		UserWalletIndex:        9,  // #9 - Sender
		UserToken1AccountIndex: 3,  // #3 - User Token X
		UserToken2AccountIndex: 4,  // #4 - User Token Y
		UserLpAccountIndex:     -1, // 无 LP Token
		PoolToken1AccountIndex: 5,  // #5 - Reserve X
		PoolToken2AccountIndex: 6,  // #6 - Reserve Y
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// Method5
// 示例交易：https://solscan.io/tx/3rGR5kQvByHFnXxqJC8VaYAMXrEn5H4N12ecuDjfFZgDkbqnP1bdbo63frsJwNigiPejb5oQQQvFtMt53qJRNEkG
//
// Meteora DLMM - RemoveAllLiquidity 指令账户布局：
//
// #0  - Position                    // 用户流动性头寸账户（用于记录 LP 持仓）
// #1  - Lb Pair                     // 流动性池主账户（PVE-WSOL）
// #2  - Bin Array Bitmap Extension // Bin 位图扩展账户
// #3  - User Token X Account       // 用户的 Token X（PVE）账户
// #4  - User Token Y Account       // 用户的 Token Y（WSOL）账户
// #5  - Reserve X                  // 池子的 Token X 储备账户
// #6  - Reserve Y                  // 池子的 Token Y 储备账户
// #7  - Token X Mint               // Token X 的 Mint（PVE）
// #8  - Token Y Mint               // Token Y 的 Mint（WSOL）
// #9  - Bin Array Lower            // Bin Lower 数组（用于精度控制）
// #10 - Bin Array Upper            // Bin Upper 数组
// #11 - Sender                     // 用户主账户（签名者 / 费用支付）
// #12 - Token X Program            // SPL Token 程序地址（Token X）
// #13 - Token Y Program            // SPL Token 程序地址（Token Y）
// #14 - Event Authority            // 事件权限 PDA（用于事件验证）
// #15 - Program                    // Meteora DLMM 主程序地址
func extractEventForRemoveAllLiquidity(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 12
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:RemoveAllLiquidity] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "RemoveAllLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // #1 - Lb Pair
		TokenMint1Index:        7,  // #7 - Token X Mint
		TokenMint2Index:        8,  // #8 - Token Y Mint
		UserWalletIndex:        11, // #11 - Sender
		UserToken1AccountIndex: 3,  // #3 - User Token X
		UserToken2AccountIndex: 4,  // #4 - User Token Y
		UserLpAccountIndex:     -1, // 无 LP Token
		PoolToken1AccountIndex: 5,  // #5 - Reserve X
		PoolToken2AccountIndex: 6,  // #6 - Reserve Y
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

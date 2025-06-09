package meteoradlmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
)

// Method1
// 示例交易：https://solscan.io/tx/3kwXqfbmpPBwz7XV1oG94fNooVShaWmSmtrPoNKyCsXwUUsJhAFjq2D1BSK5zGmxBY3WQbtKMZniQGkvSEa5iHbF
//
// Meteora DLMM - AddLiquidity2 指令账户布局：
//
// #0  - Position                             // 流动性头寸账户
// #1  - Lb Pair                              // 流动性池主账户
// #2  - Bin Array Bitmap Extension           // Bin 位图扩展账户
// #3  - User Token X                         // 用户的 Token X 账户
// #4  - User Token Y                         // 用户的 Token Y 账户
// #5  - Reserve X                            // 池子的 Token X 账户
// #6  - Reserve Y                            // 池子的 Token Y 账户
// #7  - Token Mint X                         // 第一个代币的 Mint（如 DYORHUB）
// #8  - Token Mint Y                         // 第二个代币的 Mint（如 WSOL）
// #9  - Sender                               // 操作发起人（Signer + Fee Payer）
// #10 - Token Program X                      // 第一个代币的 Token 程序地址
// #11 - Token Program Y                      // 第二个代币的 Token 程序地址
// #12 - Event Authority                      // 事件权限 PDA
// #13 - Program                              // Meteora DLMM 程序地址
// #14 - 附加账户 1                            // 额外辅助账户，可能为临时 PDA（具体需结合代码或交易分析）
// #15 - 附加账户 2                            // 同上，进一步策略辅助账户（如记录策略或索引状态）
func extractEventForAddLiquidity2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 10
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidity2] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "AddLiquidity2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,
		TokenMint1Index:        7,
		TokenMint2Index:        8,
		UserWalletIndex:        9,
		UserToken1AccountIndex: 3,
		UserToken2AccountIndex: 4,
		UserLpAccountIndex:     -1,
		PoolToken1AccountIndex: 5,
		PoolToken2AccountIndex: 6,
		LpMintIndex:            -1,
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// Method2
// 示例交易：https://solscan.io/tx/2idCVHNaxYX7psVZ1eqdtBpYzw9w5kRUnhH2DLC9zL7SG6hUXeLSUFSa894F32PruY8irqDfkRBD167cwiqW3XsN
//
// Meteora DLMM - AddLiquidityByWeight 指令账户布局：
//
// #0  - Position                            // 流动性头寸账户（用户持仓位置）
// #1  - Lb Pair                             // 流动性池主账户
// #2  - Bin Array Bitmap Extension          // Bin 位图扩展账户
// #3  - User Token X                        // 用户的 Token X 账户
// #4  - User Token Y                        // 用户的 Token Y 账户
// #5  - Reserve X                           // 池子的 Token X 储备账户
// #6  - Reserve Y                           // 池子的 Token Y 储备账户
// #7  - Token Mint X                        // 第一个代币的 Mint（如 COSTCO）
// #8  - Token Mint Y                        // 第二个代币的 Mint（如 WSOL）
// #9  - Bin Array Lower                     // Bin 下界数组，用于分配权重
// #10 - Bin Array Upper                     // Bin 上界数组，用于分配权重
// #11 - Sender                              // 操作发起人（Signer + Fee Payer）
// #12 - Token Program X                     // 第一个代币的 Token 程序地址
// #13 - Token Program Y                     // 第二个代币的 Token 程序地址
// #14 - Event Authority                     // 事件权限 PDA
// #15 - Program                             // Meteora DLMM 程序地址
func extractEventForAddLiquidityByWeight(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 12
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidityByWeight] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "AddLiquidityByWeight", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // Lb Pair
		TokenMint1Index:        7,  // Token Mint X
		TokenMint2Index:        8,  // Token Mint Y
		UserWalletIndex:        11, // Sender（Signer + Fee Payer）
		UserToken1AccountIndex: 3,  // User Token X
		UserToken2AccountIndex: 4,  // User Token Y
		UserLpAccountIndex:     -1, // 没有 LP Token 账户
		PoolToken1AccountIndex: 5,  // Reserve X
		PoolToken2AccountIndex: 6,  // Reserve Y
		LpMintIndex:            -1, // 没有 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// Method3
// 示例交易：https://solscan.io/tx/8kMZVDAxQUQkciXc5z9igqPDDTh3erDZEcpb11cP734zEBArBbqu8u2txjHoDCp8o9j49KLC9rAyJnPXmE64864
//
// Meteora DLMM - AddLiquidityByStrategy 指令账户布局：
//
// #0  - Position                            // 用户的流动性头寸账户（用于记录 LP 持仓）
// #1  - Lb Pair                             // 流动性池主账户
// #2  - Bin Array Bitmap Extension          // Bin 位图扩展账户（用于扩展 Bin 空间）
// #3  - User Token X                        // 用户的 Token X SPL 账户
// #4  - User Token Y                        // 用户的 Token Y SPL 账户
// #5  - Reserve X                           // 池子的 Token X 储备账户
// #6  - Reserve Y                           // 池子的 Token Y 储备账户
// #7  - Token Mint X                        // 第一个代币的 Mint（如 DTR）
// #8  - Token Mint Y                        // 第二个代币的 Mint（如 WSOL）
// #9  - Bin Array Lower                     // Bin 下界数组（用于控制流动性分布）
// #10 - Bin Array Upper                     // Bin 上界数组
// #11 - Sender                              // 发起人钱包地址（Signer + Fee Payer）
// #12 - Token Program X                     // 第一个代币的 SPL Token 程序地址
// #13 - Token Program Y                     // 第二个代币的 SPL Token 程序地址
// #14 - Event Authority                     // 事件权限 PDA（用于事件溯源标记）
// #15 - Program                             // Meteora DLMM 程序地址本身
func extractEventForAddLiquidityByStrategy(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 12
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidityByStrategy] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "AddLiquidityByStrategy", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // Lb Pair
		TokenMint1Index:        7,  // Token Mint X
		TokenMint2Index:        8,  // Token Mint Y
		UserWalletIndex:        11, // Sender
		UserToken1AccountIndex: 3,  // User Token X
		UserToken2AccountIndex: 4,  // User Token Y
		UserLpAccountIndex:     -1, // 无 LP 账户
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
// 示例交易：https://solscan.io/tx/492GGJaZ3fdTDLtJDnhwPabR5njcQ2MK99LSZByzQadPD9JaHcz6Bw9PVBgt3CJoe1iaAyK9uMb4qUWUpthnAA6g
//
// Meteora DLMM - AddLiquidityByStrategy2 指令账户布局：
//
// #0  - Position                            // 用户的流动性头寸账户
// #1  - Lb Pair                             // 流动性池账户（如 KBBB-WSOL）
// #2  - Bin Array Bitmap Extension          // Bin 位图扩展账户
// #3  - User Token X                        // 用户的 Token X SPL 账户
// #4  - User Token Y                        // 用户的 Token Y SPL 账户
// #5  - Reserve X                           // 池子的 Token X 储备账户
// #6  - Reserve Y                           // 池子的 Token Y 储备账户
// #7  - Token Mint X                        // 第一个代币的 Mint（如 KBBB）
// #8  - Token Mint Y                        // 第二个代币的 Mint（如 WSOL）
// #9  - Sender                              // 发起人钱包地址（Signer + Fee Payer）
// #10 - Token Program X                     // 第一个代币的 SPL Token 程序地址
// #11 - Token Program Y                     // 第二个代币的 SPL Token 程序地址
// #12 - Event Authority                     // 事件权限 PDA（标记事件来源）
// #13 - Program                             // Meteora DLMM 程序地址本身
// #14 - Account                             // 附加账户（用途待确认，可能为策略辅助 PDA）
// #15 - Account                             // 附加账户（用途待确认，可能为临时状态或权限控制）
func extractEventForAddLiquidityByStrategy2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 10
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidityByStrategy2] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "AddLiquidityByStrategy2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // Lb Pair
		TokenMint1Index:        7,  // Token Mint X
		TokenMint2Index:        8,  // Token Mint Y
		UserWalletIndex:        9,  // Sender
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

// Method5
// 示例交易：https://solscan.io/tx/3UPXH5xbbVRzh28VgYMBWbcdAiJM3mE9K9k3zHuMYALiZN2e2QukEPYta7EeNrCkBPFP7nBghfhjjyPiXon8Z2Hv
//
// Meteora DLMM - AddLiquidityByStrategyOneSide 指令账户布局：
//
// #0  - Position                       // 用户头寸账户（记录 LP 头寸），也是签名者
// #1  - Lb Pair                        // 流动性池账户（如 LAUNCHCOIN-WSOL 市场）
// #2  - Bin Array Bitmap Extension     // Bin 位图扩展账户（用于精细流动性分布）
// #3  - User Token                     // 用户提供单边流动性的 SPL Token 账户
// #4  - Reserve                        // 池子中对应 Token 的储备账户
// #5  - Token Mint                     // 上述 Token 对应的 Mint（如 WSOL）
// #6  - Bin Array Lower                // Bin 下界数组账户（表示添加的 Bin 范围起点）
// #7  - Bin Array Upper                // Bin 上界数组账户（表示添加的 Bin 范围终点）
// #8  - Sender                         // 用户主账户，签名者 + 手续费支付者
// #9  - Token Program                  // SPL Token 程序地址
// #10 - Event Authority                // 事件权限 PDA，用于事件记录校验
// #11 - Program                        // Meteora DLMM 主程序地址
func extractEventForAddLiquidityByStrategyOneSide(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 9
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidityByStrategyOneSide] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	pool := ix.Accounts[1]
	user := ix.Accounts[8]
	providedMint := ix.Accounts[5]

	// Step 1: 找到池子的两个 token mint 和对应的余额
	var poolBalances []*core.TokenBalance
	for _, b := range ctx.Balances {
		if b.PostOwner == pool {
			poolBalances = append(poolBalances, b)
		}
	}
	if len(poolBalances) != 2 {
		return -1 // 不符合 DLMM 池结构
	}

	var otherMint types.Pubkey
	var otherPoolBalance *core.TokenBalance
	if poolBalances[0].Token == providedMint {
		otherMint = poolBalances[1].Token
		otherPoolBalance = poolBalances[1]
	} else if poolBalances[1].Token == providedMint {
		otherMint = poolBalances[0].Token
		otherPoolBalance = poolBalances[0]
	} else {
		return -1 // providedMint 不在池子里
	}

	// Step 2: 判断能否确认 quote
	if _, ok := utils.ChooseQuote(providedMint, otherMint); !ok {
		return -1
	}

	// Step 3: 找到用户的另一个 token 账户（即未提供的一侧）
	var userOtherTokenAccount types.Pubkey
	found := false
	for _, b := range ctx.Balances {
		if b.PostOwner == user && b.Token == otherMint {
			userOtherTokenAccount = b.TokenAccount
			found = true
			break
		}
	}
	if !found {
		return -1
	}

	// Step 4: 填补另一边的账户
	originalLen := len(ix.Accounts)
	ix.Accounts = append(ix.Accounts, otherMint, userOtherTokenAccount, otherPoolBalance.TokenAccount)

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "AddLiquidityByStrategyOneSide", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,
		TokenMint1Index:        5,
		TokenMint2Index:        originalLen,
		UserWalletIndex:        8,
		UserToken1AccountIndex: 3,
		UserToken2AccountIndex: originalLen + 1,
		UserLpAccountIndex:     -1,
		PoolToken1AccountIndex: 4,
		PoolToken2AccountIndex: originalLen + 2,
		LpMintIndex:            -1,
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// Method6
// 示例交易：https://solscan.io/tx/3M22Q7yWrs3PjDRq8LW4Kb23Zr2qMSuVEUfyudeBcsJPVN2jn8b9L8GrG6avujALLQenDwNavzsjstdCQw3dfMty
//
// Meteora DLMM - AddLiquidity 指令账户布局：
//
// #0  - Position                       // 用户头寸账户（记录流动性头寸），签名者
// #1  - Lb Pair                        // 流动性池账户（如 USDC-WSOL）
// #2  - Bin Array Bitmap Extension     // Bin 位图扩展账户
// #3  - User Token X                   // 用户 Token X 的账户（如 USDC）
// #4  - User Token Y                   // 用户 Token Y 的账户（如 WSOL）
// #5  - Reserve X                      // 池子 Token X 储备账户
// #6  - Reserve Y                      // 池子 Token Y 储备账户
// #7  - Token X Mint                   // Token X 的 Mint（如 USDC）
// #8  - Token Y Mint                   // Token Y 的 Mint（如 WSOL）
// #9  - Bin Array Lower                // Bin 下界数组账户（流动性添加的价格起点）
// #10 - Bin Array Upper                // Bin 上界数组账户（流动性添加的价格终点）
// #11 - Sender                         // 用户主账户，签名者 + 手续费支付者
// #12 - Token X Program                // Token X 所属的 SPL Token 程序
// #13 - Token Y Program                // Token Y 所属的 SPL Token 程序
// #14 - Event Authority                // 事件权限 PDA（事件记录认证标志）
func extractEventForAddLiquidity(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 12
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidity] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexMeteoraDLMM, "AddLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       1,  // Lb Pair
		TokenMint1Index:        7,  // Token X Mint
		TokenMint2Index:        8,  // Token Y Mint
		UserWalletIndex:        11, // Sender
		UserToken1AccountIndex: 3,  // User Token X
		UserToken2AccountIndex: 4,  // User Token Y
		UserLpAccountIndex:     -1, // 无 LP token
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

// Method7
// 示例交易：https://solscan.io/tx/4ag2xA4r9bEGC6cuXCx3WcceuoeC2JsV432RTvXGrJes5zMSeBY51AoynktuHRABW4oszqzNAQehLodF58p2qT3z
//
// Meteora DLMM - AddLiquidityOneSidePrecise 指令账户布局：
//
// #0  - Position                     // 用户头寸账户（记录此次添加的流动性头寸）
// #1  - Lb Pair                      // 流动性池账户（如 BRSTL-USDC）
// #2  - Bin Array Bitmap Extension   // Bin 位图扩展账户
// #3  - User Token                   // 用户提供的单边 Token 账户（如 BRSTL 或 USDC）
// #4  - Reserve                      // 池子中对应的单边 Token 储备账户
// #5  - Token Mint                   // 用户提供的 Token 的 Mint（如 BRSTL）
// #6  - Bin Array Lower              // Bin 下界数组账户（流动性添加的价格起点）
// #7  - Bin Array Upper              // Bin 上界数组账户（流动性添加的价格终点）
// #8  - Sender                       // 用户主账户，签名者 + 手续费支付者
// #9  - Token Program                // SPL Token 程序地址
// #10 - Event Authority              // 事件权限 PDA，用于记录链上事件
// #11 - Program                      // Meteora DLMM 主程序 ID
func extractEventForAddLiquidityOneSidePrecise(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	const requiredAccounts = 9
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidityOneSidePrecise] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	// 暂不处理 AddLiquidityOneSide 系列（单边添加流动性），
	// 因其仅提供一侧 Token，需特殊事件结构支持。
	return -1
}

// Method8
// 示例交易：https://solscan.io/tx/2f2NLpMqXy69cBp5yDUNx1bNPNcLQw1GKgU5h1p9JgUtrv4LrPjabPQWDT2TekaXCn5KHLetuCm4M9dnN4Gz8iJa
//
// Meteora DLMM - AddLiquidityOneSide 指令账户布局：
//
// #0  - Position                     // 用户头寸账户，记录此次添加流动性的状态
// #1  - Lb Pair                      // 流动性池账户（如 ZBCN-USDC）
// #2  - Bin Array Bitmap Extension   // Bin 位图扩展账户（追踪哪些 Bin 有流动性）
// #3  - User Token                   // 用户提供的 Token 账户（仅提供一侧，如 ZBCN）
// #4  - Reserve                      // 池子中对应的 Token 储备账户
// #5  - Token Mint                   // 用户提供的 Token 的 Mint（如 ZBCN）
// #6  - Bin Array Lower              // Bin 下界数组账户（价格区间下限）
// #7  - Bin Array Upper              // Bin 上界数组账户（价格区间上限）
// #8  - Sender                       // 用户钱包账户（Signer + Fee Payer）
// #9  - Token Program                // SPL Token 程序地址
// #10 - Event Authority              // 用于事件记录的 PDA（Program Derived Address）
// #11 - Program                      // Meteora DLMM 主程序地址
func extractEventForAddLiquidityOneSide(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	const requiredAccounts = 9
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidityOneSide] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	// 暂不处理 AddLiquidityOneSide 系列（单边添加流动性），
	// 因其仅提供一侧 Token，需特殊事件结构支持。
	return -1
}

// Method9
// 示例交易：https://solscan.io/tx/59k5XMDoN6iZN9EwoLuY8WtQxsNDGcbVzg8Zr9r4Pf8xQbXJyWajfDbndhrvWPNkVroGBEziUhRBpHFpwn7p7M6p
//
// Meteora DLMM - AddLiquidityOneSidePrecise2 指令账户布局：
//
// #0  - Position                      // 用户头寸账户，记录本次添加流动性的状态
// #1  - Lb Pair                       // 流动性池账户（如 AIKI-WSOL）
// #2  - Bin Array Bitmap Extension   // Bin 位图扩展账户，用于追踪活跃 Bin 状态
// #3  - User Token                   // 用户提供的一侧 Token 账户（如 AIKI）
// #4  - Reserve                      // 池子中对应的储备账户（匹配 Token Mint）
// #5  - Token Mint                   // 用户提供 Token 的 Mint（如 AIKI）
// #6  - Sender                       // 用户钱包账户（Signer + Fee Payer）
// #7  - Token Program                // SPL Token 程序地址
// #8  - Event Authority              // 用于事件记录的 PDA（Program Derived Address）
// #9  - Program                      // Meteora DLMM 主程序地址
// #10 - Account                      // 附加账户 1（功能待查，一般为内部辅助状态）
// #11 - Account                      // 附加账户 2（功能待查，可能与策略/奖励等机制相关）
func extractEventForAddLiquidityOneSidePrecise2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	const requiredAccounts = 8
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:AddLiquidityOneSidePrecise2] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	// 暂不处理 AddLiquidityOneSide 系列（单边添加流动性），
	// 因其仅提供一侧 Token，需特殊事件结构支持。
	return -1
}

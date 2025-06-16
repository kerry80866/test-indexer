package meteoradlmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
)

// 示例交易：https://solscan.io/tx/o7D5om3waD7MXBH3mkGRjUxW1UyuwPu3gmLHW2hYjL9j6vqwecqGowrgsJvFJ445bjZZuXhWk2KhrRAmSDr42ev
//
// Meteora DLMM - InitializePair2 指令账户布局：
//
// #0  - Lb Pair                               // 池子主账户
// #1  - Bin Array Bitmap Extension            // Bin 扩展数据（用于扩展 bin 位图）
// #2  - Token Mint X                          // 第一个代币（如 PUFF）
// #3  - Token Mint Y                          // 第二个代币（如 WSOL）
// #4  - Reserve X                             // 第一个代币池子账户（Pool 1）
// #5  - Reserve Y                             // 第二个代币池子账户（Pool 2）
// #6  - Oracle                                // 预言机账户（通常是池子内部使用）
// #7  - Preset Parameter                      // 池子参数预设值（tick spacing、fee 等）
// #8  - Funder                                // 池子创建者（钱包地址，Signer + Fee Payer）
// #9  - Token Badge X                         // 第一个代币的徽章账户（Badge PDA）
// #10 - Token Badge Y                         // 第二个代币的徽章账户
// #11 - Token Program X                       // 第一个代币的 SPL Token 程序
// #12 - Token Program Y                       // 第二个代币的 SPL Token 程序
// #13 - System Program                        // 系统程序
// #14 - Event Authority                       // 事件授权 PDA
// #15 - Program                               // Meteora DLMM 程序自身
func extractEventForInitializePair2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 实际使用到了 index 0~12
	const requiredAccounts = 13
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:InitializePair2] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	createPooleEvent := common.ExtractCreatePoolEvent(ctx, ix, consts.DexMeteoraDLMM, "InitializePair2", &common.CreatePoolLayout{
		PoolAddressIndex:   0,
		TokenMint1Index:    2,
		TokenMint2Index:    3,
		TokenProgram1Index: 11,
		TokenProgram2Index: 12,
		PoolVault1Index:    4,
		PoolVault2Index:    5,
		UserWalletIndex:    8,
	})
	if createPooleEvent == nil {
		return -1
	}

	ctx.AddEvent(createPooleEvent)
	return current + 1
}

// 示例交易：https://solscan.io/tx/hjJyjps53KtdPRiqkkdzKRV9SvfdUFe4vdZeTqQWYTifME4M7skgnr4M9z8MFJzVBUSJmSeuxAXPFpXNCWg3sgB
//
// Meteora DLMM - InitializeCustomPair 指令账户布局：
//
// #0  - Lb Pair                               // 池子主账户（Meteora (DOGGER-USDC) Market）
// #1  - Bin Array Bitmap Extension            // Bin 位图扩展账户，用于存储更多 bin 数据
// #2  - Token Mint X                          // 第一个代币的 Mint（如 DOGGER）
// #3  - Token Mint Y                          // 第二个代币的 Mint（如 USDC）
// #4  - Reserve X                             // 第一个代币的资金池账户（Pool X）
// #5  - Reserve Y                             // 第二个代币的资金池账户（Pool Y）
// #6  - Oracle                                // Oracle 账户，用于内部价格控制（通常不对外）
// #7  - User Token X                          // 用户持有的 Token X 的账户（用于初始化时可能转账）
// #8  - Funder                                // 创建者钱包地址（Signer + Fee Payer）
// #9  - Token Program                         // SPL Token 程序地址（主 SPL Token）
// #10 - System Program                        // 系统程序地址，用于账户初始化等系统操作
// #11 - User Token Y                          // 用户持有的 Token Y 的账户（用于初始化时可能转账）
// #12 - Event Authority                       // Meteora DLMM 事件权限 PDA（用于链上事件标记）
// #13 - Program                               // Meteora DLMM 程序自身
func extractEventForInitializeCustomPair(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 14 {
		logger.Errorf("[MeteoraDLMM:InitializeCustomPair] 指令账户长度不足: got=%d, expect>=14, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	createPooleEvent := common.ExtractCreatePoolEvent(ctx, ix, consts.DexMeteoraDLMM, "InitializeCustomPair", &common.CreatePoolLayout{
		PoolAddressIndex:   0,
		TokenMint1Index:    2,
		TokenMint2Index:    3,
		TokenProgram1Index: 9,
		TokenProgram2Index: -1,
		PoolVault1Index:    4,
		PoolVault2Index:    5,
		UserWalletIndex:    8,
	})
	if createPooleEvent == nil {
		return -1
	}

	ctx.AddEvent(createPooleEvent)
	return current + 1
}

// 示例交易：https://solscan.io/tx/iGqUmXi5KimTit3BNWRDZoiQpHmC7M5eu81yHBcizJBafLzdH2YuTdXpBjFzoHJMUe5Zhy8TP3RVdJ8STHQecXK
//
// Meteora DLMM - InitializeCustomPair2 指令账户布局：
//
// #0  - Lb Pair                               // 池子主账户（如：Meteora (TOKENX-WSOL) Market）
// #1  - Bin Array Bitmap Extension            // Bin 位图扩展账户（支持更多 bin 数据）
// #2  - Token Mint X                          // 第一个代币的 Mint（如 TOKENX）
// #3  - Token Mint Y                          // 第二个代币的 Mint（如 WSOL）
// #4  - Reserve X                             // 第一个代币的池子账户（Pool X）
// #5  - Reserve Y                             // 第二个代币的池子账户（Pool Y）
// #6  - Oracle                                // Oracle 账户（用于内部定价逻辑）
// #7  - User Token X                          // 用户的 Token X 账户
// #8  - Funder                                // 创建者钱包地址（Signer + Fee Payer）
// #9  - Token Badge X                         // 第一个代币的徽章账户（Badge PDA）
// #10 - Token Badge Y                         // 第二个代币的徽章账户
// #11 - Token Program X                       // 第一个代币的 SPL Token 程序
// #12 - Token Program Y                       // 第二个代币的 SPL Token 程序
// #13 - System Program                        // 系统程序（用于初始化账户等操作）
// #14 - User Token Y                          // 用户的 Token Y 账户
// #15 - Event Authority                       // Meteora DLMM 事件权限 PDA
// #16 - Program                               // Meteora DLMM 程序地址本身
func extractEventForInitializeCustomPair2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 实际使用到了 index 0~12
	const requiredAccounts = 13
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:InitializeCustomPair2] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	createPooleEvent := common.ExtractCreatePoolEvent(ctx, ix, consts.DexMeteoraDLMM, "InitializeCustomPair2", &common.CreatePoolLayout{
		PoolAddressIndex:   0,
		TokenMint1Index:    2,
		TokenMint2Index:    3,
		TokenProgram1Index: 11,
		TokenProgram2Index: 12,
		PoolVault1Index:    4,
		PoolVault2Index:    5,
		UserWalletIndex:    8,
	})
	if createPooleEvent == nil {
		return -1
	}

	ctx.AddEvent(createPooleEvent)
	return current + 1
}

// 示例交易：https://solscan.io/tx/3VS2kS4fZeEF3mtwN5ec7U83aY1kikhiiKeA7ujanKohVCcFNUSSAxdQMzknmQGyG91gLW52c8uBCpquvwC6n1fy
//
// Meteora DLMM - InitializePermissionLbPair 指令账户布局：
//
// #0  - Base                                   // 用作 lb_pair 派生种子的 Signer PDA（权限管理基础）
// #1  - Lb Pair                                // 流动性池主账户，Writable
// #2  - Bin Array Bitmap Extension             // 扩展的 Bin 位图账户，Writable
// #3  - Token Mint X                           // 第一个代币的 Mint（如 TOKENX）
// #4  - Token Mint Y                           // 第二个代币的 Mint（如 USDC）
// #5  - Reserve X                              // 第一个代币的池子 TokenAccount，Writable
// #6  - Reserve Y                              // 第二个代币的池子 TokenAccount，Writable
// #7  - Oracle                                 // Oracle PDA，用于内部定价或 TWAP，Writable
// #8  - Admin                                  // 管理员钱包地址（Signer + Fee Payer），Writable
// #9  - Token Badge X                          // Token X 对应的徽章账户（Badge PDA）
// #10 - Token Badge Y                          // Token Y 对应的徽章账户
// #11 - Token Program X                        // SPL Token 程序地址（通常为主 SPL Token）
// #12 - Token Program Y                        // SPL Token 程序地址
// #13 - System Program                         // 系统程序（初始化账户所需）
// #14 - Rent                                   // Rent 程序地址（部分账户创建需要）
// #15 - Event Authority                        // DLMM 事件记录权限 PDA
// #16 - Program                                // 当前程序本身（Meteora DLMM）
func extractEventForInitializePermissionLbPair(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 实际使用到了 index 0~12
	const requiredAccounts = 13
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[MeteoraDLMM:InitializePermissionLbPair] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	createPooleEvent := common.ExtractCreatePoolEvent(ctx, ix, consts.DexMeteoraDLMM, "InitializePermissionLbPair", &common.CreatePoolLayout{
		PoolAddressIndex:   1,
		TokenMint1Index:    3,
		TokenMint2Index:    4,
		TokenProgram1Index: 11,
		TokenProgram2Index: 12,
		PoolVault1Index:    5,
		PoolVault2Index:    6,
		UserWalletIndex:    8,
	})
	if createPooleEvent == nil {
		return -1
	}

	ctx.AddEvent(createPooleEvent)
	return current + 1
}

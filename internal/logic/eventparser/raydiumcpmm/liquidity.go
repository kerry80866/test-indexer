package raydiumcpmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pb"
)

// 示例交易：https://solscan.io/tx/2Gaqukq8fCjR5SMy9XKPp2LqYZXk1RckD1HibUs5dAw2TBTT4rQFbHXGk7DpqFZm1jVxkTb75mr93UDdyXzs1x5g
//
// Raydium CPMM 添加流动性指令账户布局：
//
// #0  - Owner                                 // 用户钱包地址（Signer + Fee Payer）
// #1  - Authority                             // Raydium 池子 Authority PDA
// #2  - Pool State                            // 池子主账户（含池子状态信息）
// #3  - Owner LP Token Account                // 用户接收 LP Token 的账户
// #4  - Token 0 Account                       // 用户提供的 Token0 账户
// #5  - Token 1 Account                       // 用户提供的 Token1 账户
// #6  - Token 0 Vault                         // 池子中 Token0 的存储账户
// #7  - Token 1 Vault                         // 池子中 Token1 的存储账户
// #8  - Token Program                         // SPL Token 标准程序地址（Tokenkeg...）
// #9  - Token Program 2022                    // SPL Token-2022 程序地址（TokenzQd...）
// #10 - Vault 0 Mint                          // Token0 的 Mint 地址
// #11 - Vault 1 Mint                          // Token1 的 Mint 地址
// #12 - LP Mint                               // LP Token 的 Mint 地址
func extractAddLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 13 {
		logger.Errorf("[RaydiumCPMM:AddLiquidity] 账户数不足: got=%d, expect>=13, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, mintEvent, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCPMM, "AddLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       2,
		TokenMint1Index:        10,
		TokenMint2Index:        11,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 4,
		UserToken2AccountIndex: 5,
		UserLpAccountIndex:     3,
		PoolToken1AccountIndex: 6,
		PoolToken2AccountIndex: 7,
		LpMintIndex:            12,
	})
	if liquidityEvent == nil || mintEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	ctx.AddEvent(mintEvent)
	return maxIndex + 1
}

// 示例交易：https://solscan.io/tx/3j36kyPWRBPQbc81kaB5MbhUycu6ZX294QuhSDNTFis62MP3hZBDsrrbqoXKY6Phc5bVZw3GemsHrhh2KFxYmaHN
//
// Raydium CPMM 移除流动性指令账户布局：
//
// #0  - Owner                          // 用户钱包地址
// #1  - Authority                      // Raydium 池子的 Authority PDA
// #2  - Pool State                     // 池子主账户（含池子状态信息）
// #3  - Owner LP Token Account         // 用户接收 LP Token 的账户
// #4  - Token 0 Account                // 用户提供的 Token0 账户
// #5  - Token 1 Account                // 用户提供的 Token1 账户
// #6  - Token 0 Vault                  // 池子中 Token0 的存储账户
// #7  - Token 1 Vault                  // 池子中 Token1 的存储账户
// #8  - Token Program                  // SPL Token 标准程序地址
// #9 - Token Program 2022            	// SPL Token-2022 程序地址
// #10 - Vault 0 Mint                  	// Token0 的 Mint
// #11 - Vault 1 Mint                  	// Token1 的 Mint
// #12 - LP Mint                       	// LP Token 的 Mint 账户
// #13 - Memo Program (可选)           	// Memo Program v2（可忽略）
func extractRemoveLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 13 {
		logger.Errorf("[RaydiumCPMM:RemoveLiquidity] 指令账户长度不足: got=%d, expect>=13, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, burnEvent, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCPMM, "RemoveLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       2,
		TokenMint1Index:        10,
		TokenMint2Index:        11,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 4,
		UserToken2AccountIndex: 5,
		UserLpAccountIndex:     3,
		PoolToken1AccountIndex: 6,
		PoolToken2AccountIndex: 7,
		LpMintIndex:            12,
	})
	if liquidityEvent == nil || burnEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	ctx.AddEvent(burnEvent)
	return maxIndex + 1
}

// 示例交易：https://solscan.io/tx/wyaQtPVNKpbKkMAkC8dWdk7tvYTq9uK99RuebPhTCTAXg9o9Sef8LgqSjvsb6LYzwhuk1EQnfDgf5dMqb2LMSmX
//
// Raydium CPMM initialize指令账户布局：
//
// #0  - Creator                           // 创建者地址（Signer + Fee Payer）
// #1  - Amm Config                        // Raydium 配置账户（常量地址）
// #2  - Authority                         // 池子的 Authority PDA
// #3  - Pool State                        // 池子状态账户（包含交易对配置）
// #4  - Token 0 Mint                      // Token0 的 Mint
// #5  - Token 1 Mint                      // Token1 的 Mint
// #6  - LP Mint                           // LP Token 的 Mint 账户
// #7  - Creator Token 0                   // 创建者提供的 Token0 账户
// #8  - Creator Token 1                   // 创建者提供的 Token1 账户
// #9  - Creator LP Token                  // 创建者接收 LP Token 的账户
// #10 - Token 0 Vault                     // 池子中 Token0 的存储账户
// #11 - Token 1 Vault                     // 池子中 Token1 的存储账户
// #12 - Create Pool Fee                   // 创建池子手续费接收账户（Raydium 官方账户）
// #13 - Observation State                 // 价格观察账户（用于 TWAP 等分析）
// #14 - Token Program                     // SPL Token 程序地址
// #15 - Token 0 Program                   // Token0 使用的 Token 程序地址（可能是 Token 或 Token2022）
// #16 - Token 1 Program                   // Token1 使用的 Token 程序地址（可能是 Token 或 Token2022）
// #17 - Associated Token Program          // ATA 程序地址
// #18 - System Program                    // Solana 系统程序地址
// #19 - Rent                              // Rent 程序地址
func extractInitializeEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 12 {
		logger.Errorf("[RaydiumCPMM:Initialize] 账户数不足: got=%d, expect>=12, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, mintEvent, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCPMM, "Initialize", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       3,
		TokenMint1Index:        4,
		TokenMint2Index:        5,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 7,
		UserToken2AccountIndex: 8,
		UserLpAccountIndex:     9,
		PoolToken1AccountIndex: 10,
		PoolToken2AccountIndex: 11,
		LpMintIndex:            6,
	})
	if liquidityEvent == nil || mintEvent == nil {
		return -1
	}

	// 克隆构建 CreatePoolEvent（金额清零）
	createPoolEvent := common.CloneLiquidityEvent(liquidityEvent)
	createPoolEvent.EventType = uint32(pb.EventType_CREATE_POOL)
	createPoolEvent.Event.GetLiquidity().Type = pb.EventType_CREATE_POOL
	createPoolEvent.Event.GetLiquidity().TokenAmount = 0
	createPoolEvent.Event.GetLiquidity().QuoteTokenAmount = 0

	// 为 AddLiquidityEvent 设置新的 EventID（避免与 CreatePoolEvent 冲突）
	liquidityEvent.ID += 1
	liquidityEvent.Event.GetLiquidity().EventId = liquidityEvent.ID

	ctx.AddEvent(createPoolEvent)
	ctx.AddEvent(liquidityEvent)

	if mintEvent.ID != liquidityEvent.ID {
		ctx.AddEvent(mintEvent)
	} else {
		logger.Warnf("[RaydiumCPMM:Initialize2] MintToEvent ID 冲突，已跳过: tx=%s", ctx.TxHashString())
	}
	return maxIndex + 1
}

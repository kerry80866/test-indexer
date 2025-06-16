package raydiumclmm

import (
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/internal/logic/eventparser/common"
	"github.com/dex-indexer-sol/pkg/logger"
)

// 示例交易：https://solscan.io/tx/29RtWoTifJDAEoV3gihDb1vyp1WnJnaiDK6aCCMf9BmZ1LmJQCmC6sMuFR6fuJiJJeGXPjchxxv5EkyhnrYseJoh
//
// Raydium CLMM IncreaseLiquidity（V1）指令账户布局：
//
// #0  - Nft Owner                      // NFT 持有者（即 position 所属用户，Signer）
// #1  - Nft Account                    // NFT Token Account（持有 LP Position NFT 的账户）
// #2  - Pool State                     // 池子状态账户记录 tokenA/tokenB、tick spacing、fee 等）
// #3  - Protocol Position              // 协议层 position PDA（由 NFT 推导出）
// #4  - Personal Position              // 用户层 position PDA（可独立更新 Liquidity）
// #5  - Tick Array Lower               // 低价区间 tick array（CLMM 特有）
// #6  - Tick Array Upper               // 高价区间 tick array（CLMM 特有）
// #7  - Token Account 0                // 用户提供的 token0 账户用于添加流动性）
// #8  - Token Account 1                // 用户提供的 token1 账户
// #9  - Token Vault 0                  // 池子中 token0 的托管账户
// #10 - Token Vault 1                  // 池子中 token1 的托管账户
// #11 - Token Program                  // SPL Token 标准程序地址
// #12 - Account                        // 临时 PDA（增加流动性指令中的辅助账户，用于权限校验等）
func extractEventForIncreaseLiquidity(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 11 {
		logger.Errorf("[RaydiumCLMM:IncreaseLiquidity] 账户数不足: got=%d, expect>=11, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCLMM, "IncreaseLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       2,
		TokenMint1Index:        -1,
		TokenMint2Index:        -1,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 7,
		UserToken2AccountIndex: 8,
		UserLpAccountIndex:     -1,
		PoolToken1AccountIndex: 9,
		PoolToken2AccountIndex: 10,
		LpMintIndex:            -1,
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// 示例交易：https://solscan.io/tx/4CDvRkrmCLzXshDjJWJ7hZuhKvre3tdpGbfyCtvntBNvDy3EjFqtQbuStQ2EsMkpkhLGKEbimeTX26UMhQBMdsEs
//
// Raydium CLMM IncreaseLiquidityV2 指令账户布局：
//
// #0  - Nft Owner                          // 用户钱包地址（Signer + Fee Payer），也是 Position NFT 的拥有者
// #1  - Nft Account                        // 用户用于接收 Position NFT 的 TokenAccount
// #2  - Pool State                         // 池子主账户，包含当前价格、流动性等全局状态信息
// #3  - Protocol Position                  // 协议级 Position 信息账户存储统一的头部或映射）
// #4  - Personal Position                  // 用户个人的 Position 状态账户储存具体流动性区间等）
// #5  - Tick Array Lower                   // 流动性范围下边界对应的 TickArray
// #6  - Tick Array Upper                   // 流动性范围上边界对应的 TickArray
// #7  - Token Account 0                    // 用户提供的 Token0 账户
// #8  - Token Account 1                    // 用户提供的 Token1 账户
// #9  - Token Vault 0                      // 池子中存储 Token0 的 Vault
// #10 - Token Vault 1                      // 池子中存储 Token1 的 Vault
// #11 - Token Program                      // SPL Token 标准程序地址
// #12 - Token Program 2022                 // SPL Token-2022 程序地址
// #13 - Vault 0 Mint                       // Token0 的 Mint 地址
// #14 - Vault 1 Mint                       // Token1 的 Mint 地址
func extractEventForIncreaseLiquidityV2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 15 {
		logger.Errorf("[RaydiumCLMM:IncreaseLiquidityV2] 账户数不足: got=%d, expect>=15, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCLMM, "IncreaseLiquidityV2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       2,
		TokenMint1Index:        13,
		TokenMint2Index:        14,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 7,
		UserToken2AccountIndex: 8,
		UserLpAccountIndex:     -1,
		PoolToken1AccountIndex: 9,
		PoolToken2AccountIndex: 10,
		LpMintIndex:            -1,
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// 示例交易：https://solscan.io/tx/JGEk7gfti8g5LFPqwyx8nuwdiBLgKJWuUreKEABq9Q4PJ6fZwVCTvLqN2iU7qdw51VCMMLtRWCZo6wc27dxEDiN
//
// Raydium CLMM openPositionWithToken22Nft 指令账户布局：
//
// #0  - Payer                    			// 出钱的（负责转出 Token、支付交易费，一般是用户自己）
// #1  - Position NFT Owner       			// 得利的（收到仓位 NFT、拥有流动性头寸）
// #2  - Position NFT Mint                  // 新创建的 Position NFT 的 mint 账户
// #3  - Position NFT Account               // 用户接收 Position NFT 的 TokenAccount
// #4  - Pool State                         // 池子状态账户记录 tickSpacing、token 对、fee 等）
// #5  - Protocol Position                  // 协议级 Position PDA（全局映射，通常与 NFT Mint 相关）
// #6  - Tick Array Lower                   // 流动性范围下边界对应的 TickArray
// #7  - Tick Array Upper                   // 流动性范围上边界对应的 TickArray
// #8  - Personal Position                  // 用户级别的 Position PDA，实际记录用户该 Position 的信息
// #9  - Token Account 0                    // 用户提供的 Token0
// #10 - Token Account 1                    // 用户提供的 Token1
// #11 - Token Vault 0                      // 池子中托管 Token0 的 Vault 账户
// #12 - Token Vault 1                      // 池子中托管 Token1 的 Vault 账户
// #13 - Rent                               // 系统租金程序地址（只读）
// #14 - System Program                     // Solana 系统程序地址（Program）
// #15 - Token Program                      // SPL Token 标准程序地址（Program）
// #16 - Associated Token Program           // ATA 程序地址（Program）
// #17 - Token Program 2022                 // SPL Token-2022 程序地址（Program）
// #18 - Vault 0 Mint                       // Token0 的 Mint
// #19 - Vault 1 Mint                       // Token1 的 Mint
func extractEventForOpenPositionWithToken22Nft(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 20 {
		logger.Errorf("[RaydiumCLMM:OpenPositionWithToken22Nft] 账户数不足: got=%d, expect>=20, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCLMM, "OpenPositionWithToken22Nft", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       4,
		TokenMint1Index:        18,
		TokenMint2Index:        19,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 9,
		UserToken2AccountIndex: 10,
		UserLpAccountIndex:     -1,
		PoolToken1AccountIndex: 11,
		PoolToken2AccountIndex: 12,
		LpMintIndex:            -1,
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// 示例交易：https://solscan.io/tx/SdWrJbjiWDA28oR5Y6XALUeSSgDVMKSMJvksUYH4xJdxhGAHPWKBdeYaNhokKd5QsNGDDvj6v3Sy5utcvgUo7d1

// Raydium CLMM openPositionV2 指令账户布局：
//
// #0  - Payer                    			// 出钱的（负责转出 Token、支付交易费，一般是用户自己）
// #1  - Position NFT Owner       			// 得利的（收到仓位 NFT、拥有流动性头寸）
// #2  - Position NFT Mint                  // Position NFT 的 Mint 账户
// #3  - Position NFT Account               // 用户接收 Position NFT 的 TokenAccount
// #4  - Metadata Account                   // Position NFT 对应的 Metadata PDA（Writable，用于写入元数据）
// #5  - Pool State                         // CLMM 池状态账户，包含 token 对、tickSpacing、fee 等信息
// #6  - Protocol Position                  // 协议层 Position PDA，基于 NFT Mint 派生
// #7  - Tick Array Lower                   // 流动性范围下边界的 TickArray
// #8  - Tick Array Upper                   // 流动性范围上边界的 TickArray
// #9  - Personal Position                  // 用户个人的 Position PDA，记录该仓位的具体状态
// #10 - Token Account 0                    // 用户提供的 Token0 账户
// #11 - Token Account 1                    // 用户提供的 Token1 账户
// #12 - Token Vault 0                      // 池子中托管 Token0 的 Vault 账户
// #13 - Token Vault 1                      // 池子中托管 Token1 的 Vault 账户
// #14 - Rent                               // Rent 程序账户
// #15 - System Program                     // Solana 系统程序（Program）
// #16 - Token Program                      // SPL Token 标准程序（Program）
// #17 - Associated Token Program           // Associated Token Account 程序（Program）
// #18 - Metadata Program                   // Metaplex Metadata 程序（Program），用于写入 Position NFT 元数据
// #19 - Token Program 2022                 // SPL Token-2022 程序地址（Program）
// #20 - Vault 0 Mint                       // Token0 的 Mint
// #21 - Vault 1 Mint                       // Token1 的 Mint
func extractEventForOpenPositionV2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 22 {
		logger.Errorf("[RaydiumCLMM:OpenPositionV2] 账户数不足: got=%d, expect>=22, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCLMM, "OpenPositionV2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       5,
		TokenMint1Index:        20,
		TokenMint2Index:        21,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 10,
		UserToken2AccountIndex: 11,
		UserLpAccountIndex:     -1,
		PoolToken1AccountIndex: 12,
		PoolToken2AccountIndex: 13,
		LpMintIndex:            -1,
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

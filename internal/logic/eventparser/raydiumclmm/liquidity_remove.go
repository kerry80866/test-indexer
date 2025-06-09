package raydiumclmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
)

// 示例交易：https://solscan.io/tx/4joy9CZnCcpMdTZtbH3jCskhRGxvU69Fvtsr8NkCXcE5tpjq7R7qYWnv8QNVwcH9U5iMEtZTm4e5jmq1Scjsb2c8
//
// Raydium CLMM DecreaseLiquidityV2 指令账户布局：
//
// #0  - Nft Owner                         // Position NFT 拥有者地址
// #1  - Nft Account                       // 持有 Position NFT 的 TokenAccount
// #2  - Personal Position                 // 用户级别的 Position PDA
// #3  - Pool State                        // 池子状态账户（记录 tickSpacing、fee 等）
// #4  - Protocol Position                 // 协议级 Position PDA（由 NFT Mint 派生）
// #5  - Token Vault 0                     // 池子中托管 Token0 的 Vault
// #6  - Token Vault 1                     // 池子中托管 Token1 的 Vault
// #7  - Tick Array Lower                  // 流动性区间下界对应的 TickArray
// #8  - Tick Array Upper                  // 流动性区间上界对应的 TickArray
// #9  - Recipient Token Account 0         // 用户接收 Token0 的账户
// #10 - Recipient Token Account 1         // 用户接收 Token1 的账户
// #11 - Token Program                     // SPL Token 标准程序地址
// #12 - Token Program 2022                // SPL Token-2022 程序地址（用于新型扩展 token）
// #13 - Memo Program                      // Memo Program v2（可选，用于链上备注）
// #14 - Vault 0 Mint                      // Token0 的 Mint（如 GRIFFAIN）
// #15 - Vault 1 Mint                      // Token1 的 Mint（如 arc）
func extractEventForDecreaseLiquidityV2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 16 {
		logger.Errorf("[RaydiumCLMM:DecreaseLiquidityV2] 指令账户长度不足: got=%d, expect>=16, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCLMM, "DecreaseLiquidityV2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       3,
		TokenMint1Index:        14,
		TokenMint2Index:        15,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 9,
		UserToken2AccountIndex: 10,
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

// 示例交易：https://solscan.io/tx/3yoF4Mo4wxr2pQXxY3gJ2qFmv2b9sLu1udZc84Zh6BGRiBP4ffx763yPdcdpfuuFR6tgKR5jctWvKQbU4WQDdCXw
//
// Raydium CLMM DecreaseLiquidity 指令账户布局：
//
// #0  - Nft Owner                         // Position NFT 的拥有者，一般为用户地址
// #1  - Nft Account                       // 存放 Position NFT 的 TokenAccount
// #2  - Personal Position                 // 用户个人 Position PDA，记录该仓位的状态
// #3  - Pool State                        // CLMM 池状态账户（记录当前价格、tick spacing 等）
// #4  - Protocol Position                 // 协议级 Position PDA（由 NFT Mint 派生）
// #5  - Token Vault 0                     // 池子中托管 Token0 的 Vault 账户
// #6  - Token Vault 1                     // 池子中托管 Token1 的 Vault 账户
// #7  - Tick Array Lower                  // 流动性范围下边界 TickArray
// #8  - Tick Array Upper                  // 流动性范围上边界 TickArray
// #9  - Recipient Token Account 0        // 用户接收 Token0 的账户
// #10 - Recipient Token Account 1        // 用户接收 Token1 的账户
// #11 - Token Program                     // SPL Token 程序地址
// #12 - Account                           // 临时 PDA（权限检查或中间辅助账户）
// #13 - Account                           // 临时 PDA（权限检查或中间辅助账户）
func extractEventForDecreaseLiquidity(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 11 {
		logger.Errorf("[RaydiumCLMM:DecreaseLiquidity] 指令账户长度不足: got=%d, expect>=11, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexRaydiumCLMM, "DecreaseLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       3,
		TokenMint1Index:        -1,
		TokenMint2Index:        -1,
		UserWalletIndex:        0,
		UserToken1AccountIndex: 9,
		UserToken2AccountIndex: 10,
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

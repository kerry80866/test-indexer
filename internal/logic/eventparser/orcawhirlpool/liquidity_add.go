package orcawhirlpool

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
)

// 示例交易：https://solscan.io/tx/4hP6pnim4Mdaqe3HGsnU23VETYXYKtpP7xfknDn6MUaoshdZTqR1zkPqs2JiK7DMBTGsoddbFbmekhc56XNcARXV
//
// OrcaWhirlpool - IncreaseLiquidity 指令账户布局：
//
// #0  - Whirlpool                 // 流动性池主账户（SOL-Fartcoin）
// #1  - Token Program             // SPL Token 程序地址
// #2  - Position Authority        // 用户主账户（签名者，用于授权 Position 操作）
// #3  - Position                  // 用户流动性头寸账户（记录 LP 持仓）
// #4  - Position Token Account    // 用户持有 Position NFT 的 Token Account
// #5  - Token Owner Account A     // 用户的 Token A（SOL）账户
// #6  - Token Owner Account B     // 用户的 Token B（Fartcoin）账户
// #7  - Token Vault A             // 池子的 Token A 储备账户（SOL）
// #8  - Token Vault B             // 池子的 Token B 储备账户（Fartcoin）
// #9  - Tick Array Lower          // 对应 LP 下边界价格的 TickArray
// #10 - Tick Array Upper          // 对应 LP 上边界价格的 TickArray
func extractEventForIncreaseLiquidity(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 9
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[OrcaWhirlpool:IncreaseLiquidity] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexOrcaWhirlpool, "IncreaseLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       0,  // #0 - Whirlpool
		TokenMint1Index:        -1, // 无 Mint 信息
		TokenMint2Index:        -1,
		UserWalletIndex:        2,  // #2 - 用户签名者
		UserToken1AccountIndex: 5,  // #5 - 用户 Token A 账户
		UserToken2AccountIndex: 6,  // #6 - 用户 Token B 账户
		UserLpAccountIndex:     -1, // 无需校验 LP NFT
		PoolToken1AccountIndex: 7,  // #7 - 池子 Token A 储备账户
		PoolToken2AccountIndex: 8,  // #8 - 池子 Token B 储备账户
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// 示例交易：https://solscan.io/tx/66ckXzkKUoqajkUEEbXk1kKhZp7XJu53Bghd3crC68nYeEgHC75nTJny3r9pTjYNuY4QdNHHi2zXGAfprrS7TpRd
//
// OrcaWhirlpool - IncreaseLiquidityV2 指令账户布局：
//
// #0  - Whirlpool                 // 流动性池账户（WSOL-Fartcoin）
// #1  - Token Program A           // Token A 对应的 SPL Token 程序（一般为 WSOL）
// #2  - Token Program B           // Token B 对应的 SPL Token 程序（例如 Fartcoin）
// #3  - Memo Program              // Memo 程序，用于链上附加备注（可忽略）
// #4  - Position Authority        // 用户主账户（签名者，授权操作）
// #5  - Position                  // 用户的 LP 头寸账户（由 NFT 表示）
// #6  - Position Token Account    // 用户持有 Position NFT 的 TokenAccount
// #7  - Token Mint A              // Token A 的 Mint（如 WSOL）
// #8  - Token Mint B              // Token B 的 Mint（如 Fartcoin）
// #9  - Token Owner Account A     // 用户的 WSOL 账户
// #10 - Token Owner Account B     // 用户的 Fartcoin 账户
// #11 - Token Vault A             // 池子的 WSOL 储备账户
// #12 - Token Vault B             // 池子的 Fartcoin 储备账户
// #13 - Tick Array Lower          // TickArray：定义价格区间下边界
// #14 - Tick Array Upper          // TickArray：定义价格区间上边界
func extractEventForIncreaseLiquidityV2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 13
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[OrcaWhirlpool:IncreaseLiquidityV2] 账户数不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexOrcaWhirlpool, "IncreaseLiquidityV2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       0,  // #0 - Whirlpool
		TokenMint1Index:        7,  // #7 - Token Mint A (WSOL)
		TokenMint2Index:        8,  // #8 - Token Mint B (Fartcoin)
		UserWalletIndex:        4,  // #4 - Position Authority
		UserToken1AccountIndex: 9,  // #9 - 用户 WSOL 账户
		UserToken2AccountIndex: 10, // #10 - 用户 Fartcoin 账户
		UserLpAccountIndex:     -1, // 无需校验 LP NFT
		PoolToken1AccountIndex: 11, // #11 - WSOL 储备账户
		PoolToken2AccountIndex: 12, // #12 - Fartcoin 储备账户
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

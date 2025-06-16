package orcawhirlpool

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
)

// 示例交易：https://solscan.io/tx/5Q1PhZ669qxWhCsibKh6GR3yGZCvJinBzrwF5LXB3PavLjaGA6k31AMtfSFedVLSrgFeQjQ61avi4xhVNd6qL9Un
//
// OrcaWhirlpool - DecreaseLiquidity 指令账户布局：
//
// #0  - Whirlpool                 // 流动性池账户（如 SOL-USDC）
// #1  - Token Program             // SPL Token 程序
// #2  - Position Authority        // 用户主账户（签名者）
// #3  - Position                  // 用户的 LP 头寸账户（表示 LP NFT）
// #4  - Position Token Account    // 用户持有 LP NFT 的 TokenAccount
// #5  - Token Owner Account A     // 用户的 Token A 账户（如 WSOL）
// #6  - Token Owner Account B     // 用户的 Token B 账户（如 USDC）
// #7  - Token Vault A             // 池子的 Token A 储备账户
// #8  - Token Vault B             // 池子的 Token B 储备账户
// #9  - Tick Array Lower          // TickArray：流动性范围下界
// #10 - Tick Array Upper          // TickArray：流动性范围上界
func extractEventForRemoveDecreaseLiquidity(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 9
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[OrcaWhirlpool:DecreaseLiquidity] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexOrcaWhirlpool, "DecreaseLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       0,  // Whirlpool
		TokenMint1Index:        -1, // 无 Mint 索引
		TokenMint2Index:        -1, // 无 Mint 索引
		UserWalletIndex:        2,  // Position Authority
		UserToken1AccountIndex: 5,  // 用户 Token A 账户
		UserToken2AccountIndex: 6,  // 用户 Token B 账户
		UserLpAccountIndex:     -1, // 忽略 Lp 检查
		PoolToken1AccountIndex: 7,  // 池子 Token A 储备账户
		PoolToken2AccountIndex: 8,  // 池子 Token B 储备账户
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

// 示例交易：https://solscan.io/tx/4SyHcE39eFiwzKBuZE7VaJmfY5AtV4ps5BEExRk7NWPayFPZKnfcLERCYQiZdLJxvnRscEVSfgkwtZ7gGsbSPiEQ
//
// OrcaWhirlpool - DecreaseLiquidityV2 指令账户布局：
//
// #0  - Whirlpool                 // 流动性池账户（如 WSOL-STARTUP）
// #1  - Token Program A           // Token A 的 SPL Token 程序（通常为标准 SPL Token 程序）
// #2  - Token Program B           // Token B 的 SPL Token 程序
// #3  - Memo Program              // Memo 程序（一般用于附加交易标记，可忽略）
// #4  - Position Authority        // 用户主账户（签名者）
// #5  - Position                  // 用户的 LP 头寸账户
// #6  - Position Token Account    // 用户持有 LP NFT 的账户
// #7  - Token Mint A              // Token A 的 Mint（如 WSOL）
// #8  - Token Mint B              // Token B 的 Mint（如 STARTUP）
// #9  - Token Owner Account A     // 用户的 Token A 接收账户
// #10 - Token Owner Account B     // 用户的 Token B 接收账户
// #11 - Token Vault A             // 池子的 Token A 储备账户
// #12 - Token Vault B             // 池子的 Token B 储备账户
// #13 - Tick Array Lower          // TickArray：流动性范围下界
// #14 - Tick Array Upper          // TickArray：流动性范围上界
func extractEventForRemoveDecreaseLiquidityV2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 13
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[OrcaWhirlpool:DecreaseLiquidityV2] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	liquidityEvent, _, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexOrcaWhirlpool, "DecreaseLiquidityV2", &common.LiquidityLayout{
		RequireBothTransfer:    false,
		PoolAddressIndex:       0,  // Whirlpool
		TokenMint1Index:        7,  // Token Mint A
		TokenMint2Index:        8,  // Token Mint B
		UserWalletIndex:        4,  // Position Authority
		UserToken1AccountIndex: 9,  // 用户 Token A 账户
		UserToken2AccountIndex: 10, // 用户 Token B 账户
		UserLpAccountIndex:     -1, // 忽略 Lp 检查
		PoolToken1AccountIndex: 11, // 池子 Token A 账户
		PoolToken2AccountIndex: 12, // 池子 Token B 账户
		LpMintIndex:            -1, // 无 LP Mint
	})
	if liquidityEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	return maxIndex + 1
}

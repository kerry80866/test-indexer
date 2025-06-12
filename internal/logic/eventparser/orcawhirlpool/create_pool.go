package orcawhirlpool

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pkg/logger"
)

// 示例交易：https://solscan.io/tx/Miz5QpAfzCXHAuaZB9erP2xjy66PgGzJokyaS8yBzkgozzPGafivYLbNhBx1f4cMu14cfifEEPfeDNDKqVjfMYi
//
// OrcaWhirlpool - InitializePool 指令账户布局：
//
// #0  - Whirlpools Config         // 全局配置账户
// #1  - Token Mint A              // Token A 的 Mint（如 WSOL）
// #2  - Token Mint B              // Token B 的 Mint（如 MAT）
// #3  - Funder                    // 资金提供者（用于创建池子和初始租金支付）
// #4  - Whirlpool                 // 即将创建的流动性池账户（如 WSOL-MAT）
// #5  - Token Vault A             // 池子的 Token A 储备账户（如 WSOL）
// #6  - Token Vault B             // 池子的 Token B 储备账户（如 MAT）
// #7  - Fee Tier                  // 费率层账户（表示该池子的手续费参数）
// #8  - Token Program             // SPL Token 程序地址
// #9  - System Program            // 系统程序地址（用于分配账户）
// #10 - Rent                      // Rent 程序（用于计算账户所需最小租金）
func extractEventForInitializePool(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 9
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[OrcaWhirlpool:InitializePool] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	createPooleEvent := common.ExtractCreatePoolEvent(ctx, ix, consts.DexOrcaWhirlpool, "InitializePool", &common.CreatePoolLayout{
		PoolAddressIndex:   4,  // #4 - Whirlpool
		TokenMint1Index:    1,  // #1 - Token Mint A
		TokenMint2Index:    2,  // #2 - Token Mint B
		TokenProgram1Index: 8,  // #8 - SPL Token 程序
		TokenProgram2Index: -1, //
		PoolVault1Index:    5,  // #5 - Token Vault A
		PoolVault2Index:    6,  // #6 - Token Vault B
		UserWalletIndex:    3,  // #3 - Funder
	})
	if createPooleEvent == nil {
		return -1
	}

	ctx.AddEvent(createPooleEvent)
	return current + 1
}

// 示例交易：https://solscan.io/tx/t1kFBWA9e4iNytwDunyy2xu48UgUxfkj1kPLm8d9cvLM6bRYwusr6A2A8K26GPwt4QL2mv3gaerpS4xUJMd8pbg
//
// OrcaWhirlpool - InitializePoolV2 指令账户布局：
//
// #0  - Whirlpools Config         // 全局配置账户（包含全局参数）
// #1  - Token Mint A              // Token A 的 Mint（如 WSOL）
// #2  - Token Mint B              // Token B 的 Mint（如 KINGELON）
// #3  - Token Badge A             // Token A 的扩展标记账户（如用于支持 Token-2022 扩展功能）
// #4  - Token Badge B             // Token B 的扩展标记账户
// #5  - Funder                    // 资金提供者（负责支付初始化时的 rent）
// #6  - Whirlpool                 // 即将创建的池子账户（WSOL-KINGELON）
// #7  - Token Vault A             // 池子的 Token A 储备账户（如 WSOL）
// #8  - Token Vault B             // 池子的 Token B 储备账户（如 KINGELON）
// #9  - Fee Tier                  // 手续费等级账户（定义该池的手续费率）
// #10 - Token Program A           // Token A 所使用的 Token 程序（SPL Token）
// #11 - Token Program B           // Token B 所使用的 Token 程序（可能是 Token-2022）
// #12 - System Program            // 系统程序，用于创建账户
// #13 - Rent                      // Rent 程序，用于获取最小租金
func extractEventForInitializePoolV2(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 最低账户数量要求，仅保留事件解析必须字段
	const requiredAccounts = 12
	if len(ix.Accounts) < requiredAccounts {
		logger.Errorf("[OrcaWhirlpool:InitializePoolV2] 指令账户长度不足: got=%d, expect>=%d, tx=%s",
			len(ix.Accounts), requiredAccounts, ctx.TxHashString())
		return -1
	}

	createPooleEvent := common.ExtractCreatePoolEvent(ctx, ix, consts.DexOrcaWhirlpool, "InitializePoolV2", &common.CreatePoolLayout{
		PoolAddressIndex:   6,  // #6 - Whirlpool
		TokenMint1Index:    1,  // #1 - Token Mint A
		TokenMint2Index:    2,  // #2 - Token Mint B
		TokenProgram1Index: 10, // #10 - Token Program A
		TokenProgram2Index: 11, // #11 - Token Program B
		PoolVault1Index:    7,  // #7 - Token Vault A
		PoolVault2Index:    8,  // #8 - Token Vault B
		UserWalletIndex:    5,  // #5 - Funder
	})
	if createPooleEvent == nil {
		return -1
	}

	ctx.AddEvent(createPooleEvent)
	return current + 1
}

package raydiumclmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
)

// 示例交易：https://solscan.io/tx/5xyP7pb1sLGkrhXvG5JJ1zefxthzBokqQnyvR47VxS9fAniSgPL57LXPiU7G9UfrcQtVFj4gstsU31oFSicxEmk4
//
// Raydium CLMM CreatePool 指令账户布局：
//
// #0  - Pool Creator            	// 用户钱包地址（作为创建者）
// #1  - Amm Config                	// Raydium CLMM 的全局配置账户（如 tick spacing）
// #2  - Pool State                 // 池子地址
// #3  - Token Mint 0               // 第一个 token 的 mint
// #4  - Token Mint 1               // 第二个 token 的 mint
// #5  - Token Vault 0              // 用于存放 Token 0 的池子 Token Account（即 LP token 储备）
// #6  - Token Vault 1              // 用于存放 Token 1 的池子 Token Account
// #7  - Observation State          // 用于记录 TWAP 数据的观察者状态账户
// #8  - Tick Array Bitmap          // Tick 位图账户，用于 tick 管理与导航
// #9  - Token Program 0            // Token Mint 0 所属的 SPL Token 程序地址
// #10 - Token Program 1            // Token Mint 1 所属的 SPL Token 程序地址
// #11 - System Program             // 系统程序（用于创建新账户）
// #12 - Rent                       // 系统租金变量账户（用于判断账户是否需要租金豁免）
func extractEventForCreatePool(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 实际只使用前 11 个账户（#0 ～ #10），System Program 和 Rent 未参与事件构造
	if len(ix.Accounts) < 11 {
		logger.Errorf("[RaydiumCLMM:CreatePool] 指令账户长度不足: got=%d, expect>=11, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	createPooleEvent := common.ExtractCreatePoolEvent(ctx, ix, consts.DexRaydiumCLMM, "CreatePool", &common.CreatePoolLayout{
		PoolAddressIndex:   2,
		TokenMint1Index:    3,
		TokenMint2Index:    4,
		TokenProgram1Index: 9,
		TokenProgram2Index: 10,
		PoolVault1Index:    5,
		PoolVault2Index:    6,
		UserWalletIndex:    0,
	})
	if createPooleEvent == nil {
		return -1
	}

	ctx.AddEvent(createPooleEvent)
	return current + 1
}

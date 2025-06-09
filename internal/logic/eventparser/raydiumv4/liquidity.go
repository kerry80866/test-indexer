package raydiumv4

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pb"
)

// 来源：https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs
// 示例交易：https://solscan.io/tx/3XzeH4Csvw4x8QSe89yYpZ3Q9d3uaZ73PmXm7ony5yh1FXRTyWr9esvfgpBJG4DBZ7UkMt7K2LZ1JebtYiS2ZEyN
//
// Raydium V4 添加流动性指令账户布局：
//
// #0  - Token Program                         // SPL Token 程序地址
// #1  - Amm                                   // 池子地址
// #2  - Amm Authority                         // 池子的 Authority PDA
// #3  - Amm Open Orders                       // OpenOrders PDA，用于挂单管理
// #4  - Amm Target Orders                     // Target Orders PDA，部分聚合逻辑
// #5  - LP Mint Address                       // LP Token 的 mint 账户
// #6  - Pool Coin Token Account               // 池子中 token1 的存储账户
// #7  - Pool Pc Token Account                 // 池子中 token2 的存储账户
// #8  - Serum Market                          // 关联的 Serum 市场地址
// #9  - User Coin Token Account               // 用户提供的 token1 账户
// #10 - User Pc Token Account                 // 用户提供的 token2 账户
// #11 - User LP Token Account                 // 用户LP Token 的账户
// #12 - User Owner                            // 用户钱包地址（Signer + Fee Payer）
// #13 - Serum Event Queue                     // Serum 撮合事件队列
func extractAddLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 14 {
		logger.Errorf("[RaydiumV4:AddLiquidity] 账户数不足: got=%d, expect>=14, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, mintEvent, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumV4, "AddLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       1,
		TokenMint1Index:        -1,
		TokenMint2Index:        -1,
		UserWalletIndex:        12,
		UserToken1AccountIndex: 9,
		UserToken2AccountIndex: 10,
		UserLpAccountIndex:     11,
		PoolToken1AccountIndex: 6,
		PoolToken2AccountIndex: 7,
		LpMintIndex:            5,
	})
	if liquidityEvent == nil || mintEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	ctx.AddEvent(mintEvent)
	return maxIndex + 1
}

// 来源：https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs
// 示例交易：https://solscan.io/tx/3mcgoS1fCFXfcVUDy1V4Q5SDB9egQYdmywXFEQ4qSxoJ8EbbSjYbb7rFnGjJgfwchea5cNQydJSTMmQdJuWQvFpE
//
// Raydium V4 移除流动性指令账户布局：
//
// #0  - Token Program                         // SPL Token 程序地址
// #1  - Amm                                   // Raydium 池子主账户
// #2  - Amm Authority                         // 池子的 Authority PDA
// #3  - Amm Open Orders                       // OpenOrders PDA，用于挂单管理
// #4  - Amm Target Orders                     // Target Orders PDA，部分聚合逻辑
// #5  - LP Mint Address                       // LP Token 的 mint 账户
// #6  - Pool Coin Token Account               // 池子中 token1 的存储账户
// #7  - Pool Pc Token Account                 // 池子中 token2 的存储账户
// #8  - Pool Withdraw Queue                   // 提现队列账户（Raydium 自定义机制）
// #9  - Pool Temp LP Token Account            // 临时 LP Token 账户，用于中转或校验
// #10 - Serum Program                         // Serum / OpenBook 程序地址
// #11 - Serum Market                          // 关联的 Serum 市场地址
// #12 - Serum Coin Vault Account              // Serum 中对应 token1 的托管账户
// #13 - Serum Pc Vault Account                // Serum 中对应 token2 的托管账户
// #14 - Serum Vault Signer                    // Serum Vault PDA，用于权限控制
// #15 - User LP Token Account                 // 用户LP Token 的账户
// #16 - User Coin Token Account               // 用户提供的 token1 账户
// #17 - User Pc Token Account                 // 用户提供的 token2 账户、
// #18 - User Owner                            // 用户钱包地址（Signer + Fee Payer）
// #19 - Serum Event Queue                     // Serum 事件队列（撮合撮合匹配）
// #20 - Serum Bids                            // Serum Bid 订单簿
// #21 - Serum Asks                            // Serum Ask 订单簿
func extractRemoveLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 19 {
		logger.Errorf("[RaydiumV4:RemoveLiquidity] 指令账户长度不足: got=%d, expect>=19, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	// 部分交易前有额外账户
	offset := 0
	if len(ix.Accounts) >= 22 {
		offset = 2
	}

	liquidityEvent, burnEvent, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexRaydiumV4, "RemoveLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       1,
		TokenMint1Index:        -1,
		TokenMint2Index:        -1,
		UserWalletIndex:        offset + 16,
		UserToken1AccountIndex: offset + 14,
		UserToken2AccountIndex: offset + 15,
		UserLpAccountIndex:     offset + 13,
		PoolToken1AccountIndex: 6,
		PoolToken2AccountIndex: 7,
		LpMintIndex:            5,
	})
	if liquidityEvent == nil || burnEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	ctx.AddEvent(burnEvent)
	return maxIndex + 1
}

// 来源：https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs
// 示例交易：https://solscan.io/tx/3XzeH4Csvw4x8QSe89yYpZ3Q9d3uaZ73PmXm7ony5yh1FXRTyWr9esvfgpBJG4DBZ7UkMt7K2LZ1JebtYiS2ZEyN
//
// Raydium V4 initialize2指令账户布局：
//
// #0  - Token Program                         // SPL Token 程序地址
// #1  - Associated Token Account Program      // 关联账户程序地址
// #2  - System Program                        // 系统程序地址
// #3  - Rent                                  // Rent 程序地址
// #4  - Amm                                   // Raydium 池子主账户
// #5  - Amm Authority                         // Raydium V4 Authority PDA
// #6  - Amm Open Orders                       // OpenOrders PDA，用于挂单管理
// #7  - LP Mint Address                       // LP Token 的 mint 账户
// #8  - Coin Mint                             // Base Token 的 mint（
// #9  - Pc Mint                               // Quote Token 的 mint
// #10 - Pool Coin Token Account               // 池子中 token1（Coin）的存储账户
// #11 - Pool Pc Token Account                 // 池子中 token2（Pc）的存储账户
// #12 - Pool Withdraw Queue                   // 提现队列账户（Raydium 自定义机制）
// #13 - Amm Target Orders                     // Target Orders PDA，部分聚合逻辑
// #14 - Pool Temp LP Token Account            // 临时 LP Token 账户，用于中转或校验
// #15 - Serum / OpenBook Program              // OpenBook 程序地址
// #16 - Serum Market                          // 关联的 Serum 市场地址
// #17 - User Wallet                           // 用户钱包地址（Signer + Fee Payer）
// #18 - User Coin Token Account               // 用户提供的 token1（Coin）账户
// #19 - User Pc Token Account                 // 用户提供的 token2（Pc）账户
// #20 - User LP Token Account                 // 用户LP Token 的账户
func extractInitializeEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 21 {
		logger.Errorf("[RaydiumV4:Initialize2] 账户数不足: got=%d, expect>=21, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, mintEvent, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexRaydiumV4, "Initialize2", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       4,
		TokenMint1Index:        8,
		TokenMint2Index:        9,
		UserWalletIndex:        17,
		UserToken1AccountIndex: 18,
		UserToken2AccountIndex: 19,
		UserLpAccountIndex:     20,
		PoolToken1AccountIndex: 10,
		PoolToken2AccountIndex: 11,
		LpMintIndex:            7,
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
		logger.Warnf("[RaydiumV4:Initialize2] MintToEvent ID 冲突，已跳过: tx=%s", ctx.TxHashString())
	}
	return maxIndex + 1
}

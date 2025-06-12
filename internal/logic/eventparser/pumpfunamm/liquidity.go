package pumpfunamm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pb"
	"dex-indexer-sol/pkg/logger"
)

// Pump.fun AMM 添加流动性指令账户布局：
//
// #0  - Pool                                   // Pump.fun AMM (JUP-USDC) Market，池子账户
// #1  - Global Config                          // 全局配置地址
// #2  - User                                   // 用户钱包地址
// #3  - Base Mint                              // JUP Token Mint
// #4  - Quote Mint                             // USDC Token Mint
// #5  - LP Mint                                // LP Token Mint（Pump.fun AMM LP Token）
// #6  - User Base Token Account                // 用户的 JUP Token 账户
// #7  - User Quote Token Account               // 用户的 USDC Token 账户
// #8  - User Pool Token Account                // 用户的 LP Token 账户（接收 LP Token）
// #9  - Pool Base Token Account                // 池子的 JUP Token 账户
// #10 - Pool Quote Token Account               // 池子的 USDC Token 账户
// #11 - Token Program                          // SPL Token 程序地址（主 SPL Token）
// #12 - Token 2022 Program                     // Token-2022 程序地址（支持扩展功能）
// #13 - Event Authority                        // Event Authority PDA（事件记录或权限标识）
// #14 - Program                                // Pump.fun AMM 程序本身
//
// 示例交易：https://solscan.io/tx/8M1ymW67CRt4zkCRLqh9e8jK8mhtUN9JtaYFqc1e8JKpGoBjsSeHTpuGupQfTcgCKKsF6vq65tEbaTCg2zS15RS
func extractAddLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 15 {
		logger.Errorf("[PumpfunAMM:AddLiquidity] 账户数不足: got=%d, expect>=15, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, mintEvent, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexPumpfunAMM, "AddLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       0,
		TokenMint1Index:        3,
		TokenMint2Index:        4,
		UserWalletIndex:        2,
		UserToken1AccountIndex: 6,
		UserToken2AccountIndex: 7,
		UserLpAccountIndex:     8,
		PoolToken1AccountIndex: 9,
		PoolToken2AccountIndex: 10,
		LpMintIndex:            5,
	})
	if liquidityEvent == nil || mintEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	ctx.AddEvent(mintEvent)
	return maxIndex + 1
}

// Pump.fun AMM 移除流动性指令账户布局：
//
// #0  - Pool                                   // 池子账户（Pump.fun AMM (TrenchTok-WSOL) Market）
// #1  - Global Config                          // 全局配置地址
// #2  - User                                   // 用户钱包地址
// #3  - Base Mint                              // TrenchTok Token Mint
// #4  - Quote Mint                             // WSOL Token Mint
// #5  - LP Mint                                // LP Token Mint（Pump.fun AMM LP Token）
// #6  - User Base Token Account                // 用户的 TrenchTok 账户（接收 base token）
// #7  - User Quote Token Account               // 用户的 WSOL 账户（接收 quote token）
// #8  - User Pool Token Account                // 用户的 LP Token 账户（消耗 LP token）
// #9  - Pool Base Token Account                // 池子的 TrenchTok Token 账户
// #10 - Pool Quote Token Account               // 池子的 WSOL Token 账户
// #11 - Token Program                          // SPL Token 程序地址
// #12 - Token 2022 Program                     // Token-2022 程序地址（支持扩展功能）
// #13 - Event Authority                        // Event Authority PDA（事件记录或权限验证）
// #14 - Program                                // Pump.fun AMM 程序 ID
//
// 示例交易：https://solscan.io/tx/26g2MFnXChvgXVTfi4oZeUeAHySthexq1mHP62s64TJrZzAdzCzbe1d1scpUG51bPf36zeSL8JaMP28Pss7jwfFq
func extractRemoveLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 15 {
		logger.Errorf("[PumpfunAMM:RemoveLiquidity] 指令账户长度不足: got=%d, expect>=15, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, burnEvent, maxIndex := common.ExtractRemoveLiquidityEvent(ctx, instrs, current, consts.DexPumpfunAMM, "RemoveLiquidity", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       0,
		TokenMint1Index:        3,
		TokenMint2Index:        4,
		UserWalletIndex:        2,
		UserToken1AccountIndex: 6,
		UserToken2AccountIndex: 7,
		UserLpAccountIndex:     8,
		PoolToken1AccountIndex: 9,
		PoolToken2AccountIndex: 10,
		LpMintIndex:            5,
	})
	if liquidityEvent == nil || burnEvent == nil {
		return -1
	}

	ctx.AddEvent(liquidityEvent)
	ctx.AddEvent(burnEvent)
	return maxIndex + 1
}

// Pump.fun AMM Create Pool指令账户布局：
//
// #0  - Pool                                   // 池子账户（Pump.fun AMM (WSOL-URIBE) Market）
// #1  - Global Config                          // 全局配置地址
// #2  - Creator                                // 用户钱包地址（创建人）
// #3  - Base Mint                              // WSOL Token Mint
// #4  - Quote Mint                             // URIBE Token Mint
// #5  - LP Mint                                // LP Token Mint（Pump.fun AMM LP Token）
// #6  - User Base Token Account                // 创建者的 WSOL Token 账户
// #7  - User Quote Token Account               // 创建者的 URIBE Token 账户
// #8  - User Pool Token Account                // 创建者的 LP Token 账户（接收 LP Token）
// #9  - Pool Base Token Account                // 池子的 WSOL Token 账户
// #10 - Pool Quote Token Account               // 池子的 URIBE Token 账户
// #11 - System Program                         // 系统程序地址（创建账户）
// #12 - Token 2022 Program                     // Token-2022 程序地址
// #13 - Base Token Program                     // WSOL 对应的 Token 程序（通常为 SPL Token）
// #14 - Quote Token Program                    // URIBE 对应的 Token 程序（通常为 SPL Token）
// #15 - Associated Token Program               // ATA 程序地址（用于初始化关联账户）
// #16 - Event Authority                        // 事件权限 PDA（Event Authority）
// #17 - Program                                // Pump.fun AMM 程序本身
//
// 示例交易：https://solscan.io/tx/3sg7gJxwFShPWcTgadp2VxVwT542tBPorFbRFQSR7YU7UJrFc2PywxXGfH5ntHfApinxSLLUeixmJ2eTBjhvrBrN
func extractCreatePoolEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	if len(ix.Accounts) < 18 {
		logger.Errorf("[PumpfunAMM:CreatePool] 账户数不足: got=%d, expect>=18, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return -1
	}

	liquidityEvent, mintEvent, maxIndex := common.ExtractAddLiquidityEvent(ctx, instrs, current, consts.DexPumpfunAMM, "CreatePool", &common.LiquidityLayout{
		RequireBothTransfer:    true,
		PoolAddressIndex:       0,
		TokenMint1Index:        3,
		TokenMint2Index:        4,
		UserWalletIndex:        2,
		UserToken1AccountIndex: 6,
		UserToken2AccountIndex: 7,
		UserLpAccountIndex:     8,
		PoolToken1AccountIndex: 9,
		PoolToken2AccountIndex: 10,
		LpMintIndex:            5,
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
		logger.Warnf("[PumpfunAMM:CreatePool] MintToEvent ID 冲突，已跳过: tx=%s", ctx.TxHashString())
	}
	return maxIndex + 1
}

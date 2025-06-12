package common

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/tools"
	"dex-indexer-sol/pkg/logger"
	"dex-indexer-sol/pkg/types"
)

func ExtractAddLiquidityEvent(
	ctx *ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
	dex int,
	instructionName string,
	layout *LiquidityLayout,
) (*core.Event, *core.Event, int) {
	ix := instrs[current]
	dexName := consts.DexName(dex)

	// Step 1: 提取 token 和 LP mint 转账
	result := FindAddLiquidityTransfers(ctx, instrs, current, layout, 0)
	if result == nil || (result.Token1Transfer == nil && result.Token2Transfer == nil) {
		logger.Errorf("[%s:%s] 转账结构缺失: tx=%s, ix=%d, inner=%d",
			dexName, instructionName, ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return nil, nil, -1
	}

	// Step 2: 单边时补齐另一边
	if !layout.RequireBothTransfer {
		switch {
		case result.Token1Transfer != nil && result.Token2Transfer == nil:
			result.Token2Transfer = fillMissingTransferForAdd(
				ctx, ix,
				result.Token1Transfer.SrcWallet,
				ix.Accounts[layout.UserToken2AccountIndex],
				getOtherPoolToken(ix, layout, result.Token1Transfer.DestAccount),
			)
		case result.Token1Transfer == nil && result.Token2Transfer != nil:
			result.Token1Transfer = fillMissingTransferForAdd(
				ctx, ix,
				result.Token2Transfer.SrcWallet,
				ix.Accounts[layout.UserToken1AccountIndex],
				getOtherPoolToken(ix, layout, result.Token2Transfer.DestAccount),
			)
		}
	}

	// Step 3: 检查transfer
	if result.Token1Transfer == nil || result.Token2Transfer == nil {
		logger.Warnf("[%s:%s] 转账补全失败: tx=%s", dexName, instructionName, ctx.TxHashString())
		return nil, nil, -1
	}

	// Step 4: 校验用户钱包地址是否一致（即两笔 token 的来源都是指定用户）
	if layout.UserWalletIndex >= 0 {
		expected := ix.Accounts[layout.UserWalletIndex]
		if result.Token1Transfer.SrcWallet != expected || result.Token2Transfer.SrcWallet != expected {
			logger.Warnf("[%s:%s] token 来源钱包地址不一致: token1.wallet=%s token2.wallet=%s expect=%s, ix=%d, inner=%d, tx=%s",
				dexName, instructionName, result.Token1Transfer.SrcWallet, result.Token2Transfer.SrcWallet, expected,
				ix.IxIndex, ix.InnerIndex, ctx.TxHashString())
			return nil, nil, -1
		}
	}

	// Step 5: 检查 LP MintTo 是否存在
	if layout.LpMintIndex >= 0 && result.LpMintTo == nil {
		logger.Warnf("[%s:%s] 缺少 LP Mint 转账: tx=%s", dexName, instructionName, ctx.TxHashString())
		return nil, nil, -1
	}

	// Step 6: 校验 token mint
	if layout.TokenMint1Index >= 0 {
		expect := ix.Accounts[layout.TokenMint1Index]
		if result.Token1Transfer.Token != expect {
			logger.Warnf("[%s:%s] token1 mint 不匹配: got=%s expect=%s, tx=%s",
				dexName, instructionName, result.Token1Transfer.Token, expect, ctx.TxHashString())
			return nil, nil, -1
		}
	}
	if layout.TokenMint2Index >= 0 {
		expect := ix.Accounts[layout.TokenMint2Index]
		if result.Token2Transfer.Token != expect {
			logger.Warnf("[%s:%s] token2 mint 不匹配: got=%s expect=%s, tx=%s",
				dexName, instructionName, result.Token2Transfer.Token, expect, ctx.TxHashString())
			return nil, nil, -1
		}
	}

	// Step 7: 判断 base / quote
	baseTransfer, quoteTransfer := determineBaseQuoteTransfer(result.Token1Transfer, result.Token2Transfer)

	// Step 8: 构建 AddLiquidityEvent
	addEvent := BuildAddLiquidityEvent(ctx, ix, baseTransfer, quoteTransfer, ix.Accounts[layout.PoolAddressIndex], dex)
	if addEvent == nil {
		return nil, nil, -1
	}

	// Step 9: 构建可选的 LP MintToEvent
	var mintEvent *core.Event
	if result.LpMintTo != nil {
		mintEvent = BuildMintToEvent(ctx, result.LpMintTo)
		if mintEvent == nil {
			return nil, nil, -1
		}
	}

	return addEvent, mintEvent, result.MaxIndex
}

func ExtractRemoveLiquidityEvent(
	ctx *ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
	dex int,
	instructionName string,
	layout *LiquidityLayout,
) (*core.Event, *core.Event, int) {
	ix := instrs[current]
	dexName := consts.DexName(dex)

	// Step 1: 提取 token 转入 + LP burn 转账
	result := FindRemoveLiquidityTransfers(ctx, instrs, current, layout, 0)
	if result == nil || (result.Token1Transfer == nil && result.Token2Transfer == nil) {
		logger.Errorf("[%s:%s] 转账结构缺失: tx=%s, ix=%d, inner=%d",
			dexName, instructionName, ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return nil, nil, -1
	}

	// Step 2: 补齐单边
	if !layout.RequireBothTransfer {
		switch {
		case result.Token1Transfer != nil && result.Token2Transfer == nil:
			result.Token2Transfer = fillMissingTransferForRemove(
				ctx, ix,
				result.Token1Transfer.DestWallet,
				ix.Accounts[layout.UserToken2AccountIndex],
				getOtherPoolToken(ix, layout, result.Token1Transfer.SrcAccount),
			)
		case result.Token1Transfer == nil && result.Token2Transfer != nil:
			result.Token1Transfer = fillMissingTransferForRemove(
				ctx, ix,
				result.Token2Transfer.DestWallet,
				ix.Accounts[layout.UserToken1AccountIndex],
				getOtherPoolToken(ix, layout, result.Token2Transfer.SrcAccount),
			)
		}
	}

	// Step 3: 检查 transfer 是否都存在
	if result.Token1Transfer == nil || result.Token2Transfer == nil {
		logger.Warnf("[%s:%s] 转账补全失败: tx=%s", dexName, instructionName, ctx.TxHashString())
		return nil, nil, -1
	}

	// Step 4: 校验两笔 token 的接收者钱包地址是否一致
	if result.Token1Transfer.DestWallet != result.Token2Transfer.DestWallet {
		if result.Token1Transfer.Token == consts.WSOLMint || result.Token2Transfer.Token == consts.WSOLMint {
			logger.Warnf("[%s:%s] token 接收者钱包地址不一致，但包含 WSOL，放行: token1.wallet=%s token2.wallet=%s, ix=%d, inner=%d, tx=%s",
				dexName, instructionName, result.Token1Transfer.DestWallet, result.Token2Transfer.DestWallet,
				ix.IxIndex, ix.InnerIndex, ctx.TxHashString())
		} else {
			logger.Warnf("[%s:%s] token 接收者钱包地址不一致: token1.wallet=%s token2.wallet=%s, ix=%d, inner=%d, tx=%s",
				dexName, instructionName, result.Token1Transfer.DestWallet, result.Token2Transfer.DestWallet,
				ix.IxIndex, ix.InnerIndex, ctx.TxHashString())
			return nil, nil, -1
		}
	}

	// Step 5: 校验 LP Burn 是否存在
	if layout.LpMintIndex >= 0 && result.LpBurn == nil {
		logger.Warnf("[%s:%s] 缺少 LP Burn 转账: tx=%s", dexName, instructionName, ctx.TxHashString())
		return nil, nil, -1
	}

	// Step 6 校验 token mint
	if layout.TokenMint1Index >= 0 {
		expect := ix.Accounts[layout.TokenMint1Index]
		if result.Token1Transfer.Token != expect {
			logger.Warnf("[%s:%s] token1 mint 不匹配: got=%s expect=%s, tx=%s",
				dexName, instructionName, result.Token1Transfer.Token, expect, ctx.TxHashString())
			return nil, nil, -1
		}
	}
	if layout.TokenMint2Index >= 0 {
		expect := ix.Accounts[layout.TokenMint2Index]
		if result.Token2Transfer.Token != expect {
			logger.Warnf("[%s:%s] token2 mint 不匹配: got=%s expect=%s, tx=%s",
				dexName, instructionName, result.Token2Transfer.Token, expect, ctx.TxHashString())
			return nil, nil, -1
		}
	}

	// Step 7: 判断 base / quote
	baseTransfer, quoteTransfer := determineBaseQuoteTransfer(result.Token1Transfer, result.Token2Transfer)

	// Step 8: 构建 RemoveLiquidityEvent
	removeEvent := BuildRemoveLiquidityEvent(ctx, ix, baseTransfer, quoteTransfer, ix.Accounts[layout.PoolAddressIndex], dex)
	if removeEvent == nil {
		return nil, nil, -1
	}

	// Step 9: 构建 BurnEvent（可选）
	var burnEvent *core.Event
	if result.LpBurn != nil {
		burnEvent = BuildBurnEvent(ctx, result.LpBurn)
		if burnEvent == nil {
			return nil, nil, -1
		}
	}

	return removeEvent, burnEvent, result.MaxIndex
}

func fillMissingTransferForAdd(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	userWallet, userTokenAccount, poolTokenAccount types.Pubkey,
) *ParsedTransfer {
	srcInfo, ok1 := ctx.Balances[userTokenAccount]
	destInfo, ok2 := ctx.Balances[poolTokenAccount]
	if !ok1 || !ok2 {
		return nil
	}
	return &ParsedTransfer{
		IxIndex:         ix.IxIndex,
		InnerIndex:      ix.InnerIndex,
		Token:           srcInfo.Token,
		SrcAccount:      userTokenAccount,
		DestAccount:     poolTokenAccount,
		SrcWallet:       userWallet,
		DestWallet:      destInfo.PostOwner,
		Amount:          0,
		Decimals:        srcInfo.Decimals,
		SrcPostBalance:  srcInfo.PostBalance,
		DestPostBalance: destInfo.PostBalance,
	}
}

func fillMissingTransferForRemove(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	userWallet, userTokenAccount, poolTokenAccount types.Pubkey,
) *ParsedTransfer {
	srcInfo, ok1 := ctx.Balances[poolTokenAccount]
	destInfo, ok2 := ctx.Balances[userTokenAccount]
	if !ok1 || !ok2 {
		return nil
	}
	return &ParsedTransfer{
		IxIndex:         ix.IxIndex,
		InnerIndex:      ix.InnerIndex,
		Token:           srcInfo.Token,
		SrcAccount:      poolTokenAccount,
		DestAccount:     userTokenAccount,
		SrcWallet:       srcInfo.PostOwner,
		DestWallet:      userWallet,
		Amount:          0,
		Decimals:        srcInfo.Decimals,
		SrcPostBalance:  srcInfo.PostBalance,
		DestPostBalance: destInfo.PostBalance,
	}
}

func getOtherPoolToken(
	ix *core.AdaptedInstruction,
	layout *LiquidityLayout,
	knownPoolAccount types.Pubkey,
) types.Pubkey {
	if knownPoolAccount == ix.Accounts[layout.PoolToken1AccountIndex] {
		return ix.Accounts[layout.PoolToken2AccountIndex]
	}
	return ix.Accounts[layout.PoolToken1AccountIndex]
}

func determineBaseQuoteTransfer(
	transfer1, transfer2 *ParsedTransfer,
) (baseTransfer *ParsedTransfer, quoteTransfer *ParsedTransfer) {
	if quote, ok := tools.ChooseQuote(transfer1.Token, transfer2.Token); ok {
		if quote == transfer2.Token {
			return transfer1, transfer2
		}
		return transfer2, transfer1
	}
	// fallback: 无法识别 quote，保持顺序，默认 transfer1 为 base
	return transfer1, transfer2
}

package raydiumv4

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

// 来源：https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs
// 示例交易：https://solscan.io/tx/3XzeH4Csvw4x8QSe89yYpZ3Q9d3uaZ73PmXm7ony5yh1FXRTyWr9esvfgpBJG4DBZ7UkMt7K2LZ1JebtYiS2ZEyN
//
// Raydium V4 添加流动性指令账户布局：
//  0. `[]`           SPL Token Program
//  1. `[]`           AMM 主账户（池子地址）
//  2. `[]`           权限 PDA
//  3. `[]`           AMM open_orders 账户
//  4. `[]`           AMM target orders（挂单缓冲区）
//  5. `[writable]`   LP token mint 账户
//  6. `[writable]`   池子 token1（coin）vault
//  7. `[writable]`   池子 token2（pc）vault
//  8. `[]`           市场账户（由 Serum 控制）
//  9. `[writable]`   用户 token1 账户（coin）
//  10. `[writable]`   用户 token2 账户（pc）
//  11. `[writable]`   用户 LP token 账户
//  12. `[signer]`     用户钱包账户
//  13. `[writable]`   市场 event queue（Serum）
func extractRaydiumV4AddLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]

	if len(ix.Accounts) != 14 {
		logx.Errorf("[RaydiumV4:AddLiquidity] 指令账户长度不足: got=%d, expect>=14, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return nil, current + 1
	}

	result := common.FindAddLiquidityTransfers(ctx, instrs, current, &common.LiquidityInstructionIndex{
		UserToken1AccountIndex: 9,
		UserToken2AccountIndex: 10,
		PoolToken1AccountIndex: 6,
		PoolToken2AccountIndex: 7,
		LpMintIndex:            5,
	}, 3)

	if result == nil || result.Token1Transfer == nil || result.Token2Transfer == nil || result.LpMintTo == nil {
		logx.Errorf("[RaydiumV4:AddLiquidity] 转账结构缺失: tx=%s", ctx.TxHashString())
		return nil, current + 1
	}

	// 校验用户钱包一致性（双 token 的 src wallet 均等于第 12 位 wallet）
	expectedWallet := ix.Accounts[12]
	if expectedWallet != result.Token1Transfer.SrcWallet || expectedWallet != result.Token2Transfer.SrcWallet {
		logx.Errorf("[RaydiumV4:AddLiquidity] 用户钱包不一致: tx=%s, expect=%s, got1=%s, got2=%s",
			ctx.TxHashString(),
			expectedWallet,
			result.Token1Transfer.SrcWallet,
			result.Token2Transfer.SrcWallet)
		return nil, current + 1
	}

	baseTransfer, quoteTransfer := determineBaseQuoteTransfer(result.Token1Transfer, result.Token2Transfer)
	event := common.BuildAddLiquidityEvent(ctx, ix, baseTransfer, quoteTransfer, result.LpMintTo, ix.Accounts[1], consts.DexRaydiumV4)
	if event == nil {
		return event, current + 1
	}
	return event, current + 3
}

// 来源：https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs
// 示例交易：https://solscan.io/tx/3XzeH4Csvw4x8QSe89yYpZ3Q9d3uaZ73PmXm7ony5yh1FXRTyWr9esvfgpBJG4DBZ7UkMt7K2LZ1JebtYiS2ZEyN
//
// Raydium V4 移除流动性指令账户布局：
// /   Withdraw the vault tokens from the pool at the current ratio.
// /
// /   0. `[]` Spl Token program id
// /   1. `[writable]` AMM Account
// /   2. `[]` $authority derived from `create_program_address(&[AUTHORITY_AMM, &[nonce]])`.
// /   3. `[writable]` AMM open orders Account
// /   4. `[writable]` AMM target orders Account
// /   5. `[writable]` AMM lp mint Account. Owned by $authority.
// /   6. `[writable]` AMM coin vault Account to withdraw FROM,
// /   7. `[writable]` AMM pc vault Account to withdraw FROM,
// /   8. `[]` Market program id
// /   9. `[writable]` Market Account. Market program is the owner.
// /   10. `[writable]` Market coin vault Account
// /   11. `[writable]` Market pc vault Account
// /   12. '[]` Market vault signer Account
// /   13. `[writable]` User lp token Account.
// /   14. `[writable]` User token coin Account. user Account to credit.
// /   15. `[writable]` User token pc Account. user Account to credit.
// /   16. `[signer]` User wallet Account
// /   17. `[writable]` Market event queue Account
// /   18. `[writable]` Market bids Account
// /   19. `[writable]` Market asks Account
func extractRaydiumV4RemoveLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]

	if len(ix.Accounts) < 20 {
		logx.Errorf("[RaydiumV4:AddLiquidity] 指令账户长度不足: got=%d, expect>=14, tx=%s",
			len(ix.Accounts), ctx.TxHashString())
		return nil, current + 1
	}

	//offset := 0
	//if len(ix.Accounts) >= 22 {
	//	offset = 2
	//}
	//
	//result := common.FindRemoveLiquidityTransfers(ctx, instrs, current, &common.LiquidityInstructionIndex{
	//	UserToken1AccountIndex: 14 + offset,
	//	UserToken2AccountIndex: 15 + offset,
	//	PoolToken1AccountIndex: 6,
	//	PoolToken2AccountIndex: 7,
	//	LpMintIndex:            5,
	//}, 3)
	//
	//if result == nil || result.Token1Transfer == nil || result.Token2Transfer == nil || result.LpBurn == nil {
	//	logx.Errorf("[RaydiumV4:AddLiquidity] 转账结构缺失: tx=%s", ctx.TxHashString())
	//	return nil, current + 1
	//}
	//
	//// 校验用户钱包一致性（双 token 的 src wallet 均等于第 12 位 wallet）
	//expectedWallet := ix.Accounts[12]
	//if expectedWallet != result.Token1Transfer.SrcWallet || expectedWallet != result.Token2Transfer.SrcWallet {
	//	logx.Errorf("[RaydiumV4:AddLiquidity] 用户钱包不一致: tx=%s, expect=%s, got1=%s, got2=%s",
	//		ctx.TxHashString(),
	//		expectedWallet,
	//		result.Token1Transfer.SrcWallet,
	//		result.Token2Transfer.SrcWallet)
	//	return nil, current + 1
	//}
	//
	//baseTransfer, quoteTransfer := determineBaseQuoteTransfer(result.Token1Transfer, result.Token2Transfer)
	//event := common.BuildAddLiquidityEvent(ctx, ix, baseTransfer, quoteTransfer, result.LpMintTo, ix.Accounts[1], consts.DexRaydiumV4)
	//if event == nil {
	//	return event, current + 1
	//}
	return nil, current + 3
}

func determineBaseQuoteTransfer(
	transfer1 *common.ParsedTransfer,
	transfer2 *common.ParsedTransfer,
) (baseTransfer *common.ParsedTransfer, quoteTransfer *common.ParsedTransfer) {
	if quote, ok := utils.ChooseQuote(transfer1.Token, transfer2.Token); ok {
		if quote == transfer2.Token {
			return transfer1, transfer2
		}
		return transfer2, transfer1
	}
	// fallback：无明确 quote，默认按输入顺序返回
	return transfer1, transfer2
}

package common

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/pkg/types"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
)

type AddLiquidityResult struct {
	Token1Transfer *ParsedTransfer
	Token2Transfer *ParsedTransfer
	LpMintTo       *ParsedMintTo
	MaxIndex       int // 涉及的最大指令序号
}

type RemoveLiquidityResult struct {
	Token1Transfer *ParsedTransfer
	Token2Transfer *ParsedTransfer
	LpBurn         *ParsedBurn
	MaxIndex       int // 涉及的最大指令序号
}

type LiquidityLayout struct {
	RequireBothTransfer bool
	PoolAddressIndex    int // 必须存在, 不能为-1
	TokenMint1Index     int // 可选, -1表示忽略
	TokenMint2Index     int // 可选, -1表示忽略
	UserWalletIndex     int // 可选, -1表示忽略

	UserToken1AccountIndex int // 必须存在, 不能为-1
	UserToken2AccountIndex int // 必须存在, 不能为-1
	UserLpAccountIndex     int // 可选, -1表示忽略
	PoolToken1AccountIndex int // 必须存在, 不能为-1
	PoolToken2AccountIndex int // 必须存在, 不能为-1
	LpMintIndex            int // 可选, -1表示忽略
}

// validateLiquidityInstructionIndex 校验 LiquidityInstructionIndex 中各字段合法性。
// - 必选字段必须存在且 index 在 accounts 范围内。
// - 可选字段（LpMintIndex）允许为 -1，表示未提供。
func validateLiquidityInstructionIndex(layout *LiquidityLayout, accountsLen int) bool {
	isValid := func(index int) bool {
		return index >= 0 && index < accountsLen
	}
	isOptional := func(index int) bool {
		return index == -1 || isValid(index)
	}
	return isValid(layout.UserToken1AccountIndex) &&
		isValid(layout.UserToken2AccountIndex) &&
		isValid(layout.PoolToken1AccountIndex) &&
		isValid(layout.PoolToken2AccountIndex) &&
		isOptional(layout.UserLpAccountIndex) &&
		isOptional(layout.LpMintIndex)
}

// FindAddLiquidityTransfers 尝试从主指令开始向后匹配添加流动性相关的转账（Transfer）和铸造（MintTo）操作，
// 适用于AMM在用户添加流动性时的典型指令结构解析。
//
// 参数说明：
//   - ctx          : 当前交易解析上下文（包含账户余额、Token 结构等信息）。
//   - instrs       : 展平后的指令列表（包含主指令和 inner 指令）。
//   - current      : 当前主指令在 instrs 中的索引（作为匹配起点）。
//   - layout       : 表示用户提供和池子使用的 Token 账户索引结构，包括 LP Mint（可选）。
//   - maxLookahead : 向后最多检查的指令数量（不包括主指令本身）；
//     若为 0，表示不限制，遍历当前主指令的所有 inner 指令（IxIndex 不变）。
//
// 返回值：
// - Token1Transfer : 用户支付的 Token1 的转账记录（用户 → 池子）。
// - Token2Transfer : 用户支付的 Token2 的转账记录（用户 → 池子）。
// - LpMint         : 用户收到的 LP Token 的铸造记录（MintTo）。
// - MaxIndex       : 匹配到的所有相关指令中的最大索引位置（用于标记事件范围）。
func FindAddLiquidityTransfers(
	ctx *ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
	layout *LiquidityLayout,
	maxLookahead int,
) *AddLiquidityResult {
	mainIx := instrs[current]
	if !validateLiquidityInstructionIndex(layout, len(mainIx.Accounts)) {
		return nil
	}

	// 提取关键账户
	userToken1Account := mainIx.Accounts[layout.UserToken1AccountIndex]
	userToken2Account := mainIx.Accounts[layout.UserToken2AccountIndex]
	poolToken1Account := mainIx.Accounts[layout.PoolToken1AccountIndex]
	poolToken2Account := mainIx.Accounts[layout.PoolToken2AccountIndex]

	var token1Transfer, token2Transfer *ParsedTransfer
	var lpMint *ParsedMintTo
	maxIndex := current

	hasLp := layout.LpMintIndex >= 0
	lpToken := types.Pubkey{}
	if hasLp {
		lpToken = mainIx.Accounts[layout.LpMintIndex]
	}

	hasUserLpAccount := layout.UserLpAccountIndex >= 0
	userLpAccount := types.Pubkey{}
	if hasUserLpAccount {
		userLpAccount = mainIx.Accounts[layout.UserLpAccountIndex]
	}

	looked := 0
	for i := current + 1; i < len(instrs); i++ {
		ix := instrs[i]

		// 只遍历当前主指令的 inner 指令
		if ix.IxIndex != mainIx.IxIndex {
			break
		}
		if maxLookahead > 0 {
			if looked >= maxLookahead {
				break
			}
			looked++
		}

		// 跳过空指令 / 非 Token Program
		if len(ix.Data) == 0 ||
			(ix.ProgramID != consts.TokenProgram && ix.ProgramID != consts.TokenProgram2022) {
			continue
		}

		switch ix.Data[0] {
		case byte(sdktoken.InstructionTransfer), byte(sdktoken.InstructionTransferChecked):
			pt, ok := ParseTransferInstruction(ctx, ix)
			if !ok {
				continue
			}

			// 用户 → 池子：Token1
			if token1Transfer == nil &&
				pt.SrcAccount == userToken1Account &&
				(pt.DestAccount == poolToken1Account || pt.DestAccount == poolToken2Account) {
				// 若 token2 已匹配到相同目标，冲突跳过
				if isTransferConflict(pt, token2Transfer) {
					continue
				}
				token1Transfer = pt
				maxIndex = i
				continue
			}

			// 用户 → 池子：Token2
			if token2Transfer == nil &&
				pt.SrcAccount == userToken2Account &&
				(pt.DestAccount == poolToken1Account || pt.DestAccount == poolToken2Account) {
				// 若 token1 已匹配到相同目标，冲突跳过
				if isTransferConflict(pt, token1Transfer) {
					continue
				}
				token2Transfer = pt
				maxIndex = i
				continue
			}

		case byte(sdktoken.InstructionMintTo), byte(sdktoken.InstructionMintToChecked):
			if hasLp && lpMint == nil {
				mt, ok := ParseMintToInstruction(ctx, ix)
				if ok && mt.Token == lpToken && (!hasUserLpAccount || mt.DestAccount == userLpAccount) {
					lpMint = mt
					maxIndex = i
					continue
				}
			}
		}

		if token1Transfer != nil && token2Transfer != nil && (!hasLp || lpMint != nil) {
			break
		}
	}

	// 至少需要匹配到一个 token 的支付
	if token1Transfer == nil && token2Transfer == nil {
		return nil
	}

	return &AddLiquidityResult{
		Token1Transfer: token1Transfer,
		Token2Transfer: token2Transfer,
		LpMintTo:       lpMint,
		MaxIndex:       maxIndex,
	}
}

// FindRemoveLiquidityTransfers 尝试从主指令开始向后匹配移除流动性相关的转账（Transfer）和销毁（Burn）操作。
// 适用于AMM在用户移除流动性时的典型指令结构解析。
//
// 参数说明：
//   - ctx          : 当前交易解析上下文（包含账户余额、Token 结构等信息）。
//   - instrs       : 展平后的指令列表（包含主指令和 inner 指令）。
//   - current      : 当前主指令在 instrs 中的索引（作为匹配起点）。
//   - layout       : 表示用户提供和池子使用的 Token 账户索引结构，包括 LP Mint（可选）。
//   - maxLookahead : 向后最多检查的指令数量（不包括主指令本身）；
//     若为 0，表示不限制，遍历当前主指令的所有 inner 指令（IxIndex 不变）。
//
// 返回值：
// - Token1Transfer : 用户收到的 Token1 的转账记录（池子 → 用户）。
// - Token2Transfer : 用户收到的 Token2 的转账记录（池子 → 用户）。
// - LpBurn         : 用户销毁 LP Token 的记录（代表移除份额）。
// - MaxIndex       : 匹配到的所有相关指令中的最大索引位置（用于标记事件范围）。
func FindRemoveLiquidityTransfers(
	ctx *ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
	layout *LiquidityLayout,
	maxLookahead int,
) *RemoveLiquidityResult {
	mainIx := instrs[current]
	if !validateLiquidityInstructionIndex(layout, len(mainIx.Accounts)) {
		return nil
	}

	// 提取关键账户
	userToken1Account := mainIx.Accounts[layout.UserToken1AccountIndex]
	userToken2Account := mainIx.Accounts[layout.UserToken2AccountIndex]
	poolToken1Account := mainIx.Accounts[layout.PoolToken1AccountIndex]
	poolToken2Account := mainIx.Accounts[layout.PoolToken2AccountIndex]

	var token1Transfer, token2Transfer *ParsedTransfer
	var lpBurn *ParsedBurn
	maxIndex := current

	hasLp := layout.LpMintIndex >= 0
	lpToken := types.Pubkey{}
	if hasLp {
		lpToken = mainIx.Accounts[layout.LpMintIndex]
	}

	hasUserLpAccount := layout.UserLpAccountIndex >= 0
	userLpAccount := types.Pubkey{}
	if hasUserLpAccount {
		userLpAccount = mainIx.Accounts[layout.UserLpAccountIndex]
	}

	looked := 0
	for i := current + 1; i < len(instrs); i++ {
		ix := instrs[i]

		// 同一个主指令范围内查找
		if ix.IxIndex != mainIx.IxIndex {
			break
		}
		if maxLookahead > 0 {
			if looked >= maxLookahead {
				break
			}
			looked++
		}

		// 非 Token Program 指令直接跳过
		if len(ix.Data) == 0 ||
			(ix.ProgramID != consts.TokenProgram && ix.ProgramID != consts.TokenProgram2022) {
			continue
		}

		switch ix.Data[0] {
		case byte(sdktoken.InstructionTransfer), byte(sdktoken.InstructionTransferChecked):
			pt, ok := ParseTransferInstruction(ctx, ix)
			if !ok {
				continue
			}

			// 尝试匹配池子 → 用户 的 token1
			if token1Transfer == nil &&
				pt.DestAccount == userToken1Account &&
				(pt.SrcAccount == poolToken1Account || pt.SrcAccount == poolToken2Account) {
				// 若 token2 已匹配到相同目标，冲突跳过
				if isTransferConflict(pt, token2Transfer) {
					continue
				}
				token1Transfer = pt
				maxIndex = i
				continue
			}

			// 尝试匹配池子 → 用户 的 token2
			if token2Transfer == nil &&
				pt.DestAccount == userToken2Account &&
				(pt.SrcAccount == poolToken1Account || pt.SrcAccount == poolToken2Account) {
				// 若 token1 已匹配到相同目标，冲突跳过
				if isTransferConflict(pt, token1Transfer) {
					continue
				}
				token2Transfer = pt
				maxIndex = i
				continue
			}

		case byte(sdktoken.InstructionBurn), byte(sdktoken.InstructionBurnChecked):
			if hasLp && lpBurn == nil {
				burn, ok := ParseBurnInstruction(ctx, ix)
				if ok && burn.Token == lpToken && (!hasUserLpAccount || burn.SrcAccount == userLpAccount) {
					lpBurn = burn
					maxIndex = i
					continue
				}
			}
		}

		if token1Transfer != nil && token2Transfer != nil && (!hasLp || lpBurn != nil) {
			break
		}
	}

	if token1Transfer == nil && token2Transfer == nil {
		return nil
	}

	return &RemoveLiquidityResult{
		Token1Transfer: token1Transfer,
		Token2Transfer: token2Transfer,
		LpBurn:         lpBurn,
		MaxIndex:       maxIndex,
	}
}

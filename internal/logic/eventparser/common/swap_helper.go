package common

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
)

// SwapInstructionIndex 表示 Swap 操作中涉及的关键账户索引。
// 所有字段对应主指令中的 accounts 列表索引位置。
type SwapInstructionIndex struct {
	UserToken1AccountIndex int // 用户提供的 token1 账户索引（可能为支付或接收）
	UserToken2AccountIndex int // 用户提供的 token2 账户索引（可能为支付或接收）
	PoolToken1AccountIndex int // 池子的 token1 账户索引
	PoolToken2AccountIndex int // 池子的 token2 账户索引
}

// SwapTransferResult 表示成功识别出的 Swap 中两个方向的转账记录。
type SwapTransferResult struct {
	UserToPool *ParsedTransfer // 用户支付 → 池子（表示卖出 token）
	PoolToUser *ParsedTransfer // 池子支付 → 用户（表示买入 token）
	MaxIndex   int             // 涉及的最大指令序号（用于标记事件范围）
}

// validateSwapInstructionIndex 校验 SwapInstructionIndex 各字段是否在账户范围内。
func validateSwapInstructionIndex(indexes *SwapInstructionIndex, accountsLen int) bool {
	isValid := func(index int) bool {
		return index >= 0 && index < accountsLen
	}
	return isValid(indexes.UserToken1AccountIndex) &&
		isValid(indexes.UserToken2AccountIndex) &&
		isValid(indexes.PoolToken1AccountIndex) &&
		isValid(indexes.PoolToken2AccountIndex)
}

// FindSwapTransfersByIndex 尝试从主指令开始向后匹配 Swap 相关的转账操作（Transfer）。
// 适用于 AMM 中 Swap 操作的典型指令结构（用户支付 + 用户接收）。
//
// 参数说明：
//   - ctx          : 当前交易解析上下文（包含账户余额、Token 结构等信息）。
//   - instrs       : 展平后的指令列表（包含主指令和 inner 指令）。
//   - current      : 当前主指令在 instrs 中的索引（作为匹配起点）。
//   - indexes      : 表示用户和池子之间 Token 账户的索引结构。
//   - maxLookahead : 向后最多检查的指令数量（不包括主指令本身）；
//     若为 0，表示不限制，遍历当前主指令的所有 inner 指令（IxIndex 不变）。
//
// 返回值：若同时成功匹配两个方向的转账，返回 SwapTransferResult；否则返回 nil。
func FindSwapTransfersByIndex(
	ctx *ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
	indexes *SwapInstructionIndex,
	maxLookahead int,
) *SwapTransferResult {
	mainIx := instrs[current]
	if !validateSwapInstructionIndex(indexes, len(mainIx.Accounts)) {
		return nil
	}

	// 提取关键账户
	userToken1 := mainIx.Accounts[indexes.UserToken1AccountIndex]
	userToken2 := mainIx.Accounts[indexes.UserToken2AccountIndex]
	poolToken1 := mainIx.Accounts[indexes.PoolToken1AccountIndex]
	poolToken2 := mainIx.Accounts[indexes.PoolToken2AccountIndex]

	var userToPool, poolToUser *ParsedTransfer
	maxIndex := current
	looked := 0

	for i := current + 1; i < len(instrs); i++ {
		ix := instrs[i]

		// 只处理当前主指令的 inner 指令
		if ix.IxIndex != mainIx.IxIndex {
			break
		}
		if maxLookahead > 0 {
			if looked >= maxLookahead {
				break
			}
			looked++
		}

		// 跳过空指令或非 Token Program 的指令
		if len(ix.Data) == 0 ||
			(ix.ProgramID != consts.TokenProgram && ix.ProgramID != consts.TokenProgram2022) {
			continue
		}
		// 仅处理 Transfer 类型的指令
		if ix.Data[0] != byte(sdktoken.InstructionTransfer) &&
			ix.Data[0] != byte(sdktoken.InstructionTransferChecked) {
			continue
		}

		pt, ok := ParseTransferInstruction(ctx, ix)
		if !ok {
			continue
		}

		// 用户 → 池子（支付方向）
		if userToPool == nil &&
			(pt.SrcAccount == userToken1 || pt.SrcAccount == userToken2) &&
			(pt.DestAccount == poolToken1 || pt.DestAccount == poolToken2) {
			// 避免与 poolToUser 冲突（如地址重叠）
			if isTransferConflict(pt, poolToUser) {
				continue
			}
			userToPool = pt
			maxIndex = i
			continue
		}

		// 池子 → 用户（接收方向）
		if poolToUser == nil &&
			(pt.SrcAccount == poolToken1 || pt.SrcAccount == poolToken2) &&
			(pt.DestAccount == userToken1 || pt.DestAccount == userToken2) {
			// 避免与 userToPool 冲突（如地址重叠）
			if isTransferConflict(pt, userToPool) {
				continue
			}
			poolToUser = pt
			maxIndex = i
			continue
		}

		// 两个方向都已匹配到，提前结束
		if userToPool != nil && poolToUser != nil {
			break
		}
	}

	if userToPool == nil || poolToUser == nil {
		return nil
	}
	if userToPool.Token == poolToUser.Token {
		return nil
	}

	return &SwapTransferResult{
		UserToPool: userToPool,
		PoolToUser: poolToUser,
		MaxIndex:   maxIndex,
	}
}

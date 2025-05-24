package spltoken

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
)

// RegisterHandlers 注册 token 的所有指令处理逻辑
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.TokenProgram] = handleTokenInstruction
	m[consts.TokenProgram2022] = handleTokenInstruction
}

// handleTokenInstruction 负责识别并解析 Token Transfer 类型的指令。
// 返回生成的事件（若有）和下一条待处理的指令索引。
func handleTokenInstruction(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]
	if len(ix.Data) == 0 {
		return nil, current + 1
	}

	switch ix.Data[0] {
	case byte(sdktoken.InstructionTransfer), byte(sdktoken.InstructionTransferChecked):
		return extractTokenTransferEvent(ctx, instrs, current)

	case byte(sdktoken.InstructionInitializeAccount),
		byte(sdktoken.InstructionInitializeAccount2),
		byte(sdktoken.InstructionInitializeAccount3):
		tryFillBalanceFromInitAccount(ctx, ix)
		return nil, current + 1

	default:
		// 非关心的 TokenProgram 指令，忽略
		return nil, current + 1
	}
}

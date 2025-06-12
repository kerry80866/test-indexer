package spltoken

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pkg/types"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
)

// RegisterHandlers 注册 token 的所有指令处理逻辑
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.TokenProgram] = handleTokenInstruction
	m[consts.TokenProgram2022] = handleTokenInstruction
}

// handleTokenInstruction 根据 SPL Token 指令类型分派至对应解析函数。
func handleTokenInstruction(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]
	if len(ix.Data) == 0 {
		return -1
	}

	switch ix.Data[0] {
	case byte(sdktoken.InstructionTransfer),
		byte(sdktoken.InstructionTransferChecked):
		return extractTokenTransferEvent(ctx, instrs, current)

	case byte(sdktoken.InstructionMintTo),
		byte(sdktoken.InstructionMintToChecked):
		return extractTokenMintToEvent(ctx, instrs, current)

	case byte(sdktoken.InstructionBurn),
		byte(sdktoken.InstructionBurnChecked):
		return extractTokenBurnEvent(ctx, instrs, current)

	default:
		// 忽略非关心的 TokenProgram 指令
		return -1
	}
}

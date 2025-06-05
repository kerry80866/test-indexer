package raydiumcpmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	"encoding/binary"
)

const (
	SwapBaseInput uint64 = 0x8fbe5adac41e33de
	SwapBaseOut   uint64 = 0x37d96256a34ab4ad
)

// RegisterHandlers 注册 RaydiumV4 相关 Program 的指令解析器（仅处理 CLMM Program）
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.RaydiumCPMMProgram] = handleInstruction
}

func handleInstruction(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]

	// 指令 data 至少应包含 8 字节方法 ID
	if len(ix.Data) < 8 {
		return nil, current + 1
	}

	// 解析方法ID
	methodID := binary.BigEndian.Uint64(ix.Data[:8])

	// 解析方法ID
	switch methodID {
	case SwapBaseInput, SwapBaseOut:
		return extractSwapEvent(ctx, instrs, current, methodID)
	default:
		// 未知方法ID，跳过该指令
		return nil, current + 1
	}
}

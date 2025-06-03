package raydiumclmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	"encoding/binary"
)

const (
	Swap   uint64 = 0xf8c69e91e17587c8
	SwapV2 uint64 = 0x2b04ed0b1ac91e62
)

// RegisterHandlers 注册 RaydiumV4 相关 Program 的指令解析器（仅处理 CLMM Program）
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.RaydiumCLMMProgram] = handleRaydiumClmmInstruction
}

func handleRaydiumClmmInstruction(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]

	// 指令 data 至少应包含 8 字节方法 ID
	if len(ix.Data) < 8 {
		return nil, current + 1
	}

	// 提取前 8 字节方法编号，进行分发
	switch binary.BigEndian.Uint64(ix.Data[:8]) {
	case Swap, SwapV2:
		return extractSwapEvent(ctx, instrs, current)

	default:
		// 未识别的指令，直接跳过
		return nil, current + 1
	}
}

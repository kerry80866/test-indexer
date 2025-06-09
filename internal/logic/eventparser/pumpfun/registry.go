package pumpfun

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	"encoding/binary"
)

const (
	Create  uint64 = 0x181ec828051c0777
	Buy     uint64 = 0x66063d1201daebea
	Sell    uint64 = 0x33e685a4017f83ad
	Migrate uint64 = 0x9beae792ec9ea21e
)

// RegisterHandlers 注册 RaydiumV4 相关 Program 的指令解析器（仅处理 CLMM Program）
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.PumpFunProgram] = handleInstruction
}

func handleInstruction(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 指令 data 至少应包含 8 字节方法 ID
	if len(ix.Data) < 8 {
		return -1
	}

	switch binary.BigEndian.Uint64(ix.Data[:8]) {
	case Create:
		return extractCreateEvent(ctx, instrs, current)
	case Buy:
		return extractSwapEvent(ctx, instrs, current, true)
	case Sell:
		return extractSwapEvent(ctx, instrs, current, false)
	case Migrate:
		return extractMigrateEvent(ctx, instrs, current)
	default:
		return -1
	}
}

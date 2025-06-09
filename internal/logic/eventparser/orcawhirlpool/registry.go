package orcawhirlpool

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	"encoding/binary"
)

const (
	// Swap 系列
	Swap  uint64 = 0xf8c69e91e17587c8
	Swap2 uint64 = 0x2b04ed0b1ac91e62

	// Create Pool
	InitializePool   uint64 = 0x5fb40aac54aee828
	InitializePoolV2 uint64 = 0xcf2d57f21b3fcc43

	// 添加流动性
	IncreaseLiquidity   uint64 = 0x2e9cf3760dcdfbb2
	IncreaseLiquidityV2 uint64 = 0x851d59df45eeb00a

	// 添加流动性
	DecreaseLiquidity   uint64 = 0xa026d06f685b2c01
	DecreaseLiquidityV2 uint64 = 0x3a7fbc3e4f52c460
)

// RegisterHandlers 注册 RaydiumV4 相关 Program 的指令解析器（仅处理 CLMM Program）
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.OrcaWhirlpoolProgram] = handleInstruction
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

	// 提取前 8 字节方法编号，进行分发
	switch binary.BigEndian.Uint64(ix.Data[:8]) {
	case Swap:
		return extractSwapEvent(ctx, instrs, current)

	case Swap2:
		return extractSwap2Event(ctx, instrs, current)

	case InitializePool:
		return extractEventForInitializePool(ctx, instrs, current)

	case InitializePoolV2:
		return extractEventForInitializePoolV2(ctx, instrs, current)

	case IncreaseLiquidity:
		return extractEventForIncreaseLiquidity(ctx, instrs, current)

	case IncreaseLiquidityV2:
		return extractEventForIncreaseLiquidityV2(ctx, instrs, current)

	case DecreaseLiquidity:
		return extractEventForRemoveDecreaseLiquidity(ctx, instrs, current)

	case DecreaseLiquidityV2:
		return extractEventForRemoveDecreaseLiquidityV2(ctx, instrs, current)

	default:
		// 未识别的指令，直接跳过
		return -1
	}
}

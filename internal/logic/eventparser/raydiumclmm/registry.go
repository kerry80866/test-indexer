package raydiumclmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/types"
	"encoding/binary"
)

const (
	Swap                       uint64 = 0xf8c69e91e17587c8
	SwapV2                     uint64 = 0x2b04ed0b1ac91e62
	IncreaseLiquidity          uint64 = 0x2e9cf3760dcdfbb2
	IncreaseLiquidityV2        uint64 = 0x851d59df45eeb00a
	OpenPositionWithToken22Nft uint64 = 0x4dffae527d1dc92e
	OpenPositionV2             uint64 = 0x4db84ad67056f1c7
	DecreaseLiquidityV2        uint64 = 0x3a7fbc3e4f52c460
	DecreaseLiquidity          uint64 = 0xa026d06f685b2c01
	CreatePool                 uint64 = 0xe992d18ecf6840bc
)

// RegisterHandlers 注册 RaydiumV4 相关 Program 的指令解析器（仅处理 CLMM Program）
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.RaydiumCLMMProgram] = handleInstruction
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
	case Swap, SwapV2: // 交易指令（买/卖）
		return extractSwapEvent(ctx, instrs, current)

	case IncreaseLiquidity: // 添加流动性（老版）
		return extractEventForIncreaseLiquidity(ctx, instrs, current)

	case IncreaseLiquidityV2: // 添加流动性（新版）
		return extractEventForIncreaseLiquidityV2(ctx, instrs, current)

	case OpenPositionWithToken22Nft: // 创建 Position 并添加流动性
		return extractEventForOpenPositionWithToken22Nft(ctx, instrs, current)

	case OpenPositionV2: // 创建 Position 并添加流动性
		return extractEventForOpenPositionV2(ctx, instrs, current)

	case DecreaseLiquidityV2: // 移除流动性（新版）
		return extractEventForDecreaseLiquidityV2(ctx, instrs, current)

	case DecreaseLiquidity: // 移除流动性（老版）
		return extractEventForDecreaseLiquidity(ctx, instrs, current)

	case CreatePool:
		return extractEventForCreatePool(ctx, instrs, current)

	default:
		return -1
	}
}

package meteoradlmm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/types"
	"encoding/binary"
)

const (
	// Swap 系列
	Swap                 uint64 = 0xf8c69e91e17587c8
	Swap2                uint64 = 0x414b3f4ceb5b5b88
	SwapExactOut         uint64 = 0xfa49652126cf4bb8
	SwapExactOut2        uint64 = 0x2bd7f784893cf351
	SwapWithPriceImpact2 uint64 = 0x4a62c0d6b1334b33

	// Create Pool
	InitializePair2          uint64 = 0x493b2478ed536cc6
	InitializeCustomPair     uint64 = 0x2e2729876fb7c840
	InitializeCustomPair2    uint64 = 0xf349817e3313f16b
	InitializePermissionPair uint64 = 0x6c66d555fb033515

	// 添加流动性
	AddLiquidity2                 uint64 = 0xe4a24e1c46db7473
	AddLiquidityByWeight          uint64 = 0x1c8cee63e7a21595
	AddLiquidityByStrategy        uint64 = 0x0703967f94283dc8
	AddLiquidityByStrategy2       uint64 = 0x03dd95da6f8d76d5
	AddLiquidityByStrategyOneSide uint64 = 0x2905eeaf64e106cd
	AddLiquidity                  uint64 = 0xb59d59438fb63448
	AddLiquidityOneSide           uint64 = 0x5e9b6797465fdca5
	AddLiquidityOneSidePrecise    uint64 = 0xa1c26754ab47fa9a
	AddLiquidityOneSidePrecise2   uint64 = 0x2133a3c975627de7

	// 添加流动性
	RemoveLiquidity         uint64 = 0x5055d14818ceb16c
	RemoveLiquidity2        uint64 = 0xe6d7527ff165e392
	RemoveLiquidityByRange  uint64 = 0x1a526698f04a691a
	RemoveLiquidityByRange2 uint64 = 0xcc02c391359191cd
	RemoveAllLiquidity      uint64 = 0x0a333d2370691855
)

// RegisterHandlers 注册 RaydiumV4 相关 Program 的指令解析器（仅处理 CLMM Program）
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.MeteoraDLMMProgram] = handleInstruction
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
	case Swap, Swap2, SwapExactOut, SwapExactOut2, SwapWithPriceImpact2:
		return extractSwapEvent(ctx, instrs, current)

		// Create Pool 系列
	case InitializePair2:
		return extractEventForInitializePair2(ctx, instrs, current)
	case InitializeCustomPair:
		return extractEventForInitializeCustomPair(ctx, instrs, current)
	case InitializeCustomPair2:
		return extractEventForInitializeCustomPair2(ctx, instrs, current)
	case InitializePermissionPair:
		return extractEventForInitializePermissionLbPair(ctx, instrs, current)

		// AddLiquidity 系列
	case AddLiquidity2:
		return extractEventForAddLiquidity2(ctx, instrs, current)
	case AddLiquidityByWeight:
		return extractEventForAddLiquidityByWeight(ctx, instrs, current)
	case AddLiquidityByStrategy:
		return extractEventForAddLiquidityByStrategy(ctx, instrs, current)
	case AddLiquidityByStrategy2:
		return extractEventForAddLiquidityByStrategy2(ctx, instrs, current)
	case AddLiquidityByStrategyOneSide:
		return extractEventForAddLiquidityByStrategyOneSide(ctx, instrs, current)
	case AddLiquidity:
		return extractEventForAddLiquidity(ctx, instrs, current)
	case AddLiquidityOneSide:
		return extractEventForAddLiquidityOneSide(ctx, instrs, current)
	case AddLiquidityOneSidePrecise:
		return extractEventForAddLiquidityOneSidePrecise(ctx, instrs, current)
	case AddLiquidityOneSidePrecise2:
		return extractEventForAddLiquidityOneSidePrecise2(ctx, instrs, current)

		// RemoveLiquidity 系列
	case RemoveLiquidity:
		return extractEventForRemoveLiquidity(ctx, instrs, current)
	case RemoveLiquidity2:
		return extractEventForRemoveLiquidity2(ctx, instrs, current)
	case RemoveLiquidityByRange:
		return extractEventForRemoveLiquidityByRange(ctx, instrs, current)
	case RemoveLiquidityByRange2:
		return extractEventForRemoveLiquidityByRange2(ctx, instrs, current)
	case RemoveAllLiquidity:
		return extractEventForRemoveAllLiquidity(ctx, instrs, current)

	default:
		// 未识别的指令，直接跳过
		return -1
	}
}

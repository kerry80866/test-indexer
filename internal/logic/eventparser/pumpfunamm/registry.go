package pumpfunamm

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/pkg/types"
	"encoding/binary"
)

const (
	Buy        uint64 = 0x66063d1201daebea
	Sell       uint64 = 0x33e685a4017f83ad
	Deposit    uint64 = 0xf223c68952e1f2b6
	Withdraw   uint64 = 0xb712469c946da122
	CreatePool uint64 = 0xe992d18ecf6840bc
)

// RegisterHandlers 注册 RaydiumV4 相关 Program 的指令解析器（仅处理 CLMM Program）
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.PumpFunAMMProgram] = handleInstruction
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
	case Buy, Sell:
		return extractSwapEvent(ctx, instrs, current)

	case Deposit:
		return extractAddLiquidityEvent(ctx, instrs, current)

	case Withdraw:
		return extractRemoveLiquidityEvent(ctx, instrs, current)

	case CreatePool:
		return extractCreatePoolEvent(ctx, instrs, current)

	default:
		// 未识别的指令，直接跳过
		return -1
	}
}

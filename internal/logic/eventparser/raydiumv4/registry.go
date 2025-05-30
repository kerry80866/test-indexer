package raydiumv4

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
)

// 来源, https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs
const (
	Initialize2 = 1
	Deposit     = 3
	Withdraw    = 4
	SwapBaseIn  = 9
	SwapBaseOut = 11
)

// RegisterHandlers 注册 RaydiumV4 的所有指令处理逻辑
func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.RaydiumV4Program] = handleRaydiumV4Instruction
}

// handleRaydiumV4Instruction 是 RaydiumV4 的主分发入口
func handleRaydiumV4Instruction(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	ix := instrs[current]
	if len(ix.Data) == 0 {
		return nil, current + 1
	}

	switch ix.Data[0] {
	case SwapBaseIn, SwapBaseOut:
		return extractRaydiumV4SwapEvent(ctx, instrs, current)

	case Deposit:
		return extractRaydiumV4AddLiquidityEvent(ctx, instrs, current)

	case Withdraw:
		// TODO: return extractLiquidity(...)
		return nil, current + 1

	default:
		return nil, current + 1
	}
}

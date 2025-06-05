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
	m[consts.RaydiumV4Program] = handleInstruction
}

// handleInstruction 是 RaydiumV4 的主分发入口
func handleInstruction(
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
		return extractSwapEvent(ctx, instrs, current)

	case Deposit:
		return extractAddLiquidityEvent(ctx, instrs, current)

	case Withdraw:
		return nil, current + 1

	default:
		return nil, current + 1
	}
}

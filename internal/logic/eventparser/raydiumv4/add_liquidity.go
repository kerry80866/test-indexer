package raydiumv4

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
)

func extractRaydiumV4AddLiquidityEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (*core.Event, int) {
	return nil, current + 3
}

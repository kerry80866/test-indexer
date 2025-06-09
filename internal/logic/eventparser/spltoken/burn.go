package spltoken

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
)

// extractTokenBurnEvent 尝试将当前指令解析为 SPL Token 的 Burn 或 BurnChecked 操作。
// 若解析成功，则构造 BurnEvent 并添加至上下文。
func extractTokenBurnEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 解析 Burn / BurnChecked 指令
	parsedBurn, ok := common.ParseBurnInstruction(ctx, ix)
	if !ok {
		return -1
	}

	ctx.AddEvent(common.BuildBurnEvent(ctx, parsedBurn))
	return current + 1
}

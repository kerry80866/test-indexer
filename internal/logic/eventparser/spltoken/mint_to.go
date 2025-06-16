package spltoken

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
)

// extractTokenMintToEvent 尝试将当前指令解析为 SPL Token 的 MintTo 或 MintToChecked 操作。
// 若解析成功，则构造 MintToEvent 并添加至上下文。
func extractTokenMintToEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 解析 MintTo / MintToChecked 指令
	parsedMintTo, ok := common.ParseMintToInstruction(ctx, ix)
	if !ok {
		return -1
	}

	ctx.AddEvent(common.BuildMintToEvent(ctx, parsedMintTo))
	return current + 1
}

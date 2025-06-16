package spltoken

import (
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/internal/logic/eventparser/common"
)

// extractTokenTransferEvent 尝试将当前指令解析为 SPL Token 的 Transfer 或 TransferChecked 事件。
// 若解析成功，则构造 TransferEvent 并添加至上下文。
func extractTokenTransferEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]

	// 解析 Transfer / TransferChecked 指令
	parsedTransfer, ok := common.ParseTransferInstruction(ctx, ix)
	if !ok {
		return -1
	}

	ctx.AddEvent(common.BuildTransferEvent(ctx, parsedTransfer))
	return current + 1
}

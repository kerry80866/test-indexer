package eventparser

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/logic/eventparser/raydiumv4"
	"dex-indexer-sol/internal/logic/eventparser/spltoken"
	"dex-indexer-sol/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"runtime/debug"
)

// handlers 是 Solana ProgramID → 对应事件解析 handler 的路由表。
// 所有协议模块通过 RegisterHandlers 注册进该表。
var handlers = map[types.Pubkey]common.InstructionHandler{}

// Init 初始化所有 handler 注册器等解析所需状态
func Init() {
	spltoken.RegisterHandlers(handlers)
	raydiumv4.RegisterHandlers(handlers)
}

func ExtractEventsFromTx(adaptedTx *core.AdaptedTx) []*core.Event {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("panic in ExtractEventsFromTx: %+v\nstack: %s", r, debug.Stack())
		}
	}()

	ctx := common.BuildParserContext(adaptedTx)
	instrs := ctx.Tx.Instructions

	// 预处理：扫描 InitializeAccount 指令，补全 TokenAccount → Mint → Owner 映射
	common.PreScanInitAccountBalances(ctx, instrs)

	result := make([]*core.Event, 0, len(instrs))
	for i := 0; i < len(instrs); {
		ix := instrs[i]
		if handler, ok := handlers[ix.ProgramID]; ok {
			event, next := handler(ctx, instrs, i)
			if event != nil {
				result = append(result, event)
			}
			i = next
		} else {
			i++
		}
	}
	return result
}

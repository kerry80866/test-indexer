package eventparser

import (
	"dex-indexer-sol/internal/logger"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/logic/eventparser/meteoradlmm"
	"dex-indexer-sol/internal/logic/eventparser/orcawhirlpool"
	"dex-indexer-sol/internal/logic/eventparser/pumpfun"
	"dex-indexer-sol/internal/logic/eventparser/pumpfunamm"
	"dex-indexer-sol/internal/logic/eventparser/raydiumclmm"
	"dex-indexer-sol/internal/logic/eventparser/raydiumcpmm"
	"dex-indexer-sol/internal/logic/eventparser/raydiumv4"
	"dex-indexer-sol/internal/logic/eventparser/spltoken"
	"dex-indexer-sol/internal/types"
	"github.com/mr-tron/base58"
	"runtime/debug"
)

// handlers 是 Solana ProgramID → 对应事件解析 handler 的路由表。
// 所有协议模块通过 RegisterHandlers 注册进该表。
var handlers = map[types.Pubkey]common.InstructionHandler{}

// Init 初始化所有 handler 注册器等解析所需状态
func Init() {
	spltoken.RegisterHandlers(handlers)
	raydiumv4.RegisterHandlers(handlers)
	raydiumclmm.RegisterHandlers(handlers)
	raydiumcpmm.RegisterHandlers(handlers)
	pumpfunamm.RegisterHandlers(handlers)
	pumpfun.RegisterHandlers(handlers)
	meteoradlmm.RegisterHandlers(handlers)
	orcawhirlpool.RegisterHandlers(handlers)
}

func ExtractEventsFromTx(adaptedTx *core.AdaptedTx) (result []*core.Event) {
	defer func() {
		if r := recover(); r != nil {
			txHash := base58.Encode(adaptedTx.Signature)
			logger.Errorf("[eventparser::ExtractEventsFromTx] panic tx=%s: %+v\nstack: %s", txHash, r, debug.Stack())
			result = nil
		}
	}()

	ctx := common.BuildParserContext(adaptedTx)
	instrs := ctx.Tx.Instructions

	// 扫描 InitializeAccount 指令，补全 TokenAccount → Mint → Owner 映射
	common.PreScanInitAccountBalances(ctx, instrs)

	for i := 0; i < len(instrs); {
		ix := instrs[i]
		if handler, ok := handlers[ix.ProgramID]; ok {
			if next := handler(ctx, instrs, i); next > i {
				i = next
				continue
			}
		}
		i++
	}
	return ctx.TakeEvents()
}

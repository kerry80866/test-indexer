package common

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"github.com/mr-tron/base58"
)

// ParserContext 是事件解析时传入每个 handler 的上下文。
// 包含当前交易的完整信息、日志、Token 精度、事件列表等。
type ParserContext struct {
	Tx        *core.AdaptedTx // 当前交易（含 Slot、指令、账户等）
	TxIndex   uint32          // 交易在区块中的序号
	TxHash    []byte          // 交易签名（64 字节原始数据）
	Signers   [][]byte        // 签名者列表
	BlockTime int64           // 区块时间（Unix 秒）
	Slot      uint64          // 区块 Slot 高度

	LogMessages []string                            // tx.Meta.LogMessages，用于日志判断
	Balances    map[types.Pubkey]*core.TokenBalance // tokenAccount → TokenBalance 映射

	Events []*core.Event // 当前交易解析出的事件
}

// TxHashString 返回交易签名的 Base58 编码形式。
func (ctx *ParserContext) TxHashString() string {
	return base58.Encode(ctx.TxHash)
}

// AddEvent 添加一个事件到当前上下文中。
func (ctx *ParserContext) AddEvent(event *core.Event) {
	ctx.Events = append(ctx.Events, event)
}

// TakeEvents 返回并清空当前上下文中的事件。
func (ctx *ParserContext) TakeEvents() []*core.Event {
	events := ctx.Events
	ctx.Events = nil
	return events
}

// InstructionHandler 定义事件指令的解析函数签名。
// 参数：
//   - ctx:     当前解析上下文
//   - instrs:  当前交易中的扁平化指令列表（含 inner 指令）
//   - current: 当前处理的指令索引（instrs[current]）
//
// 返回值：
//   - 若返回值 > current：表示成功处理，返回下一条待处理指令的索引（支持跳过多条指令）
//   - 若返回值 <= current：表示未匹配或处理失败
type InstructionHandler func(ctx *ParserContext, instrs []*core.AdaptedInstruction, current int) int

// BuildParserContext 构造标准的事件解析上下文。
func BuildParserContext(tx *core.AdaptedTx) *ParserContext {
	return &ParserContext{
		Tx:          tx,
		TxIndex:     tx.TxIndex,
		Slot:        tx.TxCtx.Slot,
		BlockTime:   tx.TxCtx.BlockTime,
		TxHash:      tx.Signature,
		Signers:     tx.Signers,
		LogMessages: tx.LogMessages,
		Balances:    tx.Balances,
		Events:      make([]*core.Event, 0, len(tx.Instructions)),
	}
}

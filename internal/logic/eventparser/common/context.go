package common

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
)

// ParserContext 是传入每个事件 handler 的解析上下文。
// 它包含当前交易的完整结构、事件通用字段模板、日志信息、Token 精度等，
// 用于支持事件识别、构造与序列化。
type ParserContext struct {
	Tx        *core.AdaptedTx // 原始交易上下文，包含 slot、指令、账户等
	TxIndex   uint32          // 当前交易在区块中的序号（基于 Geyser TransactionIndex）
	TxHash    []byte          // 交易签名（64 字节原始数据）
	TxFrom    []byte          // 交易发起者（通常为 accountKeys[0]）
	BlockTime int64           // 区块时间戳（Unix 秒级）
	Slot      uint64          // 当前区块 Slot（Solana 高度单位）

	LogMessages []string                            // tx.Meta.LogMessages，用于部分协议日志判定
	Balances    map[types.Pubkey]*core.TokenBalance // tokenAccount → TokenBalance
}

// InstructionHandler 定义了统一的事件指令解析函数签名。
// 用于从扁平化的 Solana 指令序列中解析事件。
//
// 参数：
//   - ctx:     当前解析上下文（包含 txIndex、BaseEvent 等基础信息）
//   - instrs:  当前交易中已展平的指令列表（含主指令与对应 inner 指令）
//   - current: 当前正在处理的指令索引（instrs[current]）
//
// 返回值：
//   - event: 若成功解析出事件，返回成功解析的事件；否则为 nil
//   - next:  返回下一条待处理的指令索引（通常为 current+1，可跳过多条）
type InstructionHandler func(ctx *ParserContext, instrs []*core.AdaptedInstruction, current int) (event *core.Event, next int)

// BuildParserContext 构造标准化的事件解析上下文。
// 提前设置好 BaseEvent 模板和其他字段。
func BuildParserContext(tx *core.AdaptedTx) *ParserContext {
	return &ParserContext{
		Tx:          tx,
		TxIndex:     tx.TxIndex,
		Slot:        tx.TxCtx.Slot,
		BlockTime:   tx.TxCtx.BlockTime,
		TxHash:      tx.Signature,
		TxFrom:      tx.Signer[:],
		LogMessages: tx.LogMessages,
		Balances:    tx.Balances,
	}
}

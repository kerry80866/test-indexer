package core

import (
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/pb"
)

// ParsedTxResult 表示某笔交易解析后的中间结构，包含余额和事件
type ParsedTxResult struct {
	TxIndex  int                            // 当前交易在 block 中的序号
	Balances map[types.Pubkey]*TokenBalance // TokenAccount → 余额变动信息
	Events   []*Event                       // 已解析出的事件（Trade / Transfer / Balance 等）
}

type Event struct {
	ID        uint32    // slot 内唯一事件 ID（txIndex、ixIndex、innerIndex 组合）
	EventType uint32    // 枚举型，表示事件类别（Trade/Transfer/Balance）
	Key       []byte    // Kafka 分区 key，建议为 owner 或 base token
	Event     *pb.Event // Protobuf 封装的实际事件内容（包含 Transfer、Trade 等变体）
}

// BuildEventID 构造事件唯一标识 ID（uint32），由 txIndex、ixIndex、innerIndex 组合而成：
//   - txIndex    (16 bits): 当前交易在区块中的序号，范围 0 ~ 65535
//   - ixIndex    (8 bits) : 当前交易中的主指令序号，范围 0 ~ 255
//   - innerIndex (8 bits) : inner 指令的序号，主指令时为 -1，编码时加 1 变为 0，顺序对齐
//
// 编码结构：
//
//	[ 16 bits txIndex ] [ 8 bits ixIndex ] [ 8 bits (innerIndex + 1) ]
func BuildEventID(txIndex uint32, ixIndex int16, innerIndex int16) uint32 {
	return (txIndex << 16) | (uint32(ixIndex) << 8) | uint32(innerIndex+1)
}

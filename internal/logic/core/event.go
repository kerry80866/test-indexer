package core

import (
	"dex-indexer-sol/internal/pkg/types"
	"dex-indexer-sol/pb"
)

// ParsedTxResult 表示某笔交易解析后的中间结构，包含余额和事件
type ParsedTxResult struct {
	Balances    map[types.Pubkey]*TokenBalance // TokenAccount → 余额变动信息
	Events      []*Event                       // 已解析出的事件（Trade / Transfer / Balance 等）
	PriceEvents []*PriceEvent
}

type Event struct {
	ID        uint64    // 唯一事件 ID
	EventType uint32    // 枚举型，表示事件类别（Trade/Transfer/Balance）
	Key       []byte    // Kafka 分区 key
	Event     *pb.Event // Protobuf 封装的实际事件内容（包含 Transfer、Trade 等变体）
}

type PriceEvent struct {
	ID          uint64       // 唯一事件 ID
	PriceUsd    float64      // USD 价格
	PublishTime int64        // 发布事件
	TokenMint   types.Pubkey // 哪个 token
}

// BuildEventID 构造事件唯一标识 ID（uint64），由 slot、txIndex、ixIndex、innerIndex 组合而成：
//   - slot       (32 bits) : 区块高度，放在高位，确保唯一性
//   - txIndex    (16 bits) : 当前交易在区块中的序号，范围 0 ~ 65535
//   - ixIndex    (8 bits)  : 当前交易中的主指令序号，范围 0 ~ 255
//   - innerIndex (8 bits)  : inner 指令的序号
func BuildEventID(slot uint64, txIndex uint32, ixIndex uint16, innerIndex uint16) uint64 {
	return slot<<32 | uint64(txIndex)<<16 | uint64(ixIndex)<<8 | uint64(innerIndex)
}

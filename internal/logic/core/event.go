package core

import "dex-indexer-sol/pb"

type Event struct {
	EventId   uint32    // 构造的唯一事件 ID，基于 txIndex、ixIndex、innerIndex
	EventType uint32    // 自定义事件类型（枚举）
	Key       []byte    // Kafka分区key：余额事件是owner，其它类型是base token
	Event     *pb.Event // 实际事件的 proto 内容
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

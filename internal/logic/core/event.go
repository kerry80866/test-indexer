package core

type Event struct {
	Tx        *AdaptedTx // 所属交易上下文，sendEvent期间只读，用于调试与定位
	EventId   uint32     // ← 来自 proto 的 BaseEvent.event_index
	EventType uint32     // 自定义枚举，如 Swap, Mint 等
	Token     []byte     // 主 token（常用于分区 / 过滤）
	Data      []byte     // 事件原始序列化内容（protobuf / JSON）
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

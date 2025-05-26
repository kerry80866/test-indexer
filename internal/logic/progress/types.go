package progress

// SlotStatus 表示 slot 的处理状态（统一 Redis 与 DB 编码）
type SlotStatus int

const (
	SlotUnknown   SlotStatus = 0 // Redis 不存在
	SlotProcessed SlotStatus = 1 // ✅ 已处理成功
	SlotInvalid   SlotStatus = 2 // ❌ 明确结构错误、跳过
	SlotPending   SlotStatus = 3 // 🕒 Redis 标记中，暂未完成（仅 Redis 用）
)

// EventType 表示不同类型的进度事件（用于区分 Redis key、表名）
type EventType int

const (
	EventSlot EventType = 0
)

func (et EventType) TableName() string {
	return "progress_slot"
}

// Source 表示事件来源模块（grpc、rpc）
const (
	SourceUnknown int16 = 0
	SourceGrpc    int16 = 1
	SourceRpc     int16 = 2
)

func SourceName(src int16) string {
	switch src {
	case SourceGrpc:
		return "grpc"
	case SourceRpc:
		return "rpc"
	default:
		return "unknown"
	}
}

// SlotRecord 表示一条待写入 DB 的 slot 记录
type SlotRecord struct {
	Slot      uint64     // Solana slot
	Source    int16      // 来源：1=grpc, 2=rpc
	BlockTime int64      // Unix timestamp（秒）
	Status    SlotStatus // 处理状态：1=已处理，2=无效
}

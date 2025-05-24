// logic/progress/manager.go
package progress

import (
	"context"
	"time"
)

// ProgressManager 统一封装 Redis + DB + 缓存，控制进度判重与写入
type ProgressManager struct {
	redis           *RedisProgressStore
	db              *DBProgressStore
	buffer          *slotBuffer
	recentThreshold time.Duration // 新 block 的判断阈值
}

func NewProgressManager(redis *RedisProgressStore, db *DBProgressStore, recentThresholdSec int) *ProgressManager {
	return &ProgressManager{
		redis:           redis,
		db:              db,
		buffer:          newSlotBuffer(),
		recentThreshold: time.Duration(recentThresholdSec) * time.Second,
	}
}

// ShouldProcessSlot 用于判断是否需要处理该 slot：
// - 如果 block 是“最近的”，直接处理
// - 否则 Redis 查状态 + fallback 到 DB
func (pm *ProgressManager) ShouldProcessSlot(ctx context.Context, slot uint64, eventType EventType, blockTime int64) (bool, error) {
	if time.Since(time.Unix(blockTime, 0)) <= pm.recentThreshold {
		return true, nil // 近期 block，直接处理
	}

	// 旧 block 判重：先查 Redis
	status, err := pm.redis.GetSlotStatus(ctx, slot, eventType)
	if err != nil {
		return false, err
	}
	if status == SlotProcessed || status == SlotInvalid {
		return false, nil
	}

	// fallback 到 DB
	exists, err := pm.db.CheckSlotExists(ctx, slot, eventType)
	if err != nil {
		return false, err
	}
	if exists {
		_ = pm.redis.MarkSlotProcessed(ctx, slot, eventType)
		return false, nil
	} else {
		_ = pm.redis.MarkSlotInvalid(ctx, slot, eventType)
		return false, nil
	}
}

// MarkSlotStatus 标记某 slot 的处理状态（如已处理、结构非法等）
// 会同时更新 Redis 与 slotBuffer（供后续批量写入 DB）
func (pm *ProgressManager) MarkSlotStatus(
	ctx context.Context,
	slot uint64,
	eventType EventType,
	source int16,
	blockTime int64,
	status SlotStatus,
) error {
	var err error

	// 写入 Redis 状态
	switch status {
	case SlotProcessed:
		err = pm.redis.MarkSlotProcessed(ctx, slot, eventType)
	case SlotInvalid:
		err = pm.redis.MarkSlotInvalid(ctx, slot, eventType)
	default:
		return nil // SlotUnknown / SlotPending 不参与记录
	}
	if err != nil {
		return err
	}

	// 加入缓冲区，待后续批量持久化 DB
	pm.buffer.Add(eventType, &SlotRecord{
		Slot:      slot,
		Source:    source,
		BlockTime: blockTime,
		Status:    status,
	})
	return nil
}

// StartFlushLoop 启动后台定时 flush
func (pm *ProgressManager) StartFlushLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			flushed := pm.buffer.Flush()
			for et, list := range flushed {
				if len(list) == 0 {
					continue
				}
				err := pm.db.BatchInsertProcessedSlots(ctx, list, et)
				if err != nil {
					// 打日志即可，buffer 已清空
					// 可扩展重试或告警
				}
			}
		}
	}
}

// StartGCLoop 启动后台 GC 清理（每 interval 执行一次，对所有事件类型清理历史 slot 记录）
func (pm *ProgressManager) StartGCLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, et := range []EventType{EventTrade, EventBalance} {
					_ = pm.db.DeleteOldSlots(ctx, et)
				}
			}
		}
	}()
}

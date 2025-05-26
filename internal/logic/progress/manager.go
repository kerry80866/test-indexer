package progress

import (
	"context"
	"time"
)

type ProgressManager struct {
	redis           *RedisProgressStore
	db              *DBProgressStore
	recentThreshold time.Duration
}

func NewProgressManager(redis *RedisProgressStore, db *DBProgressStore, recentThresholdSec int) *ProgressManager {
	return &ProgressManager{
		redis:           redis,
		db:              db,
		recentThreshold: time.Duration(recentThresholdSec) * time.Second,
	}
}

// ShouldProcessSlot 判断是否需要处理该 slot：
// - 近期 block 永远处理（不判重）
// - 否则查 Redis + fallback DB 判重
func (pm *ProgressManager) ShouldProcessSlot(ctx context.Context, slot uint64, blockTime int64) (bool, error) {
	if time.Since(time.Unix(blockTime, 0)) <= pm.recentThreshold {
		return true, nil
	}

	// Step 1: Redis
	status, err := pm.redis.GetSlotStatus(ctx, slot)
	if err != nil {
		return false, err
	}
	if status == SlotProcessed || status == SlotInvalid {
		return false, nil
	}

	// Step 2: fallback DB
	exists, err := pm.db.CheckSlotExists(ctx, slot)
	if err != nil {
		return false, err
	}
	if exists {
		_ = pm.redis.MarkSlotProcessed(ctx, slot)
		return false, nil
	}
	_ = pm.redis.MarkSlotInvalid(ctx, slot)
	return false, nil
}

// MarkSlotStatus 持久化 slot 状态（支持 SlotProcessed / SlotInvalid）
func (pm *ProgressManager) MarkSlotStatus(
	ctx context.Context,
	source int16,
	slot uint64,
	blockTime int64,
	status SlotStatus,
) error {
	if status != SlotProcessed && status != SlotInvalid {
		return nil
	}

	// Step 1: 落库 DB（Insert or Update）
	err := pm.db.InsertOrUpdateSlot(ctx, &SlotRecord{
		Slot:      slot,
		Source:    source,
		BlockTime: blockTime,
		Status:    status,
	})
	if err != nil {
		return err
	}

	// Step 2: Redis 标记状态（幂等）
	switch status {
	case SlotProcessed:
		return pm.redis.MarkSlotProcessed(ctx, slot)
	case SlotInvalid:
		return pm.redis.MarkSlotInvalid(ctx, slot)
	default:
		return nil // 理论不会进入
	}
}

// StartGCLoop 启动后台 GC 清理历史进度（单表清理）
func (pm *ProgressManager) StartGCLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = pm.db.DeleteOldSlots(ctx)
			}
		}
	}()
}

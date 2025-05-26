package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	progressPrefix = "progress:slot"
	defaultTTL     = 24 * time.Hour
)

// RedisProgressStore 管理 Redis 中的 slot 状态记录（幂等控制）
type RedisProgressStore struct {
	rdb *redis.Client
}

// NewRedisProgressStore 创建 Redis 判重管理器
func NewRedisProgressStore(rdb *redis.Client) *RedisProgressStore {
	return &RedisProgressStore{rdb: rdb}
}

func (r *RedisProgressStore) getKey(slot uint64) string {
	return fmt.Sprintf("%s:%d", progressPrefix, slot)
}

// GetSlotStatus 获取 slot 的处理状态
func (r *RedisProgressStore) GetSlotStatus(ctx context.Context, slot uint64) (SlotStatus, error) {
	val, err := r.rdb.Get(ctx, r.getKey(slot)).Int()
	switch {
	case err == redis.Nil:
		return SlotUnknown, nil
	case err != nil:
		return SlotUnknown, fmt.Errorf("redis get error: %w", err)
	case val == int(SlotProcessed):
		return SlotProcessed, nil
	case val == int(SlotInvalid):
		return SlotInvalid, nil
	case val == int(SlotPending):
		return SlotPending, nil
	default:
		return SlotUnknown, nil
	}
}

// MarkSlotStatus 设置 slot 的处理状态
func (r *RedisProgressStore) MarkSlotStatus(ctx context.Context, slot uint64, status SlotStatus) error {
	return r.rdb.Set(ctx, r.getKey(slot), status, defaultTTL).Err()
}

func (r *RedisProgressStore) MarkSlotProcessed(ctx context.Context, slot uint64) error {
	return r.MarkSlotStatus(ctx, slot, SlotProcessed)
}

func (r *RedisProgressStore) MarkSlotInvalid(ctx context.Context, slot uint64) error {
	return r.MarkSlotStatus(ctx, slot, SlotInvalid)
}

func (r *RedisProgressStore) MarkSlotPending(ctx context.Context, slot uint64) error {
	return r.MarkSlotStatus(ctx, slot, SlotPending)
}

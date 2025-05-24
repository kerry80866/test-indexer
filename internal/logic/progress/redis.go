package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisProgressStore 管理 Redis 中的 slot 状态记录（幂等控制）
type RedisProgressStore struct {
	rdb *redis.Client
}

// Redis key 前缀
const (
	tradePrefix   = "progress:trade:slot"
	balancePrefix = "progress:balance:slot"
	unknownPrefix = "progress:unknown:slot"
)

// 每类事件的 slot TTL（可调）
const (
	tradeTTL   = 7 * 24 * time.Hour
	balanceTTL = 3 * 24 * time.Hour
	defaultTTL = 24 * time.Hour
)

// NewRedisProgressStore 创建 Redis 判重管理器
func NewRedisProgressStore(rdb *redis.Client) *RedisProgressStore {
	return &RedisProgressStore{rdb: rdb}
}

// getKey 构造 Redis key，按事件类型区分
func (r *RedisProgressStore) getKey(slot uint64, eventType EventType) string {
	var prefix string
	switch eventType {
	case EventTrade:
		prefix = tradePrefix
	case EventBalance:
		prefix = balancePrefix
	default:
		prefix = unknownPrefix
	}
	return fmt.Sprintf("%s:%d", prefix, slot)
}

// getTTL 获取 Redis key 的 TTL，按事件类型区分
func (r *RedisProgressStore) getTTL(eventType EventType) time.Duration {
	switch eventType {
	case EventTrade:
		return tradeTTL
	case EventBalance:
		return balanceTTL
	default:
		return defaultTTL
	}
}

// GetSlotStatus 获取 slot 的状态（Unknown / Processed / Invalid / Pending）
func (r *RedisProgressStore) GetSlotStatus(ctx context.Context, slot uint64, eventType EventType) (SlotStatus, error) {
	key := r.getKey(slot, eventType)
	val, err := r.rdb.Get(ctx, key).Int()
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
		return SlotUnknown, nil // 容错处理
	}
}

// MarkSlotStatus 通用设置 slot 的状态
func (r *RedisProgressStore) MarkSlotStatus(ctx context.Context, slot uint64, eventType EventType, status SlotStatus) error {
	key := r.getKey(slot, eventType)
	ttl := r.getTTL(eventType)
	return r.rdb.Set(ctx, key, status, ttl).Err()
}

// MarkSlotProcessed 标记 slot 为已处理
func (r *RedisProgressStore) MarkSlotProcessed(ctx context.Context, slot uint64, eventType EventType) error {
	return r.MarkSlotStatus(ctx, slot, eventType, SlotProcessed)
}

// MarkSlotInvalid 标记 slot 为无效（结构失败、跳过）
func (r *RedisProgressStore) MarkSlotInvalid(ctx context.Context, slot uint64, eventType EventType) error {
	return r.MarkSlotStatus(ctx, slot, eventType, SlotInvalid)
}

// MarkSlotPending 标记 slot 为正在处理（幂等控制）
func (r *RedisProgressStore) MarkSlotPending(ctx context.Context, slot uint64, eventType EventType) error {
	return r.MarkSlotStatus(ctx, slot, eventType, SlotPending)
}

package progress

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DBProgressStore 管理 slot 的 DB 存储
// 写入用于持久记录进度，服务恢复后可用
// 不做高频幂等判重，只 fallback 使用

type DBProgressStore struct {
	db *sql.DB
}

func NewDBProgressStore(db *sql.DB) *DBProgressStore {
	return &DBProgressStore{db: db}
}

// CheckSlotExists 判定某 slot 是否已存在于 DB 中（用于 RPC fallback）
func (d *DBProgressStore) CheckSlotExists(ctx context.Context, slot uint64, eventType EventType) (bool, error) {
	table := getTableName(eventType)
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE slot = $1", table)
	var dummy int
	err := d.db.QueryRowContext(ctx, query, slot).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// BatchInsertProcessedSlots 批量插入 slot 记录，按 batchLimit 分批写入数据库。
// 如果 slot 冲突，交由 insertChunk 中的 ON CONFLICT 策略处理。
func (d *DBProgressStore) BatchInsertProcessedSlots(ctx context.Context, slots []*SlotRecord, eventType EventType) error {
	if len(slots) == 0 {
		return nil
	}

	const batchLimit = 1000
	for i := 0; i < len(slots); i += batchLimit {
		end := i + batchLimit
		if end > len(slots) {
			end = len(slots)
		}
		err := d.insertChunk(ctx, slots[i:end], eventType)
		if err != nil {
			return err
		}
	}
	return nil
}

// insertChunk 插入一批 slot 记录（最多 1000 条）。
// 若主键 slot 冲突，仅更新 status 和 updated_at 字段。
func (d *DBProgressStore) insertChunk(ctx context.Context, slots []*SlotRecord, eventType EventType) error {
	table := getTableName(eventType)
	query := fmt.Sprintf("INSERT INTO %s (slot, source, block_time, status, updated_at) VALUES ", table)

	args := make([]interface{}, 0, len(slots)*5)
	placeholders := ""
	for i, slot := range slots {
		placeholders += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d),", i*5+1, i*5+2, i*5+3, i*5+4, i*5+5)
		args = append(args,
			slot.Slot,
			slot.Source,
			slot.BlockTime,
			slot.Status,
			time.Now().Unix(),
		)
	}

	// 对所有冲突 slot，执行更新：仅更新 status 和 updated_at，其余字段保持原值
	query += placeholders[:len(placeholders)-1] +
		" ON CONFLICT (slot) DO UPDATE SET " +
		"status = EXCLUDED.status, " +
		"updated_at = EXCLUDED.updated_at"

	_, err := d.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteOldSlots 删除历史 slot 记录（用于进度 GC）。
// 会保留最近 7 天的数据，老数据按 slot 值判断。
// 为防止锁表和长事务，采用分批删除（每批最多 1000 条）。
func (d *DBProgressStore) DeleteOldSlots(ctx context.Context, eventType EventType) error {
	table := getTableName(eventType)

	// Step 1: 获取当前最新的 slot，用于计算安全保留下限
	var latestSlot uint64
	latestQuery := fmt.Sprintf("SELECT MAX(slot) FROM %s", table)
	err := d.db.QueryRowContext(ctx, latestQuery).Scan(&latestSlot)
	if err != nil {
		return fmt.Errorf("fetch latest slot failed: %w", err)
	}

	// Step 2: 计算安全保留 slot（保留 7 天）
	// 估算公式：7天 × 每秒 2.5 slot = ~1,512,000 slots
	days := 7 * 24 * 3600
	safeSlot := latestSlot - uint64(float64(days)*2.5)

	// Step 3: 分批删除早于 safeSlot 的历史记录
	batchSize := 1000
	for {
		deleteQuery := fmt.Sprintf(
			"DELETE FROM %s WHERE slot < $1 ORDER BY slot LIMIT %d",
			table, batchSize,
		)

		res, err := d.db.ExecContext(ctx, deleteQuery, safeSlot)
		if err != nil {
			return fmt.Errorf("delete old slots failed: %w", err)
		}

		// 没有更多记录可删，提前退出
		affected, _ := res.RowsAffected()
		if affected == 0 {
			break
		}

		// 打印每轮删除日志，便于监控
		fmt.Printf("[GC] deleted %d rows from %s\n", affected, table)
	}

	return nil
}

func getTableName(eventType EventType) string {
	switch eventType {
	case EventTrade:
		return "progress_trade"
	case EventBalance:
		return "progress_balance"
	default:
		return "progress_unknown"
	}
}

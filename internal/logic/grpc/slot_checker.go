package grpc

import (
	"context"
	"dex-indexer-sol/internal/pkg/logger"
	"github.com/blocto/solana-go-sdk/rpc"
	"sort"
	"time"
)

type SlotRange struct {
	From     uint64
	To       uint64
	SubmitAt time.Time
}

type SlotChecker struct {
	client   *rpc.RpcClient
	rangeCh  chan SlotRange
	ctx      context.Context
	cancel   context.CancelFunc
	endpoint string
}

func NewSlotChecker(endpoint string) *SlotChecker {
	ctx, cancel := context.WithCancel(context.Background())
	client := rpc.NewRpcClient(endpoint)
	return &SlotChecker{
		client:   &client,
		rangeCh:  make(chan SlotRange, 300),
		ctx:      ctx,
		cancel:   cancel,
		endpoint: endpoint,
	}
}

func (s *SlotChecker) Start() {
	go s.run()
}

func (s *SlotChecker) Stop() {
	s.cancel()
}

// Submit 提交一个 slot 范围进行空块检测，闭区间 [from, to]
func (s *SlotChecker) Submit(from, to uint64) {
	if from > to {
		logger.Warnf("[SlotChecker] invalid slot range: from (%d) > to (%d)", from, to)
		return
	}

	r := SlotRange{
		From:     from,
		To:       to,
		SubmitAt: time.Now(),
	}
	select {
	case s.rangeCh <- r:
	default:
		logger.Warnf("[SlotChecker] slot range channel full, dropped: [%d, %d]", from, to)
	}
}

func (s *SlotChecker) run() {
	const maxPendingRanges = 200 // 可调上限
	const delayBeforeCheck = 30 * time.Second

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	ranges := make([]SlotRange, 0, 32)
	ready := make([]SlotRange, 0, 32)
	pending := make([]SlotRange, 0, 32)

	for {
		select {
		case <-s.ctx.Done():
			logger.Infof("[SlotChecker] stopped")
			return

		case r := <-s.rangeCh:
			if len(ranges) >= maxPendingRanges {
				logger.Warnf("[SlotChecker] too many pending ranges (%d), drop [%d, %d]",
					len(ranges), r.From, r.To)
			} else {
				ranges = append(ranges, r)
			}

		case <-ticker.C:
			drainTicker(ticker)
			if len(ranges) == 0 {
				continue
			}

			now := time.Now()

			// 重用切片，避免重复分配
			ready = ready[:0]
			pending = pending[:0]

			for _, r := range ranges {
				if now.Sub(r.SubmitAt) >= delayBeforeCheck {
					ready = append(ready, r)
				} else {
					pending = append(pending, r)
				}
			}

			if len(ready) == 0 {
				continue
			}

			// 串行执行，防止 goroutine 累积
			s.checkSlotRanges(ready)
			ranges = pending
		}
	}
}

func drainTicker(t *time.Ticker) {
	for {
		select {
		case <-t.C:
			// 丢弃多余 tick
		default:
			return
		}
	}
}

func (s *SlotChecker) checkSlotRanges(ranges []SlotRange) {
	merged := mergeRanges(ranges)
	if len(merged) == 0 {
		return
	}

	maxEmptySlots := 0
	for _, r := range ranges {
		maxEmptySlots += int(r.To - r.From + 1)
	}
	confirmedEmptySlots := make(map[uint64]struct{}, maxEmptySlots)

	// 记录失败的查询范围
	failedRanges := make([]SlotRange, 0)

	for _, r := range merged {
		select {
		case <-s.ctx.Done():
			logger.Infof("[SlotChecker] stopped while checking slot range [%d, %d]", r.From, r.To)
			return
		default:
		}

		blocks, err := s.getBlocksWithRetry(r.From, r.To, 3)
		if err != nil {
			logger.Warnf("[SlotChecker] getBlocks [%d, %d] failed after retries: %v", r.From, r.To, err)
			failedRanges = append(failedRanges, r)
			continue
		}

		fillEmptySlots(r.From, r.To, blocks, confirmedEmptySlots)
	}

	for _, r := range ranges {
		for slot := r.From; slot <= r.To; slot++ {
			// merge后的range不会有交集, 并且是排好序的,所以可以直接用二分查找
			if len(failedRanges) > 0 && slotInFailedRanges(slot, failedRanges) {
				continue
			}

			if _, ok := confirmedEmptySlots[slot]; ok {
				logger.Infof("[SlotChecker] slot %d is confirmed empty", slot)
			} else {
				logger.Errorf("[SlotChecker] slot %d is missing，疑似漏扫", slot)
			}
		}
	}
}

func slotInFailedRanges(slot uint64, failedRanges []SlotRange) bool {
	// 二分查找：找第一个 From > slot 的范围
	i := sort.Search(len(failedRanges), func(i int) bool {
		return failedRanges[i].From > slot
	})
	if i == 0 {
		return false
	}
	// 看 slot 是否在前一个范围内
	r := failedRanges[i-1]
	return slot >= r.From && slot <= r.To
}

func (s *SlotChecker) getBlocksWithRetry(from, to uint64, maxRetries int) ([]uint64, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[SlotChecker] panic during getBlocks: %v", r)
		}
	}()

	var (
		delay   = 300 * time.Millisecond
		attempt int
	)

	for {
		select {
		case <-s.ctx.Done():
			return nil, context.Canceled
		default:
		}

		ctx, cancel := context.WithTimeout(s.ctx, 6*time.Second)
		blocks, err := s.client.GetBlocks(ctx, from, to)
		cancel()

		if err == nil {
			return blocks.Result, nil
		}

		attempt++
		if attempt >= maxRetries {
			return nil, err
		}

		time.Sleep(delay)
	}
}

// mergeRanges 拆分并合并 SlotRange，使每段长度不超过 maxRangeSize，且尽可能合并相邻段。
// 处理流程：
// 1. 拆分：对于每个输入 SlotRange，按 maxRangeSize 拆成多个小段（每段长度 ≤ maxRangeSize）。
// 2. 排序：将所有段按起始 From 升序排列，保证合并时顺序正确。
// 3. 合并：连续的小段会尝试合并为大段，但合并后的段长度仍不得超过 maxRangeSize。
// 注意：这是为控制 RPC 查询规模（如 getBlocks）设计的强约束逻辑。
func mergeRanges(ranges []SlotRange) []SlotRange {
	if len(ranges) == 0 {
		return nil
	}

	const maxRangeSize = 10000

	// Step 1: 拆分所有 SlotRange，使每个段的长度不超过 maxRangeSize
	newRanges := make([]SlotRange, 0, len(ranges))
	for _, r := range ranges {
		from := r.From
		to := r.To
		for {
			maxTo := from + maxRangeSize - 1
			if to > maxTo {
				newRanges = append(newRanges, SlotRange{
					From:     from,
					To:       maxTo,
					SubmitAt: r.SubmitAt,
				})
				from = maxTo + 1
				continue
			}

			newRanges = append(newRanges, SlotRange{
				From:     from,
				To:       to,
				SubmitAt: r.SubmitAt,
			})
			break
		}
	}

	// Step 2: 按 From 升序排序；From 相同时按 To 升序，方便后续合并
	sort.Slice(newRanges, func(i, j int) bool {
		if newRanges[i].From == newRanges[j].From {
			return newRanges[i].To < newRanges[j].To
		}
		return newRanges[i].From < newRanges[j].From
	})

	// Step 3: 合并相邻段，合并后的段仍需满足 maxRangeSize 限制
	merged := make([]SlotRange, 1, len(newRanges))
	merged[0] = newRanges[0]

	for _, r := range newRanges[1:] {
		last := &merged[len(merged)-1]

		maxTo := last.From + maxRangeSize - 1
		if r.To <= maxTo {
			// 可以合并进当前段，不超过 maxRangeSize
			last.To = r.To
		} else {
			// 当前段合并至 maxRangeSize 后拆分出新段
			last.To = maxTo
			merged = append(merged, SlotRange{
				From:     maxTo + 1,
				To:       r.To,
				SubmitAt: r.SubmitAt,
			})
		}
	}
	return merged
}

func fillEmptySlots(from, to uint64, confirmed []uint64, empty map[uint64]struct{}) {
	expectedCount := int(to - from + 1)
	actualCount := len(confirmed)
	missing := expectedCount - actualCount
	if missing <= 0 {
		return
	}

	if actualCount == 0 {
		for slot := from; slot <= to; slot++ {
			empty[slot] = struct{}{}
		}
		return
	}

	// 排序检查 - 如果已经有序就不排序
	if !sort.SliceIsSorted(confirmed, func(i, j int) bool {
		return confirmed[i] < confirmed[j]
	}) {
		sort.Slice(confirmed, func(i, j int) bool {
			return confirmed[i] < confirmed[j]
		})
	}

	found := 0

	// 开头缺失部分
	if confirmed[0] > from {
		for slot := from; slot < confirmed[0]; slot++ {
			empty[slot] = struct{}{}
			found++
			if found == missing {
				return // 已经找到所有缺失的，直接返回
			}
		}
	}

	// 结尾缺失部分
	if confirmed[actualCount-1] < to {
		for slot := confirmed[actualCount-1] + 1; slot <= to; slot++ {
			empty[slot] = struct{}{}
			found++
			if found == missing {
				return // 已经找到所有缺失的，直接返回
			}
		}
	}

	// 中间缺失部分
	for i := 1; i < actualCount && found < missing; i++ {
		start := confirmed[i-1]
		end := confirmed[i]
		for slot := start + 1; slot < end; slot++ {
			empty[slot] = struct{}{}
			found++
		}
	}
}

package cache

import (
	"sort"
	"sync"

	"dex-indexer-sol/internal/pkg/types"
)

type TokenPricePoint struct {
	Timestamp int64
	PriceUsd  float64
}

type PriceCache struct {
	mu      sync.RWMutex
	history map[types.Pubkey][]TokenPricePoint // 保持历史价格点按时间升序排列，便于后续快速查找/插值
}

func NewPriceCache() *PriceCache {
	return &PriceCache{
		history: make(map[types.Pubkey][]TokenPricePoint),
	}
}

func (pc *PriceCache) UpdateFrom(newPoints map[string][]TokenPricePoint) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for str, points := range newPoints {
		if len(points) == 0 {
			continue
		}
		pubKey, err := types.TryPubkeyFromBase58(str)
		if err != nil {
			continue
		}
		// 按 Timestamp 升序排列（即 blockTime 越小越靠前）
		sort.Slice(points, func(i, j int) bool {
			return points[i].Timestamp < points[j].Timestamp
		})
		pc.history[pubKey] = points
	}
}

func (pc *PriceCache) Insert(newPoints map[types.Pubkey]TokenPricePoint) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	const maxCapacity = 400
	const retainCount = 300

	for token, point := range newPoints {
		pricePoints, ok := pc.history[token]
		if !ok {
			pricePoints = make([]TokenPricePoint, 0, maxCapacity)
			pricePoints = append(pricePoints, point)
			pc.history[token] = pricePoints
			continue
		}

		if len(pricePoints) >= maxCapacity {
			// 将后半段复制到前半段，截断为 retainCount 长度
			copy(pricePoints[:retainCount], pricePoints[len(pricePoints)-retainCount:])
			pricePoints = pricePoints[:retainCount]
			pc.history[token] = pricePoints
		}

		// 顺序插入优化
		lastPricePoint := pricePoints[len(pricePoints)-1]
		if point.Timestamp == lastPricePoint.Timestamp {
			continue
		}
		if point.Timestamp > lastPricePoint.Timestamp {
			pricePoints = append(pricePoints, point)
			pc.history[token] = pricePoints
			continue
		}

		// 插入到中间
		insertIdx := sort.Search(len(pricePoints), func(i int) bool {
			return pricePoints[i].Timestamp >= point.Timestamp
		})
		if insertIdx < len(pricePoints) && pricePoints[insertIdx].Timestamp == point.Timestamp {
			continue // 跳过当前 token，继续处理其它 token
		}

		// 插入到 insertIdx
		pricePoints = append(pricePoints, TokenPricePoint{})
		copy(pricePoints[insertIdx+1:], pricePoints[insertIdx:])
		pricePoints[insertIdx] = point
		pc.history[token] = pricePoints
	}
}

func (pc *PriceCache) GetQuotePricesAt(tokens []types.Pubkey, blockTime int64) ([]float64, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	prices := make([]float64, len(tokens))
	for i, token := range tokens {
		if price, found := pc.getPriceAtUnsafe(token, blockTime); found {
			prices[i] = price
		} else {
			return nil, false
		}
	}
	return prices, true
}

func (pc *PriceCache) getPriceAtUnsafe(token types.Pubkey, blockTime int64) (float64, bool) {
	points, ok := pc.history[token]
	if !ok || len(points) == 0 {
		return 0, false
	}

	count := len(points)

	// 边界快速判断：比最老还早 or 比最新还晚
	if blockTime >= points[count-1].Timestamp {
		return points[count-1].PriceUsd, true
	}
	if blockTime < points[0].Timestamp {
		return points[0].PriceUsd, true
	}

	// 二分查找：找到第一个 >= blockTime 的点
	idx := sort.Search(len(points), func(i int) bool {
		return points[i].Timestamp >= blockTime
	})
	if idx < count && points[idx].Timestamp == blockTime {
		return points[idx].PriceUsd, true // 精准命中
	}

	// 否则取前一个点（即 < blockTime 的最大点）
	if idx > 0 {
		idx--
	}
	return points[idx].PriceUsd, true
}

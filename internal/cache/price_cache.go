package cache

import (
	"sync"

	"dex-indexer-sol/internal/types"
)

type TokenPricePoint struct {
	Timestamp int64
	PriceUsd  float64
}

type PriceCache struct {
	mu      sync.RWMutex
	history map[types.Pubkey][]TokenPricePoint
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
		pc.history[pubKey] = points
	}
}

func (pc *PriceCache) Size() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.history)
}

func (pc *PriceCache) GetQuotePricesAt(tokens []types.Pubkey, blockTime int64) (map[types.Pubkey]float64, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	out := make(map[types.Pubkey]float64, len(tokens))
	for _, token := range tokens {
		if price, ok := pc.getPriceAtUnsafe(token, blockTime); ok {
			out[token] = price
		} else {
			return nil, false
		}
	}
	return out, true
}

func (pc *PriceCache) GetPriceAt(token types.Pubkey, blockTime int64) (float64, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return pc.getPriceAtUnsafe(token, blockTime)
}

func (pc *PriceCache) getPriceAtUnsafe(token types.Pubkey, blockTime int64) (float64, bool) {
	points, ok := pc.history[token]
	if !ok || len(points) == 0 {
		return 0, false
	}

	// 特殊情况1：blockTime >= 最新的点
	if blockTime >= points[0].Timestamp {
		return points[0].PriceUsd, true
	}

	// 特殊情况2：blockTime 比最老的点还早
	if blockTime < points[len(points)-1].Timestamp {
		return points[len(points)-1].PriceUsd, true
	}

	// 找到第一个不超过blockTime的点
	l, r := 0, len(points)-1
	for l < r {
		mid := (l + r) / 2
		if points[mid].Timestamp == blockTime {
			return points[mid].PriceUsd, true
		} else if points[mid].Timestamp > blockTime {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}

	return points[l].PriceUsd, true
}

package utils

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/types"
)

const (
	WSOLDecimals = 9
	USDCDecimals = 6
	USDTDecimals = 6
)

// QuotePriority 定义系统内置 quote token 的优先级（数值越小优先级越高）。
var QuotePriority = map[types.Pubkey]int{
	consts.WSOLMint: 1, // 优先级最高，最推荐作为 quote（右对）
	consts.USDCMint: 2,
	consts.USDTMint: 3,
}

var QuoteTokenDecimals = map[types.Pubkey]uint8{
	consts.WSOLMint: WSOLDecimals,
	consts.USDCMint: USDCDecimals,
	consts.USDTMint: USDTDecimals,
}

var QuoteTokenAddresses = []string{
	consts.WSOLMintStr,
	consts.USDCMintStr,
	consts.USDTMintStr,
}

var QuoteTokens = []types.Pubkey{
	consts.WSOLMint,
	consts.USDCMint,
	consts.USDTMint,
}

// ChooseBaseQuote 根据 quotePriority 判断 base 和 quote 的关系。
// 返回 (base, quote, true) 表示成功判断，false 表示双方都不是 quote。
func ChooseBaseQuote(a, b types.Pubkey) (base, quote types.Pubkey, ok bool) {
	pa, oka := QuotePriority[a]
	pb, okb := QuotePriority[b]

	switch {
	case oka && okb:
		if pa < pb {
			return b, a, true // a 优先级更高 → 更适合当 quote
		}
		if pb < pa {
			return a, b, true
		}
	case oka:
		return b, a, true
	case okb:
		return a, b, true
	}

	return types.Pubkey{}, types.Pubkey{}, false
}

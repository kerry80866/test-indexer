package tools

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/pkg/types"
)

const (
	WSOLDecimals = 9
	USDCDecimals = 6
	USDTDecimals = 6
)

// USDQuoteMints 表示具有稳定美元价格参考的常用报价币（右对），用于估值、价格折算等场景。
var USDQuoteMints = []types.Pubkey{
	consts.WSOLMint,
	consts.USDCMint,
	consts.USDTMint,
}

// USDQuoteMintStrs 是 USDQuoteMints 对应的字符串形式（常用于配置、日志、比较等）
var USDQuoteMintStrs = []string{
	consts.WSOLMintStr,
	consts.USDCMintStr,
	consts.USDTMintStr,
}

var QuoteDecimals = map[types.Pubkey]uint8{
	consts.WSOLMint: WSOLDecimals,
	consts.USDCMint: USDCDecimals,
	consts.USDTMint: USDTDecimals,
}

// QuotePriority 定义系统内置 quote token 的优先级（数值越小优先级越高）。
var QuotePriority = map[types.Pubkey]int{
	consts.WSOLMint: 1, // 优先级最高，最推荐作为 quote（右对）
	consts.USDCMint: 2,
	consts.USDTMint: 3,

	consts.JitoSOLMint: 101,
	consts.MSOLMint:    102,
	consts.JupSOLMint:  102,
	consts.BSOLMint:    103,
}

// ChooseBaseQuote 根据 quotePriority 判断 base 和 quote 的关系。
// 返回 (base, quote, true) 表示成功判断，false 表示双方都不是 quote。
func ChooseQuote(a, b types.Pubkey) (quote types.Pubkey, ok bool) {
	pa, oka := QuotePriority[a]
	pb, okb := QuotePriority[b]

	switch {
	case oka && okb:
		if pa < pb {
			return a, true // a 优先级更高 → 更适合当 quote
		}
		if pb < pa {
			return b, true
		}
	case oka:
		return a, true
	case okb:
		return b, true
	}

	return types.Pubkey{}, false
}

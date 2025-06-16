package txadapter

import (
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/internal/tools"
	"github.com/dex-indexer-sol/pkg/types"
)

// mintKV 表示缓存中的一个条目：mint base58 → Pubkey + decimals。
type mintKV struct {
	base58   string       // 原始 mint 字符串（base58 编码）
	pubkey   types.Pubkey // 解码后的 32 字节公钥
	decimals uint8        // Token 精度
}

// mintResolver 用于将 base58 编码的 mint 字符串解析为 Pubkey，并缓存对应的 decimals。
type mintResolver struct {
	cache []mintKV
}

// 参数 capacity 为预估的额外动态 token 数量，用于预分配缓存容量。
func newMintResolver(capacity int) *mintResolver {
	return &mintResolver{cache: make([]mintKV, 0, capacity)}
}

// resolve 返回指定 mintStr 对应的 Pubkey。
// 若缓存命中则直接返回，否则进行 base58 解码并缓存后返回。
func (r *mintResolver) resolve(mintStr string, decimals uint8) types.Pubkey {
	switch mintStr {
	case consts.WSOLMintStr:
		return consts.WSOLMint
	case consts.USDCMintStr:
		return consts.USDCMint
	case consts.USDTMintStr:
		return consts.USDTMint
	}
	for _, item := range r.cache {
		if item.base58 == mintStr {
			return item.pubkey
		}
	}
	pk := types.PubkeyFromBase58(mintStr)
	r.cache = append(r.cache, mintKV{base58: mintStr, pubkey: pk, decimals: decimals})
	return pk
}

// buildTokenDecimals 返回当前交易中涉及的所有 mint → decimals 映射。
func (r *mintResolver) buildTokenDecimals() []core.TokenDecimals {
	list := make([]core.TokenDecimals, 0, len(r.cache)+3)

	// 先添加 WSOL（通常最常用）
	list = append(list, core.TokenDecimals{
		Token: consts.WSOLMint, Decimals: tools.WSOLDecimals,
	})

	// 添加动态 token
	for _, kv := range r.cache {
		list = append(list, core.TokenDecimals{
			Token: kv.pubkey, Decimals: kv.decimals,
		})
	}

	// 最后添加 USDC 和 USDT
	list = append(list,
		core.TokenDecimals{Token: consts.USDCMint, Decimals: tools.USDCDecimals},
		core.TokenDecimals{Token: consts.USDTMint, Decimals: tools.USDTDecimals},
	)
	return list
}

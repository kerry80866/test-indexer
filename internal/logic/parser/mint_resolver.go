package parser

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/domain"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
)

// mintKV 表示缓存中的一个条目：mint base58 → Pubkey + decimals。
type mintKV struct {
	base58   string       // 原始 mint 字符串（base58 编码）
	pubkey   types.Pubkey // 解码后的 32 字节公钥
	decimals uint8        // Token 精度
}

// MintResolver 用于将 base58 编码的 mint 字符串解析为 Pubkey，并缓存对应的 decimals。
// 每个 resolver 实例生命周期限定于单笔交易内，非并发安全。
type MintResolver struct {
	hasCachedCustomMint bool     // 是否已写入自定义（非系统）mint：标记 cached[0] 是否被占用
	cached              []mintKV // 缓存结构：[0]首个动态 mint，[1~3]系统 mint，[4+]更多动态 mint
}

// NewMintResolver 创建 resolver 并初始化内置系统 mint（WSOL、USDC、USDT）。
// 参数 capacity 为预估的额外动态 token 数量，用于预分配缓存容量。
func NewMintResolver(capacity int) *MintResolver {
	cached := make([]mintKV, 0, 4+capacity)
	cached = append(cached,
		mintKV{}, // cached[0]：预留首个动态 mint（避免初期 append）
		mintKV{base58: consts.WSOLMintStr, pubkey: consts.WSOLMint, decimals: utils.WSOLDecimals},
		mintKV{base58: consts.USDCMintStr, pubkey: consts.USDCMint, decimals: utils.USDCDecimals},
		mintKV{base58: consts.USDTMintStr, pubkey: consts.USDTMint, decimals: utils.USDTDecimals},
	)
	return &MintResolver{cached: cached}
}

// Resolve 返回指定 mintStr 对应的 Pubkey。
// 若缓存命中则直接返回，否则进行 base58 解码并缓存后返回。
func (r *MintResolver) Resolve(mintStr string, decimals uint8) types.Pubkey {
	if r.hasCachedCustomMint {
		// 已写入过动态 mint：查找整个缓存列表
		for _, kv := range r.cached {
			if kv.base58 == mintStr {
				return kv.pubkey
			}
		}
		// 未命中：append 至末尾
		pk := types.PubkeyFromBase58(mintStr)
		r.cached = append(r.cached, mintKV{base58: mintStr, pubkey: pk, decimals: decimals})
		return pk
	} else {
		// 尚未使用 cached-0，仅匹配系统 token（cached[1~3]）
		for i := 1; i < len(r.cached); i++ {
			if r.cached[i].base58 == mintStr {
				return r.cached[i].pubkey
			}
		}
		// 非系统 mint：写入 cached-0 作为首个动态 token
		pk := types.PubkeyFromBase58(mintStr)
		r.cached[0] = mintKV{base58: mintStr, pubkey: pk, decimals: decimals}
		r.hasCachedCustomMint = true
		return pk
	}
}

// BuildTokenDecimals 返回当前交易中涉及的所有 mint → decimals 映射。
func (r *MintResolver) BuildTokenDecimals() []domain.TokenDecimals {
	if r.hasCachedCustomMint {
		// 包含动态 mint：返回全部 cached（含 cached[0]）
		list := make([]domain.TokenDecimals, 0, len(r.cached))
		for _, kv := range r.cached {
			list = append(list, domain.TokenDecimals{
				Token:    kv.pubkey,
				Decimals: kv.decimals,
			})
		}
		return list
	} else {
		// 仅使用系统 mint：跳过 cached[0]
		list := make([]domain.TokenDecimals, 0, len(r.cached)-1)
		for i := 1; i < len(r.cached); i++ {
			list = append(list, domain.TokenDecimals{
				Token:    r.cached[i].pubkey,
				Decimals: r.cached[i].decimals,
			})
		}
		return list
	}
}

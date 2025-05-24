package txadapter

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/types"
)

type ownerKV struct {
	base58 string       // 原始 owner 字符串
	pubkey types.Pubkey // 解码后的公钥
}

// ownerResolver 解析 base58 owner 地址 → types.Pubkey，仅缓存解码结果，无 decimals。
type ownerResolver struct {
	cached []ownerKV
}

// newOwnerResolver 创建 resolver，容量为预估 owner 数量。
func newOwnerResolver(capacity int) *ownerResolver {
	return &ownerResolver{cached: make([]ownerKV, 0, capacity)}
}

// resolveOwner 解码 base58 owner 字符串，命中则返回缓存值，否则解码后加入缓存。
func (r *ownerResolver) resolve(base58Str string) types.Pubkey {
	if base58Str == consts.RaydiumV4AuthorityStr {
		return consts.RaydiumV4Authority
	}
	for _, kv := range r.cached {
		if kv.base58 == base58Str {
			return kv.pubkey
		}
	}
	pk := types.PubkeyFromBase58(base58Str)
	r.cached = append(r.cached, ownerKV{base58: base58Str, pubkey: pk})
	return pk
}

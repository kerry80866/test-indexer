package txadapter

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/pkg/types"
)

type ownerResolver struct {
	owners map[string]types.Pubkey // 外部传入 + 协程私有，无需加锁
}

func newOwnerResolver(owners map[string]types.Pubkey) *ownerResolver {
	return &ownerResolver{owners: owners}
}

// resolve 解码 base58 owner 字符串，命中则返回缓存值，否则解码后加入缓存。
func (r *ownerResolver) resolve(base58Str string) types.Pubkey {
	// 1. 优先处理热点地址
	switch base58Str {
	case consts.RaydiumV4AuthorityStr:
		return consts.RaydiumV4Authority
	case consts.RaydiumCPMMAuthorityStr:
		return consts.RaydiumCPMMAuthority
	case consts.PumpFunAMMFee1Str:
		return consts.PumpFunAMMFee1
	case consts.PumpFunAMMFee4Str:
		return consts.PumpFunAMMFee4
	case consts.PumpFunAMMFee5Str:
		return consts.PumpFunAMMFee5
	case consts.PumpFunAMMFee6Str:
		return consts.PumpFunAMMFee6
	case consts.PumpFunAMMFee8Str:
		return consts.PumpFunAMMFee8
	}

	// 2. 再走缓存
	if pk, ok := r.owners[base58Str]; ok {
		return pk
	}

	// 3. 未命中缓存：解码 + 回写缓存
	pk := types.PubkeyFromBase58(base58Str)
	r.owners[base58Str] = pk
	return pk
}

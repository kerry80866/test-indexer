package domain

import "dex-indexer-sol/internal/types"

// TokenBalance 表示某个 SPL Token 账户在交易执行前后的余额信息。
type TokenBalance struct {
	Decimals     uint8  // Token 精度（如 USDC 是 6，SOL 是 9）
	PreBalance   uint64 // 交易执行前余额（最小单位，例如 lamports）
	PostBalance  uint64 // 交易执行后余额
	TokenAccount types.Pubkey
	Token        types.Pubkey
	Owner        string
}

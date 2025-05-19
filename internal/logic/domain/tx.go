package domain

import (
	"dex-indexer-sol/internal/types"
)

// TxContext 表示交易所属区块的上下文信息，提供时间、高度、价格等元信息。
type TxContext struct {
	BlockTime   int64                    // 区块时间戳（Unix 秒级）
	Slot        uint64                   // 当前区块 Slot（Solana 高度单位）
	ParentSlot  uint64                   // 父区块 Slot（用于分叉检测与回滚处理）
	BlockHeight uint64                   // 区块高度（便于比对，非必须字段）
	BlockHash   types.Hash               // 区块哈希（辅助唯一性及 fork 识别）
	QuotesPrice map[types.Pubkey]float64 // 常见 quote token 的 USD 价格（如 USDC、USDT、WSOL）
}

// TranslatedTx 表示已解析的链上交易结构，包含上下文、指令及余额变动信息。
type TranslatedTx struct {
	TxCtx     *TxContext   // 所属区块上下文（使用共享指针以节省内存）
	TxIndex   uint64       // 当前交易在区块中的序号（基于 geyser TransactionIndex）
	Signature []byte       // 交易签名（64 字节原始数据）
	Signer    types.Pubkey // 交易发起者（通常为 accountKeys[0]）

	Instructions  []*TranslatedInstruction       // 主指令及其关联的 inner 指令
	Balances      map[types.Pubkey]*TokenBalance // 涉及 Token 账户的余额变更
	TokenDecimals []TokenDecimals                // 本交易中涉及的 mint → decimals 映射，TokenDecimals 更轻量，用值类型更好
}

// TokenDecimals 表示一个 mint → decimals 的映射项，来源于解析阶段。
type TokenDecimals struct {
	Token    types.Pubkey
	Decimals uint8
}

// GetTokenDecimals 返回指定 mint 对应的精度（若存在于当前交易中）。
func (tx *TranslatedTx) GetTokenDecimals(mint types.Pubkey) (uint8, bool) {
	for _, v := range tx.TokenDecimals {
		if v.Token == mint {
			return v.Decimals, true
		}
	}
	return 0, false
}

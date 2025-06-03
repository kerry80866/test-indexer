package core

import (
	"dex-indexer-sol/internal/types"
)

type QuotePrice struct {
	Token    types.Pubkey // quote token（如 WSOL、USDC、USDT）
	PriceUsd float64      // 美元价格（非最小单位，例如 160.0）
}

// TxContext 表示交易所属区块的上下文信息，包含时间、高度、价格等元数据。
type TxContext struct {
	BlockTime   int64        // 区块时间戳（Unix 秒）
	Slot        uint64       // 当前 Slot（Solana 高度单位）
	ParentSlot  uint64       // 父 Slot（用于分叉检测和回滚）
	BlockHeight uint64       // 区块高度（辅助比对）
	BlockHash   types.Hash   // 区块哈希（辅助去重与 fork 检测）
	QuotesPrice []QuotePrice // quote token 的美元价格（数组更适合小量查找）
}

// AdaptedInstruction 表示一条主指令或 inner 指令，来源于 Solana Transaction 中的 message.instructions 或 innerInstructions。
// 所有指令在预处理阶段已展平，并补充了位置信息（IxIndex、InnerIndex），以支持顺序遍历与事件定位。
type AdaptedInstruction struct {
	IxIndex    uint16         // 主指令索引（从 0 开始）
	InnerIndex uint16         // Inner 指令在主指令中的序号，主指令本身为 0，CPI 调用从 1 开始
	ProgramID  types.Pubkey   // 指令对应的程序 ID
	Accounts   []types.Pubkey // 指令涉及的账户列表，保持原始顺序
	Data       []byte         // 指令原始数据，用于 handler 判断指令类型与解析参数
}

// TokenBalance 表示某个 SPL Token 账户在交易执行前后的余额信息。
type TokenBalance struct {
	Decimals     uint8
	HasPreOwner  bool
	TxIndex      uint16
	InnerIndex   uint16
	PreBalance   uint64 // 交易执行前余额（最小单位，例如 lamports）
	PostBalance  uint64 // 交易执行后余额
	TokenAccount types.Pubkey
	Token        types.Pubkey
	PreOwner     types.Pubkey
	PostOwner    types.Pubkey
}

// TokenDecimals 表示某 mint 的精度信息（通常用于解析金额）。
type TokenDecimals struct {
	Token    types.Pubkey
	Decimals uint8
}

// AdaptedTx 表示已解析的链上交易结构，包含上下文、指令、日志与余额变动信息。
// 是事件解析流程的核心输入结构体。
type AdaptedTx struct {
	TxCtx     *TxContext // 所属区块上下文（包含 Unix 时间、Slot、高度等元数据）
	TxIndex   uint32     // 当前交易在区块中的序号（基于 Geyser TransactionIndex）
	Signature []byte     // 交易签名（64 字节原始数据）
	Signers   [][]byte   // 交易签名者列表

	// Instructions 表示交易中的所有指令（包括主指令和 inner 指令），已按 Solana 执行顺序展平。
	// 每条指令都使用 AdaptedInstruction 表示，并标注其所属主指令位置（IxIndex）与 inner 索引（InnerIndex）。
	// 扁平结构可直接顺序遍历，便于事件解析器基于游标消费多条组合指令（如 swap + transfer）。
	Instructions []*AdaptedInstruction

	// LogMessages 表示交易执行过程中产生的 Program 日志。
	// 用途说明：
	// 1. 并非所有协议依赖 logs，当前仅 Pump.fun、Mango 等协议需要从 logs 中提取成交价格或内部转账等事件信息。
	// 2. 日志会在交易适配（Adapt）阶段统一挂载，事件解析阶段可按需使用。
	LogMessages []string

	// 涉及 Token 账户的余额变更（SPL Token 转账与扣款）
	Balances map[types.Pubkey]*TokenBalance

	// TokenDecimals 表示本交易中涉及的 mint → decimals 精度映射，用于解析 Token 金额。
	// 设计说明：
	// 1. decimals 主要用于将事件中的 raw amount 转换为实际金额（如 USDC 为 6、WSOL 为 9）。
	// 2. 使用切片而非 map 是因为单笔交易涉及的 mint 数量极少（通常 2~3 个），顺序查找在小规模场景下更高效，避免哈希运算与分支预测开销。
	// 3. 使用值类型而非指针，是因为结构体体积小（Pubkey + uint8），直接存值可减少间接寻址，提高 CPU cache 命中率，加快遍历与查找性能。
	TokenDecimals []TokenDecimals
}

func (tx *AdaptedTx) GetDecimalsByMint(mint types.Pubkey) (uint8, bool) {
	for _, v := range tx.TokenDecimals {
		if v.Token == mint {
			return v.Decimals, true
		}
	}
	return 0, false
}

// AddTokenDecimals 添加一个 mint 和 decimals，重复则跳过
func (tx *AdaptedTx) AddTokenDecimals(mint types.Pubkey, decimals uint8) {
	for _, v := range tx.TokenDecimals {
		if v.Token == mint {
			return
		}
	}
	tx.TokenDecimals = append(tx.TokenDecimals, TokenDecimals{
		Token:    mint,
		Decimals: decimals,
	})
}

func (ctx *TxContext) GetQuoteUsd(token types.Pubkey) (float64, bool) {
	for _, q := range ctx.QuotesPrice {
		if q.Token == token {
			return q.PriceUsd, true
		}
	}
	return 0, false
}

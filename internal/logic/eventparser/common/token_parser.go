package common

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"encoding/binary"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
)

// ParsedTransfer 表示一次 SPL Token 转账操作（Transfer 或 TransferChecked）的解析结果。
type ParsedTransfer struct {
	Token           types.Pubkey // Token 的 mint 地址
	SrcAccount      types.Pubkey // 来源 SPL Token 账户地址（转出）
	DestAccount     types.Pubkey // 目标 SPL Token 账户地址（转入）
	SrcWallet       types.Pubkey // 来源钱包地址（TokenAccount 的 owner）
	DestWallet      types.Pubkey // 目标钱包地址（TokenAccount 的 owner）
	Amount          uint64       // 转账金额（最小单位）
	Decimals        uint8        // Token 精度（来自 source）
	SrcPostBalance  uint64       // 转账后，来源账户余额（PostBalance）
	DestPostBalance uint64       // 转账后，目标账户余额（PostBalance）
}

// ParseTransferInstruction 尝试从一条指令中解析出 SPL Token 转账信息（Transfer / TransferChecked）。
// 若指令类型、格式或账户结构不符合预期，返回 (nil, false)。
func ParseTransferInstruction(ctx *ParserContext, ix *core.AdaptedInstruction) (*ParsedTransfer, bool) {
	if len(ix.Data) == 0 {
		return nil, false
	}

	switch ix.Data[0] {
	case byte(sdktoken.InstructionTransfer):
		// Transfer 指令格式：
		//   [0]      = instruction (1 byte)
		//   [1:9]    = amount (8 bytes)
		//   accounts = [src_account, dest_account, authority_wallet]
		if len(ix.Data) != 9 || len(ix.Accounts) < 3 {
			return nil, false
		}

		srcInfo, ok1 := ctx.Balances[ix.Accounts[0]]
		destInfo, ok2 := ctx.Balances[ix.Accounts[1]]
		if !ok1 || !ok2 {
			return nil, false
		}

		return &ParsedTransfer{
			Token:           srcInfo.Token,
			SrcAccount:      ix.Accounts[0],
			DestAccount:     ix.Accounts[1],
			SrcWallet:       ix.Accounts[2],
			DestWallet:      destInfo.Owner,
			Amount:          binary.LittleEndian.Uint64(ix.Data[1:9]),
			Decimals:        srcInfo.Decimals,
			SrcPostBalance:  srcInfo.PostBalance,
			DestPostBalance: destInfo.PostBalance,
		}, true

	case byte(sdktoken.InstructionTransferChecked):
		// TransferChecked 指令格式：
		//   [0]      = instruction (1 byte)
		//   [1:9]    = amount (8 bytes)
		//   [9]      = decimals (1 byte)
		//   accounts = [src_account, mint, dest_account, authority_wallet]
		if len(ix.Data) != 10 || len(ix.Accounts) < 4 {
			return nil, false
		}

		srcInfo, ok1 := ctx.Balances[ix.Accounts[0]]
		destInfo, ok2 := ctx.Balances[ix.Accounts[2]]
		if !ok1 || !ok2 {
			return nil, false
		}

		return &ParsedTransfer{
			Token:           srcInfo.Token, // 通常等于 ix.Accounts[1]（mint），此处从 balances 获取更安全
			SrcAccount:      ix.Accounts[0],
			DestAccount:     ix.Accounts[2],
			SrcWallet:       ix.Accounts[3],
			DestWallet:      destInfo.Owner,
			Amount:          binary.LittleEndian.Uint64(ix.Data[1:9]),
			Decimals:        srcInfo.Decimals, // ix.Data[9] 可作为校验备用
			SrcPostBalance:  srcInfo.PostBalance,
			DestPostBalance: destInfo.PostBalance,
		}, true
	}

	return nil, false
}

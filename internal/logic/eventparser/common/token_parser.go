package common

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/pkg/types"
	"encoding/binary"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
)

// 合约源代码:
// SplToken: https://github.com/solana-program/token/blob/main/program/src/instruction.rs
// Token2022: https://github.com/solana-program/token-2022

// ParsedTransfer 表示一次 SPL Token 的 Transfer 或 TransferChecked 操作。
type ParsedTransfer struct {
	IxIndex         uint16       // 主指令
	InnerIndex      uint16       // 内部指令
	Token           types.Pubkey // Token mint 地址
	SrcAccount      types.Pubkey // 来源 TokenAccount
	DestAccount     types.Pubkey // 目标 TokenAccount
	SrcWallet       types.Pubkey // 来源账户所有者
	DestWallet      types.Pubkey // 目标账户所有者
	Amount          uint64       // 转账数量（最小单位）
	Decimals        uint8        // Token 精度
	SrcPostBalance  uint64       // 来源账户转账后余额
	DestPostBalance uint64       // 目标账户转账后余额
}

// ParsedMintTo 表示一次 SPL Token 的 MintTo 或 MintToChecked 操作。
type ParsedMintTo struct {
	IxIndex         uint16       // 主指令
	InnerIndex      uint16       // 内部指令
	Token           types.Pubkey // Token mint 地址
	DestAccount     types.Pubkey // 目标 TokenAccount
	DestWallet      types.Pubkey // 目标账户所有者
	Amount          uint64       // 铸币数量
	Decimals        uint8        // Token 精度
	DestPostBalance uint64       // 铸币后账户余额
}

// ParsedBurn 表示一次 SPL Token 的 Burn 或 BurnChecked 操作。
type ParsedBurn struct {
	IxIndex        uint16       // 主指令
	InnerIndex     uint16       // 内部指令
	Token          types.Pubkey // Token mint 地址
	SrcAccount     types.Pubkey // 来源 TokenAccount
	SrcWallet      types.Pubkey // 来源账户所有者
	Amount         uint64       // 销毁数量
	Decimals       uint8        // Token 精度
	SrcPostBalance uint64       // 销毁后账户余额
}

// ParseTransferInstruction 解析 Transfer / TransferChecked 指令
func ParseTransferInstruction(ctx *ParserContext, ix *core.AdaptedInstruction) (*ParsedTransfer, bool) {
	if len(ix.Data) < 9 || len(ix.Accounts) < 3 {
		return nil, false
	}

	switch ix.Data[0] {
	// Transfer: [0]=instr, [1:9]=amount
	// accounts = [src_account, dest_account, authority_wallet]
	case byte(sdktoken.InstructionTransfer):
		srcInfo, ok1 := ctx.Balances[ix.Accounts[0]]
		destInfo, ok2 := ctx.Balances[ix.Accounts[1]]
		if !ok1 || !ok2 {
			logger.Errorf("[Token::Transfer] tx=%s: balance missing src=%s ok=%v dest=%s ok=%v",
				ctx.TxHashString(), ix.Accounts[0], ok1, ix.Accounts[1], ok2)
			return nil, false
		}
		return &ParsedTransfer{
			IxIndex:         ix.IxIndex,
			InnerIndex:      ix.InnerIndex,
			Token:           srcInfo.Token,
			SrcAccount:      ix.Accounts[0],
			DestAccount:     ix.Accounts[1],
			SrcWallet:       ix.Accounts[2],
			DestWallet:      destInfo.PostOwner,
			Amount:          binary.LittleEndian.Uint64(ix.Data[1:9]),
			Decimals:        srcInfo.Decimals,
			SrcPostBalance:  srcInfo.PostBalance,
			DestPostBalance: destInfo.PostBalance,
		}, true

	// TransferChecked: [0]=instr, [1:9]=amount, [9]=decimals
	// accounts = [src_account, mint, dest_account, authority_wallet]
	case byte(sdktoken.InstructionTransferChecked):
		if len(ix.Data) < 10 || len(ix.Accounts) < 4 {
			return nil, false
		}
		srcInfo, ok1 := ctx.Balances[ix.Accounts[0]]
		destInfo, ok2 := ctx.Balances[ix.Accounts[2]]
		if !ok1 || !ok2 {
			logger.Errorf("[Token::TransferChecked] tx=%s: balance missing src=%s ok=%v dest=%s ok=%v",
				ctx.TxHashString(), ix.Accounts[0], ok1, ix.Accounts[2], ok2)
			return nil, false
		}
		if srcInfo.Token != ix.Accounts[1] {
			logger.Errorf("[Token::TransferChecked] tx=%s: mint mismatch, balance.token=%s, ix.mint=%s (account=%s)",
				ctx.TxHashString(), srcInfo.Token, ix.Accounts[1], ix.Accounts[0])
		}
		return &ParsedTransfer{
			IxIndex:         ix.IxIndex,
			InnerIndex:      ix.InnerIndex,
			Token:           srcInfo.Token,
			SrcAccount:      ix.Accounts[0],
			DestAccount:     ix.Accounts[2],
			SrcWallet:       ix.Accounts[3],
			DestWallet:      destInfo.PostOwner,
			Amount:          binary.LittleEndian.Uint64(ix.Data[1:9]),
			Decimals:        srcInfo.Decimals,
			SrcPostBalance:  srcInfo.PostBalance,
			DestPostBalance: destInfo.PostBalance,
		}, true
	}
	return nil, false
}

// ParseMintToInstruction 解析 MintTo / MintToChecked 指令
func ParseMintToInstruction(ctx *ParserContext, ix *core.AdaptedInstruction) (*ParsedMintTo, bool) {
	// MintTo: [0]=instr, [1:9]=amount, [9]=decimals (可选)
	// accounts = [mint, dest_token_account, authority_wallet]
	if len(ix.Data) < 9 || len(ix.Accounts) < 3 {
		return nil, false
	}
	if ix.Data[0] != byte(sdktoken.InstructionMintTo) &&
		ix.Data[0] != byte(sdktoken.InstructionMintToChecked) {
		return nil, false
	}
	info, ok := ctx.Balances[ix.Accounts[1]]
	if !ok {
		logger.Errorf("[Token::MintTo] tx=%s: dest_token_account missing: %s",
			ctx.TxHashString(), ix.Accounts[1])
		return nil, false
	}
	if info.Token != ix.Accounts[0] {
		logger.Errorf("[Token::MintTo] tx=%s: mint mismatch, balance.token=%s, ix.mint=%s (account=%s)",
			ctx.TxHashString(), info.Token, ix.Accounts[0], ix.Accounts[1])
	}
	return &ParsedMintTo{
		IxIndex:         ix.IxIndex,
		InnerIndex:      ix.InnerIndex,
		Token:           ix.Accounts[0], // Accounts[0]更贴近指令定义，因此mintTo更倾向于Accounts[0],而不是info.Token
		DestAccount:     ix.Accounts[1],
		DestWallet:      ix.Accounts[2],
		Amount:          binary.LittleEndian.Uint64(ix.Data[1:9]),
		Decimals:        info.Decimals,
		DestPostBalance: info.PostBalance,
	}, true
}

// ParseBurnInstruction 解析 Burn / BurnChecked 指令
func ParseBurnInstruction(ctx *ParserContext, ix *core.AdaptedInstruction) (*ParsedBurn, bool) {
	// Burn: [0]=instr, [1:9]=amount, [9]=decimals (仅 BurnChecked 有)
	// accounts = [src_token_account, mint, authority_wallet]
	if len(ix.Data) < 9 || len(ix.Accounts) < 3 {
		return nil, false
	}
	if ix.Data[0] != byte(sdktoken.InstructionBurn) &&
		ix.Data[0] != byte(sdktoken.InstructionBurnChecked) {
		return nil, false
	}
	info, ok := ctx.Balances[ix.Accounts[0]]
	if !ok {
		logger.Errorf("[Token::Burn] tx=%s: src_token_account missing: %s",
			ctx.TxHashString(), ix.Accounts[0])
		return nil, false
	}
	if info.Token != ix.Accounts[1] {
		logger.Errorf("[Token::Burn] tx=%s: mint mismatch, balance.token=%s, ix.mint=%s (account=%s)",
			ctx.TxHashString(), info.Token, ix.Accounts[1], ix.Accounts[0])
	}
	return &ParsedBurn{
		IxIndex:        ix.IxIndex,
		InnerIndex:     ix.InnerIndex,
		Token:          info.Token,
		SrcAccount:     ix.Accounts[0],
		SrcWallet:      ix.Accounts[2],
		Amount:         binary.LittleEndian.Uint64(ix.Data[1:9]),
		Decimals:       info.Decimals,
		SrcPostBalance: info.PostBalance,
	}, true
}

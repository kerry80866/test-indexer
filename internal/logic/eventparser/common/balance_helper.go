package common

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
	"github.com/mr-tron/base58"
	"github.com/zeromicro/go-zero/core/logx"
)

// PreScanInitAccountBalances 扫描指令列表中 InitializeAccount 系指令，并尝试补充 ctx.Balances 信息。
func PreScanInitAccountBalances(ctx *ParserContext, instrs []*core.AdaptedInstruction) {
	for _, ix := range instrs {
		if ix.ProgramID != consts.TokenProgram && ix.ProgramID != consts.TokenProgram2022 {
			continue
		}
		if len(ix.Data) == 0 {
			continue
		}

		switch ix.Data[0] {
		case byte(sdktoken.InstructionInitializeAccount),
			byte(sdktoken.InstructionInitializeAccount2),
			byte(sdktoken.InstructionInitializeAccount3):
			tryFillBalanceFromInitAccount(ctx, ix)
		}
	}
}

// tryFillBalanceFromInitAccount 尝试从初始化账户指令中提取 TokenAccount → Token (mint) → Owner 映射。
// 仅当 ctx.Balances 中尚未包含该 TokenAccount 时生效。
func tryFillBalanceFromInitAccount(ctx *ParserContext, ix *core.AdaptedInstruction) {
	if len(ix.Data) == 0 {
		return
	}

	var (
		mint, tokenAccount, owner types.Pubkey
		err                       error
	)

	switch ix.Data[0] {
	case byte(sdktoken.InstructionInitializeAccount):
		// Layout: accounts = [tokenAccount, mint, owner]
		if len(ix.Accounts) < 3 {
			return
		}
		tokenAccount = ix.Accounts[0]
		mint = ix.Accounts[1]
		owner = ix.Accounts[2]

	case byte(sdktoken.InstructionInitializeAccount2), byte(sdktoken.InstructionInitializeAccount3):
		// Layout: accounts = [tokenAccount, mint], owner in Data[1:33]
		if len(ix.Accounts) < 2 || len(ix.Data) < 33 {
			return
		}
		tokenAccount = ix.Accounts[0]
		mint = ix.Accounts[1]
		owner, err = types.PubkeyFromBytes(ix.Data[1:33])
		if err != nil {
			logx.Errorf("[tryFillBalance] tx=%s: failed to decode owner pubkey from Data[1:33], err=%v, ix: program=%s ixIndex=%d innerIndex=%d",
				base58.Encode(ctx.TxHash), err, ix.ProgramID, ix.IxIndex, ix.InnerIndex)
			return
		}

	default:
		return
	}

	// 如果该 TokenAccount 尚未在 ctx.Balances 中，则补全余额信息
	if _, found := ctx.Balances[tokenAccount]; !found {
		if decimals, ok := ctx.Tx.GetDecimalsByMint(mint); ok {
			ctx.Balances[tokenAccount] = &core.TokenBalance{
				TokenAccount: tokenAccount,
				Token:        mint,
				Owner:        owner,
				Decimals:     decimals,
				TxIndex:      ctx.TxIndex,
			}
		} else {
			logx.Errorf("[tryFillBalance] tx=%s: missing decimals for mint=%s (tokenAccount=%s), ix: program=%s ixIndex=%d innerIndex=%d",
				base58.Encode(ctx.TxHash), mint, tokenAccount, ix.ProgramID, ix.IxIndex, ix.InnerIndex)
		}
	}
}

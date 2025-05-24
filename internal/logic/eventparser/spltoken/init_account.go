package spltoken

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/types"
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
	"github.com/zeromicro/go-zero/core/logx"
)

// tryFillBalanceFromInitAccount 尝试从初始化账户指令中提取 TokenAccount → Token/Mint → Owner 映射。
// 仅当 ctx.balances 中尚未包含该 TokenAccount 时生效。
func tryFillBalanceFromInitAccount(ctx *common.ParserContext, ix *core.AdaptedInstruction) {
	if len(ix.Data) == 0 {
		return
	}

	var (
		mint, tokenAccount, owner types.Pubkey
		err                       error
	)

	switch ix.Data[0] {
	case byte(sdktoken.InstructionInitializeAccount):
		// Layout: [tokenAccount, mint, owner]
		if len(ix.Accounts) < 3 {
			return
		}
		tokenAccount = ix.Accounts[0]
		mint = ix.Accounts[1]
		owner = ix.Accounts[2]

	case byte(sdktoken.InstructionInitializeAccount2), byte(sdktoken.InstructionInitializeAccount3):
		// Layout: [tokenAccount, mint], owner in Data[1:33]
		if len(ix.Accounts) < 2 || len(ix.Data) < 33 {
			return
		}
		tokenAccount = ix.Accounts[0]
		mint = ix.Accounts[1]

		owner, err = types.PubkeyFromBytes(ix.Data[1:33])
		if err != nil {
			return
		}

	default:
		return
	}

	// 若尚未缓存该 TokenAccount，则补全 balances 信息
	if _, found := ctx.Balances[tokenAccount]; !found {
		if decimals, ok := ctx.Tx.GetDecimalsByMint(mint); ok {
			ctx.Balances[tokenAccount] = &core.TokenBalance{
				TokenAccount: tokenAccount,
				Token:        mint,
				Owner:        owner,
				Decimals:     decimals,
			}
		} else {
			logx.Errorf(
				"tryFillBalance: missing decimals for mint %s (tokenAccount: %s), from ix: [program: %s, ixIndex: %d, innerIndex: %d]",
				mint, tokenAccount, ix.ProgramID, ix.IxIndex, ix.InnerIndex,
			)
		}
	}
}

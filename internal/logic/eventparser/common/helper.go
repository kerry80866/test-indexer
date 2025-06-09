package common

import (
	"bytes"
	"dex-indexer-sol/internal/types"
)

func isTransferConflict(pt, other *ParsedTransfer) bool {
	if other == nil {
		return false
	}
	return pt.DestAccount == other.DestAccount || pt.SrcAccount == other.SrcAccount
}

// patchWSOLBalanceIfNeeded 检查是否为临时 WSOL 账户，若是则使用钱包 SOL 补充 quote 余额。
// setBalance 是一个回调，用于设置 quote balance，可适配不同类型的事件结构。
func patchWSOLBalanceIfNeeded(
	ctx *ParserContext,
	walletAccount, wallet types.Pubkey,
	setBalance func(uint64),
) {
	bal, ok := ctx.Balances[walletAccount]
	if !ok {
		return
	}
	if bal.PreBalance != 0 || bal.PostBalance != 0 {
		return
	}
	for _, signer := range ctx.Signers {
		if bytes.Equal(wallet[:], signer) {
			if solBal, ok := ctx.Tx.SolBalances[wallet]; ok {
				setBalance(solBal.PostBalance)
			}
			return
		}
	}
}

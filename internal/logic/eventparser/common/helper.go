package common

import (
	"bytes"
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/pkg/types"
)

func isTransferConflict(pt, other *ParsedTransfer) bool {
	if other == nil {
		return false
	}
	return pt.DestAccount == other.DestAccount || pt.SrcAccount == other.SrcAccount || pt.Token == other.Token
}

// isTempWSOLAccount 判断给定账户是否为“临时 WSOL 账户”
func isTempWSOLAccount(
	balances map[types.Pubkey]*core.TokenBalance,
	token types.Pubkey,
	account types.Pubkey,
) bool {
	if token != consts.WSOLMint {
		return false
	}
	if info, ok := balances[account]; ok {
		return info.PreBalance == 0 && info.PostBalance == 0
	}
	return false
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

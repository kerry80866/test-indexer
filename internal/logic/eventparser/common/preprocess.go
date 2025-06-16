package common

import (
	sdktoken "github.com/blocto/solana-go-sdk/program/token"
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/pkg/logger"
	"github.com/dex-indexer-sol/pkg/types"
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
		case byte(sdktoken.InstructionInitializeMint),
			byte(sdktoken.InstructionInitializeMint2):
			tryFillMintDecimalsDeFromInitMint(ctx, ix)

		case byte(sdktoken.InstructionInitializeAccount),
			byte(sdktoken.InstructionInitializeAccount2),
			byte(sdktoken.InstructionInitializeAccount3):
			tryFillBalanceFromInitAccount(ctx, ix)
		}
	}
}

func tryFillMintDecimalsDeFromInitMint(ctx *ParserContext, ix *core.AdaptedInstruction) {
	// 判断数据长度和账户数是否符合预期
	if len(ix.Data) < 2 || len(ix.Accounts) == 0 {
		return
	}

	// ix.Accounts[0] 是 Mint 账号
	mint := ix.Accounts[0]

	// ix.Data[1] 是 decimals，uint8 类型，表示该 Token 的小数位数
	decimals := ix.Data[1]

	// 将 mint 和 decimals 加入当前交易的 TokenDecimals 缓存
	ctx.Tx.AddTokenDecimals(mint, decimals)
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
			logger.Errorf("[PreScanInitAccount::tryFillBalance] tx=%s: failed to decode owner pubkey from Data[1:33], err=%v, ix: program=%s ixIndex=%d innerIndex=%d",
				ctx.TxHashString(), err, ix.ProgramID, ix.IxIndex, ix.InnerIndex)
			return
		}

	default:
		return
	}

	// 如果该 TokenAccount 尚未在 ctx.Balances 中，则补全余额信息
	if _, exists := ctx.Balances[tokenAccount]; exists {
		return
	}

	decimals, ok := ctx.Tx.GetDecimalsByMint(mint)
	if !ok {
		logger.Errorf("[PreScanInitAccount::tryFillBalance] tx=%s: missing decimals for mint=%s (tokenAccount=%s), ix: program=%s ixIndex=%d innerIndex=%d",
			ctx.TxHashString(), mint, tokenAccount, ix.ProgramID, ix.IxIndex, ix.InnerIndex)
		return
	}

	innerIndex := uint16(len(ctx.Balances))
	ctx.Balances[tokenAccount] = &core.TokenBalance{
		Decimals:     decimals,
		HasPreOwner:  false,
		TxIndex:      uint16(ctx.TxIndex),
		InnerIndex:   innerIndex,
		TokenAccount: tokenAccount,
		PreBalance:   0,
		PostBalance:  0,
		Token:        mint,
		PostOwner:    owner,
	}
}

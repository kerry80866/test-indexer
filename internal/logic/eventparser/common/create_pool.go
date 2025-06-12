package common

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/tools"
	"dex-indexer-sol/pb"
	"dex-indexer-sol/pkg/logger"
	"dex-indexer-sol/pkg/types"
)

type CreatePoolLayout struct {
	PoolAddressIndex   int
	TokenMint1Index    int
	TokenMint2Index    int
	TokenProgram1Index int
	TokenProgram2Index int
	PoolVault1Index    int
	PoolVault2Index    int
	UserWalletIndex    int
}

func ExtractCreatePoolEvent(
	ctx *ParserContext,
	ix *core.AdaptedInstruction,
	dex int,
	instructionName string,
	layout *CreatePoolLayout,
) *core.Event {
	dexName := consts.DexName(dex)

	if !validateCreatePoolLayout(ctx, dexName, instructionName, layout, len(ix.Accounts)) {
		return nil
	}

	tokenMint1 := ix.Accounts[layout.TokenMint1Index]
	tokenMint2 := ix.Accounts[layout.TokenMint2Index]
	poolVault1 := ix.Accounts[layout.PoolVault1Index]
	poolVault2 := ix.Accounts[layout.PoolVault2Index]

	bal1 := validateVaultMint(ctx, dexName, instructionName, poolVault1, tokenMint1, "vault1")
	if bal1 == nil {
		return nil
	}
	bal2 := validateVaultMint(ctx, dexName, instructionName, poolVault2, tokenMint2, "vault2")
	if bal2 == nil {
		return nil
	}

	if layout.TokenProgram1Index >= 0 && !tools.IsSPLTokenProgram(ix.Accounts[layout.TokenProgram1Index]) {
		logger.Errorf("[%s:%s] invalid TokenProgram1: %s, tx=%s",
			dexName, instructionName, ix.Accounts[layout.TokenProgram1Index], ctx.TxHashString())
		return nil
	}
	if layout.TokenProgram2Index >= 0 && !tools.IsSPLTokenProgram(ix.Accounts[layout.TokenProgram2Index]) {
		logger.Errorf("[%s:%s] invalid TokenProgram2: %s, tx=%s",
			dexName, instructionName, ix.Accounts[layout.TokenProgram2Index], ctx.TxHashString())
		return nil
	}

	event := pb.LiquidityEvent{
		Type:             pb.EventType_CREATE_POOL,
		EventId:          core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		Slot:             ctx.Slot,
		BlockTime:        ctx.BlockTime,
		TxHash:           ctx.TxHash[:],
		Signers:          ctx.Signers,
		Dex:              uint32(dex),
		UserWallet:       ix.Accounts[layout.UserWalletIndex][:],
		PairAddress:      ix.Accounts[layout.PoolAddressIndex][:],
		TokenAmount:      0,
		QuoteTokenAmount: 0,
		UserTokenBalance: 0,
		UserQuoteBalance: 0,
	}

	quote, ok := tools.ChooseQuote(tokenMint1, tokenMint2)
	if !ok {
		quote = tokenMint2
	}

	if quote == tokenMint2 {
		event.Token = tokenMint1[:]
		event.QuoteToken = tokenMint2[:]
		event.TokenDecimals = uint32(bal1.Decimals)
		event.QuoteDecimals = uint32(bal2.Decimals)
		event.TokenAccount = poolVault1[:]
		event.QuoteTokenAccount = poolVault2[:]
		event.TokenAccountOwner = bal1.PostOwner[:]
		event.QuoteTokenAccountOwner = bal2.PostOwner[:]
		event.PairTokenBalance = bal1.PostBalance
		event.PairQuoteBalance = bal2.PostBalance
	} else {
		event.Token = tokenMint2[:]
		event.QuoteToken = tokenMint1[:]
		event.TokenDecimals = uint32(bal2.Decimals)
		event.QuoteDecimals = uint32(bal1.Decimals)
		event.TokenAccount = poolVault2[:]
		event.QuoteTokenAccount = poolVault1[:]
		event.TokenAccountOwner = bal2.PostOwner[:]
		event.QuoteTokenAccountOwner = bal1.PostOwner[:]
		event.PairTokenBalance = bal2.PostBalance
		event.PairQuoteBalance = bal1.PostBalance
	}

	return &core.Event{
		ID:        event.EventId,
		EventType: uint32(event.Type),
		Key:       event.PairAddress,
		Event: &pb.Event{
			Event: &pb.Event_Liquidity{
				Liquidity: &event,
			},
		},
	}
}

func validateCreatePoolLayout(
	ctx *ParserContext,
	dexName string,
	instructionName string,
	layout *CreatePoolLayout,
	accountCount int,
) bool {

	if layout.PoolAddressIndex < 0 || layout.PoolAddressIndex >= accountCount {
		logger.Errorf("[%s:%s] invalid PoolAddressIndex=%d (total=%d), tx=%s",
			dexName, instructionName, layout.PoolAddressIndex, accountCount, ctx.TxHashString())
		return false
	}
	if layout.TokenMint1Index < 0 || layout.TokenMint1Index >= accountCount {
		logger.Errorf("[%s:%s] invalid TokenMint1Index=%d (total=%d), tx=%s",
			dexName, instructionName, layout.TokenMint1Index, accountCount, ctx.TxHashString())
		return false
	}
	if layout.TokenMint2Index < 0 || layout.TokenMint2Index >= accountCount {
		logger.Errorf("[%s:%s] invalid TokenMint2Index=%d (total=%d), tx=%s",
			dexName, instructionName, layout.TokenMint2Index, accountCount, ctx.TxHashString())
		return false
	}
	if layout.PoolVault1Index < 0 || layout.PoolVault1Index >= accountCount {
		logger.Errorf("[%s:%s] invalid PoolVault1Index=%d (total=%d), tx=%s",
			dexName, instructionName, layout.PoolVault1Index, accountCount, ctx.TxHashString())
		return false
	}
	if layout.PoolVault2Index < 0 || layout.PoolVault2Index >= accountCount {
		logger.Errorf("[%s:%s] invalid PoolVault2Index=%d (total=%d), tx=%s",
			dexName, instructionName, layout.PoolVault2Index, accountCount, ctx.TxHashString())
		return false
	}
	if layout.UserWalletIndex < 0 || layout.UserWalletIndex >= accountCount {
		logger.Errorf("[%s:%s] invalid UserWalletIndex=%d (total=%d), tx=%s",
			dexName, instructionName, layout.UserWalletIndex, accountCount, ctx.TxHashString())
		return false
	}
	if layout.TokenProgram1Index != -1 && (layout.TokenProgram1Index < 0 || layout.TokenProgram1Index >= accountCount) {
		logger.Errorf("[%s:%s] invalid TokenProgram1Index=%d (total=%d), tx=%s",
			dexName, instructionName, layout.TokenProgram1Index, accountCount, ctx.TxHashString())
		return false
	}
	if layout.TokenProgram2Index != -1 && (layout.TokenProgram2Index < 0 || layout.TokenProgram2Index >= accountCount) {
		logger.Errorf("[%s:%s] invalid TokenProgram2Index=%d (total=%d), tx=%s",
			dexName, instructionName, layout.TokenProgram2Index, accountCount, ctx.TxHashString())
		return false
	}
	return true
}

func validateVaultMint(
	ctx *ParserContext,
	dexName, instructionName string,
	vault types.Pubkey,
	expectedMint types.Pubkey,
	label string, // 如 "vault1" 或 "vault2"
) *core.TokenBalance {
	bal, ok := ctx.Balances[vault]
	if !ok {
		logger.Errorf("[%s:%s] %s balance not found: vault=%s, expected mint=%s, tx=%s",
			dexName, instructionName, label, vault, expectedMint, ctx.TxHashString())
		return nil
	}
	if bal.Token != expectedMint {
		logger.Errorf("[%s:%s] %s mint mismatch: vault=%s, expected=%s, got=%s, tx=%s",
			dexName, instructionName, label, vault, expectedMint, bal.Token, ctx.TxHashString())
		return nil
	}
	return bal
}

package common

import (
	"fmt"
	"github.com/dex-indexer-sol/pb"
	"github.com/mr-tron/base58"
)

// Base58Encode 应该替换为你实际的实现
func Base58Encode(b []byte) string {
	return base58.Encode(b)
}

func PrintTradeEvent(event *pb.Event) {
	tradeWrapper, ok := event.Event.(*pb.Event_Trade)
	if !ok || tradeWrapper.Trade == nil {
		fmt.Println("⚠️ Event is not a TradeEvent")
		return
	}

	e := tradeWrapper.Trade

	fmt.Println("📦 TradeEvent ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓")
	fmt.Printf("🆔 EventID: %d, Type: %v, Slot: %d, BlockTime: %d\n", e.EventId, e.Type, e.Slot, e.BlockTime)
	fmt.Printf("🧾 TokenAmount: %d, QuoteAmount: %d\n", e.TokenAmount, e.QuoteTokenAmount)
	fmt.Printf("💰 AmountUSD: %.4f, PriceUSD: %.14f\n", e.AmountUsd, e.PriceUsd)
	fmt.Printf("📈 TokenDecimals: %d, QuoteDecimals: %d\n", e.TokenDecimals, e.QuoteDecimals)
	fmt.Printf("🏦 Dex: %d\n", e.Dex)
	fmt.Printf("🎯 Token: %s\n", Base58Encode(e.Token))
	fmt.Printf("🎯 Quote: %s\n", Base58Encode(e.QuoteToken))
	fmt.Printf("🏷️  PairAddress: %s\n", Base58Encode(e.PairAddress))
	fmt.Printf("🏦 TokenAccount: %s\n", Base58Encode(e.TokenAccount))
	fmt.Printf("🏦 QuoteTokenAccount: %s\n", Base58Encode(e.QuoteTokenAccount))
	fmt.Printf("👛 UserWallet: %s\n", Base58Encode(e.UserWallet))
	fmt.Printf("📊 PairTokenBalance: %d, PairQuoteBalance: %d\n", e.PairTokenBalance, e.PairQuoteBalance)
	fmt.Printf("📊 UserTokenBalance: %d, UserQuoteBalance: %d\n", e.UserTokenBalance, e.UserQuoteBalance)

	for i, signer := range e.Signers {
		fmt.Printf("✍️ Signer[%d]: %s\n", i, Base58Encode(signer))
	}
	fmt.Printf("🧾 TxHash: %s\n", Base58Encode(e.TxHash))
	a := 0
	_ = a
}

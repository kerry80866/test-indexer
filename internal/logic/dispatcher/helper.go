package dispatcher

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/tools"
	"dex-indexer-sol/pb"
	"fmt"
)

func buildEventsProto(ctx *core.TxContext, events []*pb.Event, source int32) *pb.Events {
	return &pb.Events{
		Version:     1,
		ChainId:     consts.ChainIDSolana,
		Slot:        ctx.Slot,
		Source:      source,
		Events:      events,
		BlockHash:   ctx.BlockHash[:],
		QuotePrices: convertToPbTokenPrices(ctx.QuotesPrice),
	}
}

func convertToPbTokenPrices(quotes []core.QuotePrice) []*pb.TokenPrice {
	result := make([]*pb.TokenPrice, 0, len(quotes)+1)

	var (
		hasNative    = false
		hasWSOL      = false
		wsolPrice    float64
		wsolDecimals uint32
	)

	for _, q := range quotes {
		decimals, ok := tools.QuoteDecimals[q.Token]
		if !ok {
			panic(fmt.Sprintf("未配置报价币 %s 的 decimals", q.Token.String()))
		}

		result = append(result, &pb.TokenPrice{
			Token:    q.Token[:],
			Price:    q.PriceUsd,
			Decimals: uint32(decimals),
		})

		switch q.Token {
		case consts.NativeSOLMint:
			hasNative = true
		case consts.WSOLMint:
			hasWSOL = true
			wsolPrice = q.PriceUsd
			wsolDecimals = uint32(decimals)
		}
	}

	if !hasNative && hasWSOL {
		result = append(result, &pb.TokenPrice{
			Token:    consts.NativeSOLMint[:],
			Price:    wsolPrice,
			Decimals: wsolDecimals,
		})
	}

	return result
}

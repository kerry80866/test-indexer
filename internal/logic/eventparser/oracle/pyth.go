package oracle

import (
	"bytes"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/logger"
	"dex-indexer-sol/internal/pkg/types"
	"encoding/binary"
	"github.com/near/borsh-go"
	"math"
	"time"
)

const (
	PythPostUpdate uint64 = 0x855fcfaf0b4f762c
)

func handlePythInstruction(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) int {
	ix := instrs[current]
	if len(ix.Data) < 8 {
		return -1
	}
	switch binary.BigEndian.Uint64(ix.Data[:8]) {
	case PythPostUpdate:
		return extraPostUpdateEvent(ctx, instrs, current)

	default:
		return -1
	}
}

type MerklePriceUpdate struct {
	Message []byte
	Proof   [][20]byte
}

type PostUpdateParams struct {
	MerklePriceUpdate MerklePriceUpdate
	TreasuryId        uint8
}

type PriceFeedMessage struct {
	FeedID      []byte
	Price       float64
	Confidence  float64
	PublishTime int64
}

const PriceFeedVariant = 0

// 参考: https://github.com/pyth-network/pyth-crosschain/blob/main/target_chains/solana/sdk/js/pyth_solana_receiver/src/PythSolanaReceiver.ts - buildPostPriceUpdateInstructions
func extraPostUpdateEvent(
	ctx *common.ParserContext,
	instrs []*core.AdaptedInstruction,
	current int,
) (next int) {
	ix := instrs[current]

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[pyth][panic] borsh.Deserialize panic: %v, tx=%v, ixIndex=%d, innerIndex=%d",
				r, ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
			next = -1
		}
	}()

	var params PostUpdateParams
	err := borsh.Deserialize(&params, ix.Data[8:])
	if err != nil {
		logger.Errorf("[pyth][panic] failed to deserialize PostUpdateParams: %v, tx=%v, ixIndex=%d, innerIndex=%d",
			err, ctx.TxHashString(), ix.IxIndex, ix.InnerIndex)
		return -1
	}

	msg := parsePriceFeedMessage(params.MerklePriceUpdate.Message)
	if msg == nil {
		return -1
	}

	var token string
	var tokenMint types.Pubkey
	if bytes.Equal(msg.FeedID, consts.PythSOLFeedID) {
		token = "sol"
		tokenMint = consts.WSOLMint
	} else if bytes.Equal(msg.FeedID, consts.PythUSDCFeedID) {
		token = "usdc"
		tokenMint = consts.USDCMint
	} else if bytes.Equal(msg.FeedID, consts.PythUSDTFeedID) {
		token = "usdt"
		tokenMint = consts.USDTMint
	} else {
		return current + 1
	}

	if isConfidenceTooLow(token, msg.Price, msg.Confidence) {
		logger.Warnf("[pyth][catch] low confidence: token=%s, price=%.6f, conf=%.6f (%.2f%%), tx=%s",
			token, msg.Price, msg.Confidence, 100*msg.Confidence/msg.Price, ctx.TxHashString())
		return current + 1
	}

	logger.Infof("[pyth] parsed price → token=%s, priceUsd=%.6f, publishTime=%s, slot=%d, txIndex=%d",
		token, msg.Price, time.Unix(msg.PublishTime, 0).Format("2006-01-02 15:04:05"), ctx.Slot, ctx.TxIndex)

	ctx.AddPriceEvent(&core.PriceEvent{
		ID:          core.BuildEventID(ctx.Slot, ctx.TxIndex, ix.IxIndex, ix.InnerIndex),
		PriceUsd:    msg.Price,
		PublishTime: msg.PublishTime,
		TokenMint:   tokenMint,
	})
	return current + 1
}

// 参考: https://github.com/pyth-network/pyth-crosschain/blob/main/price_service/sdk/js/src/AccumulatorUpdateData.ts - parsePriceFeedMessage
func parsePriceFeedMessage(msg []byte) *PriceFeedMessage {
	if len(msg) != 85 {
		logger.Errorf("[pyth] invalid message length: got %d, expected 85", len(msg))
		return nil
	}
	if msg[0] != PriceFeedVariant {
		logger.Errorf("[pyth] unsupported variant: got %d, expected %d", msg[0], PriceFeedVariant)
		return nil
	}

	offset := 1
	feedID := msg[offset : offset+32]

	offset += 32
	price := int64(binary.BigEndian.Uint64(msg[offset : offset+8]))
	offset += 8
	confidence := binary.BigEndian.Uint64(msg[offset : offset+8])
	offset += 8
	exponent := int32(binary.BigEndian.Uint32(msg[offset : offset+4]))
	offset += 4
	publishTime := int64(binary.BigEndian.Uint64(msg[offset : offset+8]))
	//offset += 8
	//prevPublishTime := int64(binary.BigEndian.Uint64(msg[offset : offset+8]))
	//offset += 8
	//emaPrice := int64(binary.BigEndian.Uint64(msg[offset : offset+8]))
	//offset += 8
	//emaConfidence := binary.BigEndian.Uint64(msg[offset : offset+8])

	return &PriceFeedMessage{
		FeedID:      feedID,
		Price:       float64(price) * math.Pow10(int(exponent)),
		Confidence:  float64(confidence) * math.Pow10(int(exponent)),
		PublishTime: publishTime,
	}
}

func isConfidenceTooLow(token string, price, conf float64) bool {
	switch token {
	case "usdc", "usdt":
		return conf > 0.005*price // 稳定币允许最大 0.5% 的置信误差
	case "sol":
		return conf > 0.02*price // SOL 允许最大 2% 的误差
	default:
		return conf > 0.05*price // 其他未知 token，允许最大 5%
	}
}

package common

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/pb"
)

func BuildBalanceUpdateEvents(results []core.ParsedTxResult, slot uint64, blockTime int64) []*core.Event {
	type Pubkey = types.Pubkey

	estimatedSize := 0
	for _, res := range results {
		estimatedSize += len(res.Balances)
	}

	accountMap := make(map[Pubkey]*core.TokenBalance, estimatedSize)
	orderedKeys := make([]Pubkey, 0, estimatedSize)

	for _, res := range results {
		for acct, bal := range res.Balances {
			// 过滤无变动账户
			if bal.PreBalance == 0 && bal.PostBalance == 0 {
				continue
			}

			// 保留首次出现顺序
			if _, exists := accountMap[acct]; !exists {
				orderedKeys = append(orderedKeys, acct)
			}

			// 直接使用最后一次 TokenBalance（我们只关心最终状态）
			accountMap[acct] = bal
		}
	}

	events := make([]*core.Event, 0, len(orderedKeys))

	for _, acct := range orderedKeys {
		tb := accountMap[acct]

		pbEvent := &pb.Event{
			Event: &pb.Event_Balance{
				Balance: &pb.BalanceUpdateEvent{
					Type:        pb.EventType_BALANCE_UPDATE,
					TxIndex:     tb.TxIndex,
					Slot:        slot,
					BlockTime:   blockTime,
					Token:       tb.Token[:],
					Account:     tb.TokenAccount[:],
					Owner:       tb.Owner[:],
					PreBalance:  tb.PreBalance,
					PostBalance: tb.PostBalance,
					Decimals:    uint32(tb.Decimals),
				},
			},
		}

		events = append(events, &core.Event{
			ID:        tb.TxIndex,
			EventType: uint32(pb.EventType_BALANCE_UPDATE),
			Key:       tb.Owner[:], // Kafka 分区 key 建议使用 Owner
			Event:     pbEvent,
		})
	}
	return events
}

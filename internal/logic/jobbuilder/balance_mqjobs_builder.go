package jobbuilder

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/pkg/mq"
	"dex-indexer-sol/internal/pkg/types"
	"dex-indexer-sol/internal/pkg/utils"
	"dex-indexer-sol/pb"
	"fmt"
	"sort"
	"sync"
)

const maxCapPerBucket = 4096
const maxSafePartitions = 64

var staticBuckets [][]*core.TokenBalance

func reuseStaticBuckets(partitions int) [][]*core.TokenBalance {
	if partitions > maxSafePartitions {
		panic(fmt.Sprintf("too many partitions: %d > safe max=%d", partitions, maxSafePartitions))
	}

	if staticBuckets == nil {
		staticBuckets = make([][]*core.TokenBalance, partitions)
		for i := 0; i < partitions; i++ {
			staticBuckets[i] = make([]*core.TokenBalance, 0, maxCapPerBucket)
		}
	} else {
		// 扩容 bucket
		if partitions > len(staticBuckets) {
			for i := len(staticBuckets); i < partitions; i++ {
				staticBuckets = append(staticBuckets, make([]*core.TokenBalance, 0, maxCapPerBucket))
			}
		}
		// 清空已有 bucket
		for i := 0; i < partitions; i++ {
			staticBuckets[i] = staticBuckets[i][:0]
		}
	}

	return staticBuckets[:partitions]
}

// BuildBalanceKafkaJobs 构造 TokenBalance 类型的 KafkaJob。
// 每个 KafkaJob 对应一个分区，内部包含多个 BalanceUpdateEvent。
func BuildBalanceKafkaJobs(
	txCtx *core.TxContext,
	quotePrices []*pb.TokenPrice,
	source int32,
	topic string,
	partitions int,
	results []core.ParsedTxResult,
) ([]*mq.KafkaJob, int) {
	if partitions <= 0 {
		partitions = 1
	}

	// 直接按 TokenAccount 分区，跳过清除逻辑
	buckets := reuseStaticBuckets(partitions)
	for _, res := range results {
		for _, bal := range res.Balances {
			if bal.PreBalance == 0 && bal.PostBalance == 0 {
				continue // 临时账户
			}
			pid := utils.PartitionHashBytes(bal.TokenAccount[:], uint32(partitions))
			buckets[pid] = append(buckets[pid], bal)
		}
	}

	// 并发构建 KafkaJob
	jobs := make([]*mq.KafkaJob, partitions)
	var wg sync.WaitGroup
	for i := 0; i < partitions; i++ {
		j := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs[j] = buildBalancePartitionJob(txCtx, quotePrices, source, topic, j, buckets[j])
		}()
	}
	wg.Wait()

	// 去除空 Job，返回有效部分
	totalEvents := 0
	out := make([]*mq.KafkaJob, 0, partitions)
	for _, job := range jobs {
		if job != nil {
			out = append(out, job)
			if msg, ok := job.Msg.(*pb.Events); ok {
				totalEvents += len(msg.Events)
			}
		}
	}
	return out, totalEvents
}

// buildBalancePartitionJob 构建指定分区内的 KafkaJob。
func buildBalancePartitionJob(
	txCtx *core.TxContext,
	quotePrices []*pb.TokenPrice,
	source int32,
	topic string,
	partition int,
	balances []*core.TokenBalance,
) *mq.KafkaJob {
	if len(balances) == 0 {
		return nil
	}

	// 合并同一个 TokenAccount 的记录，保留 TxIndex 最大的一条（即最新的一条）
	merged := make(map[types.Pubkey]*core.TokenBalance, len(balances))
	for _, bal := range balances {
		key := bal.TokenAccount
		if exist, ok := merged[key]; !ok {
			merged[key] = bal
		} else if exist.TxIndex < bal.TxIndex {
			bal.PreBalance = exist.PreBalance // 仅污染 PreBalance
			merged[key] = bal                 // 替换为更新版本
		}
	}

	// 构造 Protobuf Events 列表
	slot := txCtx.Slot
	blockTime := txCtx.BlockTime
	events := make([]*pb.Event, 0, len(merged))
	for _, bal := range merged {
		id := slot<<32 | uint64(bal.TxIndex)<<16 | uint64(bal.InnerIndex)
		events = append(events, &pb.Event{
			Event: &pb.Event_Balance{
				Balance: &pb.BalanceUpdateEvent{
					Type:        pb.EventType_BALANCE_UPDATE,
					EventId:     id,
					Slot:        slot,
					BlockTime:   blockTime,
					Token:       bal.Token[:],
					Account:     bal.TokenAccount[:],
					Owner:       bal.PostOwner[:],
					PreBalance:  bal.PreBalance,
					PostBalance: bal.PostBalance,
					Decimals:    uint32(bal.Decimals),
				},
			},
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].GetBalance().EventId < events[j].GetBalance().EventId
	})

	// 封装为 KafkaJob
	return &mq.KafkaJob{
		Topic:     topic,
		Partition: int32(partition),
		Msg: &pb.Events{
			Version:     1,
			ChainId:     consts.ChainIDSolana,
			Slot:        txCtx.Slot,
			Source:      source,
			Events:      events,
			BlockHash:   txCtx.BlockHash[:],
			QuotePrices: quotePrices,
		},
	}
}

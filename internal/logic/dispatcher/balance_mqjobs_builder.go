package dispatcher

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/mq"
	"dex-indexer-sol/internal/types"
	"dex-indexer-sol/internal/utils"
	"dex-indexer-sol/pb"
	"sort"
	"sync"
)

// BuildBalanceKafkaJobs 构造 TokenBalance 类型的 KafkaJob。
// 每个 KafkaJob 对应一个分区，内部包含多个 BalanceUpdateEvent。
func BuildBalanceKafkaJobs(
	slot uint64,
	blockTime int64,
	source int32,
	topic string,
	partitions int,
	results []core.ParsedTxResult,
) ([]*mq.KafkaJob, int) {
	if partitions <= 0 {
		partitions = 1
	}

	// 预估 balance 数量，按分区分配初始容量
	total := 0
	for _, res := range results {
		total += len(res.Balances)
	}
	capacity := utils.CalcCapPerPartition(total, partitions, 10)

	// 初始化分区数组
	buckets := make([][]*core.TokenBalance, partitions)
	for i := 0; i < partitions; i++ {
		buckets[i] = make([]*core.TokenBalance, 0, capacity)
	}

	// 分发 balance 至对应分区
	for _, res := range results {
		for _, bal := range res.Balances {
			if bal.HasPreOwner && bal.PreOwner != bal.PostOwner {
				// 插入一条“老 owner”的清除记录（用于老 owner 的落库清理）
				old := &core.TokenBalance{
					Decimals:     bal.Decimals,
					HasPreOwner:  false,
					TxIndex:      bal.TxIndex,
					InnerIndex:   bal.InnerIndex,
					TokenAccount: bal.TokenAccount,
					PreBalance:   bal.PreBalance,
					PostBalance:  0,
					Token:        bal.Token,
					PostOwner:    bal.PreOwner,
				}
				i := utils.PartitionHashBytes(old.PostOwner[:], uint32(partitions))
				buckets[i] = append(buckets[i], old)

				// 原 balance 的 PreBalance 已交由 old 记录，设为 0
				bal.PreBalance = 0
			} else if bal.PreBalance == 0 && bal.PostBalance == 0 {
				continue // 忽略无效/临时账户
			}

			// 高位标记为系统生成事件（非原始交易指令）
			bal.InnerIndex |= 0x8000

			pid := utils.PartitionHashBytes(bal.PostOwner[:], uint32(partitions))
			buckets[pid] = append(buckets[pid], bal)
		}
	}

	// 并发构建 KafkaJob（每个分区一个 Job）
	jobs := make([]*mq.KafkaJob, partitions)
	var wg sync.WaitGroup
	for i := 0; i < partitions; i++ {
		j := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs[j] = buildBalancePartitionJob(slot, blockTime, source, topic, j, buckets[j])
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
	slot uint64,
	blockTime int64,
	source int32,
	topic string,
	partition int,
	balances []*core.TokenBalance,
) *mq.KafkaJob {
	if len(balances) == 0 {
		return nil
	}

	// 按 TokenAccount + PostOwner 合并余额记录
	merged := mergeBalanceByTokenAndOwner(balances)
	if len(merged) == 0 {
		return nil
	}

	// 构造 Protobuf Events 列表
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

	// 封装为 KafkaJob
	return &mq.KafkaJob{
		Topic:     topic,
		Partition: int32(partition),
		Msg: &pb.Events{
			Version: 1,
			ChainId: consts.ChainIDSolana,
			Slot:    slot,
			Source:  source,
			Events:  events,
		},
	}
}

// mergeBalanceByTokenAndOwner 合并同一 TokenAccount + PostOwner 的余额记录。
func mergeBalanceByTokenAndOwner(balances []*core.TokenBalance) []*core.TokenBalance {
	merged := make(map[types.Pubkey][]*core.TokenBalance, len(balances))

	for _, bal := range balances {
		list := merged[bal.TokenAccount]

		found := false
		for i := range list {
			if list[i].PostOwner == bal.PostOwner {
				// 更新变化字段（覆盖）
				list[i].TxIndex = bal.TxIndex
				list[i].InnerIndex = bal.InnerIndex
				list[i].PostBalance = bal.PostBalance
				found = true
				break
			}
		}
		if !found {
			merged[bal.TokenAccount] = append(list, bal)
		}
	}

	// 扁平化输出
	result := make([]*core.TokenBalance, 0, len(balances))
	for _, list := range merged {
		result = append(result, list...)
	}

	// 保证事件顺序一致性：先按 TxIndex，再按 InnerIndex 排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].TxIndex == result[j].TxIndex {
			return result[i].InnerIndex < result[j].InnerIndex
		}
		return result[i].TxIndex < result[j].TxIndex
	})
	return result
}

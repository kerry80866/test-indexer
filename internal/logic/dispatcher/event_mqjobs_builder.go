package dispatcher

import (
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/pb"
	"dex-indexer-sol/pkg/mq"
	"dex-indexer-sol/pkg/utils"
)

// BuildEventKafkaJobs 构造事件类型 KafkaJob（非余额类）。
// 每个 KafkaJob 对应一个分区，封装若干个事件（pb.Events）。
func BuildEventKafkaJobs(
	txCtx *core.TxContext,
	source int32,
	topic string,
	partitions int,
	results []core.ParsedTxResult,
) ([]*mq.KafkaJob, int, int, int, int) {
	if partitions <= 0 {
		partitions = 1
	}

	// 预估事件总数，用于初始化每个分区的容量
	total := 0
	for _, res := range results {
		total += len(res.Events)
	}
	if total == 0 {
		return nil, 0, 0, 0, 0
	}

	// 初始化每个分区的事件桶
	buckets := make([][]*pb.Event, partitions)
	capacity := utils.CalcCapPerPartition(total, partitions, 10)
	for i := range buckets {
		buckets[i] = make([]*pb.Event, 0, capacity)
	}

	// 统计 trade 和 transfer 事件数量
	tradeCount := 0
	validTradeCount := 0
	transferCount := 0

	// 按事件的 Key 分配到对应分区
	for _, res := range results {
		for _, evt := range res.Events {
			pid := utils.PartitionHashBytes(evt.Key, uint32(partitions))
			buckets[pid] = append(buckets[pid], evt.Event)

			// 统计 trade 和 transfer 事件数量
			switch evt := evt.Event.Event.(type) {
			case *pb.Event_Trade:
				tradeCount++
				if evt.Trade.PriceUsd > 0 {
					validTradeCount++
				}
			case *pb.Event_Transfer:
				transferCount++
			}
		}
	}

	// 构造每个分区的 KafkaJob
	totalEvents := 0
	jobs := make([]*mq.KafkaJob, 0, len(buckets))
	for pid, list := range buckets {
		if n := len(list); n > 0 {
			totalEvents += n
			jobs = append(jobs, &mq.KafkaJob{
				Topic:     topic,
				Partition: int32(pid),
				Msg:       buildEventsProto(txCtx, list, source),
			})
		}
	}
	return jobs, totalEvents, tradeCount, validTradeCount, transferCount
}

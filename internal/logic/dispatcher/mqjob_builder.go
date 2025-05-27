package dispatcher

import (
	"dex-indexer-sol/internal/config"
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/core"
	"dex-indexer-sol/internal/mq"
	"dex-indexer-sol/internal/utils"
	"dex-indexer-sol/pb"
)

// BuildAllKafkaJobs 构建包含交易类和余额类事件的所有 KafkaJob 列表。
// 输入为已解析的交易结果和已生成的余额事件，按照分区策略封装为 KafkaJob。
// 该模块构建后的 []*mq.KafkaJob 可直接传入 mq.SendKafkaJobs 发送。
func BuildAllKafkaJobs(
	slot uint64,
	source int32,
	results []core.ParsedTxResult,
	balanceEvents []*core.Event,
	cfg config.KafkaProducerConfig,
) []*mq.KafkaJob {
	// 构造普通事件（如交易、流动性）Kafka Jobs
	eventJobs := BuildEventKafkaJobs(slot, source, cfg.Topics.Event, cfg.Partitions.Event, results)

	// 构造余额变动事件 Kafka Jobs
	balanceJobs := BuildBalanceKafkaJobs(slot, source, cfg.Topics.Balance, cfg.Partitions.Balance, balanceEvents)

	// 合并所有 Kafka Jobs
	jobs := make([]*mq.KafkaJob, 0, len(eventJobs)+len(balanceJobs))
	jobs = append(jobs, eventJobs...)
	jobs = append(jobs, balanceJobs...)
	return jobs
}

// BuildEventKafkaJobs 构造普通事件的 KafkaJob，按分区分组。
// 每个 Job 代表一个分区内的事件聚合，包含一个 pb.Events protobuf 封装。
func BuildEventKafkaJobs(
	slot uint64,
	source int32,
	topic string,
	partitions int,
	results []core.ParsedTxResult,
) []*mq.KafkaJob {
	if partitions <= 0 {
		partitions = 1
	}

	// 统计事件总数，用于分配容量
	total := 0
	for _, res := range results {
		total += len(res.Events)
	}
	if total == 0 {
		return nil
	}

	// 按分区初始化 buckets，每个 bucket 保存属于该分区的事件列表
	buckets := make([][]*pb.Event, partitions)
	capacity := calcCapPerPartition(total, partitions, 10)
	for i := range buckets {
		buckets[i] = make([]*pb.Event, 0, capacity)
	}

	// 将事件分配至对应分区
	for _, res := range results {
		for _, evt := range res.Events {
			pid := utils.PartitionHashBytes(evt.Key, uint32(partitions))
			buckets[pid] = append(buckets[pid], evt.Event)
		}
	}

	// 将每个分区内的事件构造成 KafkaJob
	return buildJobs(topic, slot, source, buckets)
}

// BuildBalanceKafkaJobs 构造余额事件的 KafkaJob。
// 与 BuildEventKafkaJobs 逻辑一致，但输入为已聚合好的 balanceEvents。
func BuildBalanceKafkaJobs(
	slot uint64,
	source int32,
	topic string,
	partitions int,
	events []*core.Event,
) []*mq.KafkaJob {
	if partitions <= 0 {
		partitions = 1
	}
	if len(events) == 0 {
		return nil
	}

	// 初始化每个分区的事件容器
	buckets := make([][]*pb.Event, partitions)
	capacity := calcCapPerPartition(len(events), partitions, 10)
	for i := range buckets {
		buckets[i] = make([]*pb.Event, 0, capacity)
	}

	// 分区分发事件
	for _, evt := range events {
		pid := utils.PartitionHashBytes(evt.Key, uint32(partitions))
		buckets[pid] = append(buckets[pid], evt.Event)
	}

	return buildJobs(topic, slot, source, buckets)
}

// buildJobs 将每个分区 bucket 中的事件封装为 KafkaJob。
func buildJobs(topic string, slot uint64, source int32, buckets [][]*pb.Event) []*mq.KafkaJob {
	jobs := make([]*mq.KafkaJob, 0, len(buckets))
	for pid, list := range buckets {
		if len(list) == 0 {
			continue
		}
		jobs = append(jobs, &mq.KafkaJob{
			Topic:     topic,
			Partition: int32(pid),
			Msg: &pb.Events{
				Version: 1,
				ChainId: consts.ChainIDSolana,
				Slot:    slot,
				Source:  source,
				Events:  list,
			},
		})
	}
	return jobs
}

// calcCapPerPartition 根据总量和分区数，计算每个分区的预估初始容量，带一定冗余。
// 保底值由 minCap 保证，通常用于避免每个 bucket 初始容量太小。
func calcCapPerPartition(total, partitions, minCap int) int {
	if partitions <= 1 {
		return utils.Max(total, minCap)
	}
	return utils.Max(total*3/partitions, minCap)
}

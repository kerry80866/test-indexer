package utils

// PartitionHashBytes 从任意 byte slice 中选取 4 字节构造 uint32 并模 mod，用于分区选择。
// 非加密哈希，仅适合负载均匀场景。
func PartitionHashBytes(b []byte, mod uint32) uint32 {
	if len(b) < 28 || mod <= 1 {
		return 0
	}
	switch mod {
	case 2, 4, 8, 16:
		return uint32(b[27]) & (mod - 1) // 快速路径：低位掩码替代 hash + %
	}

	// fallback 路径：组合多个字节避免 hash 冲突
	hash := uint32(b[7])<<24 | uint32(b[15])<<16 | uint32(b[19])<<8 | uint32(b[27])
	return hash % mod
}

// CalcCapPerPartition 根据总量和分区数，计算每个分区的预估初始容量，带一定冗余。
// 保底值由 minCap 保证，通常用于避免每个 bucket 初始容量太小。
func CalcCapPerPartition(total, partitions, minCap int) int {
	if partitions <= 1 {
		return max(total, minCap)
	}
	if partitions < 5 {
		return max(total/2, minCap)
	}
	return max(total*3/partitions, minCap)
}

package utils

// PartitionHashBytes 从任意 byte slice 中选取 4 字节构造 uint32 并模 mod，用于分区选择。
// 非加密哈希，仅适合负载均匀场景。
func PartitionHashBytes(b []byte, mod uint32) uint32 {
	if len(b) < 28 || mod == 0 {
		return 0
	}
	hash := uint32(b[7])<<24 | uint32(b[15])<<16 | uint32(b[19])<<8 | uint32(b[27])
	return hash % mod
}

package consts

import "runtime"

const (
	ChainIDSolana uint32 = 100000
)

// CpuCount 表示逻辑 CPU 核心数，用于控制并发任务调度上限
var CpuCount = runtime.NumCPU()

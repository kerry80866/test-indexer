package consts

// GrpcAccountInclude 用于 Helius gRPC 区块订阅过滤器
// 包含：
// - 所有核心 Program（System、Token）
// - 主要 SPL Mint（USDC、USDT、WSOL）
var GrpcAccountInclude = []string{
	SystemProgramStr,
	TokenProgramStr,
	TokenProgram2022Str,
	AssociatedTokenProgramStr,
	TokenMetaProgramIdStr,
	ComputeBudgetProgramIdStr,

	WSOLMintStr,
	USDCMintStr,
	USDTMintStr,
}

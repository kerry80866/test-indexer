package consts

import "dex-indexer-sol/internal/types"

// Base58 地址常量（可读性高，适合配置与日志使用）
const (
	//  Programs
	SystemProgramStr          = "11111111111111111111111111111111"
	TokenProgramStr           = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	TokenProgram2022Str       = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
	AssociatedTokenProgramStr = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
	TokenMetaProgramIdStr     = "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
	ComputeBudgetProgramIdStr = "ComputeBudget111111111111111111111111111111"

	// USD 计价基础报价币（具有稳定市场价格）
	WSOLMintStr = "So11111111111111111111111111111111111111112"
	USDCMintStr = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	USDTMintStr = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"

	// 常见 SOL 衍生资产（非稳定计价，用作普通 quote）
	JitoSOLMintStr = "J1toso1uCk3RLmjorhTtrVwY9HJ7X8V9yYac6Y7kGCPn"
	MSOLMintStr    = "mSoLzYCxHdYgdzU16g5QSh3i5K3z3KZK7ytfqcJm7So"
	JupSOLMintStr  = "jupSoLaHXQiZZTSfEWMTRRgpnyFm8f6sZdosWBjx93v"
	BSOLMintStr    = "bSo13r4TkiE4KumL71LsHTPpL2euBYLFx6h9HP3piy1"

	// DEX: Raydium
	RaydiumV4ProgramStr   = "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8"
	RaydiumCLMMProgramStr = "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK"

	// DEX: PumpFun
	PumpFunProgramStr = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"

	// DEX: Meteora
	MeteoraDLMMProgramStr = "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"

	// Known Owner Addresses
	RaydiumV4AuthorityStr = "5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1"
)

var (
	// 特殊语义地址
	NativeSOLMint  = types.Pubkey{} // 原生 SOL（非 SPL）
	InvalidAddress = types.Pubkey{  // 表示无效地址（全 0xFF）
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}

	// Programs
	SystemProgram          = types.PubkeyFromBase58(SystemProgramStr)
	TokenProgram           = types.PubkeyFromBase58(TokenProgramStr)
	TokenProgram2022       = types.PubkeyFromBase58(TokenProgram2022Str)
	AssociatedTokenProgram = types.PubkeyFromBase58(AssociatedTokenProgramStr)

	// 稳定报价币（USD 估值）
	WSOLMint = types.PubkeyFromBase58(WSOLMintStr)
	USDCMint = types.PubkeyFromBase58(USDCMintStr)
	USDTMint = types.PubkeyFromBase58(USDTMintStr)

	// SOL 衍生资产
	JitoSOLMint = types.PubkeyFromBase58(JitoSOLMintStr)
	MSOLMint    = types.PubkeyFromBase58(MSOLMintStr)
	JupSOLMint  = types.PubkeyFromBase58(JupSOLMintStr)
	BSOLMint    = types.PubkeyFromBase58(BSOLMintStr)

	// DEX Program
	RaydiumV4Program   = types.PubkeyFromBase58(RaydiumV4ProgramStr)
	RaydiumCLMMProgram = types.PubkeyFromBase58(RaydiumCLMMProgramStr)
	PumpFunProgram     = types.PubkeyFromBase58(PumpFunProgramStr)
	MeteoraDLMMProgram = types.PubkeyFromBase58(MeteoraDLMMProgramStr)

	// Known Owner
	RaydiumV4Authority = types.PubkeyFromBase58(RaydiumV4AuthorityStr)
)

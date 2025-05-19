package consts

import (
	"dex-indexer-sol/internal/types"
)

// 公钥形式的地址常量（types.Pubkey），用于链上比对、性能优化等场景。
var (
	// Programs
	SystemProgram          types.Pubkey
	TokenProgram           types.Pubkey
	TokenProgram2022       types.Pubkey
	AssociatedTokenProgram types.Pubkey

	// SPL Tokens
	WSOLMint types.Pubkey
	USDCMint types.Pubkey
	USDTMint types.Pubkey

	// DEX: Raydium
	RaydiumV4Program   types.Pubkey
	RaydiumCLMMProgram types.Pubkey

	// DEX: PumpFun
	PumpFunProgram types.Pubkey

	// DEX: Meteora
	MeteoraDLMMProgram types.Pubkey
)

// init 自动将 base58 字符串地址转换为 types.Pubkey
func init() {
	SystemProgram = types.PubkeyFromBase58(SystemProgramStr)
	TokenProgram = types.PubkeyFromBase58(TokenProgramStr)
	TokenProgram2022 = types.PubkeyFromBase58(TokenProgram2022Str)
	AssociatedTokenProgram = types.PubkeyFromBase58(AssociatedTokenProgramStr)

	WSOLMint = types.PubkeyFromBase58(WSOLMintStr)
	USDCMint = types.PubkeyFromBase58(USDCMintStr)
	USDTMint = types.PubkeyFromBase58(USDTMintStr)

	RaydiumV4Program = types.PubkeyFromBase58(RaydiumV4ProgramStr)
	RaydiumCLMMProgram = types.PubkeyFromBase58(RaydiumCLMMProgramStr)

	PumpFunProgram = types.PubkeyFromBase58(PumpFunProgramStr)

	MeteoraDLMMProgram = types.PubkeyFromBase58(MeteoraDLMMProgramStr)
}

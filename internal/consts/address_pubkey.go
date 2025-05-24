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

	// USD 计价基础报价币（具有稳定市场价格）
	WSOLMint types.Pubkey
	USDCMint types.Pubkey
	USDTMint types.Pubkey

	// 常见 SOL 衍生资产（非稳定计价，用作普通 quote）
	JitoSOLMint types.Pubkey
	MSOLMint    types.Pubkey
	JupSOLMint  types.Pubkey
	BSOLMint    types.Pubkey

	// DEX: Raydium
	RaydiumV4Program   types.Pubkey
	RaydiumCLMMProgram types.Pubkey

	// DEX: PumpFun
	PumpFunProgram types.Pubkey

	// DEX: Meteora
	MeteoraDLMMProgram types.Pubkey

	// Known Owner
	RaydiumV4Authority types.Pubkey
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

	JitoSOLMint = types.PubkeyFromBase58(JitoSOLMintStr)
	MSOLMint = types.PubkeyFromBase58(MSOLMintStr)
	JupSOLMint = types.PubkeyFromBase58(JupSOLMintStr)
	JupSOLMint = types.PubkeyFromBase58(JupSOLMintStr)

	RaydiumV4Program = types.PubkeyFromBase58(RaydiumV4ProgramStr)
	RaydiumCLMMProgram = types.PubkeyFromBase58(RaydiumCLMMProgramStr)

	PumpFunProgram = types.PubkeyFromBase58(PumpFunProgramStr)

	MeteoraDLMMProgram = types.PubkeyFromBase58(MeteoraDLMMProgramStr)

	RaydiumV4Authority = types.PubkeyFromBase58(RaydiumV4AuthorityStr)
}

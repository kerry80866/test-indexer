package tools

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/pkg/types"
	"dex-indexer-sol/pb"
)

func IsSPLTokenStr(programId string) bool {
	return programId == consts.TokenProgramStr || programId == consts.TokenProgram2022Str
}

func IsSPLTokenPubkey(programId types.Pubkey) bool {
	return programId == consts.TokenProgram || programId == consts.TokenProgram2022
}

func ToTokenPubkey(programId string) types.Pubkey {
	if programId == consts.TokenProgramStr {
		return consts.TokenProgram
	}
	if programId == consts.TokenProgram2022Str {
		return consts.TokenProgram2022
	}
	return types.PubkeyFromBase58(programId)
}

func TokenProgramTypeOf(programId types.Pubkey) pb.TokenProgramType {
	switch programId {
	case consts.TokenProgram:
		return pb.TokenProgramType_TOKEN_SPL
	case consts.TokenProgram2022:
		return pb.TokenProgramType_TOKEN_2022
	default:
		return pb.TokenProgramType_TOKEN_OTHER
	}
}

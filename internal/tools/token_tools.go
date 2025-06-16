package tools

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/pkg/types"
)

// IsSPLToken 判断一个 ProgramId 是否为标准的 SPL Token 程序。
// 支持 Token v1（Tokenkeg...）和 Token-2022（Tokenz...）
func IsSPLToken(programId string) bool {
	return programId == consts.TokenProgramStr || programId == consts.TokenProgram2022Str
}

func IsSPLTokenProgram(programId types.Pubkey) bool {
	return programId == consts.TokenProgram || programId == consts.TokenProgram2022
}

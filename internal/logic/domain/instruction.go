package domain

import "dex-indexer-sol/internal/types"

// Instruction 表示链上的一条原始指令（可为主指令或 inner 指令）。
type Instruction struct {
	ProgramID types.Pubkey   // 所调用的程序地址（例如 TokenProgram）
	Accounts  []types.Pubkey // 指令涉及的账户列表，保持原始顺序
	Data      []byte         // 指令数据（Base64 解码后的原始字节序列）
}

// TranslatedInstruction 表示一条主指令及其关联的 inner 指令集合。
type TranslatedInstruction struct {
	Instruction Instruction   // 主指令（outer）
	Inners      []Instruction // inner 指令列表（可为空）
}

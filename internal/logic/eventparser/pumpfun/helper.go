package pumpfun

import (
	"encoding/binary"
	"github.com/dex-indexer-sol/internal/consts"
	"github.com/dex-indexer-sol/internal/logic/core"
	"github.com/dex-indexer-sol/pkg/types"
)

const Event uint64 = 0xe445a52e51cb9a1d

func findEventInstruction(
	instrs []*core.AdaptedInstruction,
	current int,
	eventAuthority types.Pubkey,
) int {
	mainIx := instrs[current]
	for i := current + 1; i < len(instrs); i++ {
		ix := instrs[i]

		// 只处理当前主指令的 inner 指令
		if ix.IxIndex != mainIx.IxIndex {
			return -1
		}

		if ix.ProgramID != consts.PumpFunProgram {
			continue
		}

		if len(ix.Data) < 8 {
			continue
		}

		if binary.BigEndian.Uint64(ix.Data[:8]) != Event {
			continue
		}

		if len(ix.Accounts) == 0 {
			continue
		}

		// 默认 Pump.fun 的事件日志指令以 eventAuthority 作为第 0 个账户。
		// 若未来协议发生变化，此逻辑可能需要调整。
		if eventAuthority == ix.Accounts[0] {
			return i
		}
	}
	return -1
}

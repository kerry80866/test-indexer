package oracle

import (
	"dex-indexer-sol/internal/consts"
	"dex-indexer-sol/internal/logic/eventparser/common"
	"dex-indexer-sol/internal/pkg/types"
)

func RegisterHandlers(m map[types.Pubkey]common.InstructionHandler) {
	m[consts.PythReceiverAddr] = handlePythInstruction
}

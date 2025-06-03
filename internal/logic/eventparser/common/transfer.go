package common

func isTransferConflict(pt, other *ParsedTransfer) bool {
	if other == nil {
		return false
	}
	return pt.DestAccount == other.DestAccount || pt.SrcAccount == other.SrcAccount
}

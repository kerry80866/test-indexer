package consts

const (
	DexRaydiumV4     = iota + 1 // 1
	DexRaydiumCLMM              // 2
	DexPumpfunAMM               // 3
	DexPumpfun                  // 4
	DexRaydiumCPMM              // 5
	DexMeteoraDLMM              // 6
	DexOrcaWhirlpool            // 7
)

var DexNames = []string{
	"Unknown",       // 0 (保留)
	"RaydiumV4",     // 1
	"RaydiumCLMM",   // 2
	"PumpfunAMM",    // 3
	"Pumpfun",       // 4
	"RaydiumCPMM",   // 5
	"MeteoraDLMM",   // 6
	"OrcaWhirlpool", // 7
}

func DexName(dex int) string {
	if dex >= 1 && dex < len(DexNames) {
		return DexNames[dex]
	}
	return DexNames[0] // Unknown
}

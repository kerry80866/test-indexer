package consts

import (
	"dex-indexer-sol/internal/pkg/types"
	"encoding/hex"
)

const (
	// https://docs.pyth.network/price-feeds/contract-addresses/solana
	PythReceiverAddrStr = "rec5EKMGg6MxZYaMdyBfgwp4d5rB9T1VQH5pJv5LtFJ"

	// https://www.pyth.network/price-feeds/crypto-sol-usd
	PythSOLFeedIDStr = "ef0d8b6fda2ceba41da15d4095d1da392a0d2f8ed0c6c7bc0f4cfac8c280b56d"
	PythSOLAccount   = "H6ARHf6YXhGYeQfUzQNGk6rDNnLBQKrenN712K4AQJEG"

	// https://www.pyth.network/price-feeds/crypto-usdc-usd
	PythUSDCFeedIDStr = "eaa020c61cc479712813461ce153894a96a6c00b21ed0cfc2798d1f9a9e9c94a"
	PythUSDCAccount   = "Gnt27xtC473ZT2Mw5u8wZ68Z3gULkSTb5DuxJy7eJotD"

	// https://www.pyth.network/price-feeds/crypto-usdt-usd
	PythUSDTFeedIDStr = "2b89b9dc8fdf9f34709a5b106b472f0f39bb6ca9ce04b0fd7f2e971688e2e53b"
	PythUSDTAccount   = "3vxLXJqLqF3JG5TCbYycbKWRBbCJQLxQmBGCkyqEEefL"
)

var (
	PythReceiverAddr  = types.PubkeyFromBase58(PythReceiverAddrStr)
	PythSOLFeedID, _  = hex.DecodeString(PythSOLFeedIDStr)
	PythUSDCFeedID, _ = hex.DecodeString(PythUSDCFeedIDStr)
	PythUSDTFeedID, _ = hex.DecodeString(PythUSDTFeedIDStr)
)

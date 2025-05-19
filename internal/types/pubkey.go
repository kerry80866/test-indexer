package types

import (
	"fmt"
	"github.com/mr-tron/base58"
)

type Pubkey [32]byte

func (p Pubkey) String() string {
	return base58.Encode(p[:])
}

func (p Pubkey) Equals(other Pubkey) bool {
	return p == other
}

// TryPubkeyFromBase58 解析 base58 字符串为 Pubkey，失败时返回 error（用于不信任输入路径）
func TryPubkeyFromBase58(s string) (Pubkey, error) {
	data, err := base58.Decode(s)
	if err != nil {
		return Pubkey{}, fmt.Errorf("failed to decode base58 pubkey %q: %w", s, err)
	}
	if len(data) != 32 {
		return Pubkey{}, fmt.Errorf("invalid pubkey length: got %d, want 32, input=%q", len(data), s)
	}
	var p Pubkey
	copy(p[:], data)
	return p, nil
}

func PubkeyFromBase58(s string) Pubkey {
	data, err := base58.Decode(s)
	if err != nil {
		panic(fmt.Errorf("failed to decode base58 pubkey %q: %w", s, err))
	}
	if len(data) != 32 {
		panic(fmt.Errorf("invalid pubkey length: got %d, want 32, input=%q", len(data), s))
	}
	var p Pubkey
	copy(p[:], data)
	return p
}

func PubkeysFromBase58(strs []string) []Pubkey {
	result := make([]Pubkey, 0, len(strs))
	for _, s := range strs {
		result = append(result, PubkeyFromBase58(s))
	}
	return result
}

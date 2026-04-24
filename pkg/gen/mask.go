package gen

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/codingconcepts/edg/pkg/random"
)

var (
	maskKey  []byte
	maskOnce sync.Once
)

func ensureMaskKey() {
	maskOnce.Do(func() {
		maskKey = make([]byte, 32)
		for i := range maskKey {
			maskKey[i] = byte(random.Rng.IntN(256))
		}
	})
}

// SetMaskKey overrides the session mask key, useful for deterministic
// seeded runs where mask output must be reproducible.
func SetMaskKey(key []byte) {
	maskKey = key
	maskOnce.Do(func() {}) // prevent lazy init from overwriting
}

// Mask produces a deterministic, pseudonymized token for the given
// value. The same input always produces the same output within a
// session. The optional length parameter controls the hex token
// length (default 16 characters).
//
//	mask('john@example.com')       → "a3f8c1d9e2b74f06"
//	mask('john@example.com', 8)    → "a3f8c1d9"
//	mask(arg(0))                   → consistent token for whatever arg(0) is
func Mask(value any, args ...any) (string, error) {
	ensureMaskKey()

	s := fmt.Sprintf("%v", value)

	mac := hmac.New(sha256.New, maskKey)
	mac.Write([]byte(s))
	hash := hex.EncodeToString(mac.Sum(nil))

	length := 16
	if len(args) > 0 {
		switch v := args[0].(type) {
		case int:
			length = v
		case float64:
			length = int(v)
		default:
			return "", fmt.Errorf("mask length: expected int, got %T", args[0])
		}
	}

	if length > len(hash) {
		length = len(hash)
	}
	if length < 1 {
		length = 1
	}

	return hash[:length], nil
}

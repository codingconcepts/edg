package gen

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
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
// session.
//
//	mask('john@example.com')                -> "a3f8c1d9e2b74f06"
//	mask('john@example.com', 8)             -> "a3f8c1d9"
//	mask('john@example.com', 'base64')      -> "o/jB2eK3TwYK..."
//	mask('john@example.com', 'asterisk', 8) -> "********"
//	mask('john@example.com', 'email')       -> "********@example.com"
//	mask('john@example.com', 'redact')      -> "[REDACTED]"
func Mask(value any, args ...any) (string, error) {
	ensureMaskKey()

	mode := "hex"
	length := 16

	switch len(args) {
	case 0:
	case 1:
		switch v := args[0].(type) {
		case string:
			mode = v
		case int:
			length = v
		case float64:
			length = int(v)
		default:
			return "", fmt.Errorf("mask: expected mode (string) or length (int), got %T", args[0])
		}
	default:
		s, ok := args[0].(string)
		if !ok {
			return "", fmt.Errorf("mask: expected mode (string) as first option, got %T", args[0])
		}
		mode = s
		if len(args) >= 2 {
			switch v := args[1].(type) {
			case int:
				length = v
			case float64:
				length = int(v)
			default:
				return "", fmt.Errorf("mask: expected length (int) as second option, got %T", args[1])
			}
		}
	}

	if length < 1 {
		length = 1
	}

	s := fmt.Sprintf("%v", value)

	switch mode {
	case "hex":
		return maskHex(s, length), nil
	case "base64":
		return maskBase64(s, length), nil
	case "base32":
		return maskBase32(s, length), nil
	case "asterisk":
		return strings.Repeat("*", length), nil
	case "redact":
		return "[REDACTED]", nil
	case "email":
		return maskEmail(s, length), nil
	default:
		return "", fmt.Errorf("mask: unknown mode %q (supported: hex, base64, base32, asterisk, redact, email)", mode)
	}
}

func hmacHash(input string) []byte {
	mac := hmac.New(sha256.New, maskKey)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func maskHex(input string, length int) string {
	encoded := hex.EncodeToString(hmacHash(input))
	if length > len(encoded) {
		length = len(encoded)
	}
	return encoded[:length]
}

func maskBase64(input string, length int) string {
	encoded := base64.StdEncoding.EncodeToString(hmacHash(input))
	if length > len(encoded) {
		length = len(encoded)
	}
	return encoded[:length]
}

func maskBase32(input string, length int) string {
	encoded := base32.StdEncoding.EncodeToString(hmacHash(input))
	if length > len(encoded) {
		length = len(encoded)
	}
	return encoded[:length]
}

func maskEmail(input string, length int) string {
	at := strings.LastIndex(input, "@")
	if at < 0 {
		return strings.Repeat("*", length)
	}
	return strings.Repeat("*", length) + input[at:]
}

package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// EnterpriseDrivers lists the driver names that require a license.
var EnterpriseDrivers = []string{"oracle", "mssql", "dsql"}

// License represents a signed license payload.
type License struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Drivers   []string  `json:"drivers"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

// envelope is the wire format before final base64 encoding.
type envelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

// Verify decodes a license string, verifies the Ed25519 signature against
// the public key, and returns the parsed License.
func Verify(publicKey ed25519.PublicKey, licenseStr string) (*License, error) {
	envBytes, err := base64.StdEncoding.DecodeString(licenseStr)
	if err != nil {
		return nil, fmt.Errorf("decoding license: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return nil, fmt.Errorf("parsing license envelope: %w", err)
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return nil, fmt.Errorf("decoding signature: %w", err)
	}

	if !ed25519.Verify(publicKey, payloadBytes, sigBytes) {
		return nil, errors.New("invalid license signature")
	}

	var lic License
	if err := json.Unmarshal(payloadBytes, &lic); err != nil {
		return nil, fmt.Errorf("parsing license payload: %w", err)
	}

	return &lic, nil
}

// HasDriver reports whether the license includes the named driver.
func (l *License) HasDriver(name string) bool {
	return slices.Contains(l.Drivers, name)
}

// IsExpired reports whether the license has passed its expiry time.
func (l *License) IsExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

// IsEnterprise reports whether the given driver name requires a license.
func IsEnterprise(driver string) bool {
	return slices.Contains(EnterpriseDrivers, driver)
}

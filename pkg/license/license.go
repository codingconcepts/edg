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

// licenseWire is the compact wire format for serialization.
type licenseWire struct {
	I string   `json:"i"` // ID
	E string   `json:"e"` // Email
	D []string `json:"d"` // Drivers
	X int64    `json:"x"` // ExpiresAt (unix seconds)
	A int64    `json:"a"` // IssuedAt (unix seconds)
}

const ed25519SigSize = 64

// Verify decodes a license string, verifies the Ed25519 signature against
// the public key, and returns the parsed License.
func Verify(publicKey ed25519.PublicKey, licenseStr string) (*License, error) {
	raw, err := base64.RawURLEncoding.DecodeString(licenseStr)
	if err != nil {
		return nil, fmt.Errorf("decoding license: %w", err)
	}

	if len(raw) <= ed25519SigSize {
		return nil, errors.New("license too short")
	}

	payloadBytes := raw[:len(raw)-ed25519SigSize]
	sig := raw[len(raw)-ed25519SigSize:]

	if !ed25519.Verify(publicKey, payloadBytes, sig) {
		return nil, errors.New("invalid license signature")
	}

	var w licenseWire
	if err := json.Unmarshal(payloadBytes, &w); err != nil {
		return nil, fmt.Errorf("parsing license payload: %w", err)
	}

	lic := &License{
		ID:        w.I,
		Email:     w.E,
		Drivers:   w.D,
		ExpiresAt: time.Unix(w.X, 0).UTC(),
		IssuedAt:  time.Unix(w.A, 0).UTC(),
	}
	return lic, nil
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

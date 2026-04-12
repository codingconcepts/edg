package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/codingconcepts/edg/pkg/license"
	"github.com/spf13/cobra"
)

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config or license",
	}

	cmd.AddCommand(validateConfigCmd(), validateLicenseCmd())
	return cmd
}

func validateConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Validate a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				return fmt.Errorf("--config flag required")
			}

			req, err := config.LoadConfig(configFile)
			if err != nil {
				return err
			}

			if _, err := env.NewEnv(nil, "", req); err != nil {
				return err
			}

			fmt.Println("Config is valid.")
			return nil
		},
	}
}

func validateLicenseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "license",
		Short: "Validate a license key",
		RunE: func(cmd *cobra.Command, args []string) error {
			licStr := flagLicense
			if licStr == "" {
				licStr = os.Getenv("EDG_LICENSE")
			}
			if licStr == "" {
				return fmt.Errorf("--license flag or EDG_LICENSE env var required")
			}

			pubBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKeyB64))
			if err != nil {
				return fmt.Errorf("decoding public key: %w", err)
			}
			publicKey := ed25519.PublicKey(pubBytes)

			lic, err := license.Verify(publicKey, licStr)
			if err != nil {
				return fmt.Errorf("invalid license: %w", err)
			}

			fmt.Println("License info:")
			fmt.Printf("  ID:         %s\n", lic.ID)
			fmt.Printf("  Email:      %s\n", lic.Email)
			fmt.Printf("  Drivers:    %v\n", lic.Drivers)
			fmt.Printf("  Issued at:  %s\n", lic.IssuedAt.Format("2006-01-02"))
			fmt.Printf("  Expires at: %s\n", lic.ExpiresAt.Format("2006-01-02"))

			if lic.IsExpired() {
				return fmt.Errorf("license expired on %s", lic.ExpiresAt.Format("2006-01-02"))
			}

			if license.IsEnterprise(flagDriver) {
				if !lic.HasDriver(flagDriver) {
					return fmt.Errorf("license does not include driver %q (licensed: %v)", flagDriver, lic.Drivers)
				}
				fmt.Printf("License is valid for driver %q.\n", flagDriver)
			} else {
				fmt.Printf("Driver %q does not require a license.\n", flagDriver)
			}
			return nil
		},
	}
}

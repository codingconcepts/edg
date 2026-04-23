package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBindEnv(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		flagVal  string
		setFlag  bool
		envVar   string
		envVal   string
		setEnv   bool
		expected string
	}{
		{
			name:     "env overrides default",
			flag:     "url",
			envVar:   "EDG_URL",
			envVal:   "postgres://from-env",
			setEnv:   true,
			expected: "postgres://from-env",
		},
		{
			name:     "flag takes precedence over env",
			flag:     "url",
			flagVal:  "postgres://from-flag",
			setFlag:  true,
			envVar:   "EDG_URL",
			envVal:   "postgres://from-env",
			setEnv:   true,
			expected: "postgres://from-flag",
		},
		{
			name:     "default when neither set",
			flag:     "driver",
			envVar:   "EDG_DRIVER",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var val string
			cmd.Flags().StringVar(&val, tt.flag, "", "")

			if tt.setFlag {
				cmd.Flags().Set(tt.flag, tt.flagVal)
			}

			if tt.setEnv {
				t.Setenv(tt.envVar, tt.envVal)
			}

			bindEnv(cmd, tt.flag, tt.envVar)

			assert.Equal(t, tt.expected, val)
		})
	}
}

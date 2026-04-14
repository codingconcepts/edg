package test

import (
	"os"
	"testing"
)

func CleanupEnv(t *testing.T, keys ...string) {
	f := func() {
		for _, k := range keys {
			os.Unsetenv(k)
		}
	}

	t.Cleanup(f)
}

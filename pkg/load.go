package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads a YAML config file, resolves any !include directives,
// and unmarshals the result into a Request.
func LoadConfig(path string) (*Request, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path %s: %w", path, err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	baseDir := filepath.Dir(abs)
	visited := map[string]bool{abs: true}
	if err := resolveIncludes(&doc, baseDir, visited); err != nil {
		return nil, fmt.Errorf("resolving includes in %s: %w", path, err)
	}

	var req Request
	if err := doc.Decode(&req); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}

	return &req, nil
}

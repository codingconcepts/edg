package config

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

	req.GlobalsOrder = extractGlobalsOrder(&doc)

	return &req, nil
}

// ParseConfig unmarshals a YAML config from raw bytes.
func ParseConfig(data []byte) (*Request, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	var req Request
	if err := doc.Decode(&req); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	req.GlobalsOrder = extractGlobalsOrder(&doc)

	return &req, nil
}

// extractGlobalsOrder returns the keys of the globals mapping in
// YAML document order. This allows expression-valued globals to
// reference earlier globals deterministically.
func extractGlobalsOrder(doc *yaml.Node) []string {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "globals" {
			node := root.Content[i+1]
			if node.Kind != yaml.MappingNode {
				return nil
			}
			keys := make([]string, 0, len(node.Content)/2)
			for j := 0; j < len(node.Content)-1; j += 2 {
				keys = append(keys, node.Content[j].Value)
			}
			return keys
		}
	}
	return nil
}

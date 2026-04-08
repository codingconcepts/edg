package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// resolveIncludes walks a yaml.Node tree and replaces any node tagged
// with !include by parsing the referenced file. Paths are resolved
// relative to baseDir. The visited set detects circular includes.
func resolveIncludes(node *yaml.Node, baseDir string, visited map[string]bool) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if err := resolveIncludes(child, baseDir, visited); err != nil {
				return err
			}
		}

	case yaml.MappingNode:
		// Content is [key, value, key, value, ...]
		for i := 1; i < len(node.Content); i += 2 {
			val := node.Content[i]
			if val.Tag == "!include" {
				replacement, err := loadInclude(val.Value, baseDir, visited)
				if err != nil {
					return err
				}
				node.Content[i] = replacement
			} else {
				if err := resolveIncludes(val, baseDir, visited); err != nil {
					return err
				}
			}
		}

	case yaml.SequenceNode:
		var expanded []*yaml.Node
		for _, item := range node.Content {
			if item.Tag == "!include" {
				replacement, err := loadInclude(item.Value, baseDir, visited)
				if err != nil {
					return err
				}
				// If the included file is a sequence, splice its items
				// into the parent sequence instead of nesting.
				if replacement.Kind == yaml.SequenceNode {
					expanded = append(expanded, replacement.Content...)
				} else {
					expanded = append(expanded, replacement)
				}
			} else {
				if err := resolveIncludes(item, baseDir, visited); err != nil {
					return err
				}
				expanded = append(expanded, item)
			}
		}
		node.Content = expanded

	case yaml.AliasNode:
		if err := resolveIncludes(node.Alias, baseDir, visited); err != nil {
			return err
		}
	}

	return nil
}

// loadInclude reads and parses a YAML file, resolving any nested
// !include directives, and returns the root content node.
func loadInclude(path, baseDir string, visited map[string]bool) (*yaml.Node, error) {
	abs := path
	if !filepath.IsAbs(path) {
		abs = filepath.Join(baseDir, path)
	}
	abs = filepath.Clean(abs)

	if visited[abs] {
		return nil, fmt.Errorf("circular include detected: %s", abs)
	}
	visited[abs] = true
	defer delete(visited, abs)

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("reading include %s: %w", abs, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing include %s: %w", abs, err)
	}

	newBaseDir := filepath.Dir(abs)
	if err := resolveIncludes(&doc, newBaseDir, visited); err != nil {
		return nil, err
	}

	// Unwrap the document node to get the actual content.
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0], nil
	}
	return &doc, nil
}

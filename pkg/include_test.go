package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NoIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.yaml", `
globals:
  batch_size: 100
up:
  - name: create_table
    query: CREATE TABLE t (id INT)
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 100 {
		t.Errorf("globals.batch_size = %v, want 100", req.Globals["batch_size"])
	}
	if len(req.Up) != 1 || req.Up[0].Name != "create_table" {
		t.Errorf("unexpected up queries: %v", req.Up)
	}
}

func TestLoadConfig_IncludeMapping(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/globals.yaml", `
batch_size: 500
workers: 4
`)
	writeFile(t, dir, "main.yaml", `
globals: !include shared/globals.yaml
up:
  - name: create_table
    query: CREATE TABLE t (id INT)
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 500 {
		t.Errorf("globals.batch_size = %v, want 500", req.Globals["batch_size"])
	}
	if req.Globals["workers"] != 4 {
		t.Errorf("globals.workers = %v, want 4", req.Globals["workers"])
	}
}

func TestLoadConfig_IncludeSequence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/schema.yaml", `
- name: create_users
  query: CREATE TABLE users (id INT)
- name: create_orders
  query: CREATE TABLE orders (id INT)
`)
	writeFile(t, dir, "main.yaml", `
up: !include shared/schema.yaml
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(req.Up) != 2 {
		t.Fatalf("expected 2 up queries, got %d", len(req.Up))
	}
	if req.Up[0].Name != "create_users" {
		t.Errorf("up[0].name = %q, want %q", req.Up[0].Name, "create_users")
	}
	if req.Up[1].Name != "create_orders" {
		t.Errorf("up[1].name = %q, want %q", req.Up[1].Name, "create_orders")
	}
}

func TestLoadConfig_IncludeSequenceItem(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/transfer.yaml", `
- name: make_transfer
  type: exec
  query: UPDATE account SET balance = balance + 1
`)
	writeFile(t, dir, "main.yaml", `
run:
  - name: check_balance
    type: query
    query: SELECT balance FROM account WHERE id = 1
  - !include shared/transfer.yaml
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(req.Run) != 2 {
		t.Fatalf("expected 2 run queries, got %d", len(req.Run))
	}
	if req.Run[0].Name != "check_balance" {
		t.Errorf("run[0].name = %q, want %q", req.Run[0].Name, "check_balance")
	}
	if req.Run[1].Name != "make_transfer" {
		t.Errorf("run[1].name = %q, want %q", req.Run[1].Name, "make_transfer")
	}
}

func TestLoadConfig_NestedIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "level2.yaml", `
batch_size: 42
`)
	writeFile(t, dir, "level1.yaml", `
globals: !include level2.yaml
`)
	// level1 itself is included from main
	writeFile(t, dir, "main.yaml", `
globals: !include level2.yaml
up:
  - name: t
    query: CREATE TABLE t (id INT)
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 42 {
		t.Errorf("globals.batch_size = %v, want 42", req.Globals["batch_size"])
	}
}

func TestLoadConfig_CircularInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", `
globals: !include b.yaml
`)
	writeFile(t, dir, "b.yaml", `
batch_size: !include a.yaml
`)

	_, err := LoadConfig(filepath.Join(dir, "a.yaml"))
	if err == nil {
		t.Fatal("expected circular include error, got nil")
	}
}

func TestLoadConfig_MissingInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.yaml", `
globals: !include nonexistent.yaml
`)

	_, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err == nil {
		t.Fatal("expected error for missing include, got nil")
	}
}

func TestLoadConfig_MultipleIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared/globals.yaml", `
batch_size: 100
`)
	writeFile(t, dir, "shared/schema.yaml", `
- name: create_table
  query: CREATE TABLE t (id INT)
`)
	writeFile(t, dir, "shared/teardown.yaml", `
- name: drop_table
  type: exec
  query: DROP TABLE t
`)
	writeFile(t, dir, "main.yaml", `
globals: !include shared/globals.yaml
up: !include shared/schema.yaml
down: !include shared/teardown.yaml
`)

	req, err := LoadConfig(filepath.Join(dir, "main.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if req.Globals["batch_size"] != 100 {
		t.Errorf("globals.batch_size = %v, want 100", req.Globals["batch_size"])
	}
	if len(req.Up) != 1 || req.Up[0].Name != "create_table" {
		t.Errorf("unexpected up: %v", req.Up)
	}
	if len(req.Down) != 1 || req.Down[0].Name != "drop_table" {
		t.Errorf("unexpected down: %v", req.Down)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

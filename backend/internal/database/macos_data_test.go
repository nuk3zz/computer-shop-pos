package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMacOSDataMigrationPreservesFilesAndPrefersDestination(t *testing.T) {
	home := t.TempDir()
	source := filepath.Join(home, "Documents", "Universal Repair POS")
	if err := os.MkdirAll(filepath.Join(source, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(source, "data", "universal-repair-pos.db-wal")
	if err := os.WriteFile(file, []byte("preserved"), 0600); err != nil {
		t.Fatal(err)
	}
	target, err := macOSDataDir(home)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(target, "data", "universal-repair-pos.db-wal"))
	if err != nil || string(content) != "preserved" {
		t.Fatalf("migration lost sidecar: %s %v", content, err)
	}
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "unrelated"), []byte("leave alone"), 0600); err != nil {
		t.Fatal(err)
	}
	again, err := macOSDataDir(home)
	if err != nil || again != target {
		t.Fatalf("destination not preferred: %s %v", again, err)
	}
	if _, err := os.Stat(filepath.Join(source, "unrelated")); err != nil {
		t.Fatal(err)
	}
}

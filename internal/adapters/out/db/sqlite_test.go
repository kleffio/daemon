package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitDB_Success(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("expected successful initialization, got error: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("database file was not created at %s", dbPath)
	}

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='servers';")
	if err != nil {
		t.Fatalf("failed to query schema: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("servers table was not created")
	}
}

func TestInitDB_InvalidPath(t *testing.T) {

	dbPath := "\x00invalid-path/test.db"

	db, err := InitDB(dbPath)
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("expected error for invalid path, got nil")
	}
}

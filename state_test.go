package scryball

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ninesl/scryball/internal/scryfall"
)

func TestWithConfig_DefaultsToMemory(t *testing.T) {
	// Test that empty DBPath defaults to in-memory
	config := ScryballConfig{
		// DBPath is empty - should default to in-memory
		AppUserAgent: "TestAgent/1.0",
	}

	sb, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("WithConfig with empty DBPath failed: %v", err)
	}
	defer sb.db.Close()

	// Verify we can use the database
	ctx := context.Background()

	// Try to insert a test query cache
	testQuery := "test:query"
	err = sb.queries.InsertQueryCache(ctx, scryfall.InsertQueryCacheParams{
		QueryText: testQuery,
		OracleIds: "[]",
	})
	if err != nil {
		t.Errorf("Failed to insert into in-memory database: %v", err)
	}

	// Verify we can retrieve it
	cached, err := sb.queries.GetCachedQuery(ctx, testQuery)
	if err != nil {
		t.Errorf("Failed to retrieve from in-memory database: %v", err)
	}
	if cached.QueryText != testQuery {
		t.Errorf("Retrieved query text mismatch: got %q, want %q", cached.QueryText, testQuery)
	}
}

func TestWithConfig_FileDatabase(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := ScryballConfig{
		DBPath: dbPath,
	}

	sb, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("WithConfig with file path failed: %v", err)
	}
	defer sb.db.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created at %s", dbPath)
	}

	// Verify database is functional
	ctx := context.Background()
	err = sb.queries.InsertQueryCache(ctx, scryfall.InsertQueryCacheParams{
		QueryText: "file:test",
		OracleIds: "[]",
	})
	if err != nil {
		t.Errorf("Failed to use file database: %v", err)
	}
}

func TestWithConfig_CreatesDirectories(t *testing.T) {
	// Test that parent directories are created
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "nested", "dirs", "test.db")

	config := ScryballConfig{
		DBPath: dbPath,
	}

	sb, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("WithConfig failed to create nested directories: %v", err)
	}
	defer sb.db.Close()

	// Verify file and directories were created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created at %s", dbPath)
	}
}

func TestSetConfig_DefaultsToMemory(t *testing.T) {
	// Store original global instance to restore later
	originalScryball := CurrentScryball
	defer func() { CurrentScryball = originalScryball }()

	// Test SetConfig with empty DBPath
	err := SetConfig(ScryballConfig{})
	if err != nil {
		t.Fatalf("SetConfig with empty config failed: %v", err)
	}

	// Verify global instance works
	ctx := context.Background()
	err = CurrentScryball.queries.InsertQueryCache(ctx, scryfall.InsertQueryCacheParams{
		QueryText: "global:test",
		OracleIds: "[]",
	})
	if err != nil {
		t.Errorf("Failed to use global in-memory database: %v", err)
	}
}

func TestDefaultInstance_UsesMemory(t *testing.T) {
	// Reset global state for this test
	originalScryball := CurrentScryball
	originalInitOnce := initOnce
	defer func() {
		CurrentScryball = originalScryball
		initOnce = originalInitOnce
	}()

	// Clear the global instance
	CurrentScryball = nil
	initOnce = sync.Once{}

	// Ensure default instance creation
	sb, err := ensureCurrentScryball()
	if err != nil {
		t.Fatalf("Failed to create default instance: %v", err)
	}

	// Verify it uses in-memory database
	ctx := context.Background()
	err = sb.queries.InsertQueryCache(ctx, scryfall.InsertQueryCacheParams{
		QueryText: "default:test",
		OracleIds: "[]",
	})
	if err != nil {
		t.Errorf("Default instance should use in-memory database: %v", err)
	}
}

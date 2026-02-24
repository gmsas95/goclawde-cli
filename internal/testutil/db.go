// Package testutil provides database testing utilities
package testutil

import (
	"testing"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDB provides a test database instance
type TestDB struct {
	DB      *gorm.DB
	Store   *store.Store
	Logger  *zap.Logger
	Cleanup func()
}

// NewTestDB creates a new test database with SQLite in-memory
func NewTestDB(t *testing.T) *TestDB {
	logger := zap.NewNop()

	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(
		// Core models
		&store.User{},
		&store.Conversation{},
		&store.Message{},
		&store.Memory{},
		&store.File{},
		&store.ScheduledJob{},
		&store.Task{},
		&store.ChatMapping{},
		&store.Config{},
		// Add other models as needed
	); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Create store (we'll need to adapt this)
	testDB := &TestDB{
		DB:     db,
		Logger: logger,
		Cleanup: func() {
			// SQLite in-memory is automatically cleaned up
		},
	}

	return testDB
}

// NewTestDBWithStore creates a test DB with full store initialization
func NewTestDBWithStore(t *testing.T) (*TestDB, *store.Store) {
	// Create a test directory
	tempDir := t.TempDir()

	// Create config with temp directory
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir:    tempDir,
			SQLitePath: tempDir + "/test.db",
			BadgerPath: tempDir + "/badger",
		},
	}

	st, err := store.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	testDB := &TestDB{
		DB:     st.DB(),
		Store:  st,
		Logger: zap.NewNop(),
		Cleanup: func() {
			st.Close()
		},
	}

	return testDB, st
}

// NewTestStore creates just a store instance for tests
func NewTestStore(t *testing.T) *store.Store {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir:    tempDir,
			SQLitePath: tempDir + "/test.db",
			BadgerPath: tempDir + "/badger",
		},
	}

	st, err := store.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	return st
}

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/config"
	"github.com/dgraph-io/badger/v4"
	_ "github.com/glebarez/go-sqlite" // Pure Go SQLite driver
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Store provides unified access to SQLite and BadgerDB
type Store struct {
	db     *gorm.DB
	badger *badger.DB
	config *config.StorageConfig
}

// New creates a new Store instance
func New(cfg *config.Config) (*Store, error) {
	// Initialize SQLite
	sqlitePath := cfg.Storage.SQLitePath
	if sqlitePath == "" {
		sqlitePath = filepath.Join(cfg.Storage.DataDir, "jimmy.db")
	}

	// Open SQLite with optimizations
	sqliteDB, err := sql.Open("sqlite", sqlitePath+"?_journal=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=-64000")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}
	
	// Configure connection pool
	sqliteDB.SetMaxOpenConns(10)
	sqliteDB.SetMaxIdleConns(5)
	sqliteDB.SetConnMaxLifetime(time.Hour)
	
	db, err := gorm.Open(sqlite.Dialector{Conn: sqliteDB}, &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Auto-migrate schemas
	if err := db.AutoMigrate(
		&Conversation{},
		&Message{},
		&Memory{},
		&File{},
		&Task{},
		&ScheduledJob{},
		&User{},
		&Config{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	// Initialize BadgerDB
	badgerPath := cfg.Storage.BadgerPath
	if badgerPath == "" {
		badgerPath = filepath.Join(cfg.Storage.DataDir, "badger")
	}

	// Open BadgerDB with optimizations
	badgerOpts := badger.DefaultOptions(badgerPath).
		WithLogger(nil). // Disable verbose logging
		WithNumVersionsToKeep(1).
		WithCompactL0OnClose(true).
		WithValueLogFileSize(16 << 20). // 16MB value log files
		WithMemTableSize(16 << 20)      // 16MB memtable
	
	badgerDB, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger: %w", err)
	}

	store := &Store{
		db:     db,
		badger: badgerDB,
		config: &cfg.Storage,
	}

	// Create default user if none exists
	if err := store.createDefaultUser(); err != nil {
		return nil, fmt.Errorf("failed to create default user: %w", err)
	}

	return store, nil
}

// Close closes all database connections
func (s *Store) Close() error {
	return s.badger.Close()
}

// DB returns the GORM database instance
func (s *Store) DB() *gorm.DB {
	return s.db
}

// Badger returns the BadgerDB instance
func (s *Store) Badger() *badger.DB {
	return s.badger
}

// createDefaultUser creates a default user if the database is empty
func (s *Store) createDefaultUser() error {
	var count int64
	if err := s.db.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		user := &User{
			ID:           "default",
			DisplayName:  "User",
			Preferences:  json.RawMessage(`{}`),
		}
		return s.db.Create(user).Error
	}

	return nil
}

// ==================== Conversation Methods ====================

// CreateConversation creates a new conversation
func (s *Store) CreateConversation(conv *Conversation) error {
	return s.db.Create(conv).Error
}

// GetConversation retrieves a conversation by ID
func (s *Store) GetConversation(id string) (*Conversation, error) {
	var conv Conversation
	if err := s.db.First(&conv, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// ListConversations lists all conversations with pagination
func (s *Store) ListConversations(limit, offset int) ([]Conversation, error) {
	var convs []Conversation
	err := s.db.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&convs).Error
	return convs, err
}

// UpdateConversation updates a conversation
func (s *Store) UpdateConversation(conv *Conversation) error {
	return s.db.Save(conv).Error
}

// DeleteConversation soft-deletes a conversation
func (s *Store) DeleteConversation(id string) error {
	return s.db.Model(&Conversation{}).Where("id = ?", id).Update("is_archived", true).Error
}

// ==================== Message Methods ====================

// CreateMessage creates a new message
func (s *Store) CreateMessage(msg *Message) error {
	return s.db.Create(msg).Error
}

// GetMessages retrieves messages for a conversation
func (s *Store) GetMessages(conversationID string, limit, offset int) ([]Message, error) {
	var msgs []Message
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&msgs).Error
	return msgs, err
}

// GetMessageCount returns the number of messages in a conversation
func (s *Store) GetMessageCount(conversationID string) (int64, error) {
	var count int64
	err := s.db.Model(&Message{}).Where("conversation_id = ?", conversationID).Count(&count).Error
	return count, err
}

// ==================== Memory Methods ====================

// CreateMemory creates a new memory entry
func (s *Store) CreateMemory(mem *Memory) error {
	return s.db.Create(mem).Error
}

// SearchMemories searches memories by content (simple LIKE search, vector search in future)
func (s *Store) SearchMemories(query string, limit int) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("content LIKE ?", "%"+query+"%").
		Order("importance DESC, created_at DESC").
		Limit(limit).
		Find(&memories).Error
	return memories, err
}

// GetRecentMemories retrieves recent memories
func (s *Store) GetRecentMemories(limit int) ([]Memory, error) {
	var memories []Memory
	err := s.db.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

// ==================== Task Methods ====================

// CreateTask creates a new background task
func (s *Store) CreateTask(task *Task) error {
	return s.db.Create(task).Error
}

// GetPendingTasks retrieves pending tasks
func (s *Store) GetPendingTasks(limit int) ([]Task, error) {
	var tasks []Task
	err := s.db.Where("status = ?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// UpdateTask updates a task
func (s *Store) UpdateTask(task *Task) error {
	return s.db.Save(task).Error
}

// ==================== Session Methods (BadgerDB) ====================

// SetSession stores session data in BadgerDB
func (s *Store) SetSession(key string, value []byte, ttl time.Duration) error {
	return s.badger.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte("session:"+key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

// GetSession retrieves session data from BadgerDB
func (s *Store) GetSession(key string) ([]byte, error) {
	var val []byte
	err := s.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("session:" + key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			val = append([]byte{}, v...)
			return nil
		})
	})
	return val, err
}

// DeleteSession removes session data
func (s *Store) DeleteSession(key string) error {
	return s.badger.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("session:" + key))
	})
}

// ==================== Queue Methods (BadgerDB) ====================

// Enqueue adds a job to the queue
func (s *Store) Enqueue(queue string, job []byte) error {
	return s.badger.Update(func(txn *badger.Txn) error {
		// Use timestamp as key for FIFO
		key := fmt.Sprintf("queue:%s:%d", queue, time.Now().UnixNano())
		return txn.Set([]byte(key), job)
	})
}

// Dequeue retrieves and removes a job from the queue
func (s *Store) Dequeue(queue string) ([]byte, error) {
	var job []byte
	prefix := []byte("queue:" + queue + ":")

	err := s.badger.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		it.Seek(prefix)
		if !it.ValidForPrefix(prefix) {
			return fmt.Errorf("queue empty")
		}

		item := it.Item()
		key := item.Key()

		if err := item.Value(func(v []byte) error {
			job = append([]byte{}, v...)
			return nil
		}); err != nil {
			return err
		}

		return txn.Delete(key)
	})

	return job, err
}

// ==================== KV Methods (BadgerDB) ====================

// SetKV stores a key-value pair
func (s *Store) SetKV(key string, value []byte) error {
	return s.badger.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("kv:"+key), value)
	})
}

// GetKV retrieves a value by key
func (s *Store) GetKV(key string) ([]byte, error) {
	var val []byte
	err := s.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("kv:" + key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			val = append([]byte{}, v...)
			return nil
		})
	})
	return val, err
}

// ==================== Context Methods ====================

// WithContext returns a new context with store attached
func (s *Store) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "store", s)
}

// FromContext retrieves store from context
func FromContext(ctx context.Context) (*Store, bool) {
	s, ok := ctx.Value("store").(*Store)
	return s, ok
}

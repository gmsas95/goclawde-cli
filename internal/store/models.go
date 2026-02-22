package store

import (
	"crypto/rand"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// User represents a user (single user in self-hosted mode)
type User struct {
	ID          string          `gorm:"primaryKey" json:"id"`
	DisplayName string          `json:"display_name"`
	Preferences json.RawMessage `json:"preferences"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	TokensUsed   int64     `json:"tokens_used"`
	MessageCount int       `json:"message_count"`
	IsArchived   bool      `json:"is_archived"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationships
	Messages []Message `json:"messages,omitempty" gorm:"foreignKey:ConversationID"`
}

// Message represents a chat message
type Message struct {
	ID             string          `gorm:"primaryKey" json:"id"`
	ConversationID string          `gorm:"index:idx_conv_created" json:"conversation_id"`
	Role           string          `json:"role"` // user, assistant, system, tool
	Content        string          `json:"content"`
	Tokens         int             `json:"tokens"`
	ToolCalls      json.RawMessage `json:"tool_calls,omitempty" gorm:"type:text"`
	ToolResults    json.RawMessage `json:"tool_results,omitempty" gorm:"type:text"`
	ToolCallID     string          `json:"tool_call_id,omitempty"` // For tool role messages
	LatencyMs      int             `json:"latency_ms"`
	CreatedAt      time.Time       `gorm:"index:idx_conv_created" json:"created_at"`
}

// Memory represents a stored fact or preference
type Memory struct {
	ID           string     `gorm:"primaryKey" json:"id"`
	Type         string     `json:"type"` // fact, preference, task, person
	Content      string     `json:"content"`
	Embedding    []byte     `json:"-" gorm:"type:blob"` // Vector embedding for semantic search
	Importance   int        `json:"importance"`         // 1-10
	AccessCount  int        `json:"access_count"`
	LastAccessed *time.Time `json:"last_accessed"`
	Source       string     `json:"source"` // conversation_id or import
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// File represents an uploaded file
type File struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	Filename       string    `json:"filename"`
	MimeType       string    `json:"mime_type"`
	SizeBytes      int64     `json:"size_bytes"`
	StoragePath    string    `json:"storage_path"`
	ConversationID *string   `json:"conversation_id,omitempty"`
	SourceChatID   *int64    `json:"source_chat_id,omitempty"` // Which chat uploaded this file
	ProcessedText  string    `json:"processed_text,omitempty" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
}

// Task represents a background task
type Task struct {
	ID          string          `gorm:"primaryKey" json:"id"`
	Type        string          `json:"type"`
	Status      string          `json:"status"` // pending, running, completed, failed
	Title       string          `json:"title"`
	Prompt      string          `json:"prompt" gorm:"type:text"`
	Result      json.RawMessage `json:"result,omitempty" gorm:"type:text"`
	Error       string          `json:"error,omitempty"`
	StartedAt   *time.Time      `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ScheduledJob represents a recurring job
type ScheduledJob struct {
	ID             string     `gorm:"primaryKey" json:"id"`
	Name           string     `json:"name"`
	CronExpression string     `json:"cron_expression"`
	Prompt         string     `json:"prompt" gorm:"type:text"`
	IsActive       bool       `json:"is_active"`
	LastRunAt      *time.Time `json:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at"`
	RunCount       int        `json:"run_count"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ChatMapping stores the mapping between chat IDs and conversation IDs for persistence across restarts
type ChatMapping struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	ChatID         int64     `gorm:"uniqueIndex:idx_chat_mapping" json:"chat_id"`
	ChatType       string    `json:"chat_type"` // telegram, discord, etc.
	ConversationID string    `gorm:"index" json:"conversation_id"`
	IsActive       bool      `json:"is_active"` // Whether this is the currently active conversation
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BeforeCreate hook for ChatMapping
func (cm *ChatMapping) BeforeCreate(tx *gorm.DB) error {
	if cm.ID == "" {
		cm.ID = generateID("chatmap")
	}
	if cm.ChatType == "" {
		cm.ChatType = "telegram"
	}
	return nil
}

// Config stores key-value configuration
type Config struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the table name for Config
func (Config) TableName() string {
	return "config"
}

// BeforeCreate hook for Conversation
func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = generateID("conv")
	}
	if c.Model == "" {
		c.Model = "kimi-k2.5"
	}
	return nil
}

// BeforeCreate hook for Message
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = generateID("msg")
	}
	return nil
}

// BeforeCreate hook for Memory
func (m *Memory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = generateID("mem")
	}
	if m.Type == "" {
		m.Type = "fact"
	}
	if m.Importance == 0 {
		m.Importance = 5
	}
	return nil
}

// BeforeCreate hook for Task
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = generateID("task")
	}
	if t.Status == "" {
		t.Status = "pending"
	}
	return nil
}

// BeforeCreate hook for File
func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = generateID("file")
	}
	return nil
}

// BeforeCreate hook for ScheduledJob
func (s *ScheduledJob) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = generateID("job")
	}
	return nil
}

// generateID creates a unique ID with nanosecond precision
func generateID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a cryptographically secure random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

// ToJSON converts struct to JSON bytes
func ToJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// FromJSON parses JSON bytes into struct
func FromJSON(data json.RawMessage, v interface{}) error {
	return json.Unmarshal(data, v)
}

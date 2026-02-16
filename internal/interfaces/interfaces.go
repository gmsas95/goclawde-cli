package interfaces

import "context"

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

type Skill interface {
	Name() string
	Version() string
	Description() string
	Tools() []Tool
	Execute(ctx context.Context, tool string, params map[string]any) (any, error)
}

type Channel interface {
	Start() error
	Stop() error
	SendMessage(ctx context.Context, chatID, message string) error
}

type MemoryStore interface {
	CreateMemory(memory *Memory) error
	GetMemory(id string) (*Memory, error)
	ListMemories(limit, offset int) ([]*Memory, error)
	DeleteMemory(id string) error
}

type Memory struct {
	ID         string
	Content    string
	Type       string
	Importance int
	CreatedAt  int64
}

type ConversationStore interface {
	CreateConversation(conv *Conversation) error
	GetConversation(id string) (*Conversation, error)
	ListConversations(limit, offset int) ([]*Conversation, error)
	DeleteConversation(id string) error
}

type Conversation struct {
	ID        string
	Title     string
	Model     string
	CreatedAt int64
}

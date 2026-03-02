// Package storev2 provides the new conversation storage layer
// built on top of the content block architecture.
package storev2

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/types"
	"github.com/google/uuid"
)

// ConversationStore provides storage for conversations using content blocks
type ConversationStore struct {
	db *sql.DB
}

// NewConversationStore creates a new conversation store
func NewConversationStore(db *sql.DB) *ConversationStore {
	return &ConversationStore{db: db}
}

// MessageRecord represents a message in the database
type MessageRecord struct {
	ID             string
	ConversationID string
	Role           string
	Content        []byte // JSON array of content blocks
	Metadata       []byte // JSON metadata
	CreatedAt      time.Time
}

// CreateConversation creates a new conversation
func (s *ConversationStore) CreateConversation(ctx context.Context, userID string) (*types.Conversation, error) {
	conv := &types.Conversation{
		ID:        uuid.New().String(),
		UserID:    userID,
		Messages:  []types.Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO conversations_v2 (id, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4)`,
		conv.ID, conv.UserID, conv.CreatedAt, conv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *ConversationStore) GetConversation(ctx context.Context, id string) (*types.Conversation, error) {
	var conv types.Conversation
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, created_at, updated_at FROM conversations_v2 WHERE id = $1`,
		id,
	).Scan(&conv.ID, &conv.UserID, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	conv.CreatedAt = createdAt
	conv.UpdatedAt = updatedAt

	// Load messages
	messages, err := s.GetMessages(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}
	conv.Messages = messages

	return &conv, nil
}

// GetMessages retrieves all messages for a conversation
func (s *ConversationStore) GetMessages(ctx context.Context, conversationID string) ([]types.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, role, content, metadata, created_at FROM messages_v2 
		 WHERE conversation_id = $1 ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var msg types.Message
		var contentJSON, metadataJSON []byte
		var createdAt time.Time

		err := rows.Scan(&msg.ID, &msg.Role, &contentJSON, &metadataJSON, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Deserialize content blocks
		var contentArray []json.RawMessage
		if err := json.Unmarshal(contentJSON, &contentArray); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content: %w", err)
		}

		for _, blockJSON := range contentArray {
			block, err := types.UnmarshalContentBlock(blockJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal content block: %w", err)
			}
			msg.Content = append(msg.Content, block)
		}

		// Deserialize metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		msg.Timestamp = createdAt
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// SaveMessage saves a message to the database
func (s *ConversationStore) SaveMessage(ctx context.Context, conversationID string, msg *types.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Serialize content blocks
	contentJSON, err := json.Marshal(msg.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	// Serialize metadata
	metadataJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO messages_v2 (id, conversation_id, role, content, metadata, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE SET 
		   role = EXCLUDED.role,
		   content = EXCLUDED.content,
		   metadata = EXCLUDED.metadata`,
		msg.ID, conversationID, msg.Role, contentJSON, metadataJSON, msg.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Update conversation timestamp
	_, err = s.db.ExecContext(ctx,
		`UPDATE conversations_v2 SET updated_at = $1 WHERE id = $2`,
		time.Now(), conversationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	return nil
}

// DeleteMessage deletes a message by ID
func (s *ConversationStore) DeleteMessage(ctx context.Context, messageID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM messages_v2 WHERE id = $1`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}

// GetRecentMessages retrieves the most recent n messages for a conversation
func (s *ConversationStore) GetRecentMessages(ctx context.Context, conversationID string, limit int) ([]types.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, role, content, metadata, created_at FROM messages_v2 
		 WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT $2`,
		conversationID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var msg types.Message
		var contentJSON, metadataJSON []byte
		var createdAt time.Time

		err := rows.Scan(&msg.ID, &msg.Role, &contentJSON, &metadataJSON, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Deserialize content blocks
		var contentArray []json.RawMessage
		if err := json.Unmarshal(contentJSON, &contentArray); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content: %w", err)
		}

		for _, blockJSON := range contentArray {
			block, err := types.UnmarshalContentBlock(blockJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal content block: %w", err)
			}
			msg.Content = append(msg.Content, block)
		}

		// Deserialize metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		msg.Timestamp = createdAt
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// DeleteConversation deletes a conversation and all its messages
func (s *ConversationStore) DeleteConversation(ctx context.Context, id string) error {
	// Delete messages first
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM messages_v2 WHERE conversation_id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete conversation
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM conversations_v2 WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

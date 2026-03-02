// Package migration provides data migration from v1 to v2 schema
package migration

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/types"
	"github.com/google/uuid"
)

// Migrator handles v1 to v2 data migration
type Migrator struct {
	db *sql.DB
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// MigrateConversation migrates a single conversation from v1 to v2
func (m *Migrator) MigrateConversation(oldConvID string) (string, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate new conversation ID
	newConvID := uuid.New().String()

	// Get old conversation
	var userID string
	var createdAt, updatedAt time.Time
	err = tx.QueryRow(
		"SELECT user_id, created_at, updated_at FROM conversations WHERE id = ?",
		oldConvID,
	).Scan(&userID, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("conversation not found: %s", oldConvID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get old conversation: %w", err)
	}

	// Insert into v2
	_, err = tx.Exec(
		"INSERT INTO conversations_v2 (id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?)",
		newConvID, userID, createdAt, updatedAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert conversation: %w", err)
	}

	// Migrate messages
	rows, err := tx.Query(
		"SELECT id, role, content, tool_calls, tool_call_id, tokens, reasoning_content, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC",
		oldConvID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msgID, role, content, toolCallID string
		var toolCallsJSON []byte
		var tokens int
		var reasoningContent string
		var msgCreatedAt time.Time

		err := rows.Scan(&msgID, &role, &content, &toolCallsJSON, &toolCallID, &tokens, &reasoningContent, &msgCreatedAt)
		if err != nil {
			return "", fmt.Errorf("failed to scan message: %w", err)
		}

		// Build content blocks
		var contentBlocks []map[string]interface{}

		// Add reasoning content first (if present)
		if reasoningContent != "" {
			contentBlocks = append(contentBlocks, map[string]interface{}{
				"type":     "thinking",
				"thinking": reasoningContent,
			})
		}

		// Add text content (if present)
		if content != "" {
			contentBlocks = append(contentBlocks, map[string]interface{}{
				"type": "text",
				"text": content,
			})
		}

		// Add tool calls (if present)
		if len(toolCallsJSON) > 0 {
			var toolCalls []map[string]interface{}
			if err := json.Unmarshal(toolCallsJSON, &toolCalls); err == nil {
				for _, tc := range toolCalls {
					contentBlocks = append(contentBlocks, map[string]interface{}{
						"type":      "tool_call",
						"id":        tc["id"],
						"name":      tc["name"],
						"arguments": tc["arguments"],
					})
				}
			}
		}

		// Build metadata
		metadata := map[string]interface{}{
			"input_tokens":  tokens,
			"output_tokens": 0,
		}

		// Serialize to JSON
		contentJSON, _ := json.Marshal(contentBlocks)
		metadataJSON, _ := json.Marshal(metadata)

		// Insert into v2
		newMsgID := uuid.New().String()
		_, err = tx.Exec(
			"INSERT INTO messages_v2 (id, conversation_id, role, content, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			newMsgID, newConvID, role, string(contentJSON), string(metadataJSON), msgCreatedAt,
		)
		if err != nil {
			return "", fmt.Errorf("failed to insert message: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating messages: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	return newConvID, nil
}

// MigrateAllConversations migrates all conversations from v1 to v2
func (m *Migrator) MigrateAllConversations() (int, error) {
	rows, err := m.db.Query("SELECT id FROM conversations")
	if err != nil {
		return 0, fmt.Errorf("failed to get conversations: %w", err)
	}
	defer rows.Close()

	migrated := 0
	var errors []error

	for rows.Next() {
		var convID string
		if err := rows.Scan(&convID); err != nil {
			errors = append(errors, err)
			continue
		}

		_, err := m.MigrateConversation(convID)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to migrate %s: %w", convID, err))
			continue
		}

		migrated++
	}

	if len(errors) > 0 {
		return migrated, fmt.Errorf("migration completed with %d errors", len(errors))
	}

	return migrated, nil
}

// VerifyMigration checks that v2 data is correct
func (m *Migrator) VerifyMigration(convID string) error {
	// Check conversation exists
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM conversations_v2 WHERE id = ?", convID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify conversation: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("conversation not found in v2: %s", convID)
	}

	// Check messages exist and are valid JSON
	rows, err := m.db.Query("SELECT id, content, metadata FROM messages_v2 WHERE conversation_id = ?", convID)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msgID, contentJSON, metadataJSON string
		if err := rows.Scan(&msgID, &contentJSON, &metadataJSON); err != nil {
			return fmt.Errorf("failed to scan message %s: %w", msgID, err)
		}

		// Verify content is valid JSON
		var content []types.ContentBlock
		if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
			return fmt.Errorf("invalid content JSON for message %s: %w", msgID, err)
		}

		// Verify metadata is valid JSON
		var metadata types.MessageMetadata
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON for message %s: %w", msgID, err)
		}
	}

	return nil
}

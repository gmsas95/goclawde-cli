package compaction

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gmsas95/myrai-cli/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlidingWindowCompactor(t *testing.T) {
	compactor := NewSlidingWindowCompactor(5)

	t.Run("returns all messages when under window size", func(t *testing.T) {
		messages := []types.Message{
			{Role: "system", Content: []types.ContentBlock{types.TextBlock{Text: "Sys"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "1"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "2"}}},
		}

		result, err := compactor.Compact(context.Background(), messages, 1000)
		require.NoError(t, err)

		assert.Len(t, result, 3)
	})

	t.Run("keeps system message and recent messages", func(t *testing.T) {
		messages := []types.Message{
			{Role: "system", Content: []types.ContentBlock{types.TextBlock{Text: "Sys"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "1"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "2"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "3"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "4"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "5"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "6"}}},
		}

		result, err := compactor.Compact(context.Background(), messages, 1000)
		require.NoError(t, err)

		// Should have system + 4 recent (window size 5 - 1 for system)
		assert.Len(t, result, 5)
		assert.Equal(t, "system", result[0].Role)
		assert.Equal(t, "6", result[4].GetTextContent())
	})

	t.Run("handles messages without system", func(t *testing.T) {
		messages := make([]types.Message, 10)
		for i := range messages {
			messages[i] = types.Message{
				Role:    "user",
				Content: []types.ContentBlock{types.TextBlock{Text: string(rune('0' + i))}},
			}
		}

		result, err := compactor.Compact(context.Background(), messages, 1000)
		require.NoError(t, err)

		// Should keep last 5
		assert.Len(t, result, 5)
	})
}

func TestSimpleSummarizer(t *testing.T) {
	summarizer := &SimpleSummarizer{}

	t.Run("summarizes basic conversation", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hi"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "How are you?"}}},
		}

		summary, err := summarizer.Summarize(context.Background(), messages)
		require.NoError(t, err)

		assert.Contains(t, summary, "2 user messages")
		assert.Contains(t, summary, "1 assistant responses")
	})

	t.Run("counts tool calls", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Read file"}}},
			{
				Role: "assistant",
				Content: []types.ContentBlock{
					types.ToolCallBlock{ID: "1", Name: "read"},
					types.ToolCallBlock{ID: "2", Name: "write"},
				},
			},
		}

		summary, err := summarizer.Summarize(context.Background(), messages)
		require.NoError(t, err)

		assert.Contains(t, summary, "2 tool calls")
	})
}

func TestSmartCompactor(t *testing.T) {
	summarizer := &SimpleSummarizer{}
	compactor := NewSmartCompactor(summarizer, 1000, 3, 4)

	t.Run("returns messages when under limit", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hi"}}},
		}

		result, err := compactor.Compact(context.Background(), messages, 1000)
		require.NoError(t, err)

		assert.Len(t, result, 2)
	})

	t.Run("compacts when over limit", func(t *testing.T) {
		// Create messages with longer content to ensure they exceed token limit
		messages := []types.Message{
			{Role: "system", Content: []types.ContentBlock{types.TextBlock{Text: "System prompt with many words to increase token count"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "This is a long message with many words to ensure it consumes tokens"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "This is another long response with sufficient words to count"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Yet another lengthy message to add to the token count"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "And another response with enough words to matter for tokens"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "More long text to ensure we exceed the token limit threshold"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Final response with many words to complete the conversation"}}},
		}

		result, err := compactor.Compact(context.Background(), messages, 50)
		require.NoError(t, err)

		// Should have: system + summary + recent turns (less than original 7)
		assert.LessOrEqual(t, len(result), len(messages))
		assert.Equal(t, "system", result[0].Role)
	})

	t.Run("returns error when below minimum", func(t *testing.T) {
		// Create many messages to exceed token limit but keep below minMessages
		// minMessages is 4, so we need more than 4 messages to trigger compaction
		// but the logic should return error if we try to compact below minMessages
		messages := []types.Message{
			{Role: "system", Content: []types.ContentBlock{types.TextBlock{Text: "Sys"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "1"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "2"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "3"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "4"}}},
		}

		// Use a very low minMessages to trigger the error
		lowMinCompactor := NewSmartCompactor(summarizer, 1000, 3, 10)
		_, err := lowMinCompactor.Compact(context.Background(), messages, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "minimum message count")
	})
}

func TestContextManager(t *testing.T) {
	compactor := NewSlidingWindowCompactor(5)
	manager := NewContextManager(compactor, 1000, 10)

	t.Run("prepares context without compaction when under limits", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hi"}}},
		}

		result, err := manager.PrepareContext(context.Background(), messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
	})

	t.Run("compacts when over token limit", func(t *testing.T) {
		// Create many messages with lots of text
		messages := make([]types.Message, 20)
		for i := range messages {
			messages[i] = types.Message{
				Role:    "user",
				Content: []types.ContentBlock{types.TextBlock{Text: "This is a very long message with many words to increase token count significantly beyond the limit"}},
			}
		}

		result, err := manager.PrepareContext(context.Background(), messages)
		require.NoError(t, err)

		// Should be compacted
		assert.Less(t, len(result), len(messages))
	})

	t.Run("detects when compaction is needed", func(t *testing.T) {
		// Many short messages
		messages := make([]types.Message, 15)
		for i := range messages {
			messages[i] = types.Message{
				Role:    "user",
				Content: []types.ContentBlock{types.TextBlock{Text: "Hi"}},
			}
		}

		assert.True(t, manager.ShouldCompact(messages))
	})

	t.Run("detects when compaction is not needed", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		}

		assert.False(t, manager.ShouldCompact(messages))
	})
}

func TestEstimateMessageTokens(t *testing.T) {
	tests := []struct {
		name     string
		message  types.Message
		expected int
	}{
		{
			name: "text only",
			message: types.Message{
				Content: []types.ContentBlock{
					types.TextBlock{Text: "Hello World"}, // ~11 chars / 4 = 3 tokens
				},
			},
			expected: 7, // 3 + 4 overhead
		},
		{
			name: "tool call",
			message: types.Message{
				Content: []types.ContentBlock{
					types.ToolCallBlock{
						ID:        "call_1",
						Name:      "read_file",
						Arguments: json.RawMessage(`{"path": "/test"}`),
					},
				},
			},
			expected: 21, // 10 + 2 + 5 + 4 overhead
		},
		{
			name: "mixed content",
			message: types.Message{
				Content: []types.ContentBlock{
					types.TextBlock{Text: "Hello"},
					types.ToolCallBlock{ID: "1", Name: "read"},
				},
			},
			expected: 18, // 2 + 10 + 2 + 4 overhead
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := estimateMessageTokens(tt.message)
			// Allow some variance due to rounding
			assert.InDelta(t, tt.expected, tokens, 2)
		})
	}
}

func BenchmarkEstimateMessageTokens(b *testing.B) {
	msg := types.Message{
		Content: []types.ContentBlock{
			types.TextBlock{Text: "Hello World this is a test message"},
			types.ToolCallBlock{ID: "1", Name: "read_file", Arguments: json.RawMessage(`{"path": "/test"}`)},
			types.TextBlock{Text: "More content here"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = estimateMessageTokens(msg)
	}
}

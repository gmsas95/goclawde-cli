package pipeline

import (
	"testing"

	"github.com/gmsas95/myrai-cli/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyAssistantFilter(t *testing.T) {
	filter := &EmptyAssistantFilter{}

	t.Run("removes empty assistant messages", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{Role: "assistant", Content: []types.ContentBlock{}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Bye"}}},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
		assert.Equal(t, "user", result[0].Role)
		assert.Equal(t, "user", result[1].Role)
	})

	t.Run("keeps assistant with content", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hi there"}}},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
		assert.Equal(t, "assistant", result[1].Role)
	})

	t.Run("keeps assistant with tool calls", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Read file"}}},
			{
				Role: "assistant",
				Content: []types.ContentBlock{
					types.ToolCallBlock{ID: "1", Name: "read"},
				},
			},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
	})

	t.Run("keeps error messages even if empty", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{
				Role:     "assistant",
				Content:  []types.ContentBlock{},
				Metadata: types.MessageMetadata{StopReason: "error"},
			},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
	})

	t.Run("handles empty input", func(t *testing.T) {
		result, err := filter.Sanitize([]types.Message{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("preserves non-assistant messages", func(t *testing.T) {
		messages := []types.Message{
			{Role: "system", Content: []types.ContentBlock{}},
			{Role: "user", Content: []types.ContentBlock{}},
			{Role: "tool", Content: []types.ContentBlock{}},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 3)
	})
}

func TestConsecutiveAssistantMerger(t *testing.T) {
	merger := &ConsecutiveAssistantMerger{}

	t.Run("merges consecutive assistant messages", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hi"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: " there"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Bye"}}},
		}

		result, err := merger.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 3)
		assert.Equal(t, "assistant", result[1].Role)
		assert.Len(t, result[1].Content, 2)
	})

	t.Run("does not merge non-consecutive", func(t *testing.T) {
		messages := []types.Message{
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "A"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "B"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "C"}}},
		}

		result, err := merger.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 3)
	})

	t.Run("handles empty input", func(t *testing.T) {
		result, err := merger.Sanitize([]types.Message{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("handles single message", func(t *testing.T) {
		messages := []types.Message{
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		}

		result, err := merger.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 1)
	})
}

func TestConsecutiveUserMerger(t *testing.T) {
	merger := &ConsecutiveUserMerger{}

	t.Run("merges consecutive user messages", func(t *testing.T) {
		messages := []types.Message{
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hi"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "World"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Bye"}}},
		}

		result, err := merger.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 3)
		assert.Equal(t, "user", result[1].Role)
		assert.Len(t, result[1].Content, 2)
	})
}

func TestEmptyTextBlockFilter(t *testing.T) {
	filter := &EmptyTextBlockFilter{}

	t.Run("removes empty text blocks from assistant", func(t *testing.T) {
		messages := []types.Message{
			{
				Role: "assistant",
				Content: []types.ContentBlock{
					types.TextBlock{Text: ""},
					types.TextBlock{Text: "Hello"},
					types.TextBlock{Text: "   "},
				},
			},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result[0].Content, 1)
		assert.Equal(t, "Hello", result[0].Content[0].(types.TextBlock).Text)
	})

	t.Run("keeps non-text blocks", func(t *testing.T) {
		messages := []types.Message{
			{
				Role: "assistant",
				Content: []types.ContentBlock{
					types.ToolCallBlock{ID: "1", Name: "read"},
					types.TextBlock{Text: ""},
				},
			},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result[0].Content, 2)
	})

	t.Run("does not affect non-assistant roles", func(t *testing.T) {
		messages := []types.Message{
			{
				Role: "user",
				Content: []types.ContentBlock{
					types.TextBlock{Text: ""},
					types.TextBlock{Text: "Hello"},
				},
			},
		}

		result, err := filter.Sanitize(messages)
		require.NoError(t, err)

		// User messages should not be filtered
		assert.Len(t, result[0].Content, 2)
	})
}

func TestSystemMessageNormalizer(t *testing.T) {
	normalizer := &SystemMessageNormalizer{}

	t.Run("merges multiple system messages", func(t *testing.T) {
		messages := []types.Message{
			{Role: "system", Content: []types.ContentBlock{types.TextBlock{Text: "A"}}},
			{Role: "system", Content: []types.ContentBlock{types.TextBlock{Text: "B"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		}

		result, err := normalizer.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
		assert.Equal(t, "system", result[0].Role)
		assert.Len(t, result[0].Content, 2)
	})

	t.Run("keeps single system message", func(t *testing.T) {
		messages := []types.Message{
			{Role: "system", Content: []types.ContentBlock{types.TextBlock{Text: "A"}}},
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		}

		result, err := normalizer.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
	})

	t.Run("handles no system messages", func(t *testing.T) {
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		}

		result, err := normalizer.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 1)
	})
}

func TestToolResultValidator(t *testing.T) {
	validator := &ToolResultValidator{}

	t.Run("keeps tool results with matching calls", func(t *testing.T) {
		messages := []types.Message{
			{
				Role: "assistant",
				Content: []types.ContentBlock{
					types.ToolCallBlock{ID: "call_1", Name: "read"},
				},
			},
			{
				Role: "tool",
				Content: []types.ContentBlock{
					types.ToolResultBlock{ToolCallID: "call_1", Content: []types.ContentBlock{types.TextBlock{Text: "Result"}}},
				},
			},
		}

		result, err := validator.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 2)
	})

	t.Run("removes orphaned tool results", func(t *testing.T) {
		messages := []types.Message{
			{
				Role: "assistant",
				Content: []types.ContentBlock{
					types.ToolCallBlock{ID: "call_1", Name: "read"},
				},
			},
			{
				Role: "tool",
				Content: []types.ContentBlock{
					types.ToolResultBlock{ToolCallID: "orphan", Content: []types.ContentBlock{types.TextBlock{Text: "Result"}}},
				},
			},
		}

		result, err := validator.Sanitize(messages)
		require.NoError(t, err)

		assert.Len(t, result, 1)
	})
}

func TestPipeline(t *testing.T) {
	t.Run("processes sanitizers in order", func(t *testing.T) {
		pipeline := NewPipeline("test").
			Add(&EmptyAssistantFilter{}).
			Add(&ConsecutiveAssistantMerger{})

		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "A"}}},
			{Role: "assistant", Content: []types.ContentBlock{}}, // Empty, will be filtered
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "B"}}},
			{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "C"}}},
		}

		result, err := pipeline.Process(messages)
		require.NoError(t, err)

		// Should have: user, merged assistant (B+C)
		assert.Len(t, result, 2)
		assert.Len(t, result[1].Content, 2)
	})

	t.Run("handles empty pipeline", func(t *testing.T) {
		pipeline := NewPipeline("empty")
		messages := []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		}

		result, err := pipeline.Process(messages)
		require.NoError(t, err)

		assert.Equal(t, messages, result)
	})

	t.Run("provider pipelines are configured correctly", func(t *testing.T) {
		// Test that provider pipelines don't panic
		pipelines := []*Pipeline{
			OpenAIPipeline(),
			AnthropicPipeline(),
			GeminiPipeline(),
			UniversalPipeline(),
		}

		for _, p := range pipelines {
			assert.NotNil(t, p)
			assert.NotEmpty(t, p.name)
		}
	})
}

func BenchmarkEmptyAssistantFilter(b *testing.B) {
	filter := &EmptyAssistantFilter{}
	messages := []types.Message{
		{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		{Role: "assistant", Content: []types.ContentBlock{types.TextBlock{Text: "Hi"}}},
		{Role: "assistant", Content: []types.ContentBlock{}},
		{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Bye"}}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = filter.Sanitize(messages)
	}
}

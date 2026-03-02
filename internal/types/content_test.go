package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextBlock(t *testing.T) {
	t.Run("BlockType returns text", func(t *testing.T) {
		block := TextBlock{Text: "Hello"}
		assert.Equal(t, "text", block.BlockType())
	})

	t.Run("IsEmpty returns true for empty text", func(t *testing.T) {
		block := TextBlock{Text: ""}
		assert.True(t, block.IsEmpty())
	})

	t.Run("IsEmpty returns true for whitespace only", func(t *testing.T) {
		block := TextBlock{Text: "   \n\t  "}
		assert.True(t, block.IsEmpty())
	})

	t.Run("IsEmpty returns false for non-empty text", func(t *testing.T) {
		block := TextBlock{Text: "Hello World"}
		assert.False(t, block.IsEmpty())
	})

	t.Run("JSON marshaling includes type field", func(t *testing.T) {
		block := TextBlock{Text: "Hello"}
		data, err := json.Marshal(block)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "text", result["type"])
		assert.Equal(t, "Hello", result["text"])
	})
}

func TestToolCallBlock(t *testing.T) {
	t.Run("BlockType returns tool_call", func(t *testing.T) {
		block := ToolCallBlock{
			ID:   "call_123",
			Name: "read_file",
		}
		assert.Equal(t, "tool_call", block.BlockType())
	})

	t.Run("IsEmpty always returns false", func(t *testing.T) {
		block := ToolCallBlock{}
		assert.False(t, block.IsEmpty())
	})

	t.Run("JSON marshaling includes all fields", func(t *testing.T) {
		args := json.RawMessage(`{"path": "/test.txt"}`)
		block := ToolCallBlock{
			ID:        "call_123",
			Name:      "read_file",
			Arguments: args,
		}
		data, err := json.Marshal(block)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "tool_call", result["type"])
		assert.Equal(t, "call_123", result["id"])
		assert.Equal(t, "read_file", result["name"])
	})
}

func TestToolResultBlock(t *testing.T) {
	t.Run("BlockType returns tool_result", func(t *testing.T) {
		block := ToolResultBlock{
			ToolCallID: "call_123",
			ToolName:   "read_file",
		}
		assert.Equal(t, "tool_result", block.BlockType())
	})

	t.Run("IsEmpty returns true for empty content", func(t *testing.T) {
		block := ToolResultBlock{
			ToolCallID: "call_123",
			Content:    []ContentBlock{},
		}
		assert.True(t, block.IsEmpty())
	})

	t.Run("IsEmpty returns false with content", func(t *testing.T) {
		block := ToolResultBlock{
			ToolCallID: "call_123",
			Content:    []ContentBlock{TextBlock{Text: "Result"}},
		}
		assert.False(t, block.IsEmpty())
	})
}

func TestThinkingBlock(t *testing.T) {
	t.Run("BlockType returns thinking", func(t *testing.T) {
		block := ThinkingBlock{Thinking: "Analyzing..."}
		assert.Equal(t, "thinking", block.BlockType())
	})

	t.Run("IsEmpty returns true for empty thinking", func(t *testing.T) {
		block := ThinkingBlock{Thinking: ""}
		assert.True(t, block.IsEmpty())
	})

	t.Run("IsEmpty returns false for non-empty thinking", func(t *testing.T) {
		block := ThinkingBlock{Thinking: "Step 1: Parse input"}
		assert.False(t, block.IsEmpty())
	})
}

func TestImageBlock(t *testing.T) {
	t.Run("BlockType returns image", func(t *testing.T) {
		block := ImageBlock{
			Source:    "base64",
			MediaType: "image/png",
			Data:      "iVBORw0KGgo...",
		}
		assert.Equal(t, "image", block.BlockType())
	})

	t.Run("IsEmpty returns true for empty data", func(t *testing.T) {
		block := ImageBlock{Data: ""}
		assert.True(t, block.IsEmpty())
	})

	t.Run("IsEmpty returns false with data", func(t *testing.T) {
		block := ImageBlock{Data: "some-image-data"}
		assert.False(t, block.IsEmpty())
	})
}

func TestMessage(t *testing.T) {
	t.Run("IsEmpty returns true for empty message", func(t *testing.T) {
		msg := &Message{
			Role:    "assistant",
			Content: []ContentBlock{},
		}
		assert.True(t, msg.IsEmpty())
	})

	t.Run("IsEmpty returns true for all empty blocks", func(t *testing.T) {
		msg := &Message{
			Role: "assistant",
			Content: []ContentBlock{
				TextBlock{Text: ""},
				TextBlock{Text: "   "},
			},
		}
		assert.True(t, msg.IsEmpty())
	})

	t.Run("IsEmpty returns false with content", func(t *testing.T) {
		msg := &Message{
			Role: "assistant",
			Content: []ContentBlock{
				TextBlock{Text: "Hello"},
			},
		}
		assert.False(t, msg.IsEmpty())
	})

	t.Run("IsEmpty returns false with tool calls", func(t *testing.T) {
		msg := &Message{
			Role: "assistant",
			Content: []ContentBlock{
				ToolCallBlock{ID: "1", Name: "read"},
			},
		}
		assert.False(t, msg.IsEmpty())
	})

	t.Run("HasToolCalls returns true with tool calls", func(t *testing.T) {
		msg := &Message{
			Role: "assistant",
			Content: []ContentBlock{
				TextBlock{Text: "Let me check"},
				ToolCallBlock{ID: "1", Name: "read"},
			},
		}
		assert.True(t, msg.HasToolCalls())
	})

	t.Run("HasToolCalls returns false without tool calls", func(t *testing.T) {
		msg := &Message{
			Role: "assistant",
			Content: []ContentBlock{
				TextBlock{Text: "Hello"},
			},
		}
		assert.False(t, msg.HasToolCalls())
	})

	t.Run("GetTextContent concatenates text blocks", func(t *testing.T) {
		msg := &Message{
			Role: "assistant",
			Content: []ContentBlock{
				TextBlock{Text: "Hello"},
				TextBlock{Text: "World"},
			},
		}
		assert.Equal(t, "Hello\nWorld", msg.GetTextContent())
	})

	t.Run("GetToolCalls returns all tool calls", func(t *testing.T) {
		msg := &Message{
			Role: "assistant",
			Content: []ContentBlock{
				ToolCallBlock{ID: "1", Name: "read"},
				TextBlock{Text: "and"},
				ToolCallBlock{ID: "2", Name: "write"},
			},
		}
		calls := msg.GetToolCalls()
		assert.Len(t, calls, 2)
		assert.Equal(t, "read", calls[0].Name)
		assert.Equal(t, "write", calls[1].Name)
	})

	t.Run("AddTextBlock appends text", func(t *testing.T) {
		msg := &Message{Role: "assistant"}
		msg.AddTextBlock("Hello")
		msg.AddTextBlock("World")

		assert.Len(t, msg.Content, 2)
		assert.Equal(t, "Hello", msg.Content[0].(TextBlock).Text)
		assert.Equal(t, "World", msg.Content[1].(TextBlock).Text)
	})

	t.Run("AddToolCallBlock appends tool call", func(t *testing.T) {
		msg := &Message{Role: "assistant"}
		msg.AddToolCallBlock("1", "read", json.RawMessage(`{}`))

		assert.Len(t, msg.Content, 1)
		call := msg.Content[0].(ToolCallBlock)
		assert.Equal(t, "1", call.ID)
		assert.Equal(t, "read", call.Name)
	})
}

func TestUnmarshalContentBlock(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected ContentBlock
		wantErr  bool
	}{
		{
			name:     "text block",
			json:     `{"type":"text","text":"Hello"}`,
			expected: TextBlock{Text: "Hello"},
		},
		{
			name:     "tool_call block",
			json:     `{"type":"tool_call","id":"1","name":"read"}`,
			expected: ToolCallBlock{ID: "1", Name: "read"},
		},
		{
			name:     "thinking block",
			json:     `{"type":"thinking","thinking":"Analyzing"}`,
			expected: ThinkingBlock{Thinking: "Analyzing"},
		},
		{
			name:     "image block",
			json:     `{"type":"image","source":"base64","media_type":"image/png","data":"abc"}`,
			expected: ImageBlock{Source: "base64", MediaType: "image/png", Data: "abc"},
		},
		{
			name:    "unknown type",
			json:    `{"type":"unknown"}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			json:    `{"type":}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := UnmarshalContentBlock([]byte(tt.json))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, block)
		})
	}
}

func TestUsage(t *testing.T) {
	t.Run("Total returns sum of input and output", func(t *testing.T) {
		usage := Usage{
			InputTokens:  100,
			OutputTokens: 50,
		}
		assert.Equal(t, 150, usage.Total())
	})
}

func BenchmarkMessageGetTextContent(b *testing.B) {
	msg := &Message{
		Role: "assistant",
		Content: []ContentBlock{
			TextBlock{Text: "Hello World"},
			ToolCallBlock{ID: "1", Name: "read"},
			TextBlock{Text: "How are you?"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = msg.GetTextContent()
	}
}

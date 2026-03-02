package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gmsas95/myrai-cli/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextDeltaEvent(t *testing.T) {
	event := NewTextDeltaEvent("Hello", "World")

	assert.Equal(t, "text_delta", event.EventType())
	assert.Equal(t, "Hello", event.Delta)
	assert.Equal(t, "World", event.Text)
	assert.False(t, event.Timestamp().IsZero())
}

func TestToolCallEvents(t *testing.T) {
	t.Run("ToolCallStartEvent", func(t *testing.T) {
		event := NewToolCallStartEvent("call_1", "read_file")
		assert.Equal(t, "tool_call_start", event.EventType())
		assert.Equal(t, "call_1", event.ID)
		assert.Equal(t, "read_file", event.Name)
	})

	t.Run("ToolCallDeltaEvent", func(t *testing.T) {
		event := NewToolCallDeltaEvent("call_1", `{"pa`, `{"path": "/test`)
		assert.Equal(t, "tool_call_delta", event.EventType())
		assert.Equal(t, "call_1", event.ID)
		assert.Equal(t, `{"pa`, event.Delta)
		assert.Equal(t, `{"path": "/test`, event.Arguments)
	})

	t.Run("ToolCallCompleteEvent", func(t *testing.T) {
		args := json.RawMessage(`{"path": "/test.txt"}`)
		event := NewToolCallCompleteEvent("call_1", "read_file", args)
		assert.Equal(t, "tool_call_complete", event.EventType())
		assert.Equal(t, "call_1", event.ID)
		assert.Equal(t, "read_file", event.Name)
		assert.Equal(t, args, event.Arguments)
	})
}

func TestUsageEvent(t *testing.T) {
	event := NewUsageEvent(100, 50)

	assert.Equal(t, "usage", event.EventType())
	assert.Equal(t, 100, event.InputTokens)
	assert.Equal(t, 50, event.OutputTokens)
	assert.Equal(t, 150, event.Total)
}

func TestStopEvent(t *testing.T) {
	event := NewStopEvent("stop")

	assert.Equal(t, "stop", event.EventType())
	assert.Equal(t, "stop", event.Reason)
}

func TestErrorEvent(t *testing.T) {
	err := assert.AnError
	event := NewErrorEvent(err)

	assert.Equal(t, "error", event.EventType())
	assert.Equal(t, err.Error(), event.Error)
}

func TestProgressEvent(t *testing.T) {
	event := NewProgressEvent("thinking", "Analyzing data", 50.5)

	assert.Equal(t, "progress", event.EventType())
	assert.Equal(t, "thinking", event.Stage)
	assert.Equal(t, "Analyzing data", event.Description)
	assert.Equal(t, 50.5, event.Percent)
}

func TestEventHandlerFunc(t *testing.T) {
	t.Run("OnEvent calls function", func(t *testing.T) {
		called := false
		handler := EventHandlerFunc{
			OnEventFunc: func(event StreamEvent) error {
				called = true
				return nil
			},
		}

		_ = handler.OnEvent(NewTextDeltaEvent("test", "test"))
		assert.True(t, called)
	})

	t.Run("OnError calls function", func(t *testing.T) {
		called := false
		handler := EventHandlerFunc{
			OnErrorFunc: func(err error) {
				called = true
			},
		}

		handler.OnError(assert.AnError)
		assert.True(t, called)
	})

	t.Run("OnComplete calls function", func(t *testing.T) {
		called := false
		handler := EventHandlerFunc{
			OnCompleteFunc: func() {
				called = true
			},
		}

		handler.OnComplete()
		assert.True(t, called)
	})
}

func TestAccumulator(t *testing.T) {
	t.Run("processes text deltas", func(t *testing.T) {
		acc := NewAccumulator()

		err := acc.ProcessEvent(NewTextDeltaEvent("Hello", "Hello"))
		require.NoError(t, err)

		err = acc.ProcessEvent(NewTextDeltaEvent(" World", "Hello World"))
		require.NoError(t, err)

		msg := acc.Finalize()
		assert.Equal(t, "Hello World", msg.GetTextContent())
	})

	t.Run("processes thinking deltas", func(t *testing.T) {
		acc := NewAccumulator()

		err := acc.ProcessEvent(NewThinkingDeltaEvent("Step 1", "Step 1"))
		require.NoError(t, err)

		msg := acc.Finalize()
		assert.Equal(t, "Step 1", msg.GetThinkingContent())
	})

	t.Run("processes tool calls", func(t *testing.T) {
		acc := NewAccumulator()

		args := json.RawMessage(`{"path": "/test.txt"}`)
		err := acc.ProcessEvent(NewToolCallCompleteEvent("call_1", "read_file", args))
		require.NoError(t, err)

		msg := acc.Finalize()
		calls := msg.GetToolCalls()
		assert.Len(t, calls, 1)
		assert.Equal(t, "read_file", calls[0].Name)
	})

	t.Run("processes usage", func(t *testing.T) {
		acc := NewAccumulator()

		err := acc.ProcessEvent(NewUsageEvent(100, 50))
		require.NoError(t, err)

		msg := acc.Finalize()
		assert.Equal(t, 100, msg.Metadata.Usage.InputTokens)
		assert.Equal(t, 50, msg.Metadata.Usage.OutputTokens)
	})

	t.Run("processes stop reason", func(t *testing.T) {
		acc := NewAccumulator()

		err := acc.ProcessEvent(NewStopEvent("tool_calls"))
		require.NoError(t, err)

		msg := acc.Finalize()
		assert.Equal(t, "tool_calls", msg.Metadata.StopReason)
	})

	t.Run("GetMessage returns current state", func(t *testing.T) {
		acc := NewAccumulator()

		err := acc.ProcessEvent(NewTextDeltaEvent("Hello", "Hello"))
		require.NoError(t, err)

		msg := acc.GetMessage()
		assert.Equal(t, "Hello", msg.GetTextContent())
		assert.Equal(t, "assistant", msg.Role)
	})

	t.Run("handles mixed content", func(t *testing.T) {
		acc := NewAccumulator()

		// Text
		err := acc.ProcessEvent(NewTextDeltaEvent("Let me check", "Let me check"))
		require.NoError(t, err)

		// Thinking
		err = acc.ProcessEvent(NewThinkingDeltaEvent("Analyzing", "Analyzing"))
		require.NoError(t, err)

		// Tool call
		args := json.RawMessage(`{}`)
		err = acc.ProcessEvent(NewToolCallCompleteEvent("1", "read", args))
		require.NoError(t, err)

		msg := acc.Finalize()

		assert.Equal(t, "Let me check", msg.GetTextContent())
		assert.Equal(t, "Analyzing", msg.GetThinkingContent())
		assert.Len(t, msg.GetToolCalls(), 1)
	})
}

func TestEventSerialization(t *testing.T) {
	tests := []struct {
		name  string
		event StreamEvent
	}{
		{
			name:  "text_delta",
			event: NewTextDeltaEvent("Hello", "Hello"),
		},
		{
			name:  "thinking_delta",
			event: NewThinkingDeltaEvent("Think", "Think"),
		},
		{
			name:  "tool_call_start",
			event: NewToolCallStartEvent("1", "read"),
		},
		{
			name:  "tool_call_delta",
			event: NewToolCallDeltaEvent("1", "{", "{"),
		},
		{
			name:  "tool_call_complete",
			event: NewToolCallCompleteEvent("1", "read", json.RawMessage(`{}`)),
		},
		{
			name:  "usage",
			event: NewUsageEvent(100, 50),
		},
		{
			name:  "stop",
			event: NewStopEvent("stop"),
		},
		{
			name:  "error",
			event: NewErrorEvent(assert.AnError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := EventToJSON(tt.event)
			require.NoError(t, err)

			// Deserialize
			restored, err := JSONToEvent(data)
			require.NoError(t, err)

			// Verify type matches
			assert.Equal(t, tt.event.EventType(), restored.EventType())
		})
	}
}

func TestIsStreamingError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsStreamingError(tt.err))
		})
	}
}

func TestStreamRequest(t *testing.T) {
	req := StreamRequest{
		Messages: []types.Message{
			{Role: "user", Content: []types.ContentBlock{types.TextBlock{Text: "Hello"}}},
		},
		Tools: []*ToolDefinition{
			{Name: "read", Description: "Read file"},
		},
		Model: "gpt-4",
		Options: map[string]interface{}{
			"temperature": 0.7,
		},
	}

	assert.Len(t, req.Messages, 1)
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "gpt-4", req.Model)
	assert.Equal(t, 0.7, req.Options["temperature"])
}

func BenchmarkAccumulator_ProcessEvent(b *testing.B) {
	acc := NewAccumulator()
	event := NewTextDeltaEvent("Hello", "Hello")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = acc.ProcessEvent(event)
	}
}

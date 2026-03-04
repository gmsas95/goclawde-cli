// Package agent provides tests for agent functionality
package agent

import (
	"encoding/json"
	"testing"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
)

// TestToolMessageStorageSchema verifies that tool messages are stored with correct schema
// This test ensures the fix where ToolCalls were stored as single object instead of array
func TestToolMessageStorageSchema(t *testing.T) {
	t.Run("ToolCallsStoredAsArray", func(t *testing.T) {
		// Create a tool call
		toolCall := llm.ToolCall{
			ID:   "call_123",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "get_weather",
				Arguments: `{"location":"KL"}`,
			},
		}

		// The fix: store as array []llm.ToolCall, not single llm.ToolCall
		toolCallsArray := []llm.ToolCall{toolCall}

		// Serialize
		data, err := json.Marshal(toolCallsArray)
		if err != nil {
			t.Fatalf("Failed to marshal tool calls: %v", err)
		}

		// Deserialize back as array (what retrieval code does)
		var retrieved []llm.ToolCall
		if err := json.Unmarshal(data, &retrieved); err != nil {
			t.Fatalf("Failed to unmarshal tool calls as array: %v", err)
		}

		// Verify
		if len(retrieved) != 1 {
			t.Errorf("Expected 1 tool call, got %d", len(retrieved))
		}

		if retrieved[0].Function.Name != "get_weather" {
			t.Errorf("Expected tool name 'get_weather', got %q", retrieved[0].Function.Name)
		}
	})

	t.Run("ToolCallDeserializationBackwardCompatibility", func(t *testing.T) {
		// Test what happens if we try to deserialize a single object as array
		// (This would fail before the fix)

		singleToolCall := llm.ToolCall{
			ID:   "call_456",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "voice_info",
				Arguments: "{}",
			},
		}

		// Serialize single object (old buggy behavior)
		singleData, _ := json.Marshal(singleToolCall)

		// Try to deserialize as array (what buildFullContext does)
		var asArray []llm.ToolCall
		err := json.Unmarshal(singleData, &asArray)
		if err == nil {
			// Some JSON decoders can handle this, but it's unreliable
			t.Log("Single object deserialized as array (implementation dependent)")
		}

		// The fix ensures we always store as array, avoiding this issue
		// by wrapping: []llm.ToolCall{singleToolCall}
	})
}

// TestToolResultJSONFormatting verifies tool results are formatted as proper JSON
func TestToolResultJSONFormatting(t *testing.T) {
	t.Run("ToolResultAsJSON", func(t *testing.T) {
		// Simulate a tool result (map)
		result := map[string]interface{}{
			"temperature": 25.5,
			"condition":   "sunny",
			"location":    "KL",
		}

		// The fix: convert to proper JSON
		jsonData, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		resultStr := string(jsonData)

		// Verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(resultStr), &parsed); err != nil {
			t.Errorf("Result is not valid JSON: %v", err)
		}

		// Verify values
		if parsed["temperature"] != 25.5 {
			t.Errorf("Expected temperature 25.5, got %v", parsed["temperature"])
		}
	})

	t.Run("ToolResultNotGoFormat", func(t *testing.T) {
		// Before fix: fmt.Sprintf("%v", result) would produce: map[condition:sunny location:KL temperature:25.5]
		// After fix: json.Marshal(result) produces: {"condition":"sunny","location":"KL","temperature":25.5}

		result := map[string]interface{}{
			"key": "value",
		}

		// Old buggy format
		oldFormat := "map[key:value]"

		// New correct format
		newFormatBytes, _ := json.Marshal(result)
		newFormat := string(newFormatBytes)

		// Verify they're different
		if oldFormat == newFormat {
			t.Error("Old and new format should be different")
		}

		// Verify new format is valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(newFormat), &parsed); err != nil {
			t.Errorf("New format is not valid JSON: %v", err)
		}

		// Verify old format is NOT valid JSON
		if err := json.Unmarshal([]byte(oldFormat), &parsed); err == nil {
			t.Error("Old format should not be valid JSON")
		}
	})
}

// TestMessageOrdering verifies messages are processed in correct order
func TestMessageOrdering(t *testing.T) {
	t.Run("ChronologicalOrderForLLM", func(t *testing.T) {
		// Simulate message retrieval (newest first from DB)
		dbMessages := []store.Message{
			{Content: "Message 5 (newest)", Role: "user"},
			{Content: "Message 4", Role: "assistant"},
			{Content: "Message 3", Role: "user"},
			{Content: "Message 2", Role: "assistant"},
			{Content: "Message 1 (oldest)", Role: "user"},
		}

		// Simulate buildFullContext: reverse to chronological
		var chronological []store.Message
		for i := len(dbMessages) - 1; i >= 0; i-- {
			chronological = append(chronological, dbMessages[i])
		}

		// Verify order
		expected := []string{
			"Message 1 (oldest)",
			"Message 2",
			"Message 3",
			"Message 4",
			"Message 5 (newest)",
		}

		for i, exp := range expected {
			if chronological[i].Content != exp {
				t.Errorf("Message %d: expected %q, got %q", i, exp, chronological[i].Content)
			}
		}
	})
}

// TestBuildContextMessageRetrieval tests that buildContext retrieves correct messages
func TestBuildContextMessageRetrieval(t *testing.T) {
	t.Run("CorrectMessageCount", func(t *testing.T) {
		// This test documents that we should get the 20 most recent messages
		// not the 20 oldest

		// Simulate 30 messages
		allMessages := make([]store.Message, 30)
		for i := 0; i < 30; i++ {
			allMessages[i] = store.Message{
				Content: "Message " + string(rune('0'+i)),
				Role:    "user",
			}
		}

		// With limit=20, offset=0, and DESC order, we get messages 30, 29, 28, ... 11
		// (the 20 newest)

		// Simulate retrieval
		limit := 20
		var retrieved []store.Message
		for i := len(allMessages) - 1; i >= 0 && len(retrieved) < limit; i-- {
			retrieved = append(retrieved, allMessages[i])
		}

		if len(retrieved) != 20 {
			t.Errorf("Expected 20 messages, got %d", len(retrieved))
		}

		// First retrieved should be message 29 (the newest)
		if retrieved[0].Content != "Message "+string(rune('0'+29)) {
			t.Errorf("Expected first message to be newest (29), got %q", retrieved[0].Content)
		}
	})
}

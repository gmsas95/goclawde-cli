// Package v2demo demonstrates how to use the v2 architecture components
// This shows the complete flow from message to response.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gmsas95/myrai-cli/internal/compaction"
	"github.com/gmsas95/myrai-cli/internal/conversation"
	"github.com/gmsas95/myrai-cli/internal/pipeline"
	"github.com/gmsas95/myrai-cli/internal/storev2"
	"github.com/gmsas95/myrai-cli/internal/streaming"
	"github.com/gmsas95/myrai-cli/internal/tools"
	"github.com/gmsas95/myrai-cli/internal/types"
)

func main() {
	ctx := context.Background()

	// 1. Initialize store
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := storev2.NewConversationStore(db)

	// 2. Initialize tool registry
	toolRegistry := tools.NewRegistry()
	registerBuiltinTools(toolRegistry)

	// 3. Initialize conversation manager
	// Use smart compaction with LLM summarization
	summarizer := compaction.NewSimpleSummarizer() // or NewLLMSummarizer(client)
	compactor := compaction.NewSmartCompactor(summarizer, 4000, 10, 6)

	convManager := conversation.NewManager(conversation.ManagerOptions{
		Store:       store,
		Pipeline:    pipeline.UniversalPipeline(),
		Compactor:   compactor,
		MaxMessages: 100,
		MaxTokens:   8000,
	})

	// 4. Create or get conversation
	userID := "telegram_user_123"
	conv, err := convManager.CreateConversation(ctx, userID)
	if err != nil {
		log.Fatal("Failed to create conversation:", err)
	}

	fmt.Println("Conversation created:", conv.ID)

	// 5. Add user message
	userMsg, err := convManager.AddUserMessage(ctx, conv.ID, "What's the weather in Tokyo?")
	if err != nil {
		log.Fatal("Failed to add user message:", err)
	}
	fmt.Println("User message added:", userMsg.GetTextContent())

	// 6. Get context for LLM
	context, err := convManager.GetContext(ctx, conv.ID)
	if err != nil {
		log.Fatal("Failed to get context:", err)
	}
	fmt.Printf("Context has %d messages\n", len(context))

	// 7. Stream LLM response
	client := streaming.NewOpenAIStreamingClient(
		os.Getenv("OPENAI_API_KEY"),
		"https://api.openai.com/v1",
		"gpt-4",
	)

	request := streaming.StreamRequest{
		Messages: convertToStreamingMessages(context),
		Tools:    convertToolsToStreaming(toolRegistry.List()),
		Model:    "gpt-4",
	}

	handler := &MessageAccumulatorHandler{
		convManager:    convManager,
		conversationID: conv.ID,
	}

	fmt.Println("Streaming response...")
	if err := client.Stream(ctx, request, handler); err != nil {
		log.Fatal("Streaming failed:", err)
	}

	// 8. Get conversation stats
	stats, err := convManager.GetStats(ctx, conv.ID)
	if err != nil {
		log.Fatal("Failed to get stats:", err)
	}

	fmt.Printf("\nConversation Stats:\n")
	fmt.Printf("- Messages: %d\n", stats.MessageCount)
	fmt.Printf("- Tokens: %d\n", stats.TokenEstimate)
	fmt.Printf("- Duration: %v\n", stats.Duration)
}

// MessageAccumulatorHandler accumulates streaming events into a message
type MessageAccumulatorHandler struct {
	accumulator    *streaming.Accumulator
	convManager    *conversation.Manager
	conversationID string
}

func (h *MessageAccumulatorHandler) OnEvent(event streaming.StreamEvent) error {
	switch e := event.(type) {
	case *streaming.TextDeltaEvent:
		fmt.Print(e.Delta) // Print as we receive
	case *streaming.ToolCallStartEvent:
		fmt.Printf("\n[Tool: %s]\n", e.Name)
	case *streaming.ToolCallCompleteEvent:
		// Execute the tool
		result, err := tools.Execute(context.Background(), e.Name, e.Arguments)
		if err != nil {
			fmt.Printf("[Tool error: %v]\n", err)
			return nil
		}
		fmt.Printf("[Tool result: %s]\n", extractTextFromResult(result))
	case *streaming.UsageEvent:
		fmt.Printf("\n[Tokens: %d input, %d output]\n", e.InputTokens, e.OutputTokens)
	}

	// Accumulate for final message
	if h.accumulator == nil {
		h.accumulator = streaming.NewAccumulator()
	}
	return h.accumulator.ProcessEvent(event)
}

func (h *MessageAccumulatorHandler) OnError(err error) {
	log.Println("Stream error:", err)
}

func (h *MessageAccumulatorHandler) OnComplete() {
	fmt.Println("\n[Stream complete]")

	// Save the accumulated message
	if h.accumulator != nil {
		msg := h.accumulator.Finalize()
		if err := h.convManager.AddAssistantMessage(context.Background(), h.conversationID, msg); err != nil {
			log.Println("Failed to save assistant message:", err)
		}
	}
}

// Helper functions

func initDB() (*sql.DB, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://user:password@localhost/myrai?sslmode=disable"
	}
	return sql.Open("postgres", connStr)
}

func registerBuiltinTools(registry *tools.Registry) {
	// Register built-in tools
	registry.Register(&tools.ToolDefinition{
		Name:        "weather",
		Description: "Get weather information for a location",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
		Handler: func(ctx context.Context, args json.RawMessage) (*tools.ToolResult, error) {
			// Implementation
			return &tools.ToolResult{
				Content: []types.ContentBlock{
					types.TextBlock{Text: "Sunny, 25°C in Tokyo"},
				},
			}, nil
		},
		Source: tools.ToolSourceBuiltin,
	})
}

func convertToStreamingMessages(messages []types.Message) []types.Message {
	// Messages are already in the right format
	return messages
}

func convertToolsToStreaming(toolDefs []*tools.ToolDefinition) []*streaming.ToolDefinition {
	var result []*streaming.ToolDefinition
	for _, td := range toolDefs {
		result = append(result, &streaming.ToolDefinition{
			Name:        td.Name,
			Description: td.Description,
			Parameters:  td.Parameters,
		})
	}
	return result
}

func extractTextFromResult(result *tools.ToolResult) string {
	var text string
	for _, block := range result.Content {
		if tb, ok := block.(types.TextBlock); ok {
			text += tb.Text
		}
	}
	return text
}

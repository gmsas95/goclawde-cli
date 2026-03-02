# Myrai Production Architecture Design

Based on analysis of OpenClaw's proven architecture, this document outlines the production-grade overhaul needed for Myrai.

## Current Architecture Problems

### 1. **String-Based Content Model**
```go
// Current (fragile)
type Message struct {
    Content    string  // Text only
    ToolCalls  []byte  // Raw JSON
    ToolCallID string  // Separate field
}
```

**Issues:**
- Cannot represent multi-modal content (images, audio, files)
- Awkward separation of text and tool calls
- Empty content edge cases not handled cleanly
- No support for reasoning/thinking blocks

### 2. **No Provider Abstraction**
- Single message format for all LLM providers
- OpenAI, Anthropic, Gemini have different requirements
- Hard to add new providers

### 3. **Missing Sanitization Pipeline**
- Messages sent directly to LLM without validation
- No repair of broken conversation flows
- Empty/tool-only messages cause API errors

### 4. **Static Tool System**
- Tools compiled into binary
- Cannot add/remove tools at runtime
- No skill/plugin system

### 5. **No Streaming Architecture**
- Wait for complete response before processing
- Cannot interrupt long operations
- Poor UX for tool-heavy workflows

## Proposed Architecture

### 1. Content Block Model

```go
// types/content.go
package types

// ContentBlock is the unified content representation
type ContentBlock interface {
    BlockType() string
}

// TextBlock represents text content
type TextBlock struct {
    Text string `json:"text"`
}

func (b TextBlock) BlockType() string { return "text" }

// ToolCallBlock represents a tool invocation
type ToolCallBlock struct {
    ID        string          `json:"id"`
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}

func (b ToolCallBlock) BlockType() string { return "tool_call" }

// ToolResultBlock represents tool execution result
type ToolResultBlock struct {
    ToolCallID string          `json:"tool_call_id"`
    ToolName   string          `json:"tool_name"`
    Content    []ContentBlock  `json:"content"`
    IsError    bool            `json:"is_error"`
}

func (b ToolResultBlock) BlockType() string { return "tool_result" }

// ThinkingBlock represents model reasoning
type ThinkingBlock struct {
    Thinking string `json:"thinking"`
    Signature string `json:"signature,omitempty"`
}

func (b ThinkingBlock) BlockType() string { return "thinking" }

// ImageBlock represents image content
type ImageBlock struct {
    Source    string `json:"source"` // "base64" or "url"
    MediaType string `json:"media_type"`
    Data      string `json:"data"`
}

func (b ImageBlock) BlockType() string { return "image" }

// Unified Message type
type Message struct {
    ID        string         `json:"id"`
    Role      string         `json:"role"` // "system", "user", "assistant", "tool"
    Content   []ContentBlock `json:"content"`
    Metadata  MessageMetadata `json:"metadata,omitempty"`
    Timestamp int64          `json:"timestamp"`
}

type MessageMetadata struct {
    StopReason   string `json:"stop_reason,omitempty"`
    Usage        Usage  `json:"usage,omitempty"`
    ErrorMessage string `json:"error_message,omitempty"`
}

type Usage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
}
```

### 2. Sanitization Pipeline

```go
// pipeline/sanitizer.go
package pipeline

type MessageSanitizer interface {
    Sanitize(messages []types.Message) ([]types.Message, error)
}

// Pipeline executes multiple sanitizers in sequence
type Pipeline struct {
    sanitizers []MessageSanitizer
}

func (p *Pipeline) Add(sanitizer MessageSanitizer) {
    p.sanitizers = append(p.sanitizers, sanitizer)
}

func (p *Pipeline) Process(messages []types.Message) ([]types.Message, error) {
    result := messages
    for _, s := range p.sanitizers {
        var err error
        result, err = s.Sanitize(result)
        if err != nil {
            return nil, err
        }
    }
    return result, nil
}

// Concrete sanitizers

// EmptyMessageSanitizer removes empty assistant messages
type EmptyMessageSanitizer struct{}

func (s *EmptyMessageSanitizer) Sanitize(messages []types.Message) ([]types.Message, error) {
    var result []types.Message
    for _, msg := range messages {
        if msg.Role == "assistant" && isEmptyAssistantMessage(msg) {
            continue
        }
        result = append(result, msg)
    }
    return result, nil
}

// ConsecutiveAssistantMerger merges back-to-back assistant messages
type ConsecutiveAssistantMerger struct{}

func (s *ConsecutiveAssistantMerger) Sanitize(messages []types.Message) ([]types.Message, error) {
    if len(messages) < 2 {
        return messages, nil
    }
    
    var result []types.Message
    for i, msg := range messages {
        if i > 0 && msg.Role == "assistant" && result[len(result)-1].Role == "assistant" {
            // Merge with previous
            result[len(result)-1].Content = append(
                result[len(result)-1].Content,
                msg.Content...,
            )
        } else {
            result = append(result, msg)
        }
    }
    return result, nil
}

// TurnOrderValidator ensures proper user/assistant alternation
type TurnOrderValidator struct{}

func (s *TurnOrderValidator) Sanitize(messages []types.Message) ([]types.Message, error) {
    // Implementation similar to OpenClaw's validateGeminiTurns/validateAnthropicTurns
    // Handle different provider requirements
}
```

### 3. Provider Adapters

```go
// providers/adapter.go
package providers

type ProviderAdapter interface {
    Name() string
    ToProviderFormat(messages []types.Message) (interface{}, error)
    FromProviderFormat(response interface{}) (*types.Message, error)
    SupportsStreaming() bool
}

// OpenAI adapter
type OpenAIAdapter struct{}

func (a *OpenAIAdapter) ToProviderFormat(messages []types.Message) (interface{}, error) {
    var result []openai.ChatCompletionMessage
    
    for _, msg := range messages {
        switch msg.Role {
        case "system":
            result = append(result, openai.ChatCompletionMessage{
                Role:    "system",
                Content: extractTextContent(msg.Content),
            })
        case "user":
            result = append(result, openai.ChatCompletionMessage{
                Role:    "user",
                Content: extractTextContent(msg.Content),
            })
        case "assistant":
            content := extractTextContent(msg.Content)
            toolCalls := extractToolCalls(msg.Content)
            
            result = append(result, openai.ChatCompletionMessage{
                Role:      "assistant",
                Content:   content,
                ToolCalls: convertToolCalls(toolCalls),
            })
        case "tool":
            // Tool results
            for _, block := range msg.Content {
                if tr, ok := block.(types.ToolResultBlock); ok {
                    result = append(result, openai.ChatCompletionMessage{
                        Role:       "tool",
                        Content:    extractTextContent(tr.Content),
                        ToolCallID: tr.ToolCallID,
                    })
                }
            }
        }
    }
    
    return result, nil
}

// Anthropic adapter (different format)
type AnthropicAdapter struct{}

func (a *AnthropicAdapter) ToProviderFormat(messages []types.Message) (interface{}, error) {
    // Anthropic uses different message structure
    // Implement conversion logic
}
```

### 4. Dynamic Tool Registry

```go
// tools/registry.go
package tools

// ToolDefinition defines a tool that can be invoked
type ToolDefinition struct {
    Name        string
    Description string
    Parameters  json.RawMessage
    Handler     ToolHandler
    Source      ToolSource // "builtin", "skill", "plugin"
}

type ToolHandler func(ctx context.Context, args json.RawMessage) (*ToolResult, error)

type ToolResult struct {
    Content []types.ContentBlock
    IsError bool
}

type ToolSource string

const (
    ToolSourceBuiltin ToolSource = "builtin"
    ToolSourceSkill   ToolSource = "skill"
    ToolSourcePlugin  ToolSource = "plugin"
)

// Registry manages available tools
type Registry struct {
    tools map[string]*ToolDefinition
    mu    sync.RWMutex
}

func (r *Registry) Register(tool *ToolDefinition) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.tools[tool.Name]; exists {
        return fmt.Errorf("tool %s already registered", tool.Name)
    }
    
    r.tools[tool.Name] = tool
    return nil
}

func (r *Registry) Unregister(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    delete(r.tools, name)
    return nil
}

func (r *Registry) Get(name string) (*ToolDefinition, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    tool, exists := r.tools[name]
    return tool, exists
}

func (r *Registry) List() []*ToolDefinition {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    result := make([]*ToolDefinition, 0, len(r.tools))
    for _, tool := range r.tools {
        result = append(result, tool)
    }
    return result
}

// SkillLoader loads tools from skill directories
type SkillLoader struct {
    registry *Registry
    skillsDir string
}

func (l *SkillLoader) LoadSkill(skillName string) error {
    // Load SKILL.md and any tool definitions
    // Register tools dynamically
}
```

### 5. Streaming Architecture

```go
// streaming/stream.go
package streaming

// StreamEvent represents an event in the response stream
type StreamEvent interface {
    EventType() string
}

// TextDeltaEvent represents incremental text
type TextDeltaEvent struct {
    Text string
}

func (e TextDeltaEvent) EventType() string { return "text_delta" }

// ToolCallStartEvent signals a tool call is starting
type ToolCallStartEvent struct {
    ID   string
    Name string
}

func (e ToolCallStartEvent) EventType() string { return "tool_call_start" }

// ToolCallDeltaEvent represents incremental tool call arguments
type ToolCallDeltaEvent struct {
    ID        string
    Arguments string // Partial JSON
}

func (e ToolCallDeltaEvent) EventType() string { return "tool_call_delta" }

// ToolCallCompleteEvent signals a tool call is complete
type ToolCallCompleteEvent struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

func (e ToolCallCompleteEvent) EventType() string { return "tool_call_complete" }

// UsageEvent contains token usage info
type UsageEvent struct {
    InputTokens  int
    OutputTokens int
}

func (e UsageEvent) EventType() string { return "usage" }

// StreamHandler processes streaming events
type StreamHandler interface {
    OnEvent(event StreamEvent) error
    OnError(err error)
    OnComplete()
}

// StreamingClient interface for LLM providers
type StreamingClient interface {
    Stream(ctx context.Context, messages []types.Message, tools []*ToolDefinition, handler StreamHandler) error
}
```

### 6. Conversation Management

```go
// conversation/manager.go
package conversation

type Manager struct {
    store      Store
    pipeline   *pipeline.Pipeline
    maxTokens  int
    compaction CompactionStrategy
}

type CompactionStrategy interface {
    Compact(messages []types.Message, targetTokens int) ([]types.Message, error)
}

// ContextWindowCompaction implements sliding window with summarization
type ContextWindowCompaction struct {
    summarizer Summarizer
}

func (c *ContextWindowCompaction) Compact(messages []types.Message, targetTokens int) ([]types.Message, error) {
    // Keep recent messages
    // Summarize older messages
    // Return compacted history
}
```

## Migration Strategy

### Phase 1: Content Block Model (Week 1-2)
1. Create new `types` package with content blocks
2. Create database migration to convert existing messages
3. Update store layer to handle new format
4. Maintain backward compatibility during transition

### Phase 2: Provider Adapters (Week 2-3)
1. Implement ProviderAdapter interface
2. Create OpenAI and Anthropic adapters
3. Update agent to use adapters
4. Add comprehensive tests

### Phase 3: Sanitization Pipeline (Week 3-4)
1. Implement Pipeline framework
2. Create EmptyMessageSanitizer
3. Create ConsecutiveAssistantMerger
4. Create TurnOrderValidator
5. Integrate into agent flow

### Phase 4: Tool Registry (Week 4-5)
1. Implement Registry
2. Create SkillLoader
3. Convert existing tools to new format
4. Add hot-reload capability

### Phase 5: Streaming (Week 5-6)
1. Implement streaming interfaces
2. Add streaming support to providers
3. Update Telegram bot for streaming
4. Add progress indicators

### Phase 6: Polish (Week 6-8)
1. Add comprehensive error handling
2. Implement metrics and monitoring
3. Performance optimization
4. Documentation

## Database Migration

```sql
-- Migration: Convert messages to content blocks
-- Step 1: Add new column
ALTER TABLE messages ADD COLUMN content_blocks JSONB;

-- Step 2: Migrate data
UPDATE messages SET content_blocks = CASE
    WHEN tool_calls IS NOT NULL AND tool_calls != '[]'::jsonb THEN
        jsonb_build_array(
            jsonb_build_object('type', 'text', 'text', content),
            jsonb_build_object('type', 'tool_calls', 'calls', tool_calls)
        )
    ELSE
        jsonb_build_array(jsonb_build_object('type', 'text', 'text', content))
    END;

-- Step 3: Remove old columns (after verification)
-- ALTER TABLE messages DROP COLUMN tool_calls;
-- ALTER TABLE messages DROP COLUMN tool_call_id;
```

## Success Metrics

1. **Reliability**: Zero empty message errors in production
2. **Flexibility**: Add new provider in < 1 day
3. **Extensibility**: Add new tool without code deployment
4. **Performance**: < 100ms latency for context building
5. **Maintainability**: < 50% code reduction in message handling

## Immediate Actions Needed

1. **Stop current defensive filtering approach** - it's a band-aid
2. **Approve architecture overhaul** - this is 6-8 weeks of work
3. **Freeze new features** until architecture is stable
4. **Set up proper test environment** for migration validation

**Do you want to proceed with this overhaul, or do you want to discuss/adjust any part of the design?**
</Content>
</Action>

I'll create the architecture design document. Let me also mark the first todo as complete and start on the next one.
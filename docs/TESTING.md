# Test Suite Documentation

## Overview

Myrai v2 includes comprehensive test coverage for all major components. Tests are organized by package and use the standard Go testing framework with `testify` for assertions.

## Test Structure

```
internal/
├── types/
│   └── content_test.go       # Content block tests (328 lines)
├── pipeline/
│   └── sanitizer_test.go     # Pipeline sanitizer tests (375 lines)
├── tools/
│   └── registry_test.go      # Tool registry tests (383 lines)
├── streaming/
│   └── events_test.go        # Streaming event tests (327 lines)
├── compaction/
│   └── compactor_test.go     # Context compaction tests (266 lines)
```

## Running Tests

### Run All Tests
```bash
go test ./internal/...
```

### Run Specific Package Tests
```bash
# Content types
go test ./internal/types/... -v

# Pipeline sanitizers
go test ./internal/pipeline/... -v

# Tool registry
go test ./internal/tools/... -v

# Streaming
go test ./internal/streaming/... -v

# Compaction
go test ./internal/compaction/... -v
```

### Run with Coverage
```bash
go test ./internal/... -cover

# Detailed coverage report
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Benchmarks
```bash
go test ./internal/... -bench=. -benchmem
```

## Test Categories

### 1. Content Block Tests (`internal/types/content_test.go`)

**Coverage**:
- TextBlock validation and serialization
- ToolCallBlock handling
- ToolResultBlock validation
- ThinkingBlock support
- ImageBlock support
- Message operations (IsEmpty, HasToolCalls, GetTextContent, etc.)
- JSON marshaling/unmarshaling
- Content block type detection

**Key Tests**:
- `TestTextBlock` - Empty detection, whitespace handling, JSON marshaling
- `TestToolCallBlock` - Tool call structure validation
- `TestMessage` - Message operations and validation
- `TestUnmarshalContentBlock` - Polymorphic deserialization
- `BenchmarkMessageGetTextContent` - Performance benchmark

### 2. Pipeline Sanitizer Tests (`internal/pipeline/sanitizer_test.go`)

**Coverage**:
- EmptyAssistantFilter
- ConsecutiveAssistantMerger
- ConsecutiveUserMerger
- EmptyTextBlockFilter
- SystemMessageNormalizer
- ToolResultValidator
- Pipeline orchestration

**Key Tests**:
- `TestEmptyAssistantFilter` - Removes empty assistant messages
- `TestConsecutiveAssistantMerger` - Merges back-to-back assistant messages
- `TestSystemMessageNormalizer` - Consolidates system messages
- `TestToolResultValidator` - Validates tool call/result pairing
- `TestPipeline` - Pipeline composition and ordering
- `BenchmarkEmptyAssistantFilter` - Performance benchmark

### 3. Tool Registry Tests (`internal/tools/registry_test.go`)

**Coverage**:
- Tool registration and unregistration
- Tool retrieval by name
- Tool listing and filtering
- Tool execution
- Thread-safe operations
- Global registry functions
- Error handling

**Key Tests**:
- `TestRegistry_Register` - Registration validation
- `TestRegistry_Concurrency` - Thread safety
- `TestRegistry_Execute` - Tool execution flow
- `TestGlobalRegistry` - Global functions
- `BenchmarkRegistry_Get` - Performance benchmark

### 4. Streaming Event Tests (`internal/streaming/events_test.go`)

**Coverage**:
- All event types (TextDelta, ToolCall, Usage, Stop, Error, etc.)
- Event serialization/deserialization
- Accumulator functionality
- Event handler interfaces
- Error detection

**Key Tests**:
- `TestTextDeltaEvent` - Text streaming events
- `TestToolCallEvents` - Tool call lifecycle events
- `TestAccumulator` - Event accumulation into messages
- `TestEventSerialization` - JSON round-trip
- `BenchmarkAccumulator_ProcessEvent` - Performance benchmark

### 5. Compaction Tests (`internal/compaction/compactor_test.go`)

**Coverage**:
- Sliding window compaction
- Summarization strategies
- Smart compaction with fallbacks
- Context manager operations
- Token estimation

**Key Tests**:
- `TestSlidingWindowCompactor` - Window-based message retention
- `TestSimpleSummarizer` - Rule-based summarization
- `TestSmartCompactor` - Intelligent compaction
- `TestContextManager` - Context preparation
- `TestEstimateMessageTokens` - Token counting
- `BenchmarkEstimateMessageTokens` - Performance benchmark

## Test Philosophy

### 1. Comprehensive Coverage
- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **Edge Cases**: Empty inputs, nil values, error conditions
- **Concurrency Tests**: Thread safety validation

### 2. Performance Awareness
- Benchmarks for critical paths
- Memory allocation tracking
- Concurrent load testing

### 3. Realistic Data
- Tests use realistic message structures
- Covers all content block types
- Tests error scenarios

### 4. Maintainability
- Clear test names describing behavior
- Table-driven tests for multiple cases
- Shared test helpers where appropriate

## Writing New Tests

### Pattern
```go
func TestComponent_Method(t *testing.T) {
    t.Run("descriptive test case", func(t *testing.T) {
        // Arrange
        input := setupData()
        
        // Act
        result, err := component.Method(input)
        
        // Assert
        require.NoError(t, err)
        assert.Equal(t, expected, result)
    })
}
```

### Table-Driven Tests
```go
func TestComponent_MultipleCases(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {"case 1", input1, expected1, false},
        {"case 2", input2, expected2, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := component.Method(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Benchmarks
```go
func BenchmarkComponent_Operation(b *testing.B) {
    setup := prepareData()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _ = component.Operation(setup)
    }
}
```

## Continuous Integration

### Recommended CI Configuration
```yaml
test:
  script:
    - go test ./internal/... -race -coverprofile=coverage.out
    - go tool cover -func=coverage.out | grep total | awk '{print $3}'
    - go test ./internal/... -bench=. -benchmem
  coverage:
    min: 70%
```

### Pre-commit Hooks
```bash
#!/bin/bash
# Run tests before commit
go test ./internal/... || exit 1
```

## Debugging Tests

### Verbose Output
```bash
go test ./internal/types/... -v
```

### Run Single Test
```bash
go test ./internal/types/... -run TestTextBlock -v
```

### Debug with Print Statements
```go
func TestSomething(t *testing.T) {
    t.Logf("Debug: value=%v", value)
    // ...
}
```

### Race Detection
```bash
go test ./internal/... -race
```

## Coverage Goals

| Package | Target Coverage |
|---------|----------------|
| types   | 90% |
| pipeline| 90% |
| tools   | 85% |
| streaming| 85% |
| compaction| 80% |

## Known Gaps

1. **Store v2 Tests**: Need PostgreSQL test container
2. **Provider Adapter Tests**: Need mock LLM responses
3. **Conversation Manager Tests**: Integration tests needed
4. **Migration Tests**: Database migration validation

## Future Improvements

1. Add property-based testing (fuzzing)
2. Add integration tests with testcontainers
3. Add performance regression tests
4. Add chaos engineering tests

## Total Test Coverage

- **Test Files**: 5
- **Total Lines**: ~1,700
- **Unit Tests**: 50+
- **Benchmarks**: 5
- **Test Functions**: 70+
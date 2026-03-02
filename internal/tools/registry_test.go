package tools

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/gmsas95/myrai-cli/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry()

	t.Run("registers tool successfully", func(t *testing.T) {
		tool := &ToolDefinition{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters:  json.RawMessage(`{}`),
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return &ToolResult{Content: []types.ContentBlock{types.TextBlock{Text: "OK"}}}, nil
			},
		}

		err := reg.Register(tool)
		require.NoError(t, err)

		// Verify it was registered
		retrieved, exists := reg.Get("test_tool")
		assert.True(t, exists)
		assert.Equal(t, "test_tool", retrieved.Name)
	})

	t.Run("returns error for duplicate registration", func(t *testing.T) {
		tool := &ToolDefinition{
			Name: "duplicate_tool",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		}

		err := reg.Register(tool)
		require.NoError(t, err)

		err = reg.Register(tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		tool := &ToolDefinition{
			Name: "",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		}

		err := reg.Register(tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("returns error for nil handler", func(t *testing.T) {
		tool := &ToolDefinition{
			Name:    "no_handler",
			Handler: nil,
		}

		err := reg.Register(tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no handler")
	})

	t.Run("sets created_at if not provided", func(t *testing.T) {
		tool := &ToolDefinition{
			Name: "new_tool",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		}

		err := reg.Register(tool)
		require.NoError(t, err)

		retrieved, _ := reg.Get("new_tool")
		assert.False(t, retrieved.CreatedAt.IsZero())
	})
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()

	t.Run("unregisters existing tool", func(t *testing.T) {
		tool := &ToolDefinition{
			Name: "to_remove",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		}

		err := reg.Register(tool)
		require.NoError(t, err)

		err = reg.Unregister("to_remove")
		require.NoError(t, err)

		_, exists := reg.Get("to_remove")
		assert.False(t, exists)
	})

	t.Run("returns error for non-existent tool", func(t *testing.T) {
		err := reg.Unregister("non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()

	t.Run("returns tool and true when exists", func(t *testing.T) {
		tool := &ToolDefinition{
			Name: "existing",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		}

		err := reg.Register(tool)
		require.NoError(t, err)

		retrieved, exists := reg.Get("existing")
		assert.True(t, exists)
		assert.Equal(t, "existing", retrieved.Name)
	})

	t.Run("returns nil and false when not exists", func(t *testing.T) {
		retrieved, exists := reg.Get("non_existent")
		assert.False(t, exists)
		assert.Nil(t, retrieved)
	})
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	t.Run("returns all registered tools", func(t *testing.T) {
		// Register multiple tools
		for i := 0; i < 3; i++ {
			tool := &ToolDefinition{
				Name: string(rune('a' + i)),
				Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
					return nil, nil
				},
			}
			err := reg.Register(tool)
			require.NoError(t, err)
		}

		tools := reg.List()
		assert.Len(t, tools, 3)
	})

	t.Run("returns empty slice when no tools", func(t *testing.T) {
		emptyReg := NewRegistry()
		tools := emptyReg.List()
		assert.Empty(t, tools)
	})
}

func TestRegistry_ListBySource(t *testing.T) {
	reg := NewRegistry()

	t.Run("returns tools filtered by source", func(t *testing.T) {
		// Register builtin tool
		err := reg.Register(&ToolDefinition{
			Name:   "builtin_tool",
			Source: ToolSourceBuiltin,
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		})
		require.NoError(t, err)

		// Register skill tool
		err = reg.Register(&ToolDefinition{
			Name:   "skill_tool",
			Source: ToolSourceSkill,
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		})
		require.NoError(t, err)

		builtinTools := reg.ListBySource(ToolSourceBuiltin)
		skillTools := reg.ListBySource(ToolSourceSkill)

		assert.Len(t, builtinTools, 1)
		assert.Len(t, skillTools, 1)
		assert.Equal(t, "builtin_tool", builtinTools[0].Name)
		assert.Equal(t, "skill_tool", skillTools[0].Name)
	})
}

func TestRegistry_Execute(t *testing.T) {
	reg := NewRegistry()

	t.Run("executes tool successfully", func(t *testing.T) {
		tool := &ToolDefinition{
			Name: "echo",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return &ToolResult{
					Content: []types.ContentBlock{types.TextBlock{Text: "Echo: " + string(args)}},
				}, nil
			},
		}

		err := reg.Register(tool)
		require.NoError(t, err)

		result, err := reg.Execute(context.Background(), "echo", json.RawMessage(`"hello"`))
		require.NoError(t, err)

		assert.Len(t, result.Content, 1)
		assert.Equal(t, `Echo: "hello"`, result.Content[0].(types.TextBlock).Text)
	})

	t.Run("returns error for non-existent tool", func(t *testing.T) {
		_, err := reg.Execute(context.Background(), "non_existent", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns handler errors", func(t *testing.T) {
		tool := &ToolDefinition{
			Name: "error_tool",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, errors.New("handler error")
			},
		}

		err := reg.Register(tool)
		require.NoError(t, err)

		_, err = reg.Execute(context.Background(), "error_tool", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler error")
	})
}

func TestRegistry_Concurrency(t *testing.T) {
	reg := NewRegistry()

	t.Run("handles concurrent reads and writes", func(t *testing.T) {
		// Register initial tool
		err := reg.Register(&ToolDefinition{
			Name: "initial",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		})
		require.NoError(t, err)

		var wg sync.WaitGroup
		numGoroutines := 100

		// Concurrent reads
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = reg.Get("initial")
				_ = reg.List()
			}()
		}

		// Concurrent writes
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_ = reg.Register(&ToolDefinition{
					Name: string(rune('a' + idx)),
					Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
						return nil, nil
					},
				})
			}(i)
		}

		wg.Wait()
		// If we get here without deadlock or panic, the test passes
	})
}

func TestToolDefinition_ToFunctionDefinition(t *testing.T) {
	tool := &ToolDefinition{
		Name:        "read_file",
		Description: "Reads a file",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}

	def := tool.ToFunctionDefinition()

	assert.Equal(t, "read_file", def["name"])
	assert.Equal(t, "Reads a file", def["description"])
	assert.Equal(t, json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`), def["parameters"])
}

func TestGlobalRegistry(t *testing.T) {
	t.Run("global register and get work", func(t *testing.T) {
		// Reset default registry
		DefaultRegistry = NewRegistry()

		tool := &ToolDefinition{
			Name: "global_test",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return &ToolResult{Content: []types.ContentBlock{types.TextBlock{Text: "OK"}}}, nil
			},
		}

		err := Register(tool)
		require.NoError(t, err)

		retrieved, exists := Get("global_test")
		assert.True(t, exists)
		assert.Equal(t, "global_test", retrieved.Name)
	})

	t.Run("global execute works", func(t *testing.T) {
		// Reset default registry
		DefaultRegistry = NewRegistry()

		tool := &ToolDefinition{
			Name: "global_echo",
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return &ToolResult{Content: []types.ContentBlock{types.TextBlock{Text: "OK"}}}, nil
			},
		}

		err := Register(tool)
		require.NoError(t, err)

		result, err := Execute(context.Background(), "global_echo", nil)
		require.NoError(t, err)
		assert.Equal(t, "OK", result.Content[0].(types.TextBlock).Text)
	})
}

func BenchmarkRegistry_Register(b *testing.B) {
	reg := NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := &ToolDefinition{
			Name: string(rune('a' + (i % 26))),
			Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
				return nil, nil
			},
		}
		_ = reg.Register(tool)
	}
}

func BenchmarkRegistry_Get(b *testing.B) {
	reg := NewRegistry()

	// Register a tool
	_ = reg.Register(&ToolDefinition{
		Name: "test",
		Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
			return nil, nil
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.Get("test")
	}
}

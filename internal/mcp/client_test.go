package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	config := MCPServerConfig{
		Name:      "test-server",
		Command:   "echo",
		Args:      []string{"hello"},
		Transport: TransportStdio,
		Timeout:   30,
	}

	client := NewClient(config)
	assert.NotNil(t, client)
	assert.Equal(t, config.Name, client.GetServerConfig().Name)
	assert.False(t, client.IsConnected())
}

func TestMCPServerConfig(t *testing.T) {
	config := MCPServerConfig{
		Name:        "filesystem",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		Env:         map[string]string{"KEY": "value"},
		Enabled:     true,
		Transport:   TransportStdio,
		Timeout:     30,
		Description: "Test filesystem server",
	}

	assert.Equal(t, "filesystem", config.Name)
	assert.Equal(t, "npx", config.Command)
	assert.Len(t, config.Args, 3)
	assert.Equal(t, TransportStdio, config.Transport)
}

func TestTransportTypes(t *testing.T) {
	assert.Equal(t, TransportType("stdio"), TransportStdio)
	assert.Equal(t, TransportType("sse"), TransportSSE)
}

func TestToolResult(t *testing.T) {
	result := &ToolResult{
		Content: []Content{
			{Type: "text", Text: "Hello"},
		},
		IsError: false,
	}

	assert.Len(t, result.Content, 1)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "Hello", result.Content[0].Text)
	assert.False(t, result.IsError)
}

func TestTool(t *testing.T) {
	tool := Tool{
		Name:        "read_file",
		Description: "Read a file",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	assert.Equal(t, "read_file", tool.Name)
	assert.Equal(t, "Read a file", tool.Description)
	assert.NotNil(t, tool.InputSchema)
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	time.Sleep(time.Nanosecond)
	id2 := generateRequestID()

	assert.NotEqual(t, id1, id2)
	assert.Greater(t, id2, id1)
}

func TestRemoteClientNotConnected(t *testing.T) {
	config := MCPServerConfig{
		Name:    "test",
		Command: "echo",
	}

	client := NewClient(config)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Should fail when not connected
	_, err := client.ListTools(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

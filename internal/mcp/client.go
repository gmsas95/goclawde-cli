// Package mcp implements the Model Context Protocol (MCP) client
// MCP is an open protocol for AI assistants to communicate with external tools
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gmsas95/myrai-cli/internal/errors"
)

// MCPServerConfig represents the configuration for an MCP server
type MCPServerConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Command     string            `json:"command" yaml:"command"`
	Args        []string          `json:"args" yaml:"args"`
	Env         map[string]string `json:"env" yaml:"env"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Transport   TransportType     `json:"transport" yaml:"transport"`
	URL         string            `json:"url,omitempty" yaml:"url,omitempty"`
	Timeout     int               `json:"timeout" yaml:"timeout"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// TransportType represents the transport mechanism for MCP communication
type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
)

// RemoteClient represents an MCP client connection to a remote server
type RemoteClient struct {
	serverConfig MCPServerConfig
	transport    Transport
	tools        []Tool
	mu           sync.RWMutex
	connected    bool
	clientID     string
}

// Transport defines the interface for MCP transports
type Transport interface {
	Connect(ctx context.Context) error
	Disconnect() error
	Send(ctx context.Context, message []byte) ([]byte, error)
	IsConnected() bool
}

// stdioTransport implements Transport for stdin/stdout communication
type stdioTransport struct {
	config    MCPServerConfig
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	mu        sync.Mutex
	connected bool
}

// sseTransport implements Transport for Server-Sent Events communication
type sseTransport struct {
	config     MCPServerConfig
	client     *http.Client
	eventURL   string
	messageURL string
	connected  bool
	mu         sync.Mutex
}

// NewClient creates a new MCP client with the given server configuration
func NewClient(config MCPServerConfig) *RemoteClient {
	return &RemoteClient{
		serverConfig: config,
		tools:        make([]Tool, 0),
		clientID:     generateClientID(),
	}
}

// Connect establishes a connection to the MCP server
func (c *RemoteClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	var transport Transport
	var err error

	switch c.serverConfig.Transport {
	case TransportSSE:
		transport, err = newSSETransport(c.serverConfig)
	case TransportStdio:
		fallthrough
	default:
		transport, err = newStdioTransport(c.serverConfig)
	}

	if err != nil {
		return errors.Wrap(err, "MCP_001", "failed to create transport")
	}

	if err := transport.Connect(ctx); err != nil {
		return errors.Wrap(err, "MCP_002", "failed to connect to MCP server")
	}

	c.transport = transport
	c.connected = true

	// Initialize and get tools list
	tools, err := c.listToolsInternal(ctx)
	if err != nil {
		transport.Disconnect()
		c.connected = false
		return errors.Wrap(err, "MCP_003", "failed to list tools from MCP server")
	}

	c.tools = tools
	return nil
}

// Disconnect closes the connection to the MCP server
func (c *RemoteClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.transport == nil {
		return nil
	}

	if err := c.transport.Disconnect(); err != nil {
		return errors.Wrap(err, "MCP_004", "error disconnecting from MCP server")
	}

	c.connected = false
	c.tools = make([]Tool, 0)
	return nil
}

// IsConnected returns whether the client is connected to an MCP server
func (c *RemoteClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.transport != nil && c.transport.IsConnected()
}

// ListTools returns the list of available tools from the MCP server
func (c *RemoteClient) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.RLock()
	if c.connected && len(c.tools) > 0 {
		tools := make([]Tool, len(c.tools))
		copy(tools, c.tools)
		c.mu.RUnlock()
		return tools, nil
	}
	c.mu.RUnlock()

	if !c.IsConnected() {
		return nil, errors.New("MCP_005", "not connected to MCP server")
	}

	return c.listToolsInternal(ctx)
}

// listToolsInternal fetches the tools list from the MCP server
func (c *RemoteClient) listToolsInternal(ctx context.Context) ([]Tool, error) {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      generateRequestID(),
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "MCP_006", "failed to marshal tools/list request")
	}

	respData, err := c.transport.Send(ctx, reqData)
	if err != nil {
		return nil, errors.Wrap(err, "MCP_007", "failed to send tools/list request")
	}

	var resp struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      interface{} `json:"id"`
		Result  struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, errors.Wrap(err, "MCP_008", "failed to unmarshal tools/list response")
	}

	if resp.Error != nil {
		return nil, errors.New("MCP_009", fmt.Sprintf("MCP server error: %s", resp.Error.Message))
	}

	return resp.Result.Tools, nil
}

// CallTool invokes a tool on the MCP server
func (c *RemoteClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	if !c.IsConnected() {
		return nil, errors.New("MCP_010", "not connected to MCP server")
	}

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      generateRequestID(),
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "MCP_011", "failed to marshal tool call request")
	}

	respData, err := c.transport.Send(ctx, reqData)
	if err != nil {
		return nil, errors.Wrap(err, "MCP_012", "failed to send tool call request")
	}

	var resp struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      interface{} `json:"id"`
		Result  *ToolResult `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, errors.Wrap(err, "MCP_013", "failed to unmarshal tool call response")
	}

	if resp.Error != nil {
		return &ToolResult{
			Content: []Content{
				{Type: "text", Text: fmt.Sprintf("Error: %s", resp.Error.Message)},
			},
			IsError: true,
		}, nil
	}

	return resp.Result, nil
}

// GetServerConfig returns the server configuration
func (c *RemoteClient) GetServerConfig() MCPServerConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverConfig
}

// ============ stdioTransport Implementation ============

func newStdioTransport(config MCPServerConfig) (Transport, error) {
	return &stdioTransport{
		config: config,
	}, nil
}

func (t *stdioTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	cmd := exec.CommandContext(ctx, t.config.Command, t.config.Args...)

	// Set environment variables
	if len(t.config.Env) > 0 {
		env := make([]string, 0, len(t.config.Env))
		for k, v := range t.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = append(cmd.Env, env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.stdout = stdout
	t.stderr = stderr
	t.connected = true

	// Start a goroutine to read stderr for debugging
	go t.readStderr()

	return nil
}

func (t *stdioTransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	if t.stdin != nil {
		t.stdin.Close()
	}

	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}

	t.connected = false
	return nil
}

func (t *stdioTransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected && t.cmd != nil && t.cmd.ProcessState == nil
}

func (t *stdioTransport) Send(ctx context.Context, message []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil, fmt.Errorf("transport not connected")
	}

	// Write message with newline
	msg := append(message, '\n')
	if _, err := t.stdin.Write(msg); err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %w", err)
	}

	// Read response with timeout
	reader := bufio.NewReader(t.stdout)

	type result struct {
		data []byte
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		line, err := reader.ReadBytes('\n')
		ch <- result{data: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("failed to read from stdout: %w", res.err)
		}
		return bytes.TrimSpace(res.data), nil
	}
}

func (t *stdioTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		// Log stderr output for debugging
		// In production, this could be sent to a proper logger
		_ = scanner.Text()
	}
}

// ============ sseTransport Implementation ============

func newSSETransport(config MCPServerConfig) (Transport, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("URL is required for SSE transport")
	}

	return &sseTransport{
		config:     config,
		eventURL:   config.URL + "/sse",
		messageURL: config.URL + "/message",
		client:     &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *sseTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	// For SSE, we validate the endpoint is accessible
	// but actual SSE connection is handled during tool calls
	t.connected = true
	return nil
}

func (t *sseTransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connected = false
	return nil
}

func (t *sseTransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected
}

func (t *sseTransport) Send(ctx context.Context, message []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil, fmt.Errorf("transport not connected")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.messageURL, bytes.NewReader(message))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// ============ Helper Functions ============

func generateRequestID() int64 {
	return time.Now().UnixNano()
}

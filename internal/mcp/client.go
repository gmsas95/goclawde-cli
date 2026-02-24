// Package mcp provides a Model Context Protocol (MCP) client implementation
// supporting both stdio and SSE transports
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gmsas95/myrai-cli/internal/circuitbreaker"
	"go.uber.org/zap"
)

// TransportType represents the type of MCP transport
type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
)

// MCPServerConfig holds configuration for an MCP server
type MCPServerConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Transport   TransportType     `yaml:"transport" json:"transport"`
	Command     string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	URL         string            `yaml:"url,omitempty" json:"url,omitempty"`
	Timeout     int               `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	MaxRetries  int               `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content in a tool result
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Transport is the interface for MCP transports
type Transport interface {
	Connect(ctx context.Context) error
	Disconnect() error
	Send(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error)
	IsConnected() bool
}

// RemoteClient represents an MCP client connection (alias for backward compatibility)
type RemoteClient = Client

// Client represents an MCP client
type Client struct {
	serverConfig MCPServerConfig
	transport    Transport
	tools        []Tool
	cbManager    *circuitbreaker.Manager
	logger       *zap.Logger
	mu           sync.RWMutex
	connected    bool
	requestID    int64
	requestMu    sync.Mutex
}

// NewClient creates a new MCP client
func NewClient(config MCPServerConfig) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &Client{
		serverConfig: config,
		logger:       zap.NewNop(),
		tools:        []Tool{},
		cbManager:    circuitbreaker.NewDefaultManager(zap.NewNop()),
	}
}

// NewClientWithLogger creates a new MCP client with a custom logger
func NewClientWithLogger(config MCPServerConfig, logger *zap.Logger) *Client {
	client := NewClient(config)
	client.logger = logger
	return client
}

// GetServerConfig returns the server configuration
func (c *Client) GetServerConfig() MCPServerConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverConfig
}

// Connect establishes connection to the MCP server
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	var transport Transport

	switch c.serverConfig.Transport {
	case TransportStdio:
		transport = newStdioTransport(c.serverConfig, c.logger)
	case TransportSSE:
		transport = newSSETransport(c.serverConfig, c.logger)
	default:
		return fmt.Errorf("unsupported transport type: %s", c.serverConfig.Transport)
	}

	// Connect with timeout
	timeout := time.Duration(c.serverConfig.Timeout) * time.Second
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := transport.Connect(connectCtx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.transport = transport
	c.connected = true

	c.logger.Info("MCP client connected",
		zap.String("server", c.serverConfig.Name),
		zap.String("transport", string(c.serverConfig.Transport)),
	)

	// Initialize and list tools
	if err := c.initialize(ctx); err != nil {
		c.transport.Disconnect()
		c.connected = false
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// Disconnect closes the connection to the MCP server
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.transport == nil {
		return nil
	}

	if err := c.transport.Disconnect(); err != nil {
		c.logger.Warn("Error disconnecting from MCP server", zap.Error(err))
	}

	c.connected = false
	c.tools = nil

	c.logger.Info("MCP client disconnected", zap.String("server", c.serverConfig.Name))
	return nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.transport != nil && c.transport.IsConnected()
}

// Reconnect attempts to reconnect to the MCP server
func (c *Client) Reconnect(ctx context.Context) error {
	c.Disconnect()
	return c.Connect(ctx)
}

// initialize performs MCP initialization handshake
func (c *Client) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "myrai-cli",
			"version": "2.0.0",
		},
	}

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize params: %w", err)
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "initialize",
		Params:  paramsBytes,
	}

	resp, err := c.sendWithRetry(ctx, req)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %w", resp.Error)
	}

	c.logger.Debug("MCP initialized successfully",
		zap.String("server", c.serverConfig.Name),
	)

	// List available tools
	if _, err := c.ListTools(ctx); err != nil {
		c.logger.Warn("Failed to list tools", zap.Error(err))
	}

	return nil
}

// nextRequestID generates a unique request ID
func (c *Client) nextRequestID() int64 {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.requestID++
	return c.requestID
}

// generateRequestID generates a unique request ID (package-level function)
func generateRequestID() int64 {
	return time.Now().UnixNano()
}

// sendWithRetry sends a request with retry logic
func (c *Client) sendWithRetry(ctx context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= c.serverConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("Retrying request",
				zap.String("method", req.Method),
				zap.Int("attempt", attempt),
			)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		resp, err := c.transport.Send(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if we should reconnect
		if !c.IsConnected() {
			c.logger.Warn("Connection lost, attempting to reconnect")
			if reconnectErr := c.Reconnect(ctx); reconnectErr != nil {
				c.logger.Error("Reconnection failed", zap.Error(reconnectErr))
			}
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.serverConfig.MaxRetries+1, lastErr)
}

// ListTools returns all available tools from the MCP server
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/list",
	}

	resp, err := c.sendWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("list tools error: %w", resp.Error)
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	c.logger.Debug("Listed tools",
		zap.String("server", c.serverConfig.Name),
		zap.Int("count", len(result.Tools)),
	)

	return result.Tools, nil
}

// GetTools returns the cached list of tools
func (c *Client) GetTools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tools := make([]Tool, len(c.tools))
	copy(tools, c.tools)
	return tools
}

// CallTool invokes a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool params: %w", err)
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/call",
		Params:  paramsBytes,
	}

	resp, err := c.sendWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tool call error: %w", resp.Error)
	}

	var result ToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool result: %w", err)
	}

	return &result, nil
}

// stdioTransport implements Transport for stdio-based MCP servers
type stdioTransport struct {
	config MCPServerConfig
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	logger *zap.Logger
	mu     sync.Mutex
}

func newStdioTransport(config MCPServerConfig, logger *zap.Logger) *stdioTransport {
	return &stdioTransport{
		config: config,
		logger: logger,
	}
}

func (t *stdioTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd != nil {
		return fmt.Errorf("already connected")
	}

	t.cmd = exec.CommandContext(ctx, t.config.Command, t.config.Args...)

	// Set up environment
	if len(t.config.Env) > 0 {
		env := os.Environ()
		for k, v := range t.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		t.cmd.Env = env
	}

	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	t.stdout = stdout

	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	t.stderr = stderr

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Start stderr reader for logging
	go t.readStderr()

	t.logger.Info("Stdio transport connected",
		zap.String("command", t.config.Command),
		zap.Strings("args", t.config.Args),
	)

	return nil
}

func (t *stdioTransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil {
		return nil
	}

	if t.stdin != nil {
		t.stdin.Close()
	}

	if t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}

	t.cmd = nil
	t.stdin = nil
	t.stdout = nil
	t.stderr = nil

	return nil
}

func (t *stdioTransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cmd != nil && t.cmd.Process != nil
}

func (t *stdioTransport) readStderr() {
	if t.stderr == nil {
		return
	}
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		t.logger.Debug("MCP server stderr", zap.String("line", scanner.Text()))
	}
}

func (t *stdioTransport) Send(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil || t.stdin == nil || t.stdout == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Marshal request
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write request with newline
	data = append(data, '\n')
	if _, err := t.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(t.stdout)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	var response JSONRPCResponse
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// sseTransport implements Transport for SSE-based MCP servers
type sseTransport struct {
	config     MCPServerConfig
	httpClient *http.Client
	eventURL   string
	logger     *zap.Logger
	mu         sync.Mutex
	connected  bool
}

func newSSETransport(config MCPServerConfig, logger *zap.Logger) *sseTransport {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30
	}

	return &sseTransport{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		logger: logger,
	}
}

func (t *sseTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	// SSE connection would typically start here
	// For now, we just mark as connected
	t.connected = true
	t.eventURL = t.config.URL

	t.logger.Info("SSE transport connected", zap.String("url", t.config.URL))
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

func (t *sseTransport) Send(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Marshal request
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", t.eventURL, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// Manager manages multiple MCP clients
type Manager struct {
	clients map[string]*Client
	mu      sync.RWMutex
	logger  *zap.Logger
}

// NewManager creates a new MCP manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// RegisterClient registers an MCP client
func (m *Manager) RegisterClient(config MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[config.Name]; exists {
		return fmt.Errorf("client %s already registered", config.Name)
	}

	client := NewClientWithLogger(config, m.logger)
	m.clients[config.Name] = client

	m.logger.Info("MCP client registered", zap.String("name", config.Name))
	return nil
}

// ConnectClient connects a registered client
func (m *Manager) ConnectClient(ctx context.Context, name string) error {
	m.mu.RLock()
	client, exists := m.clients[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	return client.Connect(ctx)
}

// DisconnectClient disconnects a client
func (m *Manager) DisconnectClient(name string) error {
	m.mu.RLock()
	client, exists := m.clients[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	return client.Disconnect()
}

// GetClient returns a client by name
func (m *Manager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[name]
	return client, ok
}

// ListClients returns all registered client names
func (m *Manager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// DisconnectAll disconnects all clients
func (m *Manager) DisconnectAll() {
	m.mu.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	for _, client := range clients {
		if err := client.Disconnect(); err != nil {
			m.logger.Warn("Failed to disconnect client", zap.Error(err))
		}
	}
}

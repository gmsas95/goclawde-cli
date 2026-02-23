package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gmsas95/myrai-cli/internal/skills"
)

// Adapter adapts MCP tools to internal skill format
type Adapter struct {
	clients    map[string]*RemoteClient
	registry   *skills.EnhancedRegistry
	mu         sync.RWMutex
	configPath string
}

// NewAdapter creates a new MCP adapter
func NewAdapter(registry *skills.EnhancedRegistry, configPath string) *Adapter {
	return &Adapter{
		clients:    make(map[string]*RemoteClient),
		registry:   registry,
		configPath: configPath,
	}
}

// AddServer adds an MCP server configuration
func (a *Adapter) AddServer(config MCPServerConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.clients[config.Name]; exists {
		return fmt.Errorf("MCP server '%s' already exists", config.Name)
	}

	client := NewClient(config)
	a.clients[config.Name] = client

	return nil
}

// RemoveServer removes an MCP server configuration
func (a *Adapter) RemoveServer(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	client, exists := a.clients[name]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", name)
	}

	// Disconnect if connected
	if client.transport != nil {
		client.transport.Disconnect()
	}

	delete(a.clients, name)
	return nil
}

// ConnectServer connects to an MCP server and registers its tools
func (a *Adapter) ConnectServer(ctx context.Context, name string) error {
	a.mu.RLock()
	client, exists := a.clients[name]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("MCP server '%s' not found", name)
	}

	// Connect to server
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to MCP server '%s': %w", name, err)
	}

	// List tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools from MCP server '%s': %w", name, err)
	}

	// Adapt and register tools
	for _, tool := range tools {
		adaptedTool := a.adaptTool(name, tool, client)
		if err := a.registry.RegisterMCPTool(adaptedTool); err != nil {
			return fmt.Errorf("failed to register MCP tool '%s': %w", tool.Name, err)
		}
	}

	return nil
}

// DisconnectServer disconnects from an MCP server
func (a *Adapter) DisconnectServer(name string) error {
	a.mu.RLock()
	client, exists := a.clients[name]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("MCP server '%s' not found", name)
	}

	return client.Disconnect()
}

// GetClient returns an MCP client by name
func (a *Adapter) GetClient(name string) (*RemoteClient, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	client, ok := a.clients[name]
	return client, ok
}

// ListServers returns all configured MCP servers
func (a *Adapter) ListServers() []MCPServerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()

	configs := make([]MCPServerConfig, 0, len(a.clients))
	for _, client := range a.clients {
		configs = append(configs, client.serverConfig)
	}

	return configs
}

// adaptTool converts an MCP tool to internal format
func (a *Adapter) adaptTool(serverName string, mcpTool Tool, client *RemoteClient) *skills.MCPTool {
	return &skills.MCPTool{
		Name:        mcpTool.Name,
		Description: mcpTool.Description,
		InputSchema: mcpTool.InputSchema,
		ServerName:  serverName,
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			result, err := client.CallTool(ctx, mcpTool.Name, args)
			if err != nil {
				return nil, err
			}

			// Convert result to map
			if result.IsError {
				return nil, fmt.Errorf("MCP tool error: %v", result.Content)
			}

			// Extract text content
			var texts []string
			for _, content := range result.Content {
				if content.Type == "text" {
					texts = append(texts, content.Text)
				}
			}

			return map[string]interface{}{
				"content": strings.Join(texts, "\n"),
			}, nil
		},
	}
}

// LoadServers loads MCP server configurations from the config file
func (a *Adapter) LoadServers() error {
	if a.configPath == "" {
		return nil
	}

	data, err := os.ReadFile(a.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var config struct {
		MCP struct {
			Servers []MCPServerConfig `yaml:"servers" json:"servers"`
		} `yaml:"mcp" json:"mcp"`
	}

	// Try JSON first, then YAML
	if err := json.Unmarshal(data, &config); err != nil {
		// Simple parsing fallback
		return nil
	}

	for _, serverConfig := range config.MCP.Servers {
		if err := a.AddServer(serverConfig); err != nil {
			return fmt.Errorf("failed to add server '%s': %w", serverConfig.Name, err)
		}
	}

	return nil
}

// SaveServers saves MCP server configurations to the config file
func (a *Adapter) SaveServers() error {
	if a.configPath == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(a.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	servers := a.ListServers()

	config := struct {
		MCP struct {
			Servers []MCPServerConfig `yaml:"servers" json:"servers"`
		} `yaml:"mcp" json:"mcp"`
	}{
		MCP: struct {
			Servers []MCPServerConfig `yaml:"servers" json:"servers"`
		}{
			Servers: servers,
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.configPath, data, 0644)
}

// GetConnectedServers returns a list of connected server names
func (a *Adapter) GetConnectedServers() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var connected []string
	for name, client := range a.clients {
		if client.connected {
			connected = append(connected, name)
		}
	}
	return connected
}

// GetServerStatus returns the status of an MCP server
func (a *Adapter) GetServerStatus(name string) (string, error) {
	a.mu.RLock()
	client, exists := a.clients[name]
	a.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("MCP server '%s' not found", name)
	}

	if client.connected {
		return "connected", nil
	}
	return "disconnected", nil
}

// StartAutoConnect connects to all enabled servers
func (a *Adapter) StartAutoConnect(ctx context.Context) error {
	servers := a.ListServers()

	for _, serverConfig := range servers {
		if serverConfig.Enabled {
			if err := a.ConnectServer(ctx, serverConfig.Name); err != nil {
				// Log error but continue with other servers
				continue
			}
		}
	}

	return nil
}

// ==================== Auto-Discovery ====================

// DiscoveredServer represents a discovered MCP server
type DiscoveredServer struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Category    string   `json:"category"`
	Official    bool     `json:"official"`
}

// AutoDiscover discovers available MCP servers
func AutoDiscover() []DiscoveredServer {
	return []DiscoveredServer{
		{
			Name:        "filesystem",
			Description: "File system operations - read, write, list files",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "/home"},
			Category:    "system",
			Official:    true,
		},
		{
			Name:        "github",
			Description: "GitHub operations - repos, issues, PRs",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-github"},
			Category:    "vcs",
			Official:    true,
		},
		{
			Name:        "postgres",
			Description: "PostgreSQL database operations",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-postgres"},
			Category:    "database",
			Official:    true,
		},
		{
			Name:        "sqlite",
			Description: "SQLite database operations",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-sqlite"},
			Category:    "database",
			Official:    true,
		},
		{
			Name:        "puppeteer",
			Description: "Browser automation with Puppeteer",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-puppeteer"},
			Category:    "browser",
			Official:    true,
		},
		{
			Name:        "brave-search",
			Description: "Web search using Brave Search API",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
			Category:    "search",
			Official:    true,
		},
		{
			Name:        "fetch",
			Description: "Fetch web content",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-fetch"},
			Category:    "web",
			Official:    true,
		},
		{
			Name:        "slack",
			Description: "Slack messaging operations",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-slack"},
			Category:    "communication",
			Official:    true,
		},
		{
			Name:        "google-maps",
			Description: "Google Maps API integration",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-google-maps"},
			Category:    "location",
			Official:    true,
		},
	}
}

// SearchDiscoveredServers searches discovered servers by query
func SearchDiscoveredServers(query string) []DiscoveredServer {
	allServers := AutoDiscover()
	query = strings.ToLower(query)

	var results []DiscoveredServer
	for _, server := range allServers {
		if strings.Contains(strings.ToLower(server.Name), query) ||
			strings.Contains(strings.ToLower(server.Description), query) ||
			strings.Contains(strings.ToLower(server.Category), query) {
			results = append(results, server)
		}
	}

	return results
}

// LoadMCPConfig loads MCP server configurations from a JSON file
func LoadMCPConfig(configPath string) ([]MCPServerConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []MCPServerConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read MCP config: %w", err)
	}

	var configs []MCPServerConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	return configs, nil
}

// SaveMCPConfig saves MCP server configurations to a JSON file
func SaveMCPConfig(configPath string, configs []MCPServerConfig) error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write MCP config: %w", err)
	}

	return nil
}

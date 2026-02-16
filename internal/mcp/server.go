// Package mcp implements the Model Context Protocol (MCP) server
// MCP is an open protocol for AI assistants to communicate with external tools
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/pkg/tools"
)

// Server implements an MCP server for tool exposure
type Server struct {
	config      *config.Config
	tools       *tools.Registry
	mu          sync.RWMutex
	clients     map[string]*Client
	httpServer  *http.Server
}

// Client represents an MCP client connection
type Client struct {
	ID       string
	Tools    []string
	Session  string
}

// Tool represents an MCP tool definition
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

// ToolResult represents a tool execution result
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents result content
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config, toolRegistry *tools.Registry) *Server {
	return &Server{
		config:  cfg,
		tools:   toolRegistry,
		clients: make(map[string]*Client),
	}
}

// Start starts the MCP server
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	
	// SSE endpoint for server-to-client streaming
	mux.HandleFunc("/mcp/sse", s.handleSSE)
	
	// HTTP endpoint for client-to-server messages
	mux.HandleFunc("/mcp/message", s.handleMessage)
	
	// Tool listing endpoint
	mux.HandleFunc("/mcp/tools", s.handleListTools)
	
	// Health check
	mux.HandleFunc("/mcp/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	log.Printf("MCP server starting on %s", addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the MCP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleSSE handles Server-Sent Events connections
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}
	
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = generateClientID()
	}
	
	// Register client
	s.mu.Lock()
	s.clients[clientID] = &Client{
		ID:      clientID,
		Session: generateSessionID(),
	}
	s.mu.Unlock()
	
	// Send initial endpoint message
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp/message?client_id=%s\n\n", clientID)
	flusher.Flush()
	
	// Send tools list
	tools := s.getTools()
	toolsJSON, _ := json.Marshal(tools)
	fmt.Fprintf(w, "event: tools\ndata: %s\n\n", toolsJSON)
	flusher.Flush()
	
	// Keep connection open
	<-r.Context().Done()
	
	// Cleanup
	s.mu.Lock()
	delete(s.clients, clientID)
	s.mu.Unlock()
}

// handleMessage handles incoming client messages
func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var msg struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
		ID     interface{}     `json:"id"`
	}
	
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var result interface{}
	var callErr error
	
	switch msg.Method {
	case "tools/list":
		result = s.getTools()
		
	case "tools/call":
		var call ToolCall
		if err := json.Unmarshal(msg.Params, &call); err != nil {
			callErr = err
			break
		}
		result, callErr = s.callTool(r.Context(), call)
		
	default:
		callErr = fmt.Errorf("unknown method: %s", msg.Method)
	}
	
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      msg.ID,
	}
	
	if callErr != nil {
		response["error"] = map[string]interface{}{
			"code":    -32603,
			"message": callErr.Error(),
		}
	} else {
		response["result"] = result
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListTools handles HTTP GET for tools list
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	tools := s.getTools()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

// getTools returns available tools in MCP format
func (s *Server) getTools() []Tool {
	if s.tools == nil {
		return []Tool{}
	}
	
	defs := s.tools.GetToolDefinitions()
	tools := make([]Tool, 0, len(defs))
	
	for _, def := range defs {
		if fn, ok := def["function"].(map[string]interface{}); ok {
			tool := Tool{
				Name:        getString(fn, "name"),
				Description: getString(fn, "description"),
				InputSchema: getMap(fn, "parameters"),
			}
			tools = append(tools, tool)
		}
	}
	
	return tools
}

// callTool executes a tool call
func (s *Server) callTool(ctx context.Context, call ToolCall) (*ToolResult, error) {
	if s.tools == nil {
		return nil, fmt.Errorf("no tools available")
	}
	
	argsJSON, err := json.Marshal(call.Arguments)
	if err != nil {
		return nil, err
	}
	
	result, err := s.tools.ExecuteJSON(ctx, call.Name, string(argsJSON))
	if err != nil {
		return &ToolResult{
			Content: []Content{
				{Type: "text", Text: fmt.Sprintf("Error: %v", err)},
			},
			IsError: true,
		}, nil
	}
	
	resultText := fmt.Sprintf("%v", result)
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: resultText},
		},
	}, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

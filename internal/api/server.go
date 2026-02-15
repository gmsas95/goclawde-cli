package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/agent"
	"github.com/gmsas95/goclawde-cli/internal/config"
	"github.com/gmsas95/goclawde-cli/internal/llm"
	"github.com/gmsas95/goclawde-cli/internal/persona"
	"github.com/gmsas95/goclawde-cli/internal/skills"
	"github.com/gmsas95/goclawde-cli/internal/store"
	"github.com/gmsas95/goclawde-cli/pkg/tools"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// Server handles HTTP API and WebSocket
type Server struct {
	app            *fiber.App
	config         *config.Config
	store          *store.Store
	agent          *agent.Agent
	llmClient      *llm.Client
	tools          *tools.Registry
	skillsRegistry *skills.Registry
	logger         *zap.Logger
	personaManager *persona.PersonaManager
}

// New creates a new API server
func New(cfg *config.Config, store *store.Store, logger *zap.Logger) *Server {
	// Create LLM client
	provider, err := cfg.DefaultProvider()
	if err != nil {
		logger.Fatal("Failed to get LLM provider", zap.Error(err))
	}
	llmClient := llm.NewClient(provider)

	// Create tool registry
	toolRegistry := tools.NewRegistry(cfg.Tools.AllowedCmds)

	// Create persona manager
	personaManager, err := persona.NewPersonaManager(cfg.Storage.DataDir, logger)
	if err != nil {
		logger.Warn("Failed to initialize persona manager", zap.Error(err))
		personaManager = nil
	}

	// Create agent
	agentInstance := agent.New(llmClient, toolRegistry, store, logger, personaManager)

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	s := &Server{
		app:            app,
		config:         cfg,
		store:          store,
		agent:          agentInstance,
		llmClient:      llmClient,
		tools:          toolRegistry,
		logger:         logger,
		personaManager: personaManager,
	}

	// Set skills registry on agent if available
	if s.skillsRegistry != nil {
		s.agent.SetSkillsRegistry(s.skillsRegistry)
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Middleware
	s.app.Use(recover.New())
	s.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(s.config.Security.AllowOrigins, ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Health check
	s.app.Get("/api/health", s.handleHealth)

	// API routes
	api := s.app.Group("/api")
	
	// Public routes
	api.Post("/auth/login", s.handleLogin)
	
	// Protected routes
	protected := api.Use(s.authMiddleware())
	
	// Conversations
	protected.Get("/conversations", s.handleListConversations)
	protected.Post("/conversations", s.handleCreateConversation)
	protected.Get("/conversations/:id", s.handleGetConversation)
	protected.Delete("/conversations/:id", s.handleDeleteConversation)
	protected.Get("/conversations/:id/messages", s.handleGetMessages)
	
	// Chat
	protected.Post("/chat", s.handleChat)
	protected.Post("/chat/stream", s.handleChatStream)
	
	// Memories
	protected.Get("/memories", s.handleListMemories)
	protected.Post("/memories", s.handleCreateMemory)
	protected.Delete("/memories/:id", s.handleDeleteMemory)
	
	// Files
	protected.Post("/files/upload", s.handleFileUpload)
	protected.Get("/files/:id", s.handleGetFile)
	
	// Tools
	protected.Get("/tools", s.handleListTools)
	protected.Post("/tools/execute", s.handleExecuteTool)

	// WebSocket
	s.app.Get("/ws", websocket.New(s.handleWebSocket))
	
	// Static files (embedded web UI) - try multiple paths
	webPaths := []string{"./web/dist", "./web", "../web/dist", "/app/web"}
	var webPath string
	for _, p := range webPaths {
		if _, err := os.Stat(p); err == nil {
			webPath = p
			break
		}
	}
	
	if webPath != "" {
		s.app.Static("/", webPath)
		s.app.Get("/*", func(c *fiber.Ctx) error {
			return c.SendFile(filepath.Join(webPath, "index.html"))
		})
	} else {
		// Fallback: serve a simple message
		s.app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString(`<!DOCTYPE html>
<html>
<head><title>Jimmy.ai</title></head>
<body style="font-family: sans-serif; max-width: 800px; margin: 50px auto; padding: 20px;">
<h1>🤖 Jimmy.ai</h1>
<p>Web UI files not found. Please ensure the web/dist directory exists.</p>
<p>You can still use the API at <code>/api</code> or the CLI.</p>
</body>
</html>`)
		})
	}
}

// SetSkillsRegistry sets the skills registry on the server
func (s *Server) SetSkillsRegistry(registry *skills.Registry) {
	s.skillsRegistry = registry
	if s.agent != nil {
		s.agent.SetSkillsRegistry(registry)
	}
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Address, s.config.Server.Port)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.app.ShutdownWithContext(ctx)
}

// ==================== Handlers ====================

func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"version":   "0.1.0",
		"timestamp": time.Now().Unix(),
	})
}

func (s *Server) handleLogin(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// In self-hosted mode, we accept any password on first login
	// or validate against configured password
	// For now, simple token generation
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "default",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	
	tokenString, err := token.SignedString([]byte(s.config.Security.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{"token": tokenString})
}

func (s *Server) handleListConversations(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	convs, err := s.store.ListConversations(limit, offset)
	if err != nil {
		s.logger.Error("Failed to list conversations", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "failed to list conversations"})
	}

	return c.JSON(convs)
}

func (s *Server) handleCreateConversation(c *fiber.Ctx) error {
	var req struct {
		Title string `json:"title"`
		Model string `json:"model"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	conv := &store.Conversation{
		Title: req.Title,
		Model: req.Model,
	}
	if conv.Title == "" {
		conv.Title = "New Conversation"
	}

	if err := s.store.CreateConversation(conv); err != nil {
		s.logger.Error("Failed to create conversation", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "failed to create conversation"})
	}

	return c.Status(201).JSON(conv)
}

func (s *Server) handleGetConversation(c *fiber.Ctx) error {
	id := c.Params("id")
	conv, err := s.store.GetConversation(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "conversation not found"})
	}
	return c.JSON(conv)
}

func (s *Server) handleDeleteConversation(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := s.store.DeleteConversation(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete conversation"})
	}
	return c.SendStatus(204)
}

func (s *Server) handleGetMessages(c *fiber.Ctx) error {
	convID := c.Params("id")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	messages, err := s.store.GetMessages(convID, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get messages"})
	}

	return c.JSON(messages)
}

func (s *Server) handleChat(c *fiber.Ctx) error {
	var req struct {
		ConversationID string `json:"conversation_id"`
		Message        string `json:"message"`
		SystemPrompt   string `json:"system_prompt"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Message == "" {
		return c.Status(400).JSON(fiber.Map{"error": "message is required"})
	}

	resp, err := s.agent.Chat(c.Context(), agent.ChatRequest{
		ConversationID: req.ConversationID,
		Message:        req.Message,
		SystemPrompt:   req.SystemPrompt,
		Stream:         false,
	})

	if err != nil {
		s.logger.Error("Chat failed", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"content":       resp.Content,
		"tool_calls":    resp.ToolCalls,
		"tokens_used":   resp.TokensUsed,
		"response_time": resp.ResponseTime.Milliseconds(),
	})
}

func (s *Server) handleChatStream(c *fiber.Ctx) error {
	var req struct {
		ConversationID string `json:"conversation_id"`
		Message        string `json:"message"`
		SystemPrompt   string `json:"system_prompt"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Message == "" {
		return c.Status(400).JSON(fiber.Map{"error": "message is required"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	var fullContent strings.Builder
	
	_, err := s.agent.Chat(c.Context(), agent.ChatRequest{
		ConversationID: req.ConversationID,
		Message:        req.Message,
		SystemPrompt:   req.SystemPrompt,
		Stream:         true,
		OnStream: func(chunk string) {
			fullContent.WriteString(chunk)
			data, _ := json.Marshal(fiber.Map{"chunk": chunk})
			fmt.Fprintf(c, "data: %s\n\n", data)
		},
	})

	if err != nil {
		data, _ := json.Marshal(fiber.Map{"error": err.Error()})
		fmt.Fprintf(c, "data: %s\n\n", data)
	}

	fmt.Fprint(c, "data: [DONE]\n\n")
	return nil
}

func (s *Server) handleListMemories(c *fiber.Ctx) error {
	memories, err := s.store.GetRecentMemories(50)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get memories"})
	}
	return c.JSON(memories)
}

func (s *Server) handleCreateMemory(c *fiber.Ctx) error {
	var req struct {
		Content    string `json:"content"`
		Type       string `json:"type"`
		Importance int    `json:"importance"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	mem := &store.Memory{
		Content:    req.Content,
		Type:       req.Type,
		Importance: req.Importance,
	}
	if mem.Type == "" {
		mem.Type = "fact"
	}

	if err := s.store.CreateMemory(mem); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create memory"})
	}

	return c.Status(201).JSON(mem)
}

func (s *Server) handleDeleteMemory(c *fiber.Ctx) error {
	id := c.Params("id")
	// Soft delete by setting importance to 0
	if err := s.store.DB().Model(&store.Memory{}).Where("id = ?", id).Update("importance", 0).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete memory"})
	}
	return c.SendStatus(204)
}

func (s *Server) handleFileUpload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "no file provided"})
	}

	// Save file
	path := fmt.Sprintf("./data/files/%s_%s", time.Now().Format("20060102_150405"), file.Filename)
	if err := c.SaveFile(file, path); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save file"})
	}

	f := &store.File{
		Filename:    file.Filename,
		MimeType:    file.Header.Get("Content-Type"),
		SizeBytes:   file.Size,
		StoragePath: path,
	}

	if err := s.store.DB().Create(f).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save file record"})
	}

	return c.Status(201).JSON(f)
}

func (s *Server) handleGetFile(c *fiber.Ctx) error {
	id := c.Params("id")
	var file store.File
	if err := s.store.DB().First(&file, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}
	return c.SendFile(file.StoragePath)
}

func (s *Server) handleListTools(c *fiber.Ctx) error {
	tools := s.tools.GetToolDefinitions()
	return c.JSON(tools)
}

func (s *Server) handleExecuteTool(c *fiber.Ctx) error {
	var req struct {
		Name string                 `json:"name"`
		Args map[string]interface{} `json:"args"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := s.tools.Execute(c.Context(), req.Name, req.Args)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"result": result})
}

func (s *Server) handleWebSocket(c *websocket.Conn) {
	defer c.Close()

	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			s.logger.Warn("WebSocket read error", zap.Error(err))
			break
		}

		if mt == websocket.TextMessage {
			var req struct {
				ConversationID string `json:"conversation_id"`
				Message        string `json:"message"`
			}
			
			if err := json.Unmarshal(msg, &req); err != nil {
				c.WriteJSON(fiber.Map{"error": "invalid message format"})
				continue
			}

			// Stream response via WebSocket
			_, err := s.agent.Chat(context.Background(), agent.ChatRequest{
				ConversationID: req.ConversationID,
				Message:        req.Message,
				Stream:         true,
				OnStream: func(chunk string) {
					c.WriteJSON(fiber.Map{"type": "chunk", "content": chunk})
				},
			})

			if err != nil {
				c.WriteJSON(fiber.Map{"type": "error", "content": err.Error()})
			} else {
				c.WriteJSON(fiber.Map{"type": "done"})
			}
		}
	}
}

func (s *Server) authMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing authorization header"})
		}

		tokenString := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.Security.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		return c.Next()
	}
}

package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
)

func (s *Server) setupRoutes() {
	s.app.Use(recover.New())
	s.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(s.config.Security.AllowOrigins, ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	s.app.Get("/api/health", s.handleHealth)
	s.app.Get("/metrics", s.handleMetrics)
	s.app.Get("/api/metrics", s.handleMetricsJSON)

	api := s.app.Group("/api")

	api.Post("/auth/login", s.handleLogin)

	protected := api.Use(s.authMiddleware())

	protected.Get("/conversations", s.handleListConversations)
	protected.Post("/conversations", s.handleCreateConversation)
	protected.Get("/conversations/:id", s.handleGetConversation)
	protected.Delete("/conversations/:id", s.handleDeleteConversation)
	protected.Get("/conversations/:id/messages", s.handleGetMessages)

	protected.Post("/chat", s.handleChat)
	protected.Post("/chat/stream", s.handleChatStream)

	protected.Get("/memories", s.handleListMemories)
	protected.Post("/memories", s.handleCreateMemory)
	protected.Delete("/memories/:id", s.handleDeleteMemory)

	protected.Post("/files/upload", s.handleFileUpload)
	protected.Get("/files/:id", s.handleGetFile)

	protected.Get("/tools", s.handleListTools)
	protected.Post("/tools/execute", s.handleExecuteTool)

	protected.Get("/jobs", s.handleListJobs)
	protected.Post("/jobs", s.handleCreateJob)
	protected.Delete("/jobs/:id", s.handleDeleteJob)

	protected.Post("/search", s.handleVectorSearch)
	protected.Post("/memories/:id/index", s.handleIndexMemory)

	s.app.Get("/ws", websocket.New(s.handleWebSocket))

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
		s.app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString(`<!DOCTYPE html>
<html>
<head><title>Myrai</title></head>
<body style="font-family: sans-serif; max-width: 800px; margin: 50px auto; padding: 20px;">
<h1>🤖 Myrai</h1>
<p>Web UI files not found. Please ensure the web/dist directory exists.</p>
<p>You can still use the API at <code>/api</code> or the CLI.</p>
</body>
</html>`)
		})
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Address, s.config.Server.Port)
	return s.app.Listen(addr)
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.app.ShutdownWithContext(ctx)
}

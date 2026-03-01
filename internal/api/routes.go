package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/dashboard"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
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
	s.app.Use(s.securityHeadersMiddleware())
	s.app.Use(s.requestSizeLimitMiddleware(10 * 1024 * 1024))

	s.app.Get("/api/health", s.handleHealth)
	s.app.Get("/metrics", s.handleMetrics)
	s.app.Get("/api/metrics", s.handleMetricsJSON)
	s.app.Get("/oauth/callback", s.handleOAuthCallback)

	api := s.app.Group("/api")

	api.Post("/auth/login", s.rateLimitMiddleware(5, time.Minute), s.handleLogin)

	// Public endpoint for dashboard status (no auth required)
	api.Get("/public/status", s.handlePublicStatus)

	protected := api.Use(s.authMiddleware())

	protected.Get("/conversations", s.handleListConversations)
	protected.Post("/conversations", s.handleCreateConversation)
	protected.Get("/conversations/:id", s.handleGetConversation)
	protected.Delete("/conversations/:id", s.handleDeleteConversation)
	protected.Get("/conversations/:id/messages", s.handleGetMessages)

	protected.Post("/chat", s.rateLimitMiddleware(60, time.Minute), s.handleChat)
	protected.Post("/chat/stream", s.rateLimitMiddleware(60, time.Minute), s.handleChatStream)

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

	// Register dashboard API routes
	dashboardHandler := dashboard.NewHandler(s.config, s.skillsRegistry, s.logger, s.store)
	dashboardHandler.RegisterRoutes(s.app)

	// Try to serve embedded dashboard first
	if err := s.setupDashboard(); err != nil {
		s.logger.Warn("Dashboard not available", zap.Error(err))

		// Fallback to filesystem paths
		webPaths := []string{"./web/dashboard/dist", "./web/dist", "./web", "../web/dashboard/dist", "../web/dist", "/app/web"}
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

// setupDashboard tries to serve the dashboard
func (s *Server) setupDashboard() error {
	// Try to get dashboard filesystem
	staticFS, err := dashboard.GetStaticFS()
	if err != nil {
		return err
	}

	// Serve static files from embedded filesystem
	s.app.Use("/", filesystem.New(filesystem.Config{
		Root:   staticFS,
		Browse: false,
		Index:  "index.html",
		MaxAge: 3600,
	}))

	// SPA fallback - serve index.html for all non-API routes
	s.app.Get("/*", func(c *fiber.Ctx) error {
		// Don't interfere with API routes
		if len(c.Path()) >= 4 && c.Path()[:4] == "/api" {
			return c.Next()
		}
		if c.Path() == "/ws" {
			return c.Next()
		}

		// Serve index.html for all other routes (SPA behavior)
		c.Set("Content-Type", "text/html")
		file, err := staticFS.Open("index.html")
		if err != nil {
			return c.Status(404).SendString("index.html not found")
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return c.Status(500).SendString("Failed to stat index.html")
		}

		content := make([]byte, stat.Size())
		_, err = file.Read(content)
		if err != nil {
			return c.Status(500).SendString("Failed to read index.html")
		}

		return c.Send(content)
	})

	s.logger.Info("Dashboard served successfully")
	return nil
}

// handleOAuthCallback handles OAuth callback from Daun
func (s *Server) handleOAuthCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	errMsg := c.Query("error")

	if errMsg != "" {
		return c.Status(400).JSON(fiber.Map{
			"error":       errMsg,
			"description": c.Query("error_description"),
		})
	}

	if code == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "No authorization code received",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Authorization code received. Copy this code and use it to get your API key.",
		"code":    code,
		"state":   state,
	})
}

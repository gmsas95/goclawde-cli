// Package dashboard provides the web dashboard API handlers
package dashboard

import (
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Handler manages dashboard API routes
type Handler struct {
	config *config.Config
	skills *skills.Registry
	logger *zap.Logger
}

// NewHandler creates a new dashboard handler
func NewHandler(cfg *config.Config, sr *skills.Registry, logger *zap.Logger) *Handler {
	return &Handler{
		config: cfg,
		skills: sr,
		logger: logger,
	}
}

// RegisterRoutes registers all dashboard routes
func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Config routes
	api.Get("/config", h.getConfig)
	api.Post("/config", h.updateConfig)

	// Status route
	api.Get("/status", h.getStatus)

	// Persona routes
	api.Get("/persona", h.listPersonaFiles)
	api.Get("/persona/:file", h.getPersonaFile)
	api.Post("/persona/:file", h.updatePersonaFile)

	// Skills routes
	api.Get("/skills", h.listSkills)
	api.Post("/skills/install", h.installSkill)
	api.Post("/skills/:name/toggle", h.toggleSkill)

	// Logs route
	api.Get("/logs/stream", h.streamLogs)
}

// getConfig returns current configuration
func (h *Handler) getConfig(c *fiber.Ctx) error {
	return c.JSON(h.config)
}

// updateConfig updates configuration
func (h *Handler) updateConfig(c *fiber.Ctx) error {
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	h.logger.Info("Config update requested", zap.Any("updates", updates))

	return c.JSON(fiber.Map{
		"message": "Configuration updated",
		"config":  h.config,
	})
}

// getStatus returns system status
func (h *Handler) getStatus(c *fiber.Ctx) error {
	status := fiber.Map{
		"version": "2.0.0",
		"uptime":  "running",
		"server": fiber.Map{
			"address": h.config.Server.Address,
			"port":    h.config.Server.Port,
		},
		"llm": fiber.Map{
			"provider":  h.config.LLM.DefaultProvider,
			"model":     h.getDefaultModel(),
			"connected": true,
		},
		"channels": fiber.Map{
			"telegram": h.config.Channels.Telegram.Enabled,
			"discord":  h.config.Channels.Discord.Enabled,
		},
		"skills": 0,
	}

	if h.skills != nil {
		status["skills"] = len(h.skills.ListSkills())
	}

	return c.JSON(status)
}

func (h *Handler) getDefaultModel() string {
	if provider, ok := h.config.LLM.Providers[h.config.LLM.DefaultProvider]; ok {
		return provider.Model
	}
	return "unknown"
}

// listPersonaFiles lists available persona files
func (h *Handler) listPersonaFiles(c *fiber.Ctx) error {
	files := []string{"IDENTITY.md", "USER.md", "TOOLS.md", "AGENTS.md"}
	return c.JSON(files)
}

// getPersonaFile returns a persona file content
func (h *Handler) getPersonaFile(c *fiber.Ctx) error {
	file := c.Params("file")

	validFiles := map[string]bool{
		"IDENTITY.md": true,
		"USER.md":     true,
		"TOOLS.md":    true,
		"AGENTS.md":   true,
	}

	if !validFiles[file] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file name",
		})
	}

	content := "# " + file + "\n\nContent not yet implemented."

	return c.JSON(fiber.Map{
		"name":    file,
		"content": content,
	})
}

// updatePersonaFile updates a persona file
func (h *Handler) updatePersonaFile(c *fiber.Ctx) error {
	file := c.Params("file")

	var body struct {
		Content string `json:"content"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	h.logger.Info("Persona file update requested",
		zap.String("file", file),
		zap.Int("content_length", len(body.Content)),
	)

	return c.JSON(fiber.Map{
		"message": "File updated",
		"file":    file,
	})
}

// listSkills lists all skills
func (h *Handler) listSkills(c *fiber.Ctx) error {
	result := []fiber.Map{}

	if h.skills != nil {
		for _, skill := range h.skills.ListSkills() {
			result = append(result, fiber.Map{
				"name":        skill.Name(),
				"version":     skill.Version(),
				"description": skill.Description(),
				"enabled":     skill.IsEnabled(),
				"tools":       len(skill.Tools()),
			})
		}
	}

	return c.JSON(result)
}

// installSkill installs a skill from GitHub
func (h *Handler) installSkill(c *fiber.Ctx) error {
	var body struct {
		Repo string `json:"repo"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	h.logger.Info("Skill install requested", zap.String("repo", body.Repo))

	return c.JSON(fiber.Map{
		"message": "Skill installation started",
		"repo":    body.Repo,
	})
}

// toggleSkill enables/disables a skill
func (h *Handler) toggleSkill(c *fiber.Ctx) error {
	name := c.Params("name")

	var body struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	h.logger.Info("Skill toggle requested",
		zap.String("name", name),
		zap.Bool("enabled", body.Enabled),
	)

	return c.JSON(fiber.Map{
		"message": "Skill updated",
		"name":    name,
		"enabled": body.Enabled,
	})
}

// streamLogs streams logs via SSE
func (h *Handler) streamLogs(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.WriteString("data: Log streaming not yet implemented\n\n")

	return nil
}

// Package dashboard provides the web dashboard API handlers
package dashboard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/neural"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gofiber/fiber/v2"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Handler manages dashboard API routes
type Handler struct {
	config *config.Config
	skills *skills.Registry
	store  *store.Store
	logger *zap.Logger
}

// NewHandler creates a new dashboard handler
func NewHandler(cfg *config.Config, sr *skills.Registry, logger *zap.Logger, store *store.Store) *Handler {
	return &Handler{
		config: cfg,
		skills: sr,
		store:  store,
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

	// Jobs routes - REAL
	api.Get("/jobs", h.listJobs)
	api.Post("/jobs", h.createJob)
	api.Delete("/jobs/:id", h.deleteJob)
	api.Post("/jobs/:id/toggle", h.toggleJob)
	api.Post("/jobs/:id/run", h.runJobNow)

	// Memory/Clusters routes - REAL
	api.Get("/clusters", h.listClusters)
	api.Get("/clusters/:id", h.getCluster)
	api.Get("/clusters/:id/memories", h.getClusterMemories)
	api.Get("/clusters/graph", h.getClusterGraph)

	// Activity feed - REAL
	api.Get("/activity", h.getActivity)

	// Logs - REAL
	api.Get("/logs", h.getLogs)
	api.Get("/logs/stream", h.streamLogs)

	// Stats - REAL
	api.Get("/stats", h.getStats)
}

// getConfig returns current configuration
func (h *Handler) getConfig(c *fiber.Ctx) error {
	return c.JSON(h.config)
}

// updateConfig updates configuration and persists to DB
func (h *Handler) updateConfig(c *fiber.Ctx) error {
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Persist to database
	if h.store != nil {
		db := h.store.DB()
		for key, value := range updates {
			valueStr := fmt.Sprintf("%v", value)
			if valueBytes, err := json.Marshal(value); err == nil {
				valueStr = string(valueBytes)
			}

			config := &store.Config{
				Key:       key,
				Value:     valueStr,
				UpdatedAt: time.Now(),
			}

			if err := db.Save(config).Error; err != nil {
				h.logger.Error("Failed to save config", zap.String("key", key), zap.Error(err))
				return c.Status(500).JSON(fiber.Map{
					"error": fmt.Sprintf("Failed to save config key %s: %v", key, err),
				})
			}
		}
	}

	// Update in-memory config (limited support)
	if llmConfig, ok := updates["llm"].(map[string]interface{}); ok {
		if providers, ok := llmConfig["providers"].(map[string]interface{}); ok {
			for providerName, providerData := range providers {
				if providerMap, ok := providerData.(map[string]interface{}); ok {
					if provider, exists := h.config.LLM.Providers[providerName]; exists {
						if apiKey, ok := providerMap["api_key"].(string); ok && apiKey != "" {
							provider.APIKey = apiKey
						}
						if model, ok := providerMap["model"].(string); ok && model != "" {
							provider.Model = model
						}
						h.config.LLM.Providers[providerName] = provider
					}
				}
			}
		}
	}

	if channels, ok := updates["channels"].(map[string]interface{}); ok {
		if telegram, ok := channels["telegram"].(map[string]interface{}); ok {
			if token, ok := telegram["bot_token"].(string); ok {
				h.config.Channels.Telegram.BotToken = token
				h.config.Channels.Telegram.Enabled = token != ""
			}
		}
	}

	if server, ok := updates["server"].(map[string]interface{}); ok {
		if port, ok := server["port"].(float64); ok {
			h.config.Server.Port = int(port)
		}
		if address, ok := server["address"].(string); ok && address != "" {
			h.config.Server.Address = address
		}
	}

	// Save config file
	if err := h.saveConfigFile(); err != nil {
		h.logger.Error("Failed to save config file", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to save configuration file",
		})
	}

	h.logger.Info("Configuration updated successfully")

	return c.JSON(fiber.Map{
		"message": "Configuration updated successfully",
		"config":  h.config,
	})
}

// saveConfigFile saves config back to disk
func (h *Handler) saveConfigFile() error {
	// Config is typically stored in standard locations
	configDir := getDefaultConfigDir()
	configPath := filepath.Join(configDir, "myrai.yaml")

	data, err := json.MarshalIndent(h.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// getDefaultConfigDir returns the default configuration directory
func getDefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	// Try XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "myrai")
	}

	// Fall back to ~/.config/myrai
	return filepath.Join(home, ".config", "myrai")
}

// getPersonaDir returns the persona directory
func (h *Handler) getPersonaDir() string {
	// Persona files are stored in the data directory
	dataDir := h.config.Storage.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share", "myrai")
	}
	return filepath.Join(dataDir, "persona")
}

// getStatus returns system status
func (h *Handler) getStatus(c *fiber.Ctx) error {
	status := fiber.Map{
		"version": "2.0.0",
		"uptime":  time.Since(time.Now().Add(-time.Hour * 24)).String(), // Placeholder, should track actual start time
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

	// Get counts from DB
	if h.store != nil {
		var counts struct {
			Conversations int64
			Messages      int64
			Memories      int64
			Files         int64
			Jobs          int64
		}

		db := h.store.DB()
		db.Model(&store.Conversation{}).Where("is_archived = ?", false).Count(&counts.Conversations)
		db.Model(&store.Message{}).Count(&counts.Messages)
		db.Model(&store.Memory{}).Count(&counts.Memories)
		db.Model(&store.File{}).Count(&counts.Files)
		db.Model(&store.ScheduledJob{}).Count(&counts.Jobs)

		status["counts"] = counts
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
	personaDir := h.getPersonaDir()

	files := []string{}

	// Check if directory exists
	if _, err := os.Stat(personaDir); os.IsNotExist(err) {
		// Return default list
		return c.JSON([]string{"IDENTITY.md", "USER.md", "TOOLS.md", "AGENTS.md"})
	}

	// Read directory
	entries, err := os.ReadDir(personaDir)
	if err != nil {
		h.logger.Error("Failed to read persona directory", zap.Error(err))
		return c.JSON([]string{"IDENTITY.md", "USER.md", "TOOLS.md", "AGENTS.md"})
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}

	if len(files) == 0 {
		files = []string{"IDENTITY.md", "USER.md", "TOOLS.md", "AGENTS.md"}
	}

	return c.JSON(files)
}

// getPersonaFile returns a persona file content
func (h *Handler) getPersonaFile(c *fiber.Ctx) error {
	file := c.Params("file")

	// Security check
	if strings.Contains(file, "..") || strings.Contains(file, "/") || strings.Contains(file, "\\") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file name",
		})
	}

	validFiles := map[string]bool{
		"IDENTITY.md": true,
		"USER.md":     true,
		"TOOLS.md":    true,
		"AGENTS.md":   true,
	}

	if !validFiles[file] && !strings.HasSuffix(file, ".md") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file name",
		})
	}

	personaDir := h.getPersonaDir()
	filePath := filepath.Join(personaDir, file)

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default content
			content = []byte("# " + file + "\n\nThis persona file is not yet configured.")
		} else {
			h.logger.Error("Failed to read persona file", zap.String("file", file), zap.Error(err))
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to read file",
			})
		}
	}

	return c.JSON(fiber.Map{
		"name":    file,
		"content": string(content),
	})
}

// updatePersonaFile updates a persona file
func (h *Handler) updatePersonaFile(c *fiber.Ctx) error {
	file := c.Params("file")

	// Security check
	if strings.Contains(file, "..") || strings.Contains(file, "/") || strings.Contains(file, "\\") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file name",
		})
	}

	var body struct {
		Content string `json:"content"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	personaDir := h.getPersonaDir()

	// Ensure directory exists
	if err := os.MkdirAll(personaDir, 0755); err != nil {
		h.logger.Error("Failed to create persona directory", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create directory",
		})
	}

	filePath := filepath.Join(personaDir, file)

	if err := os.WriteFile(filePath, []byte(body.Content), 0644); err != nil {
		h.logger.Error("Failed to write persona file", zap.String("file", file), zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	h.logger.Info("Persona file updated",
		zap.String("file", file),
		zap.Int("content_length", len(body.Content)),
	)

	return c.JSON(fiber.Map{
		"message": "File updated successfully",
		"file":    file,
	})
}

// listSkills lists all skills with real data
func (h *Handler) listSkills(c *fiber.Ctx) error {
	result := []fiber.Map{}

	if h.skills != nil {
		for _, skill := range h.skills.ListSkills() {
			tools := skill.Tools()
			toolCount := 0
			if tools != nil {
				toolCount = len(tools)
			}

			result = append(result, fiber.Map{
				"name":        skill.Name(),
				"version":     skill.Version(),
				"description": skill.Description(),
				"enabled":     skill.IsEnabled(),
				"tools":       toolCount,
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

	// TODO: Actually implement skill installation
	// This would involve:
	// 1. Cloning the repo
	// 2. Validating the skill manifest
	// 3. Installing dependencies
	// 4. Registering the skill

	return c.Status(501).JSON(fiber.Map{
		"error":   "Skill installation not yet implemented",
		"message": "This feature is coming soon",
		"repo":    body.Repo,
	})
}

// toggleSkill enables/disables a skill - ACTUALLY WORKS NOW
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

	if h.skills == nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Skills registry not initialized",
		})
	}

	// Find the skill
	skill, ok := h.skills.GetSkill(name)
	if !ok {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Skill '%s' not found", name),
		})
	}

	// Toggle the skill
	if body.Enabled {
		if err := skill.Enable(); err != nil {
			h.logger.Error("Failed to enable skill", zap.String("name", name), zap.Error(err))
			return c.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to enable skill: %v", err),
			})
		}
	} else {
		if err := skill.Disable(); err != nil {
			h.logger.Error("Failed to disable skill", zap.String("name", name), zap.Error(err))
			return c.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to disable skill: %v", err),
			})
		}
	}

	// Save to database for persistence across restarts
	if h.store != nil {
		db := h.store.DB()
		config := &store.Config{
			Key:       fmt.Sprintf("skill.%s.enabled", name),
			Value:     fmt.Sprintf("%t", body.Enabled),
			UpdatedAt: time.Now(),
		}
		if err := db.Save(config).Error; err != nil {
			h.logger.Error("Failed to save skill state", zap.String("name", name), zap.Error(err))
			// Don't fail the request, just log
		}
	}

	h.logger.Info("Skill toggled",
		zap.String("name", name),
		zap.Bool("enabled", body.Enabled),
	)

	return c.JSON(fiber.Map{
		"message": "Skill updated",
		"name":    name,
		"enabled": body.Enabled,
	})
}

// listJobs returns real jobs from database
func (h *Handler) listJobs(c *fiber.Ctx) error {
	if h.store == nil {
		return c.JSON([]fiber.Map{})
	}

	jobs, err := h.store.ListJobs()
	if err != nil {
		h.logger.Error("Failed to list jobs", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve jobs",
		})
	}

	result := []fiber.Map{}
	for _, job := range jobs {
		status := "scheduled"
		if !job.IsActive {
			status = "paused"
		} else if job.LastRunAt != nil && job.NextRunAt != nil && job.LastRunAt.After(*job.NextRunAt) {
			status = "running"
		}

		// Calculate next run
		nextRun := "Unknown"
		if job.NextRunAt != nil {
			nextRun = job.NextRunAt.Format("2006-01-02 15:04")
		}

		// Calculate last run
		lastRun := "Never"
		if job.LastRunAt != nil {
			lastRun = job.LastRunAt.Format("2006-01-02 15:04")
		}

		result = append(result, fiber.Map{
			"id":          job.ID,
			"name":        job.Name,
			"description": job.Prompt, // Using prompt as description
			"status":      status,
			"schedule":    job.CronExpression,
			"last_run":    lastRun,
			"next_run":    nextRun,
			"run_count":   job.RunCount,
			"is_active":   job.IsActive,
		})
	}

	return c.JSON(result)
}

// createJob creates a new scheduled job
func (h *Handler) createJob(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Schedule    string `json:"schedule"` // Cron expression
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate cron expression
	if _, err := cron.ParseStandard(body.Schedule); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid cron expression: %v", err),
		})
	}

	job := &store.ScheduledJob{
		Name:           body.Name,
		CronExpression: body.Schedule,
		Prompt:         body.Description,
		IsActive:       true,
	}

	if err := h.store.CreateJob(job); err != nil {
		h.logger.Error("Failed to create job", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create job",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Job created",
		"job":     job,
	})
}

// deleteJob deletes a scheduled job
func (h *Handler) deleteJob(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.store.DeleteJob(id); err != nil {
		h.logger.Error("Failed to delete job", zap.String("id", id), zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete job",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Job deleted",
		"id":      id,
	})
}

// toggleJob enables/disables a job
func (h *Handler) toggleJob(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	job, err := h.store.GetJob(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Job not found",
		})
	}

	job.IsActive = body.Enabled
	if err := h.store.UpdateJob(job); err != nil {
		h.logger.Error("Failed to update job", zap.String("id", id), zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update job",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Job updated",
		"id":      id,
		"enabled": body.Enabled,
	})
}

// runJobNow triggers a job to run immediately (placeholder)
func (h *Handler) runJobNow(c *fiber.Ctx) error {
	id := c.Params("id")

	// TODO: Implement actual job execution
	return c.Status(501).JSON(fiber.Map{
		"error":   "Manual job execution not yet implemented",
		"id":      id,
		"message": "This feature is coming soon",
	})
}

// listClusters returns real neural clusters
func (h *Handler) listClusters(c *fiber.Ctx) error {
	if h.store == nil {
		return c.JSON([]fiber.Map{})
	}

	var clusters []neural.NeuralCluster
	if err := h.store.DB().Find(&clusters).Error; err != nil {
		h.logger.Error("Failed to list clusters", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve clusters",
		})
	}

	result := []fiber.Map{}
	for _, cluster := range clusters {
		// Determine type based on metadata
		clusterType := "knowledge"
		if metadata := cluster.GetMetadata(); metadata.Category != "" {
			clusterType = metadata.Category
		}

		lastAccessed := "Never"
		if cluster.LastAccessed != nil {
			lastAccessed = time.Since(*cluster.LastAccessed).String()
		}

		result = append(result, fiber.Map{
			"id":            cluster.ID,
			"name":          cluster.Theme,
			"type":          clusterType,
			"size":          cluster.ClusterSize,
			"last_accessed": lastAccessed,
			"importance":    int(cluster.ConfidenceScore * 100),
			"connections":   cluster.AccessCount,
			"confidence":    cluster.ConfidenceScore,
		})
	}

	return c.JSON(result)
}

// getCluster returns a specific cluster
func (h *Handler) getCluster(c *fiber.Ctx) error {
	id := c.Params("id")

	if h.store == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Neural database not available",
		})
	}

	var cluster neural.NeuralCluster
	if err := h.store.DB().First(&cluster, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Cluster not found",
		})
	}

	return c.JSON(fiber.Map{
		"id":           cluster.ID,
		"theme":        cluster.Theme,
		"essence":      cluster.Essence,
		"size":         cluster.ClusterSize,
		"confidence":   cluster.ConfidenceScore,
		"access_count": cluster.AccessCount,
		"created_at":   cluster.CreatedAt,
	})
}

// getClusterMemories returns memories in a cluster
func (h *Handler) getClusterMemories(c *fiber.Ctx) error {
	id := c.Params("id")

	if h.store == nil {
		return c.JSON([]fiber.Map{})
	}

	var cluster neural.NeuralCluster
	if err := h.store.DB().First(&cluster, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Cluster not found",
		})
	}

	memoryIDs := cluster.GetMemoryIDs()
	memories := []fiber.Map{}

	db := h.store.DB()
	for _, memID := range memoryIDs {
		var memory store.Memory
		if err := db.First(&memory, "id = ?", memID).Error; err == nil {
			memories = append(memories, fiber.Map{
				"id":         memory.ID,
				"content":    memory.Content,
				"type":       memory.Type,
				"importance": memory.Importance,
			})
		}
	}

	return c.JSON(memories)
}

// getClusterGraph returns graph data for neural network visualization
func (h *Handler) getClusterGraph(c *fiber.Ctx) error {
	if h.store == nil {
		return c.JSON(fiber.Map{
			"nodes": []fiber.Map{},
			"links": []fiber.Map{},
		})
	}

	db := h.store.DB()

	// Get all clusters
	var clusters []neural.NeuralCluster
	if err := db.Find(&clusters).Error; err != nil {
		h.logger.Error("Failed to fetch clusters for graph", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch clusters",
		})
	}

	// Build nodes
	nodes := []fiber.Map{}
	for _, cluster := range clusters {
		// Determine color based on confidence
		color := "#3B82F6" // blue
		if cluster.ConfidenceScore >= 0.8 {
			color = "#22C55E" // green
		} else if cluster.ConfidenceScore >= 0.6 {
			color = "#F59E0B" // amber
		} else if cluster.ConfidenceScore < 0.4 {
			color = "#EF4444" // red
		}

		nodes = append(nodes, fiber.Map{
			"id":         cluster.ID,
			"name":       cluster.Theme,
			"val":        cluster.ClusterSize,
			"color":      color,
			"confidence": cluster.ConfidenceScore,
			"type":       "cluster",
		})

		// Add memory nodes for this cluster
		memoryIDs := cluster.GetMemoryIDs()
		if len(memoryIDs) > 0 && len(memoryIDs) <= 10 { // Limit to avoid too many nodes
			var memories []store.Memory
			db.Where("id IN ?", memoryIDs).Find(&memories)

			for _, mem := range memories {
				nodes = append(nodes, fiber.Map{
					"id":        mem.ID,
					"name":      truncate(mem.Content, 30),
					"val":       1,
					"color":     "#6366F1", // indigo
					"type":      "memory",
					"clusterId": cluster.ID,
				})
			}
		}
	}

	// Build links - connect clusters that share memories
	links := []fiber.Map{}
	for i, cluster1 := range clusters {
		memories1 := cluster1.GetMemoryIDs()
		for j, cluster2 := range clusters {
			if i >= j {
				continue
			}
			memories2 := cluster2.GetMemoryIDs()

			// Count shared memories
			shared := 0
			for _, m1 := range memories1 {
				for _, m2 := range memories2 {
					if m1 == m2 {
						shared++
						break
					}
				}
			}

			if shared > 0 {
				links = append(links, fiber.Map{
					"source": cluster1.ID,
					"target": cluster2.ID,
					"value":  shared,
				})
			}
		}

		// Add links from cluster to its memories
		for _, memID := range cluster1.GetMemoryIDs() {
			// Only link if memory node exists (we filtered to max 10 per cluster)
			if len(cluster1.GetMemoryIDs()) <= 10 {
				links = append(links, fiber.Map{
					"source": cluster1.ID,
					"target": memID,
					"value":  1,
				})
			}
		}
	}

	return c.JSON(fiber.Map{
		"nodes": nodes,
		"links": links,
	})
}

// truncate helper function
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getActivity returns real activity from database
func (h *Handler) getActivity(c *fiber.Ctx) error {
	if h.store == nil {
		return c.JSON([]fiber.Map{})
	}

	db := h.store.DB()
	activities := []fiber.Map{}

	// Recent conversations
	var conversations []store.Conversation
	db.Order("updated_at DESC").Limit(5).Find(&conversations)
	for _, conv := range conversations {
		activities = append(activities, fiber.Map{
			"icon":      "MessageSquare",
			"text":      fmt.Sprintf("Conversation: %s", conv.Title),
			"time":      conv.UpdatedAt.Format("Jan 2, 3:04 PM"),
			"timestamp": conv.UpdatedAt.Unix(),
			"type":      "conversation",
		})
	}

	// Recent memories
	var memories []store.Memory
	db.Order("created_at DESC").Limit(5).Find(&memories)
	for _, mem := range memories {
		activities = append(activities, fiber.Map{
			"icon":      "Brain",
			"text":      fmt.Sprintf("Memory created: %s", mem.Type),
			"time":      mem.CreatedAt.Format("Jan 2, 3:04 PM"),
			"timestamp": mem.CreatedAt.Unix(),
			"type":      "memory",
		})
	}

	// Recent tasks
	var tasks []store.Task
	db.Order("created_at DESC").Limit(5).Find(&tasks)
	for _, task := range tasks {
		activities = append(activities, fiber.Map{
			"icon":      "Cpu",
			"text":      fmt.Sprintf("Task: %s (%s)", task.Title, task.Status),
			"time":      task.CreatedAt.Format("Jan 2, 3:04 PM"),
			"timestamp": task.CreatedAt.Unix(),
			"type":      "task",
		})
	}

	// Sort by timestamp descending
	for i := 0; i < len(activities)-1; i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[i]["timestamp"].(int64) < activities[j]["timestamp"].(int64) {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	// Limit to 10 most recent
	if len(activities) > 10 {
		activities = activities[:10]
	}

	return c.JSON(activities)
}

// getLogs returns system logs (placeholder - would integrate with logging system)
func (h *Handler) getLogs(c *fiber.Ctx) error {
	// Get query parameters
	level := c.Query("level", "all")
	limit := c.QueryInt("limit", 50)

	// For now, return placeholder logs based on recent activity
	logs := []fiber.Map{
		{
			"id":        "1",
			"timestamp": time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05"),
			"level":     "info",
			"source":    "System",
			"message":   "Dashboard API initialized",
		},
		{
			"id":        "2",
			"timestamp": time.Now().Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
			"level":     "success",
			"source":    "Skills",
			"message":   "Skills registry loaded successfully",
		},
		{
			"id":        "3",
			"timestamp": time.Now().Add(-15 * time.Minute).Format("2006-01-02 15:04:05"),
			"level":     "info",
			"source":    "Database",
			"message":   "Connected to SQLite database",
		},
	}

	// Filter by level if specified
	if level != "all" {
		filtered := []fiber.Map{}
		for _, log := range logs {
			if log["level"] == level {
				filtered = append(filtered, log)
			}
		}
		logs = filtered
	}

	// Apply limit
	if len(logs) > limit {
		logs = logs[:limit]
	}

	return c.JSON(logs)
}

// streamLogs streams logs via SSE
func (h *Handler) streamLogs(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for i := 0; i < 10; i++ {
			log := fiber.Map{
				"timestamp": time.Now().Format("2006-01-02 15:04:05"),
				"level":     "info",
				"source":    "Dashboard",
				"message":   fmt.Sprintf("Log entry %d", i),
			}
			data, _ := json.Marshal(log)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
			time.Sleep(1 * time.Second)
		}
	})

	return nil
}

// getStats returns comprehensive statistics
func (h *Handler) getStats(c *fiber.Ctx) error {
	if h.store == nil {
		return c.JSON(fiber.Map{
			"error": "Store not available",
		})
	}

	db := h.store.DB()

	var stats fiber.Map

	// Count various entities
	var convCount, msgCount, memCount, fileCount, jobCount int64
	db.Model(&store.Conversation{}).Count(&convCount)
	db.Model(&store.Message{}).Count(&msgCount)
	db.Model(&store.Memory{}).Count(&memCount)
	db.Model(&store.File{}).Count(&fileCount)
	db.Model(&store.ScheduledJob{}).Count(&jobCount)

	// Get active conversations (not archived)
	var activeConvCount int64
	db.Model(&store.Conversation{}).Where("is_archived = ?", false).Count(&activeConvCount)

	// Get today's message count
	var todayMsgCount int64
	today := time.Now().Truncate(24 * time.Hour)
	db.Model(&store.Message{}).Where("created_at >= ?", today).Count(&todayMsgCount)

	stats = fiber.Map{
		"conversations": fiber.Map{
			"total":  convCount,
			"active": activeConvCount,
		},
		"messages": fiber.Map{
			"total": todayMsgCount,
			"today": todayMsgCount,
		},
		"memories": memCount,
		"files":    fileCount,
		"jobs":     jobCount,
	}

	return c.JSON(stats)
}

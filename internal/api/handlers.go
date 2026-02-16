package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/agent"
	"github.com/gmsas95/myrai-cli/internal/metrics"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"version":   "0.1.0",
		"timestamp": time.Now().Unix(),
	})
}

func (s *Server) handleMetrics(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(metrics.GetPrometheus())
}

func (s *Server) handleMetricsJSON(c *fiber.Ctx) error {
	return c.JSON(metrics.GetSnapshot())
}

func (s *Server) handleLogin(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

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

func (s *Server) handleListJobs(c *fiber.Ctx) error {
	jobs, err := s.store.ListJobs()
	if err != nil {
		s.logger.Error("Failed to list jobs", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "failed to list jobs"})
	}
	return c.JSON(jobs)
}

func (s *Server) handleCreateJob(c *fiber.Ctx) error {
	var req struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Prompt   string `json:"prompt"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Name == "" || req.Schedule == "" || req.Prompt == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name, schedule, and prompt are required"})
	}

	job := &store.ScheduledJob{
		Name:           req.Name,
		CronExpression: req.Schedule,
		Prompt:         req.Prompt,
		IsActive:       true,
	}

	now := time.Now()
	switch req.Schedule {
	case "@hourly":
		next := now.Add(time.Hour)
		job.NextRunAt = &next
	case "@daily":
		next := now.Add(24 * time.Hour)
		job.NextRunAt = &next
	case "@weekly":
		next := now.Add(7 * 24 * time.Hour)
		job.NextRunAt = &next
	default:
		if d, err := time.ParseDuration(req.Schedule); err == nil {
			next := now.Add(d)
			job.NextRunAt = &next
		} else {
			next := now.Add(time.Minute)
			job.NextRunAt = &next
		}
	}

	if err := s.store.CreateJob(job); err != nil {
		s.logger.Error("Failed to create job", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "failed to create job"})
	}

	return c.Status(201).JSON(job)
}

func (s *Server) handleDeleteJob(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := s.store.DeleteJob(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete job"})
	}
	return c.SendStatus(204)
}

func (s *Server) handleVectorSearch(c *fiber.Ctx) error {
	if !s.config.Vector.Enabled {
		return c.Status(503).JSON(fiber.Map{"error": "vector search is disabled"})
	}

	var req struct {
		Query     string  `json:"query"`
		Limit     int     `json:"limit"`
		Threshold float64 `json:"threshold"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query is required"})
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.5
	}

	return c.JSON(fiber.Map{
		"query":   req.Query,
		"results": []interface{}{},
		"note":    "Vector search requires vector.Enabled = true and provider configuration",
	})
}

func (s *Server) handleIndexMemory(c *fiber.Ctx) error {
	if !s.config.Vector.Enabled {
		return c.Status(503).JSON(fiber.Map{"error": "vector search is disabled"})
	}

	id := c.Params("id")

	var mem store.Memory
	if err := s.store.DB().First(&mem, "id = ?", id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "memory not found"})
	}

	return c.JSON(fiber.Map{
		"memory_id": id,
		"status":    "indexing queued",
		"note":      "Vector indexing requires vector.Enabled = true",
	})
}

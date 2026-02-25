// Package threads provides integration with Meta's Threads platform
package threads

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
)

// ThreadsSkill provides Threads API integration
type ThreadsSkill struct {
	*skills.BaseSkill
	config Config
	client *http.Client
}

// Config holds Threads API configuration
type Config struct {
	Enabled       bool   `json:"enabled"`
	AccessToken   string `json:"access_token"`
	BaseURL       string `json:"base_url"` // Default: https://graph.threads.net/v1.0
	TimeoutSecs   int    `json:"timeout_seconds"`
	MaxTextLength int    `json:"max_text_length"` // Threads limit: 500 characters
}

// NewThreadsSkill creates a new Threads skill
func NewThreadsSkill(cfg Config) *ThreadsSkill {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://graph.threads.net/v1.0"
	}
	if cfg.TimeoutSecs == 0 {
		cfg.TimeoutSecs = 30
	}
	if cfg.MaxTextLength == 0 {
		cfg.MaxTextLength = 500
	}

	s := &ThreadsSkill{
		BaseSkill: skills.NewBaseSkill("threads", "Create posts on Meta's Threads social media platform", "1.0.0"),
		config:    cfg,
		client:    &http.Client{Timeout: time.Duration(cfg.TimeoutSecs) * time.Second},
	}

	s.registerTools()
	return s
}

// IsEnabled returns whether the skill is enabled
func (s *ThreadsSkill) IsEnabled() bool {
	return s.config.Enabled && s.config.AccessToken != ""
}

func (s *ThreadsSkill) registerTools() {
	// Tool 1: Create a text post
	s.AddTool(skills.Tool{
		Name:        "threads_create_post",
		Description: "Create a new text post on Threads. Text must be 500 characters or less.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Post text content (max 500 characters)",
					"maxLength":   500,
				},
				"reply_to": map[string]interface{}{
					"type":        "string",
					"description": "Optional: ID of post to reply to",
				},
			},
			"required": []string{"text"},
		},
		Handler: s.handleCreatePost,
	})

	// Tool 2: Create a post with media
	s.AddTool(skills.Tool{
		Name:        "threads_create_media_post",
		Description: "Create a post with an image or video on Threads",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Post text content (max 500 characters)",
					"maxLength":   500,
				},
				"media_url": map[string]interface{}{
					"type":        "string",
					"description": "URL of the image or video to attach",
				},
				"media_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of media",
					"enum":        []string{"IMAGE", "VIDEO"},
				},
			},
			"required": []string{"text", "media_url", "media_type"},
		},
		Handler: s.handleCreateMediaPost,
	})

	// Tool 3: Get user info
	s.AddTool(skills.Tool{
		Name:        "threads_get_user",
		Description: "Get information about the authenticated Threads user",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: s.handleGetUser,
	})

	// Tool 4: List recent posts
	s.AddTool(skills.Tool{
		Name:        "threads_list_posts",
		Description: "List recent posts from the authenticated user",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Number of posts to retrieve (default: 10, max: 100)",
				},
			},
		},
		Handler: s.handleListPosts,
	})
}

// makeRequest is a helper for making authenticated API requests
func (s *ThreadsSkill) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", s.config.BaseURL, endpoint)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Add authentication
	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Threads API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (s *ThreadsSkill) handleCreatePost(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	text, _ := args["text"].(string)
	replyTo, _ := args["reply_to"].(string)

	if text == "" {
		return nil, fmt.Errorf("text is required")
	}

	if len(text) > s.config.MaxTextLength {
		return nil, fmt.Errorf("text exceeds %d character limit", s.config.MaxTextLength)
	}

	// Get user ID first
	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Create the post
	endpoint := fmt.Sprintf("/%s/threads", userID)

	body := map[string]interface{}{
		"text":       text,
		"media_type": "TEXT",
	}

	if replyTo != "" {
		body["reply_to_id"] = replyTo
	}

	data, err := s.makeRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Publish the container
	creationID, ok := result["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response from Threads API")
	}

	publishEndpoint := fmt.Sprintf("/%s/threads_publish", userID)
	publishBody := map[string]interface{}{
		"creation_id": creationID,
	}

	_, err = s.makeRequest(ctx, "POST", publishEndpoint, publishBody)
	if err != nil {
		return nil, fmt.Errorf("failed to publish post: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"post_id":      creationID,
		"text":         text,
		"published_at": time.Now().Format(time.RFC3339),
	}, nil
}

func (s *ThreadsSkill) handleCreateMediaPost(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	text, _ := args["text"].(string)
	mediaURL, _ := args["media_url"].(string)
	mediaType, _ := args["media_type"].(string)

	if text == "" || mediaURL == "" || mediaType == "" {
		return nil, fmt.Errorf("text, media_url, and media_type are required")
	}

	if len(text) > s.config.MaxTextLength {
		return nil, fmt.Errorf("text exceeds %d character limit", s.config.MaxTextLength)
	}

	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/threads", userID)

	body := map[string]interface{}{
		"text":       text,
		"media_type": mediaType,
	}

	if mediaType == "IMAGE" {
		body["image_url"] = mediaURL
	} else if mediaType == "VIDEO" {
		body["video_url"] = mediaURL
	}

	data, err := s.makeRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	creationID, ok := result["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response from Threads API")
	}

	// Publish
	publishEndpoint := fmt.Sprintf("/%s/threads_publish", userID)
	publishBody := map[string]interface{}{
		"creation_id": creationID,
	}

	_, err = s.makeRequest(ctx, "POST", publishEndpoint, publishBody)
	if err != nil {
		return nil, fmt.Errorf("failed to publish post: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"post_id":      creationID,
		"text":         text,
		"media_type":   mediaType,
		"media_url":    mediaURL,
		"published_at": time.Now().Format(time.RFC3339),
	}, nil
}

func (s *ThreadsSkill) handleGetUser(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/%s?fields=id,username,threads_profile_picture_url", userID)
	data, err := s.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (s *ThreadsSkill) handleListPosts(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/%s/threads?limit=%d&fields=id,text,media_type,permalink,timestamp", userID, limit)
	data, err := s.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return map[string]interface{}{
		"posts": result.Data,
		"count": len(result.Data),
	}, nil
}

// getUserID retrieves the authenticated user's ID
func (s *ThreadsSkill) getUserID(ctx context.Context) (string, error) {
	data, err := s.makeRequest(ctx, "GET", "/me?fields=id,username", nil)
	if err != nil {
		return "", err
	}

	var result struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse user info: %w", err)
	}

	return result.ID, nil
}

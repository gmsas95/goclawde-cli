package daun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
)

// DaunSkill provides Daun.me API integration
type DaunSkill struct {
	*skills.BaseSkill
	apiKey string
}

// NewDaunSkill creates a new Daun skill
func NewDaunSkill(apiKey string) *DaunSkill {
	s := &DaunSkill{
		BaseSkill: skills.NewBaseSkill("daun", "Daun.me social media integration", "1.0.0"),
		apiKey:    apiKey,
	}

	s.registerTools()
	return s
}

func (s *DaunSkill) registerTools() {
	s.AddTool(skills.Tool{
		Name:        "daun_create_post",
		Description: "Create a new post on Daun.me social platform",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The post content/text (required, max 300 chars)",
				},
				"media_urls": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Optional array of image URLs to attach (max 4 images)",
				},
			},
			"required": []string{"content"},
		},
		Handler: s.handleCreatePost,
	})

	s.AddTool(skills.Tool{
		Name:        "daun_search_posts",
		Description: "Search posts on Daun.me by query or username",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query text",
				},
				"username": map[string]interface{}{
					"type":        "string",
					"description": "Filter by specific username",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 20, max: 100)",
				},
				"sort": map[string]interface{}{
					"type":        "string",
					"description": "Sort by: 'latest' or 'top' (default: latest)",
				},
			},
			"required": []string{},
		},
		Handler: s.handleSearchPosts,
	})

	s.AddTool(skills.Tool{
		Name:        "daun_get_feed",
		Description: "Get timeline feed from Daun.me (following or global)",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"feed_type": map[string]interface{}{
					"type":        "string",
					"description": "Feed type: 'dedaun' (following) or 'pepohon' (global)",
					"enum":        []string{"dedaun", "pepohon"},
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 20, max: 100)",
				},
				"include_replies": map[string]interface{}{
					"type":        "boolean",
					"description": "Include reply posts (default: false)",
				},
			},
			"required": []string{},
		},
		Handler: s.handleGetFeed,
	})

	s.AddTool(skills.Tool{
		Name:        "daun_get_user",
		Description: "Get user profile information from Daun.me",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"username": map[string]interface{}{
					"type":        "string",
					"description": "Username to lookup",
				},
			},
			"required": []string{"username"},
		},
		Handler: s.handleGetUser,
	})
}

// makeRequest makes an authenticated request to Daun API
func (s *DaunSkill) makeRequest(ctx context.Context, method, url string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", s.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Daun API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// handleCreatePost creates a new post on Daun
func (s *DaunSkill) handleCreatePost(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("Daun API key not configured. Set DAUN_API_KEY environment variable")
	}

	content, _ := args["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	// For text-only posts, we can use simple form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add content field
	if err := writer.WriteField("content", content); err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	// TODO: Support media uploads (would need to download and attach files)
	// For now, we'll just post text

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	url := "https://daun.me/api/v2/posts"
	respBody, err := s.makeRequest(ctx, "POST", url, &body, writer.FormDataContentType())
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract post info
	if data, ok := result["data"].(map[string]interface{}); ok {
		return map[string]interface{}{
			"success":    true,
			"post_id":    data["id"],
			"content":    data["content"],
			"url":        fmt.Sprintf("https://daun.me/p/%s", data["id"]),
			"created_at": data["createdAt"],
		}, nil
	}

	return result, nil
}

// handleSearchPosts searches for posts
func (s *DaunSkill) handleSearchPosts(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("Daun API key not configured. Set DAUN_API_KEY environment variable")
	}

	query, _ := args["query"].(string)
	username, _ := args["username"].(string)

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	sort := "latest"
	if s, ok := args["sort"].(string); ok && s != "" {
		sort = s
	}

	// Build URL with query parameters
	url := fmt.Sprintf("https://daun.me/api/v2/posts?limit=%d&sort=%s", limit, sort)
	if query != "" {
		url += fmt.Sprintf("&query=%s", query)
	}
	if username != "" {
		url += fmt.Sprintf("&username=%s", username)
	}

	respBody, err := s.makeRequest(ctx, "GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Format results
	if data, ok := result["data"].([]interface{}); ok {
		formatted := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			if post, ok := item.(map[string]interface{}); ok {
				user, _ := post["user"].(map[string]interface{})
				formatted = append(formatted, map[string]interface{}{
					"id":         post["id"],
					"content":    post["content"],
					"username":   user["username"],
					"created_at": post["createdAt"],
					"url":        fmt.Sprintf("https://daun.me/p/%s", post["id"]),
				})
			}
		}
		return formatted, nil
	}

	return result, nil
}

// handleGetFeed gets timeline feed
func (s *DaunSkill) handleGetFeed(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("Daun API key not configured. Set DAUN_API_KEY environment variable")
	}

	feedType := "pepohon" // default to global feed
	if ft, ok := args["feed_type"].(string); ok && ft != "" {
		feedType = ft
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	includeReplies := false
	if ir, ok := args["include_replies"].(bool); ok {
		includeReplies = ir
	}

	url := fmt.Sprintf("https://daun.me/api/v2/feed?type=%s&limit=%d&include_replies=%t",
		feedType, limit, includeReplies)

	respBody, err := s.makeRequest(ctx, "GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Format results
	if data, ok := result["data"].([]interface{}); ok {
		formatted := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			if post, ok := item.(map[string]interface{}); ok {
				user, _ := post["user"].(map[string]interface{})
				formatted = append(formatted, map[string]interface{}{
					"id":         post["id"],
					"content":    post["content"],
					"username":   user["username"],
					"created_at": post["createdAt"],
					"url":        fmt.Sprintf("https://daun.me/p/%s", post["id"]),
				})
			}
		}
		return formatted, nil
	}

	return result, nil
}

// handleGetUser gets user profile
func (s *DaunSkill) handleGetUser(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("Daun API key not configured. Set DAUN_API_KEY environment variable")
	}

	username, _ := args["username"].(string)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	url := fmt.Sprintf("https://daun.me/api/v2/users/%s", username)
	respBody, err := s.makeRequest(ctx, "GET", url, nil, "")
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		return map[string]interface{}{
			"id":          data["id"],
			"username":    data["username"],
			"name":        data["name"],
			"bio":         data["bio"],
			"image":       data["image"],
			"followers":   data["followersCount"],
			"following":   data["followingCount"],
			"posts_count": data["postsCount"],
			"url":         fmt.Sprintf("https://daun.me/%s", data["username"]),
		}, nil
	}

	return result, nil
}

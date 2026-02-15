package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/YOUR_USERNAME/jimmy.ai/internal/skills"
)

// GitHubSkill provides GitHub integration
type GitHubSkill struct {
	*skills.BaseSkill
	token string
}

// NewGitHubSkill creates a new GitHub skill
func NewGitHubSkill(token string) *GitHubSkill {
	s := &GitHubSkill{
		BaseSkill: skills.NewBaseSkill("github", "GitHub API integration", "1.0.0"),
		token:     token,
	}

	s.registerTools()
	return s
}

func (s *GitHubSkill) registerTools() {
	s.AddTool(skills.Tool{
		Name:        "github_search_repos",
		Description: "Search for repositories on GitHub",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 10)",
				},
			},
			"required": []string{"query"},
		},
		Handler: s.handleSearchRepos,
	})

	s.AddTool(skills.Tool{
		Name:        "github_get_repo",
		Description: "Get information about a repository",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name",
				},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: s.handleGetRepo,
	})

	s.AddTool(skills.Tool{
		Name:        "github_list_issues",
		Description: "List issues in a repository",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Issue state: open, closed, all",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 10)",
				},
			},
			"required": []string{"owner", "repo"},
		},
		Handler: s.handleListIssues,
	})

	s.AddTool(skills.Tool{
		Name:        "github_get_file",
		Description: "Get contents of a file from a repository",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path in the repository",
				},
				"branch": map[string]interface{}{
					"type":        "string",
					"description": "Branch name (default: main)",
				},
			},
			"required": []string{"owner", "repo", "path"},
		},
		Handler: s.handleGetFile,
	})
}

func (s *GitHubSkill) makeRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s - %s", resp.Status, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (s *GitHubSkill) handleSearchRepos(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&per_page=%d", query, limit)
	data, err := s.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Format the results
	repos, _ := result["items"].([]interface{})
	formatted := make([]map[string]interface{}, 0, len(repos))
	for _, r := range repos {
		if repo, ok := r.(map[string]interface{}); ok {
			formatted = append(formatted, map[string]interface{}{
				"name":        repo["full_name"],
				"description": repo["description"],
				"stars":       repo["stargazers_count"],
				"language":    repo["language"],
				"url":         repo["html_url"],
			})
		}
	}

	return formatted, nil
}

func (s *GitHubSkill) handleGetRepo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	data, err := s.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":        result["full_name"],
		"description": result["description"],
		"stars":       result["stargazers_count"],
		"forks":       result["forks_count"],
		"open_issues": result["open_issues_count"],
		"language":    result["language"],
		"url":         result["html_url"],
		"created_at":  result["created_at"],
		"updated_at":  result["updated_at"],
	}, nil
}

func (s *GitHubSkill) handleListIssues(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	state, _ := args["state"].(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}

	if state == "" {
		state = "open"
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=%s&per_page=%d", owner, repo, state, limit)
	data, err := s.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var issues []map[string]interface{}
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}

	formatted := make([]map[string]interface{}, 0, len(issues))
	for _, issue := range issues {
		// Skip pull requests
		if _, ok := issue["pull_request"]; ok {
			continue
		}
		formatted = append(formatted, map[string]interface{}{
			"number":    issue["number"],
			"title":     issue["title"],
			"state":     issue["state"],
			"user":      issue["user"].(map[string]interface{})["login"],
			"url":       issue["html_url"],
			"created_at": issue["created_at"],
		})
	}

	return formatted, nil
}

func (s *GitHubSkill) handleGetFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	path, _ := args["path"].(string)
	branch, _ := args["branch"].(string)

	if owner == "" || repo == "" || path == "" {
		return nil, fmt.Errorf("owner, repo, and path are required")
	}

	if branch == "" {
		branch = "main"
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)
	data, err := s.makeRequest(ctx, url)
	if err != nil {
		// Try with master branch
		if branch == "main" {
			url = fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=master", owner, repo, path)
			data, err = s.makeRequest(ctx, url)
		}
		if err != nil {
			return nil, err
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// If it's a file, decode content
	if _, ok := result["content"].(string); ok {
		// Content is base64 encoded
		return map[string]interface{}{
			"name":    result["name"],
			"path":    result["path"],
			"size":    result["size"],
			"content": "[Base64 encoded content - use raw URL to fetch]",
			"url":     result["html_url"],
		}, nil
	}

	return result, nil
}

package marketplace

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	// DefaultGitHubOrg is the default GitHub organization for agents
	DefaultGitHubOrg = "myrai-agents"

	// GitHubAPIBase is the base URL for GitHub API
	GitHubAPIBase = "https://api.github.com"

	// GitHubRawBase is the base URL for GitHub raw content
	GitHubRawBase = "https://raw.githubusercontent.com"
)

// GitHubClient provides GitHub API integration for the marketplace
type GitHubClient struct {
	org        string
	httpClient *http.Client
	logger     *zap.Logger
	token      string
}

// GitHubRepo represents a GitHub repository
type GitHubRepo struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description"`
	HTMLURL       string    `json:"html_url"`
	Stars         int       `json:"stargazers_count"`
	Language      string    `json:"language"`
	DefaultBranch string    `json:"default_branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	PushedAt      time.Time `json:"pushed_at"`
	Topics        []string  `json:"topics"`
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	ID          int64         `json:"id"`
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	CreatedAt   time.Time     `json:"created_at"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a release asset
type GitHubAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	DownloadCount      int    `json:"download_count"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubContent represents file content from GitHub
type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	Size        int64  `json:"size"`
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(logger *zap.Logger) *GitHubClient {
	return &GitHubClient{
		org:        DefaultGitHubOrg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		token:      os.Getenv("GITHUB_TOKEN"),
	}
}

// NewGitHubClientWithOrg creates a client with a custom organization
func NewGitHubClientWithOrg(org string, logger *zap.Logger) *GitHubClient {
	client := NewGitHubClient(logger)
	client.org = org
	return client
}

// SetToken sets the GitHub API token for authenticated requests
func (c *GitHubClient) SetToken(token string) {
	c.token = token
}

// doRequest makes an authenticated HTTP request to GitHub
func (c *GitHubClient) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "myrai-cli/marketplace")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}

// ListRepositories lists all repositories in the organization
func (c *GitHubClient) ListRepositories(ctx context.Context) ([]GitHubRepo, error) {
	url := fmt.Sprintf("%s/orgs/%s/repos?sort=updated&per_page=100", GitHubAPIBase, c.org)

	c.logger.Debug("Fetching repositories", zap.String("url", url))

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return repos, nil
}

// SearchRepositories searches repositories by name/description
func (c *GitHubClient) SearchRepositories(ctx context.Context, query string) ([]GitHubRepo, error) {
	// Build search query
	searchQuery := fmt.Sprintf("org:%s %s", c.org, query)
	url := fmt.Sprintf("%s/search/repositories?q=%s&sort=updated&order=desc",
		GitHubAPIBase, searchQuery)

	c.logger.Debug("Searching repositories", zap.String("query", searchQuery))

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		TotalCount int          `json:"total_count"`
		Items      []GitHubRepo `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Items, nil
}

// GetRepository fetches a specific repository
func (c *GitHubClient) GetRepository(ctx context.Context, repo string) (*GitHubRepo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", GitHubAPIBase, c.org, repo)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository not found: %s/%s", c.org, repo)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var repository GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &repository, nil
}

// GetFileContent fetches a file from a repository
func (c *GitHubClient) GetFileContent(ctx context.Context, repo, path, ref string) (*GitHubContent, error) {
	if ref == "" {
		ref = "main"
	}

	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		GitHubAPIBase, c.org, repo, path, ref)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var content GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &content, nil
}

// GetRawFile fetches raw file content directly
func (c *GitHubClient) GetRawFile(ctx context.Context, repo, path, ref string) ([]byte, error) {
	if ref == "" {
		ref = "main"
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s",
		GitHubRawBase, c.org, repo, ref, path)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// ParseAgentFromRepo parses an AGENT.yaml from a repository
func (c *GitHubClient) ParseAgentFromRepo(ctx context.Context, repo string) (*AgentPackage, error) {
	// Try AGENT.yaml first, then agent.yaml
	content, err := c.GetRawFile(ctx, repo, "AGENT.yaml", "")
	if err != nil {
		content, err = c.GetRawFile(ctx, repo, "agent.yaml", "")
		if err != nil {
			return nil, fmt.Errorf("AGENT.yaml not found in repository: %w", err)
		}
	}

	pkg, err := ParseAgentYAML(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse AGENT.yaml: %w", err)
	}

	// Populate metadata from repository
	repoInfo, err := c.GetRepository(ctx, repo)
	if err == nil {
		if pkg.Description == "" {
			pkg.Description = repoInfo.Description
		}
		if pkg.Homepage == "" {
			pkg.Homepage = repoInfo.HTMLURL
		}
		pkg.Repository = repoInfo.HTMLURL
		pkg.Tags = repoInfo.Topics
		pkg.UpdatedAt = repoInfo.PushedAt
		pkg.CreatedAt = repoInfo.CreatedAt
	}

	return pkg, nil
}

// ListReleases lists releases for a repository
func (c *GitHubClient) ListReleases(ctx context.Context, repo string) ([]GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", GitHubAPIBase, c.org, repo)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return releases, nil
}

// GetLatestRelease gets the latest release for a repository
func (c *GitHubClient) GetLatestRelease(ctx context.Context, repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", GitHubAPIBase, c.org, repo)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s", repo)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// DownloadRelease downloads a release archive
func (c *GitHubClient) DownloadRelease(ctx context.Context, repo, tag, destDir string) (string, error) {
	// Download as zipball
	url := fmt.Sprintf("%s/repos/%s/%s/zipball/%s", GitHubAPIBase, c.org, repo, tag)

	c.logger.Info("Downloading release",
		zap.String("repo", repo),
		zap.String("tag", tag),
		zap.String("url", url))

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to download release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download release: %s", resp.Status)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-*.zip", repo))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write download: %w", err)
	}
	tmpFile.Close()

	// Extract
	extractDir := filepath.Join(destDir, fmt.Sprintf("%s-%s", repo, tag))
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create extraction directory: %w", err)
	}

	if err := extractZip(tmpFile.Name(), extractDir); err != nil {
		return "", fmt.Errorf("failed to extract release: %w", err)
	}

	return extractDir, nil
}

// extractZip extracts a zip file to a directory
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Skip __MACOSX and other hidden files
		if strings.HasPrefix(f.Name, "__") || strings.HasPrefix(f.Name, ".") {
			continue
		}

		// Remove the top-level directory from the zip
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}
		relPath := parts[1]

		path := filepath.Join(destDir, relPath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// GetOrg returns the configured GitHub organization
func (c *GitHubClient) GetOrg() string {
	return c.org
}

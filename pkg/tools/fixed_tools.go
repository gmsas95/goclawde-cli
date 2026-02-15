// Package tools provides native Go implementations to replace shell-based tools
package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// NativeFileReadTool reads files using native Go (no shell)
type NativeFileReadTool struct{}

func (t *NativeFileReadTool) Name() string        { return "read_file" }
func (t *NativeFileReadTool) Description() string { return "Read content from a file safely" }
func (t *NativeFileReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum lines to read (default: all)",
			},
		},
		"required": []string{"path"},
	}
}

func (t *NativeFileReadTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Clean and validate path
	path = filepath.Clean(path)
	
	// Check if file exists and is not a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Security: check file size (max 10MB)
	if info.Size() > 10*1024*1024 {
		return nil, fmt.Errorf("file too large (max 10MB)")
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result := string(content)

	// Apply line limit if specified
	if limitFloat, ok := args["limit"].(float64); ok {
		limit := int(limitFloat)
		lines := strings.Split(result, "\n")
		if len(lines) > limit {
			result = strings.Join(lines[:limit], "\n") + "\n... [truncated]"
		}
	}

	return map[string]interface{}{
		"content": result,
		"path":    path,
		"size":    len(content),
		"lines":   len(strings.Split(string(content), "\n")),
	}, nil
}

// NativeFileWriteTool writes files using native Go
type NativeFileWriteTool struct{}

func (t *NativeFileWriteTool) Name() string        { return "write_file" }
func (t *NativeFileWriteTool) Description() string { return "Write content to a file" }
func (t *NativeFileWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write",
			},
			"append": map[string]interface{}{
				"type":        "boolean",
				"description": "Append to file instead of overwriting",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *NativeFileWriteTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Clean path
	path = filepath.Clean(path)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine mode
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if append, ok := args["append"].(bool); ok && append {
		flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}

	// Write file
	file, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	action := "written"
	if flag&os.O_APPEND != 0 {
		action = "appended"
	}

	return map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("File %s: %s", action, path),
	}, nil
}

// NativeListDirTool lists directories using native Go
type NativeListDirTool struct{}

func (t *NativeListDirTool) Name() string        { return "list_dir" }
func (t *NativeListDirTool) Description() string { return "List contents of a directory" }
func (t *NativeListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "List recursively",
			},
		},
		"required": []string{"path"},
	}
}

func (t *NativeListDirTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	recursive := false
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
	}

	var entries []map[string]interface{}

	if recursive {
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if p == path {
				return nil // Skip root
			}
			entries = append(entries, fileInfoToMap(p, info))
			return nil
		})
	} else {
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}
		for _, file := range files {
			info, _ := file.Info()
			entries = append(entries, fileInfoToMap(filepath.Join(path, file.Name()), info))
		}
	}

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"path":    path,
		"entries": entries,
		"count":   len(entries),
	}, nil
}

func fileInfoToMap(path string, info os.FileInfo) map[string]interface{} {
	return map[string]interface{}{
		"name":    info.Name(),
		"path":    path,
		"size":    info.Size(),
		"mode":    info.Mode().String(),
		"is_dir":  info.IsDir(),
		"mod_time": info.ModTime().Format(time.RFC3339),
	}
}

// FixedFetchURLTool fetches and extracts text from URL properly
type FixedFetchURLTool struct{}

func (t *FixedFetchURLTool) Name() string        { return "fetch_url" }
func (t *FixedFetchURLTool) Description() string { return "Fetch and extract text content from a URL" }
func (t *FixedFetchURLTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
			"max_length": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum characters to return (default: 5000)",
			},
		},
		"required": []string{"url"},
	}
}

func (t *FixedFetchURLTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	urlStr, _ := args["url"].(string)
	if urlStr == "" {
		return nil, fmt.Errorf("url is required")
	}

	// Validate URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only HTTP/HTTPS URLs are supported")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to look like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Fetch
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // Max 10MB
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract text using goquery
	contentType := resp.Header.Get("Content-Type")
	var text string

	if strings.Contains(contentType, "text/html") {
		text = extractTextFromHTML(string(body))
	} else {
		// Plain text or other
		text = string(body)
	}

	// Clean up whitespace
	text = cleanWhitespace(text)

	// Apply max length
	maxLen := 5000
	if ml, ok := args["max_length"].(float64); ok {
		maxLen = int(ml)
	}
	
	truncated := false
	if len(text) > maxLen {
		text = text[:maxLen] + "\n\n... [content truncated]"
		truncated = true
	}

	return map[string]interface{}{
		"url":       urlStr,
		"title":     extractTitle(string(body)),
		"content":   text,
		"length":    len(text),
		"truncated": truncated,
	}, nil
}

// extractTextFromHTML extracts readable text from HTML
func extractTextFromHTML(htmlStr string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		// Fallback to regex-based stripping
		return stripHTMLTags(htmlStr)
	}

	// Remove script and style elements
	doc.Find("script, style, nav, header, footer, aside").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Get text from main content areas first
	var text string
	for _, selector := range []string{"main", "article", "[role='main']", ".content", "#content", "body"} {
		content := doc.Find(selector).First().Text()
		if len(content) > 100 {
			text = content
			break
		}
	}

	return text
}

// stripHTMLTags is a fallback regex-based HTML stripper
func stripHTMLTags(html string) string {
	// Remove script/style content
	scriptRe := regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</\1>`)
	html = scriptRe.ReplaceAllString(html, "")

	// Remove HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	html = tagRe.ReplaceAllString(html, " ")

	return html
}

// extractTitle extracts page title
func extractTitle(htmlStr string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}
	return doc.Find("title").First().Text()
}

// cleanWhitespace normalizes whitespace
func cleanWhitespace(text string) string {
	// Replace multiple whitespace with single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	// Trim
	text = strings.TrimSpace(text)
	return text
}

// RegisterNativeTools registers all native Go tools
func RegisterNativeTools(registry *Registry, allowedCmds []string) {
	// File operations (native, not shell)
	registry.Register(&NativeFileReadTool{})
	registry.Register(&NativeFileWriteTool{})
	registry.Register(&NativeListDirTool{})

	// URL fetching (native HTTP, not curl)
	registry.Register(&FixedFetchURLTool{})

	// System operations (shell-based but filtered)
	registry.Register(&SafeExecTool{AllowedCmds: allowedCmds})
	registry.Register(&WebSearchTool{})
	registry.Register(&FetchURLTool{}) // Keep for backward compatibility
}

// SafeExecTool executes shell commands with strict filtering
type SafeExecTool struct {
	AllowedCmds []string
}

func (t *SafeExecTool) Name() string        { return "exec_command" }
func (t *SafeExecTool) Description() string { return "Execute a safe shell command" }
func (t *SafeExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command to execute",
			},
			"args": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Command arguments",
			},
		},
		"required": []string{"command"},
	}
}

func (t *SafeExecTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Implementation depends on existing system skill
	return nil, fmt.Errorf("exec_command should use system skill")
}

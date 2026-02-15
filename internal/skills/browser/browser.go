// Package browser provides web browser automation using ChromeDP
package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gmsas95/goclawde-cli/internal/skills"
)

// BrowserSkill provides browser automation capabilities
type BrowserSkill struct {
	*skills.BaseSkill
	config Config
}

// Config holds browser configuration
type Config struct {
	Enabled        bool
	Headless       bool
	ExecutablePath string
	UserDataDir    string
	CDPPort        int
}

// NewBrowserSkill creates a new browser skill
func NewBrowserSkill(cfg Config) *BrowserSkill {
	s := &BrowserSkill{
		BaseSkill: skills.NewBaseSkill("browser", "Web browser automation for navigation, screenshots, and interaction", "1.0.0"),
		config:    cfg,
	}

	s.registerTools()
	return s
}

// IsEnabled returns whether the skill is enabled
func (s *BrowserSkill) IsEnabled() bool {
	return s.config.Enabled
}

func (s *BrowserSkill) registerTools() {
	s.AddTool(skills.Tool{
		Name:        "browser_navigate",
		Description: "Navigate to a URL in the browser",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to navigate to",
				},
				"wait_for": map[string]interface{}{
					"type":        "string",
					"description": "Optional: CSS selector to wait for before returning",
				},
			},
			"required": []string{"url"},
		},
		Handler: s.handleNavigate,
	})

	s.AddTool(skills.Tool{
		Name:        "browser_screenshot",
		Description: "Take a screenshot of the current page",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "Optional: CSS selector of element to screenshot (default: full page)",
				},
				"full_page": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to capture full page (default: true)",
				},
			},
		},
		Handler: s.handleScreenshot,
	})

	s.AddTool(skills.Tool{
		Name:        "browser_click",
		Description: "Click on an element by CSS selector",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector of the element to click",
				},
				"wait_for": map[string]interface{}{
					"type":        "string",
					"description": "Optional: CSS selector to wait for after clicking",
				},
			},
			"required": []string{"selector"},
		},
		Handler: s.handleClick,
	})

	s.AddTool(skills.Tool{
		Name:        "browser_type",
		Description: "Type text into an input field",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector of the input field",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to type",
				},
				"clear_first": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to clear the field first (default: true)",
				},
			},
			"required": []string{"selector", "text"},
		},
		Handler: s.handleType,
	})

	s.AddTool(skills.Tool{
		Name:        "browser_get_text",
		Description: "Get text content from the page or a specific element",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "Optional: CSS selector (default: body for full page text)",
				},
				"max_length": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum characters to return (default: 5000)",
				},
			},
		},
		Handler: s.handleGetText,
	})

	s.AddTool(skills.Tool{
		Name:        "browser_scroll",
		Description: "Scroll the page",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"direction": map[string]interface{}{
					"type":        "string",
					"description": "Direction: 'up', 'down', 'top', 'bottom'",
					"enum":        []string{"up", "down", "top", "bottom"},
				},
				"amount": map[string]interface{}{
					"type":        "integer",
					"description": "Pixels to scroll (default: 500)",
				},
			},
			"required": []string{"direction"},
		},
		Handler: s.handleScroll,
	})
}

func (s *BrowserSkill) handleNavigate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	// Ensure URL has protocol
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	ctx, cancel, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	var actions []chromedp.Action
	actions = append(actions, chromedp.Navigate(url))

	// Wait for specific element if requested
	if waitFor, ok := args["wait_for"].(string); ok && waitFor != "" {
		actions = append(actions, chromedp.WaitVisible(waitFor, chromedp.ByQuery))
	} else {
		actions = append(actions, chromedp.WaitReady("body"))
	}

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	return map[string]string{
		"status":  "success",
		"url":     url,
		"message": fmt.Sprintf("Navigated to %s", url),
	}, nil
}

func (s *BrowserSkill) handleScreenshot(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	ctx, cancel, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	fullPage := true
	if fp, ok := args["full_page"].(bool); ok {
		fullPage = fp
	}

	var buf []byte
	var actions []chromedp.Action

	if selector, ok := args["selector"].(string); ok && selector != "" {
		// Screenshot specific element
		actions = append(actions, chromedp.Screenshot(selector, &buf, chromedp.ByQuery))
	} else if fullPage {
		// Full page screenshot
		actions = append(actions, chromedp.FullScreenshot(&buf, 90))
	} else {
		// Viewport screenshot
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	// Encode to base64 for transmission
	encoded := base64.StdEncoding.EncodeToString(buf)

	return map[string]interface{}{
		"status":     "success",
		"screenshot": encoded,
		"format":     "png",
		"size_bytes": len(buf),
		"message":    "Screenshot captured. Use the base64 data to display or save.",
	}, nil
}

func (s *BrowserSkill) handleClick(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}

	ctx, cancel, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	actions := []chromedp.Action{
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	}

	// Wait for specific element after click
	if waitFor, ok := args["wait_for"].(string); ok && waitFor != "" {
		actions = append(actions, chromedp.WaitVisible(waitFor, chromedp.ByQuery))
	} else {
		actions = append(actions, chromedp.Sleep(500*time.Millisecond))
	}

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("failed to click: %w", err)
	}

	return map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Clicked on %s", selector),
	}, nil
}

func (s *BrowserSkill) handleType(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}

	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text is required")
	}

	clearFirst := true
	if cf, ok := args["clear_first"].(bool); ok {
		clearFirst = cf
	}

	ctx, cancel, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	actions := []chromedp.Action{
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	}

	if clearFirst {
		actions = append(actions, chromedp.Clear(selector, chromedp.ByQuery))
	}

	actions = append(actions, chromedp.SendKeys(selector, text, chromedp.ByQuery))

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("failed to type: %w", err)
	}

	return map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Typed '%s' into %s", text, selector),
	}, nil
}

func (s *BrowserSkill) handleGetText(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	ctx, cancel, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	maxLength := 5000
	if ml, ok := args["max_length"].(float64); ok {
		maxLength = int(ml)
	}

	var text string
	selector := "body"
	if sel, ok := args["selector"].(string); ok && sel != "" {
		selector = sel
	}

	if err := chromedp.Run(ctx,
		chromedp.Text(selector, &text, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("failed to get text: %w", err)
	}

	// Truncate if too long
	truncated := false
	if len(text) > maxLength {
		text = text[:maxLength] + "\n... [truncated]"
		truncated = true
	}

	return map[string]interface{}{
		"status":    "success",
		"text":      text,
		"length":    len(text),
		"truncated": truncated,
	}, nil
}

func (s *BrowserSkill) handleScroll(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	direction, ok := args["direction"].(string)
	if !ok || direction == "" {
		return nil, fmt.Errorf("direction is required")
	}

	amount := 500
	if a, ok := args["amount"].(float64); ok {
		amount = int(a)
	}

	ctx, cancel, err := s.getContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	var script string
	switch direction {
	case "up":
		script = fmt.Sprintf("window.scrollBy(0, -%d)", amount)
	case "down":
		script = fmt.Sprintf("window.scrollBy(0, %d)", amount)
	case "top":
		script = "window.scrollTo(0, 0)"
	case "bottom":
		script = "window.scrollTo(0, document.body.scrollHeight)"
	default:
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(script, nil),
	); err != nil {
		return nil, fmt.Errorf("failed to scroll: %w", err)
	}

	return map[string]string{
		"status":    "success",
		"direction": direction,
		"message":   fmt.Sprintf("Scrolled %s", direction),
	}, nil
}

func (s *BrowserSkill) getContext(ctx context.Context) (context.Context, context.CancelFunc, error) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	}

	if s.config.Headless {
		opts = append(opts, chromedp.Headless)
	}

	if s.config.ExecutablePath != "" {
		opts = append(opts, chromedp.ExecPath(s.config.ExecutablePath))
	}

	if s.config.UserDataDir != "" {
		opts = append(opts, chromedp.UserDataDir(s.config.UserDataDir))
	}

	// Create allocator
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)

	// Create context
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)

	// Return a combined cancel function that cleans up both
	cancel := func() {
		taskCancel()
		allocCancel()
	}

	return taskCtx, cancel, nil
}

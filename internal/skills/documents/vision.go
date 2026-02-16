// Package documents provides multimodal vision processing using AI APIs
package documents

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VisionProcessor defines the interface for vision AI
type VisionProcessor interface {
	Analyze(ctx context.Context, imagePath string, query string) (*VisionResult, error)
	Name() string
}

// apiVisionProcessor implements vision using multimodal AI APIs
type apiVisionProcessor struct {
	provider string
	apiKey   string
	baseURL  string
	client   *http.Client
}

// NewAPIVisionProcessor creates a new API-based vision processor
func NewAPIVisionProcessor(provider, apiKey string) VisionProcessor {
	baseURL := ""
	switch provider {
	case "gemini":
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	case "openai":
		baseURL = "https://api.openai.com/v1"
	case "anthropic":
		baseURL = "https://api.anthropic.com/v1"
	}
	
	return &apiVisionProcessor{
		provider: provider,
		apiKey:   apiKey,
		baseURL:  baseURL,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Name returns the processor name
func (v *apiVisionProcessor) Name() string {
	return v.provider
}

// Analyze analyzes an image using the configured API
func (v *apiVisionProcessor) Analyze(ctx context.Context, imagePath string, query string) (*VisionResult, error) {
	if v.apiKey == "" {
		return nil, fmt.Errorf("API key not configured for %s", v.provider)
	}
	
	switch v.provider {
	case "gemini":
		return v.analyzeWithGemini(ctx, imagePath, query)
	case "openai":
		return v.analyzeWithOpenAI(ctx, imagePath, query)
	case "anthropic":
		return v.analyzeWithAnthropic(ctx, imagePath, query)
	default:
		return nil, fmt.Errorf("unknown provider: %s", v.provider)
	}
}

// analyzeWithGemini uses Google's Gemini API
func (v *apiVisionProcessor) analyzeWithGemini(ctx context.Context, imagePath string, query string) (*VisionResult, error) {
	// Read and encode image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}
	
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	mimeType := getMimeType(imagePath)
	
	// Build request
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": query,
					},
					{
						"inlineData": map[string]string{
							"mimeType": mimeType,
							"data":     base64Image,
						},
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"maxOutputTokens": 2048,
		},
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	// Use Gemini 1.5 Flash for speed, Pro for accuracy
	model := "gemini-1.5-flash-latest"
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", v.baseURL, model, v.apiKey)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from API")
	}
	
	text := result.Candidates[0].Content.Parts[0].Text
	
	// Parse the response into structured data
	return v.parseVisionResponse(text, imagePath), nil
}

// analyzeWithOpenAI uses OpenAI's GPT-4 Vision
func (v *apiVisionProcessor) analyzeWithOpenAI(ctx context.Context, imagePath string, query string) (*VisionResult, error) {
	// Read and encode image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}
	
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	mimeType := getMimeType(imagePath)
	
	// Build request
	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini", // or gpt-4o for better quality
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": query,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
						},
					},
				},
			},
		},
		"max_tokens": 2048,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	url := v.baseURL + "/chat/completions"
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}
	
	text := result.Choices[0].Message.Content
	
	return v.parseVisionResponse(text, imagePath), nil
}

// analyzeWithAnthropic uses Claude's vision capabilities
func (v *apiVisionProcessor) analyzeWithAnthropic(ctx context.Context, imagePath string, query string) (*VisionResult, error) {
	// Read and encode image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}
	
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	mimeType := getMimeType(imagePath)
	
	// Build request
	reqBody := map[string]interface{}{
		"model": "claude-3-haiku-20240307", // or claude-3-sonnet for better quality
		"max_tokens": 2048,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type": "base64",
							"media_type": mimeType,
							"data": base64Image,
						},
					},
					{
						"type": "text",
						"text": query,
					},
				},
			},
		},
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	url := v.baseURL + "/messages"
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", v.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("no response from API")
	}
	
	text := result.Content[0].Text
	
	return v.parseVisionResponse(text, imagePath), nil
}

// parseVisionResponse parses the API response into structured data
func (v *apiVisionProcessor) parseVisionResponse(text string, imagePath string) *VisionResult {
	result := &VisionResult{
		Description: text,
		Text:        text,
	}
	
	// Check if it's a receipt
	textLower := strings.ToLower(text)
	if strings.Contains(textLower, "receipt") || 
	   strings.Contains(textLower, "total") && strings.Contains(textLower, "tax") {
		result.IsReceipt = true
		result.ReceiptData = v.extractReceiptFromText(text)
	}
	
	// Check if it's a document
	if strings.Contains(textLower, "document") ||
	   strings.Contains(textLower, "letter") ||
	   strings.Contains(textLower, "form") {
		result.IsDocument = true
	}
	
	// Extract entities
	result.Entities = v.extractEntities(text)
	
	return result
}

// extractReceiptFromText extracts receipt data from vision API text
func (v *apiVisionProcessor) extractReceiptFromText(text string) *ReceiptData {
	receipt := &ReceiptData{
		Items: []ReceiptItem{},
	}
	
	// Look for patterns in the text
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Try to find merchant (usually early in receipt)
		if receipt.Merchant == "" && len(line) > 2 && len(line) < 50 && 
		   !strings.Contains(line, "$") && !strings.Contains(line, "Total") {
			receipt.Merchant = line
		}
		
		// Look for totals
		if strings.Contains(strings.ToLower(line), "total") {
			receipt.Total = extractAmount(line)
		}
		if strings.Contains(strings.ToLower(line), "tax") {
			receipt.Tax = extractAmount(line)
		}
		if strings.Contains(strings.ToLower(line), "subtotal") {
			receipt.Subtotal = extractAmount(line)
		}
	}
	
	return receipt
}

// extractEntities extracts key entities from text
func (v *apiVisionProcessor) extractEntities(text string) []Entity {
	var entities []Entity
	
	// Date patterns
	if date := extractDate(text); date != "" {
		entities = append(entities, Entity{Type: "date", Value: date})
	}
	
	// Email patterns
	if email := extractEmail(text); email != "" {
		entities = append(entities, Entity{Type: "email", Value: email})
	}
	
	// Phone patterns
	if phone := extractPhone(text); phone != "" {
		entities = append(entities, Entity{Type: "phone", Value: phone})
	}
	
	return entities
}

// getMimeType returns the MIME type for an image file
func getMimeType(imagePath string) string {
	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// Helper functions
func extractAmount(line string) string {
	// Simple amount extraction - look for $X.XX pattern
	words := strings.Fields(line)
	for _, word := range words {
		if strings.HasPrefix(word, "$") {
			return word
		}
	}
	return ""
}

func extractDate(text string) string {
	// Placeholder - would use regex for real date extraction
	return ""
}

func extractEmail(text string) string {
	// Placeholder - would use regex for real email extraction
	return ""
}

func extractPhone(text string) string {
	// Placeholder - would use regex for real phone extraction
	return ""
}



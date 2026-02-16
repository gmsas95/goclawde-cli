// Package vision provides camera, image analysis, and audio skills
package vision

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/skills"
)

// VisionSkill provides visual and audio understanding capabilities
type VisionSkill struct {
	*skills.BaseSkill
	llmClient   *llm.Client
	visionModel string // Model with vision capabilities
	dataDir     string
}

// VisionSkillConfig configuration for vision skill
type VisionSkillConfig struct {
	VisionModel string `json:"vision_model"`
	DataDir     string `json:"data_dir"`
}

// NewVisionSkill creates a new vision skill
func NewVisionSkill(llmClient *llm.Client, config VisionSkillConfig) *VisionSkill {
	s := &VisionSkill{
		BaseSkill:   skills.NewBaseSkill("vision", "Visual understanding and camera/audio control", "1.0.0"),
		llmClient:   llmClient,
		visionModel: config.VisionModel,
		dataDir:     config.DataDir,
	}

	// Register tools
	s.registerTools()
	return s
}

func (s *VisionSkill) registerTools() {
	// Capture photo from camera
	s.AddTool(skills.Tool{
		Name:        "capture_photo",
		Description: "Capture a photo from the system camera and analyze it",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Specific question or focus for analysis",
					"default":     "Describe what you see in detail",
				},
			},
		},
		Handler: s.capturePhoto,
	})

	// Analyze existing image
	s.AddTool(skills.Tool{
		Name:        "analyze_image",
		Description: "Analyze an image file from the filesystem",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"image_path": map[string]interface{}{
					"type":        "string",
					"description": "Full path to the image file",
				},
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "What to look for in the image",
					"default":     "Describe what you see",
				},
			},
			"required": []string{"image_path"},
		},
		Handler: s.analyzeImage,
	})

	// Capture screenshot
	s.AddTool(skills.Tool{
		Name:        "capture_screenshot",
		Description: "Take a screenshot and optionally analyze it",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"analyze": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to analyze the screenshot",
					"default":     false,
				},
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Analysis prompt if analyze is true",
					"default":     "Describe the screen content",
				},
			},
		},
		Handler: s.captureScreenshot,
	})

	// Record and transcribe audio
	s.AddTool(skills.Tool{
		Name:        "listen",
		Description: "Record audio for specified seconds and transcribe to text",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"duration": map[string]interface{}{
					"type":        "number",
					"description": "Recording duration in seconds (max 60)",
					"default":     5,
				},
			},
		},
		Handler: s.listen,
	})

	// Describe visual content (for uploaded images)
	s.AddTool(skills.Tool{
		Name:        "describe_image",
		Description: "Describe the contents of an image file in detail",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to image file",
				},
				"detail": map[string]interface{}{
					"type":        "string",
					"description": "Level of detail: low, medium, high",
					"enum":        []string{"low", "medium", "high"},
					"default":     "medium",
				},
			},
			"required": []string{"path"},
		},
		Handler: s.describeImage,
	})
}

// capturePhoto captures from camera and analyzes
func (s *VisionSkill) capturePhoto(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	prompt := "Describe what you see in detail. If there are people, describe what they appear to be doing."
	if p, ok := args["prompt"].(string); ok && p != "" {
		prompt = p
	}

	// Capture image using platform-specific method
	capturePath := s.captureFromCamera()
	if capturePath == "" {
		return nil, fmt.Errorf("camera capture failed - ensure ffmpeg or platform tools are installed")
	}
	defer os.Remove(capturePath) // Clean up

	// Analyze the captured image
	result, err := s.analyzeImageFile(capturePath, prompt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"captured_at": time.Now().Format(time.RFC3339),
		"analysis":    result,
	}, nil
}

// analyzeImage analyzes an image file
func (s *VisionSkill) analyzeImage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	imagePath, _ := args["image_path"].(string)
	if imagePath == "" {
		return nil, fmt.Errorf("image_path is required")
	}

	// Expand path if needed
	if strings.HasPrefix(imagePath, "~/") {
		home, _ := os.UserHomeDir()
		imagePath = filepath.Join(home, imagePath[2:])
	}

	prompt := "Describe what you see"
	if p, ok := args["prompt"].(string); ok && p != "" {
		prompt = p
	}

	result, err := s.analyzeImageFile(imagePath, prompt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image_path": imagePath,
		"analysis":   result,
	}, nil
}

// captureScreenshot takes a screenshot
func (s *VisionSkill) captureScreenshot(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	shouldAnalyze := false
	if a, ok := args["analyze"].(bool); ok {
		shouldAnalyze = a
	}

	prompt := "Describe the screen content"
	if p, ok := args["prompt"].(string); ok && p != "" {
		prompt = p
	}

	// Capture screenshot
	screenshotPath := s.captureScreen()
	if screenshotPath == "" {
		return nil, fmt.Errorf("screenshot capture failed")
	}

	result := map[string]interface{}{
		"screenshot_path": screenshotPath,
		"captured_at":     time.Now().Format(time.RFC3339),
	}

	if shouldAnalyze {
		analysis, err := s.analyzeImageFile(screenshotPath, prompt)
		if err == nil {
			result["analysis"] = analysis
		}
	}

	return result, nil
}

// listen records and transcribes audio
func (s *VisionSkill) listen(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	duration := 5.0
	if d, ok := args["duration"].(float64); ok {
		duration = d
	}
	if duration > 60 {
		duration = 60 // Cap at 60 seconds
	}

	// Record audio
	audioPath, err := s.recordAudio(time.Duration(duration) * time.Second)
	if err != nil {
		return nil, fmt.Errorf("audio recording failed: %w", err)
	}
	defer os.Remove(audioPath)

	// Transcribe (placeholder - would call Whisper API)
	transcription := s.transcribeAudio(audioPath)

	return map[string]interface{}{
		"duration_seconds": duration,
		"transcription":    transcription,
		"recorded_at":      time.Now().Format(time.RFC3339),
	}, nil
}

// describeImage provides a detailed description of an image
func (s *VisionSkill) describeImage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	detail := "medium"
	if d, ok := args["detail"].(string); ok && d != "" {
		detail = d
	}

	// Build prompt based on detail level
	prompt := "Describe this image"
	switch detail {
	case "low":
		prompt = "Give a brief 1-2 sentence description of this image"
	case "high":
		prompt = "Provide a detailed, comprehensive description of this image including all visible objects, people, text, colors, layout, and context"
	default:
		prompt = "Describe what you see in this image, including main objects and any notable details"
	}

	result, err := s.analyzeImageFile(path, prompt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"description": result,
		"detail":      detail,
	}, nil
}

// =============================================================================
// Helper methods
// =============================================================================

// analyzeImageFile sends image to vision-capable LLM
func (s *VisionSkill) analyzeImageFile(imagePath, prompt string) (string, error) {
	// Read image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	// Encode to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Determine MIME type
	ext := strings.ToLower(filepath.Ext(imagePath))
	mimeType := "image/jpeg"
	if ext == ".png" {
		mimeType = "image/png"
	} else if ext == ".gif" {
		mimeType = "image/gif"
	} else if ext == ".webp" {
		mimeType = "image/webp"
	}

	// For now, return mock response (would integrate with actual vision API)
	_ = base64Image
	_ = mimeType
	_ = prompt

	// In production, this would call GPT-4V or similar:
	// return s.callVisionAPI(prompt, base64Image, mimeType)

	return fmt.Sprintf("[Vision analysis placeholder] Analyzing %s image. Prompt: %s", mimeType, prompt), nil
}

// Platform-specific camera capture (simplified - uses ffmpeg)
func (s *VisionSkill) captureFromCamera() string {
	timestamp := time.Now().Format("20060102_150405")
	captureDir := filepath.Join(s.dataDir, "captures")
	os.MkdirAll(captureDir, 0755)
	capturePath := filepath.Join(captureDir, fmt.Sprintf("camera_%s.jpg", timestamp))

	// Try to use ffmpeg for capture
	// This is a basic implementation - would need platform-specific handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Platform-specific command would go here
	// For now, return empty to indicate not implemented in this basic version
	_ = ctx
	_ = capturePath
	return ""
}

// Platform-specific screen capture
func (s *VisionSkill) captureScreen() string {
	timestamp := time.Now().Format("20060102_150405")
	screenshotDir := filepath.Join(s.dataDir, "screenshots")
	os.MkdirAll(screenshotDir, 0755)
	screenshotPath := filepath.Join(screenshotDir, fmt.Sprintf("screen_%s.png", timestamp))

	// Platform-specific commands would go here
	// Return path for now (actual implementation needs platform tools)
	return screenshotPath
}

// recordAudio records audio
func (s *VisionSkill) recordAudio(duration time.Duration) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	audioDir := filepath.Join(s.dataDir, "audio")
	os.MkdirAll(audioDir, 0755)
	audioPath := filepath.Join(audioDir, fmt.Sprintf("recording_%s.wav", timestamp))

	// Would use platform-specific audio recording
	// For now, return placeholder
	_ = duration
	return audioPath, fmt.Errorf("audio recording not yet implemented - requires platform-specific setup")
}

// transcribeAudio transcribes audio to text
func (s *VisionSkill) transcribeAudio(audioPath string) string {
	// Would call Whisper API or local model
	// For now, return placeholder
	_ = audioPath
	return "[Audio transcription placeholder - Whisper API integration needed]"
}

// callVisionAPI would call GPT-4V or similar
func (s *VisionSkill) callVisionAPI(prompt, base64Image, mimeType string) (string, error) {
	// Implement vision API call
	// This requires the LLM client to support vision models
	_ = prompt
	_ = base64Image
	_ = mimeType
	return "", fmt.Errorf("vision API not yet implemented")
}

// Ensure VisionSkill implements Skill interface
var _ skills.Skill = (*VisionSkill)(nil)

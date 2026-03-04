// Package vision provides camera, image analysis, and audio skills
package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	httpClient  *http.Client
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
		httpClient:  &http.Client{Timeout: 30 * time.Second},
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
	capturePath, err := s.captureFromCamera()
	if err != nil {
		return nil, fmt.Errorf("camera capture failed: %w", err)
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
	screenshotPath, err := s.captureScreen()
	if err != nil {
		return nil, fmt.Errorf("screenshot capture failed: %w", err)
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

	// Transcribe using Whisper API
	transcription, err := s.transcribeAudio(audioPath)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

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
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	}

	// Call vision API
	return s.callVisionAPI(prompt, base64Image, mimeType)
}

// callVisionAPI calls GPT-4V or similar vision-capable model
func (s *VisionSkill) callVisionAPI(prompt, base64Image, mimeType string) (string, error) {
	// Check if LLM client supports vision
	if s.llmClient == nil {
		return s.fallbackImageAnalysis(prompt, mimeType)
	}

	// Try to use the vision API through the LLM client
	// This sends a message with image content
	result, err := s.llmClient.ChatWithVision(prompt, base64Image, mimeType)
	if err != nil {
		// Fallback to simple analysis if vision API fails
		return s.fallbackImageAnalysis(prompt, mimeType)
	}

	return result, nil
}

// fallbackImageAnalysis provides basic analysis when vision API unavailable
func (s *VisionSkill) fallbackImageAnalysis(prompt, mimeType string) (string, error) {
	// Simple fallback that acknowledges the image type
	return fmt.Sprintf("Image analysis requested: %s. Prompt: %s (Vision API not fully configured)", mimeType, prompt), nil
}

// captureFromCamera captures image from system camera
func (s *VisionSkill) captureFromCamera() (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	captureDir := filepath.Join(s.dataDir, "captures")
	if err := os.MkdirAll(captureDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create capture directory: %w", err)
	}
	capturePath := filepath.Join(captureDir, fmt.Sprintf("camera_%s.jpg", timestamp))

	// Platform-specific camera capture
	switch runtime.GOOS {
	case "darwin":
		return s.captureFromCameraMacOS(capturePath)
	case "linux":
		return s.captureFromCameraLinux(capturePath)
	default:
		return "", fmt.Errorf("camera capture not supported on %s", runtime.GOOS)
	}
}

// captureFromCameraMacOS captures from camera on macOS
func (s *VisionSkill) captureFromCameraMacOS(capturePath string) (string, error) {
	// Try to use imagesnap if available
	if _, err := exec.LookPath("imagesnap"); err == nil {
		cmd := exec.Command("imagesnap", "-w", "1.0", capturePath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("imagesnap failed: %w", err)
		}
		return capturePath, nil
	}

	// Try ffmpeg as fallback
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-f", "avfoundation",
			"-video_size", "1280x720", "-i", "0",
			"-frames:v", "1", "-y", capturePath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("ffmpeg capture failed: %w", err)
		}
		return capturePath, nil
	}

	return "", fmt.Errorf("no camera capture tool found (install imagesnap or ffmpeg)")
}

// captureFromCameraLinux captures from camera on Linux
func (s *VisionSkill) captureFromCameraLinux(capturePath string) (string, error) {
	// Try ffmpeg first
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-f", "v4l2",
			"-video_size", "1280x720", "-i", "/dev/video0",
			"-frames:v", "1", "-y", capturePath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("ffmpeg capture failed: %w", err)
		}
		return capturePath, nil
	}

	// Try fswebcam
	if _, err := exec.LookPath("fswebcam"); err == nil {
		cmd := exec.Command("fswebcam", "-r", "1280x720", "--no-banner", capturePath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("fswebcam failed: %w", err)
		}
		return capturePath, nil
	}

	return "", fmt.Errorf("no camera capture tool found (install ffmpeg or fswebcam)")
}

// captureScreen captures screenshot
func (s *VisionSkill) captureScreen() (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	screenshotDir := filepath.Join(s.dataDir, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}
	screenshotPath := filepath.Join(screenshotDir, fmt.Sprintf("screen_%s.png", timestamp))

	switch runtime.GOOS {
	case "darwin":
		return s.captureScreenMacOS(screenshotPath)
	case "linux":
		return s.captureScreenLinux(screenshotPath)
	default:
		return "", fmt.Errorf("screenshot capture not supported on %s", runtime.GOOS)
	}
}

// captureScreenMacOS captures screenshot on macOS
func (s *VisionSkill) captureScreenMacOS(screenshotPath string) (string, error) {
	// Use built-in screencapture
	cmd := exec.Command("screencapture", "-x", screenshotPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("screencapture failed: %w", err)
	}
	return screenshotPath, nil
}

// captureScreenLinux captures screenshot on Linux
func (s *VisionSkill) captureScreenLinux(screenshotPath string) (string, error) {
	// Try gnome-screenshot
	if _, err := exec.LookPath("gnome-screenshot"); err == nil {
		cmd := exec.Command("gnome-screenshot", "-f", screenshotPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("gnome-screenshot failed: %w", err)
		}
		return screenshotPath, nil
	}

	// Try import (ImageMagick)
	if _, err := exec.LookPath("import"); err == nil {
		cmd := exec.Command("import", "-window", "root", screenshotPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("import failed: %w", err)
		}
		return screenshotPath, nil
	}

	return "", fmt.Errorf("no screenshot tool found (install gnome-screenshot or ImageMagick)")
}

// recordAudio records audio for specified duration
func (s *VisionSkill) recordAudio(duration time.Duration) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	audioDir := filepath.Join(s.dataDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create audio directory: %w", err)
	}
	audioPath := filepath.Join(audioDir, fmt.Sprintf("recording_%s.wav", timestamp))

	switch runtime.GOOS {
	case "darwin":
		return s.recordAudioMacOS(audioPath, duration)
	case "linux":
		return s.recordAudioLinux(audioPath, duration)
	default:
		return "", fmt.Errorf("audio recording not supported on %s", runtime.GOOS)
	}
}

// recordAudioMacOS records audio on macOS
func (s *VisionSkill) recordAudioMacOS(audioPath string, duration time.Duration) (string, error) {
	// Try ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-f", "avfoundation",
			"-i", ":0", "-t", fmt.Sprintf("%.0f", duration.Seconds()),
			"-acodec", "pcm_s16le", "-ar", "44100", "-ac", "1",
			"-y", audioPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("ffmpeg recording failed: %w", err)
		}
		return audioPath, nil
	}

	return "", fmt.Errorf("ffmpeg not found (required for audio recording)")
}

// recordAudioLinux records audio on Linux
func (s *VisionSkill) recordAudioLinux(audioPath string, duration time.Duration) (string, error) {
	// Try ffmpeg first
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-f", "alsa", "-i", "default",
			"-t", fmt.Sprintf("%.0f", duration.Seconds()),
			"-acodec", "pcm_s16le", "-ar", "44100", "-ac", "1",
			"-y", audioPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("ffmpeg recording failed: %w", err)
		}
		return audioPath, nil
	}

	// Try arecord (ALSA)
	if _, err := exec.LookPath("arecord"); err == nil {
		cmd := exec.Command("arecord", "-d", fmt.Sprintf("%.0f", duration.Seconds()),
			"-f", "cd", "-t", "wav", audioPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("arecord failed: %w", err)
		}
		return audioPath, nil
	}

	return "", fmt.Errorf("no audio recording tool found (install ffmpeg or arecord)")
}

// transcribeAudio transcribes audio using Whisper API
func (s *VisionSkill) transcribeAudio(audioPath string) (string, error) {
	// Read audio file
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	// Use OpenAI Whisper API
	return s.callWhisperAPI(audioData)
}

// WhisperResponse represents the API response
type WhisperResponse struct {
	Text string `json:"text"`
}

// callWhisperAPI calls OpenAI Whisper API for transcription
func (s *VisionSkill) callWhisperAPI(audioData []byte) (string, error) {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set (required for transcription)")
	}

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add model parameter
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper API error: %s", string(respBody))
	}

	// Parse response
	var result WhisperResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Text, nil
}

// Ensure VisionSkill implements Skill interface
var _ skills.Skill = (*VisionSkill)(nil)

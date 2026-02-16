// Package vision provides camera, image, and audio capabilities for AI assistant
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
)

// Capture provides vision and audio capabilities
type Capture struct {
	llmClient *llm.Client
	dataDir   string
}

// NewCapture creates a new vision capture instance
func NewCapture(llmClient *llm.Client, dataDir string) *Capture {
	return &Capture{
		llmClient: llmClient,
		dataDir:   dataDir,
	}
}

// ImageAnalysisResult holds the result of image analysis
type ImageAnalysisResult struct {
	Description string            `json:"description"`
	Objects     []string          `json:"objects"`
	Text        string            `json:"text,omitempty"`
	Actions     []SuggestedAction `json:"suggested_actions,omitempty"`
}

// SuggestedAction represents a suggested action based on image
type SuggestedAction struct {
	Action      string `json:"action"`
	Description string `json:"description"`
}

// AudioTranscriptionResult holds transcription results
type AudioTranscriptionResult struct {
	Text      string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Language  string  `json:"language"`
}

// CaptureOptions options for capture
type CaptureOptions struct {
	Device     string // Camera device (default: 0)
	Resolution string // e.g., "1920x1080"
	Duration   time.Duration // For video/audio
}

// =============================================================================
// IMAGE CAPTURE AND ANALYSIS
// =============================================================================

// CaptureSnapshot captures a photo from the camera (platform-specific)
func (c *Capture) CaptureSnapshot(ctx context.Context, opts CaptureOptions) (string, error) {
	// Create temp file
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("capture_%s.jpg", timestamp)
	fullPath := filepath.Join(c.dataDir, "captures", filename)

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	// Platform-specific capture
	switch runtime.GOOS {
	case "darwin":
		return c.captureMacOS(ctx, fullPath, opts)
	case "linux":
		return c.captureLinux(ctx, fullPath, opts)
	case "windows":
		return c.captureWindows(ctx, fullPath, opts)
	default:
		return "", fmt.Errorf("camera capture not supported on %s", runtime.GOOS)
	}
}

// captureMacOS uses imagesnap or avfoundation
func (c *Capture) captureMacOS(ctx context.Context, filepath string, opts CaptureOptions) (string, error) {
	// Try imagesnap first (popular macOS tool)
	cmd := exec.CommandContext(ctx, "imagesnap", "-w", "1.0", filepath)
	if err := cmd.Run(); err == nil {
		return filepath, nil
	}

	// Fallback to ffmpeg with avfoundation
	cmd = exec.CommandContext(ctx, "ffmpeg", "-f", "avfoundation",
		"-video_size", "1280x720", "-i", "0", "-vframes", "1", filepath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to capture image on macOS: %w", err)
	}

	return filepath, nil
}

// captureLinux uses fswebcam or ffmpeg
func (c *Capture) captureLinux(ctx context.Context, filepath string, opts CaptureOptions) (string, error) {
	// Try fswebcam first
	cmd := exec.CommandContext(ctx, "fswebcam", "-r", "1280x720", "--no-banner", filepath)
	if err := cmd.Run(); err == nil {
		return filepath, nil
	}

	// Fallback to ffmpeg
	cmd = exec.CommandContext(ctx, "ffmpeg", "-f", "v4l2",
		"-video_size", "1280x720", "-i", "/dev/video0", "-vframes", "1", filepath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to capture image on Linux: %w. Install fswebcam or ffmpeg", err)
	}

	return filepath, nil
}

// captureWindows uses ffmpeg with dshow
func (c *Capture) captureWindows(ctx context.Context, filepath string, opts CaptureOptions) (string, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-f", "dshow",
		"-i", "video=0", "-vframes", "1", filepath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to capture image on Windows: %w. Install ffmpeg", err)
	}

	return filepath, nil
}

// AnalyzeImage analyzes an image using vision-capable LLM
func (c *Capture) AnalyzeImage(ctx context.Context, imagePath string, prompt string) (*ImageAnalysisResult, error) {
	// Read image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Encode to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Determine MIME type
	mimeType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(imagePath), ".png") {
		mimeType = "image/png"
	}

	// Default prompt if not provided
	if prompt == "" {
		prompt = "Describe what you see in this image in detail."
	}

	// Build vision request (OpenAI/GPT-4V format)
	req := llm.ChatRequest{
		Model: c.llmClient.GetModel(),
		Messages: []llm.Message{
			{
				Role: "user",
				Content: fmt.Sprintf(`%s

Image: data:%s;base64,%s`, prompt, mimeType, base64Image),
			},
		},
		MaxTokens: 4096,
	}

	resp, err := c.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vision analysis failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	return &ImageAnalysisResult{
		Description: resp.Choices[0].Message.Content,
	}, nil
}

// AnalyzeImageFile analyzes an uploaded image file
func (c *Capture) AnalyzeImageFile(ctx context.Context, fileHeader *multipart.FileHeader, prompt string) (*ImageAnalysisResult, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Save temporarily
	tempPath := filepath.Join(c.dataDir, "uploads", fmt.Sprintf("upload_%d.jpg", time.Now().Unix()))
	os.MkdirAll(filepath.Dir(tempPath), 0755)
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(tempPath) // Clean up

	return c.AnalyzeImage(ctx, tempPath, prompt)
}

// =============================================================================
// AUDIO CAPTURE AND TRANSCRIPTION
// =============================================================================

// RecordAudio records audio for specified duration
func (c *Capture) RecordAudio(ctx context.Context, duration time.Duration, outputPath string) (string, error) {
	if outputPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputPath = filepath.Join(c.dataDir, "audio", fmt.Sprintf("recording_%s.wav", timestamp))
	}

	os.MkdirAll(filepath.Dir(outputPath), 0755)

	switch runtime.GOOS {
	case "darwin":
		return c.recordAudioMacOS(ctx, duration, outputPath)
	case "linux":
		return c.recordAudioLinux(ctx, duration, outputPath)
	case "windows":
		return c.recordAudioWindows(ctx, duration, outputPath)
	default:
		return "", fmt.Errorf("audio recording not supported on %s", runtime.GOOS)
	}
}

func (c *Capture) recordAudioMacOS(ctx context.Context, duration time.Duration, outputPath string) (string, error) {
	// Use ffmpeg with avfoundation
	cmd := exec.CommandContext(ctx, "ffmpeg", "-f", "avfoundation",
		"-i", ":0", "-t", fmt.Sprintf("%.0f", duration.Seconds()),
		"-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to record audio: %w", err)
	}

	return outputPath, nil
}

func (c *Capture) recordAudioLinux(ctx context.Context, duration time.Duration, outputPath string) (string, error) {
	// Use ffmpeg with alsa or pulse
	cmd := exec.CommandContext(ctx, "ffmpeg", "-f", "alsa", "-i", "default",
		"-t", fmt.Sprintf("%.0f", duration.Seconds()),
		"-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputPath)

	if err := cmd.Run(); err != nil {
		// Try pulse
		cmd = exec.CommandContext(ctx, "ffmpeg", "-f", "pulse", "-i", "default",
			"-t", fmt.Sprintf("%.0f", duration.Seconds()),
			"-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to record audio: %w", err)
		}
	}

	return outputPath, nil
}

func (c *Capture) recordAudioWindows(ctx context.Context, duration time.Duration, outputPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-f", "dshow", "-i", "audio=0",
		"-t", fmt.Sprintf("%.0f", duration.Seconds()),
		"-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to record audio: %w", err)
	}

	return outputPath, nil
}

// TranscribeAudio transcribes audio using Whisper API
func (c *Capture) TranscribeAudio(ctx context.Context, audioPath string) (*AudioTranscriptionResult, error) {
	// Read audio file
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio: %w", err)
	}

	// For now, we would need to implement Whisper API call
	// This is a placeholder - in production, call Whisper API or local model
	return c.transcribeWithAPI(ctx, audioData)
}

func (c *Capture) transcribeWithAPI(ctx context.Context, audioData []byte) (*AudioTranscriptionResult, error) {
	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return nil, err
	}
	part.Write(audioData)

	// Add model parameter
	writer.WriteField("model", "whisper-1")
	writer.Close()

	// Make request to Whisper API
	// Note: This requires API key configuration
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	// req.Header.Set("Authorization", "Bearer "+apiKey) // Need to implement

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("transcription request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("transcription failed: %s", string(body))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &AudioTranscriptionResult{
		Text:       result.Text,
		Confidence: 0.95, // Placeholder
		Language:   "en",  // Would detect from response
	}, nil
}

// =============================================================================
// SCREEN CAPTURE
// =============================================================================

// CaptureScreen captures the screen
func (c *Capture) CaptureScreen(ctx context.Context, outputPath string) (string, error) {
	if outputPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputPath = filepath.Join(c.dataDir, "screenshots", fmt.Sprintf("screen_%s.png", timestamp))
	}

	os.MkdirAll(filepath.Dir(outputPath), 0755)

	switch runtime.GOOS {
	case "darwin":
		cmd := exec.CommandContext(ctx, "screencapture", "-x", outputPath)
		if err := cmd.Run(); err != nil {
			return "", err
		}
	case "linux":
		cmd := exec.CommandContext(ctx, "gnome-screenshot", "-f", outputPath)
		if err := cmd.Run(); err != nil {
			// Try import (ImageMagick)
			cmd = exec.CommandContext(ctx, "import", "-window", "root", outputPath)
			if err := cmd.Run(); err != nil {
				return "", err
			}
		}
	case "windows":
		// Windows would need different approach - possibly PowerShell or external tool
		return "", fmt.Errorf("screen capture on Windows requires additional tools")
	}

	return outputPath, nil
}

// ProcessVoiceCommand combines audio recording + transcription + action
func (c *Capture) ProcessVoiceCommand(ctx context.Context, duration time.Duration) (string, error) {
	// Record audio
	audioPath, err := c.RecordAudio(ctx, duration, "")
	if err != nil {
		return "", fmt.Errorf("failed to record audio: %w", err)
	}
	defer os.Remove(audioPath)

	// Transcribe
	result, err := c.TranscribeAudio(ctx, audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to transcribe: %w", err)
	}

	return result.Text, nil
}

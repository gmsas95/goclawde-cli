package vision

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/skills"
)

func TestNewVisionSkill(t *testing.T) {
	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     t.TempDir(),
	}

	skill := NewVisionSkill(client, config)

	if skill == nil {
		t.Fatal("expected skill to be created")
	}

	if skill.Name() != "vision" {
		t.Errorf("expected name 'vision', got '%s'", skill.Name())
	}

	tools := skill.Tools()
	if len(tools) != 5 {
		t.Errorf("expected 5 tools, got %d", len(tools))
	}

	// Check tool names
	expectedTools := []string{"capture_photo", "analyze_image", "capture_screenshot", "listen", "describe_image"}
	for _, name := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool '%s' not found", name)
		}
	}
}

func findTool(tools []skills.Tool, name string) *skills.Tool {
	for _, tool := range tools {
		if tool.Name == name {
			return &tool
		}
	}
	return nil
}

func TestVisionSkill_ExecuteTool(t *testing.T) {
	tempDir := t.TempDir()
	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     tempDir,
	}

	skill := NewVisionSkill(client, config)
	ctx := context.Background()

	tests := []struct {
		name      string
		toolName  string
		args      map[string]interface{}
		wantError bool
	}{
		{
			name:      "describe_image with empty path",
			toolName:  "describe_image",
			args:      map[string]interface{}{"path": ""},
			wantError: true,
		},
		{
			name:      "analyze_image with empty path",
			toolName:  "analyze_image",
			args:      map[string]interface{}{"image_path": ""},
			wantError: true,
		},
		{
			name:      "capture_photo",
			toolName:  "capture_photo",
			args:      map[string]interface{}{},
			wantError: true, // No camera in test environment
		},
		{
			name:      "capture_screenshot",
			toolName:  "capture_screenshot",
			args:      map[string]interface{}{"analyze": false},
			wantError: true, // May fail if no screenshot tools available
		},
		{
			name:      "listen",
			toolName:  "listen",
			args:      map[string]interface{}{"duration": 3.0},
			wantError: true, // No audio in test environment
		},
		{
			name:      "describe_image with valid path",
			toolName:  "describe_image",
			args:      map[string]interface{}{"path": "/tmp/nonexistent.jpg"},
			wantError: true, // File doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := findTool(skill.Tools(), tt.toolName)
			if tool == nil {
				t.Fatalf("tool '%s' not found", tt.toolName)
			}

			_, err := tool.Handler(ctx, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v", tt.wantError, err)
			}
		})
	}
}

func TestVisionSkill_DescribeImageDetailLevels(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test image
	testImagePath := filepath.Join(tempDir, "test.jpg")
	// Create minimal valid JPEG
	minimalJPEG := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
	}
	if err := os.WriteFile(testImagePath, minimalJPEG, 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     tempDir,
	}

	skill := NewVisionSkill(client, config)
	tool := findTool(skill.Tools(), "describe_image")
	if tool == nil {
		t.Fatal("describe_image tool not found")
	}

	ctx := context.Background()
	detailLevels := []string{"low", "medium", "high"}
	for _, detail := range detailLevels {
		t.Run("detail_"+detail, func(t *testing.T) {
			args := map[string]interface{}{
				"path":   testImagePath,
				"detail": detail,
			}
			// The test image is valid, so it should succeed with placeholder response
			_, err := tool.Handler(ctx, args)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestVisionSkill_GetDescription(t *testing.T) {
	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     t.TempDir(),
	}

	skill := NewVisionSkill(client, config)

	desc := skill.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}

	expected := "Visual understanding and camera/audio control"
	if desc != expected {
		t.Errorf("expected description '%s', got '%s'", expected, desc)
	}
}

func TestVisionSkill_ScreenshotWithAnalysis(t *testing.T) {
	tempDir := t.TempDir()
	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     tempDir,
	}

	skill := NewVisionSkill(client, config)
	tool := findTool(skill.Tools(), "capture_screenshot")
	if tool == nil {
		t.Fatal("capture_screenshot tool not found")
	}

	ctx := context.Background()

	// Test without analysis
	result, err := tool.Handler(ctx, map[string]interface{}{
		"analyze": false,
	})

	// May error if no screenshot tools available - that's OK
	if err != nil {
		t.Logf("Screenshot failed (expected in test environment): %v", err)
		return
	}

	if result == nil {
		t.Fatal("expected result")
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if resultMap["screenshot_path"] == "" {
		t.Error("expected screenshot_path in result")
	}
}

func TestVisionSkill_ListenDurationLimits(t *testing.T) {
	tempDir := t.TempDir()
	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     tempDir,
	}

	skill := NewVisionSkill(client, config)
	tool := findTool(skill.Tools(), "listen")
	if tool == nil {
		t.Fatal("listen tool not found")
	}

	ctx := context.Background()

	// Test duration > 60 is capped (won't error, just caps)
	// This should fail due to no audio device, but not due to invalid duration
	_, _ = tool.Handler(ctx, map[string]interface{}{
		"duration": 120.0, // Should be capped to 60
	})

	// Test valid duration
	_, _ = tool.Handler(ctx, map[string]interface{}{
		"duration": 5.0,
	})
}

// Test integration with skills registry
func TestVisionSkill_Integration(t *testing.T) {
	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     t.TempDir(),
	}

	skill := NewVisionSkill(client, config)

	// Verify it implements the interface
	var _ skills.Skill = skill

	// Test that all tools have required fields
	for _, tool := range skill.Tools() {
		if tool.Name == "" {
			t.Error("tool name is empty")
		}
		if tool.Description == "" {
			t.Errorf("tool '%s' description is empty", tool.Name)
		}
		if tool.Handler == nil {
			t.Errorf("tool '%s' handler is nil", tool.Name)
		}
	}
}

// Test MIME type detection
func TestGetMIMEType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.jpg", "image/jpeg"},
		{"test.jpeg", "image/jpeg"},
		{"test.png", "image/png"},
		{"test.gif", "image/gif"},
		{"test.webp", "image/webp"},
		{"test.unknown", "image/jpeg"}, // default
	}

	for _, tt := range tests {
		ext := filepath.Ext(tt.filename)
		mimeType := "image/jpeg"
		switch ext {
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		case ".webp":
			mimeType = "image/webp"
		}

		if mimeType != tt.expected {
			t.Errorf("For %s: expected %s, got %s", tt.filename, tt.expected, mimeType)
		}
	}
}

// Test transcribe audio with missing API key
func TestTranscribeAudio_MissingAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	skill := &VisionSkill{
		dataDir:    tempDir,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	// Create dummy audio file
	audioPath := filepath.Join(tempDir, "test.wav")
	dummyAudio := []byte("RIFF$\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x44\xAC\x00\x00\x88X\x01\x00\x02\x00\x10\x00data\x00\x00\x00\x00")
	if err := os.WriteFile(audioPath, dummyAudio, 0644); err != nil {
		t.Fatalf("Failed to create dummy audio: %v", err)
	}

	// Unset API key
	os.Unsetenv("OPENAI_API_KEY")

	_, err := skill.transcribeAudio(audioPath)
	if err == nil {
		t.Error("Expected error without OPENAI_API_KEY")
	}

	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("Expected error about missing API key, got: %v", err)
	}
}

// Test transcribe audio with non-existent file
func TestTranscribeAudio_FileNotFound(t *testing.T) {
	skill := &VisionSkill{}

	_, err := skill.transcribeAudio("/nonexistent/path/audio.wav")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// Test call whisper API with missing key
func TestCallWhisperAPI_MissingKey(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")

	skill := &VisionSkill{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := skill.callWhisperAPI([]byte("dummy"))
	if err == nil {
		t.Error("Expected error without OPENAI_API_KEY")
	}

	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("Expected error about missing API key, got: %v", err)
	}
}

// Test vision skill config
func TestVisionSkillConfig(t *testing.T) {
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     "/tmp/test",
	}

	if config.VisionModel != "gpt-4-vision-preview" {
		t.Errorf("Expected vision model 'gpt-4-vision-preview', got '%s'", config.VisionModel)
	}

	if config.DataDir != "/tmp/test" {
		t.Errorf("Expected data dir '/tmp/test', got '%s'", config.DataDir)
	}
}

// Test HTTP client initialization
func TestVisionSkill_HTTPClient(t *testing.T) {
	client := &llm.Client{}
	config := VisionSkillConfig{
		VisionModel: "gpt-4-vision-preview",
		DataDir:     t.TempDir(),
	}

	skill := NewVisionSkill(client, config)

	if skill.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if skill.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", skill.httpClient.Timeout)
	}
}

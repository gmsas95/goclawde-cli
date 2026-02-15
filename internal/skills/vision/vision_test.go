package vision

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gmsas95/goclawde-cli/internal/llm"
	"github.com/gmsas95/goclawde-cli/internal/skills"
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
			wantError: false, // Just returns path
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
	if err != nil {
		t.Errorf("unexpected error: %v", err)
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

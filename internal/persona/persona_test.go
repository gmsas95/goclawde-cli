package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNewPersonaManager(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	
	logger := zap.NewNop()
	pm, err := NewPersonaManager(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create PersonaManager: %v", err)
	}
	
	// Check workspace was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Workspace directory not created")
	}
	
	// Check subdirectories
	subdirs := []string{"projects", "diary", "memory"}
	for _, dir := range subdirs {
		path := filepath.Join(tempDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Subdirectory %s not created", dir)
		}
	}
	
	// Check default identity
	if pm.GetIdentity().Name != "GoClawde" {
		t.Errorf("Expected default name 'GoClawde', got %s", pm.GetIdentity().Name)
	}
}

func TestIdentityString(t *testing.T) {
	identity := &Identity{
		Name:        "TestBot",
		Personality: "Friendly and helpful",
		Voice:       "Casual and approachable",
		Values:      []string{"honesty", "kindness"},
		Expertise:   []string{"coding", "writing"},
	}
	
	result := identity.String()
	
	if !strings.Contains(result, "Name: TestBot") {
		t.Error("Identity string missing name")
	}
	if !strings.Contains(result, "Friendly and helpful") {
		t.Error("Identity string missing personality")
	}
	if !strings.Contains(result, "honesty") {
		t.Error("Identity string missing values")
	}
}

func TestUserProfileString(t *testing.T) {
	user := &UserProfile{
		Name:               "Alice",
		CommunicationStyle: "Concise",
		Expertise:          []string{"Go", "Python"},
		Goals:              []string{"Learn Rust"},
		Preferences:        map[string]string{"theme": "dark"},
	}
	
	result := user.String()
	
	if !strings.Contains(result, "Name: Alice") {
		t.Error("User profile string missing name")
	}
	if !strings.Contains(result, "Concise") {
		t.Error("User profile string missing communication style")
	}
	if !strings.Contains(result, "theme: dark") {
		t.Error("User profile string missing preferences")
	}
}

func TestParseIdentity(t *testing.T) {
	data := `# Identity

Name: TestBot

## Personality
Friendly and helpful AI assistant

## Voice
Casual tone

## Values
- honesty
- transparency

## Expertise
- coding
- writing
`
	
	identity := parseIdentity(data)
	
	if identity.Name != "TestBot" {
		t.Errorf("Expected name 'TestBot', got '%s'", identity.Name)
	}
	if !strings.Contains(identity.Personality, "Friendly") {
		t.Error("Failed to parse personality")
	}
	if len(identity.Values) != 2 {
		t.Errorf("Expected 2 values, got %d", len(identity.Values))
	}
}

func TestParseUserProfile(t *testing.T) {
	data := `# User Profile

Name: Bob

## Communication Style
Detailed and thorough

## Expertise
- Go
- Kubernetes

## Goals
- Build scalable systems

## Preferences
- editor: vim
- theme: dark
`
	
	user := parseUserProfile(data)
	
	if user.Name != "Bob" {
		t.Errorf("Expected name 'Bob', got '%s'", user.Name)
	}
	if len(user.Expertise) != 2 {
		t.Errorf("Expected 2 expertise items, got %d", len(user.Expertise))
	}
	if user.Preferences["editor"] != "vim" {
		t.Error("Failed to parse preferences")
	}
}

func TestPersonaManagerCache(t *testing.T) {
	tempDir := t.TempDir()
	logger := zap.NewNop()
	
	pm, err := NewPersonaManager(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create PersonaManager: %v", err)
	}
	
	// First call should build and cache
	prompt1 := pm.GetSystemPrompt()
	if prompt1 == "" {
		t.Error("System prompt is empty")
	}
	
	// Second call should return cached
	prompt2 := pm.GetSystemPrompt()
	if prompt1 != prompt2 {
		t.Error("Cache not working - prompts differ")
	}
	
	// Invalidate cache
	pm.InvalidateCache()
	
	// After update, cache should be rebuilt
	pm.cacheMu.RLock()
	valid := pm.cacheValid
	pm.cacheMu.RUnlock()
	
	if valid {
		t.Error("Cache should be invalid after InvalidateCache")
	}
}

func TestGetSystemPromptContent(t *testing.T) {
	tempDir := t.TempDir()
	logger := zap.NewNop()
	
	pm, err := NewPersonaManager(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create PersonaManager: %v", err)
	}
	
	prompt := pm.GetSystemPrompt()
	
	// Should contain expected sections
	requiredSections := []string{
		"Current Context",      // Time awareness
		"Your Identity",        // Identity
		"User Profile",         // User
	}
	
	for _, section := range requiredSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("System prompt missing section: %s", section)
		}
	}
}

package search

import (
	"context"
	"testing"
)

func TestSearchSkillCreation(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Provider:    "brave",
		APIKey:      "test_key",
		MaxResults:  5,
		TimeoutSecs: 30,
	}

	skill := NewSearchSkill(cfg)

	if skill == nil {
		t.Fatal("Expected skill to be created")
	}

	if skill.Name() != "search" {
		t.Errorf("Expected name 'search', got '%s'", skill.Name())
	}

	if !skill.IsEnabled() {
		t.Error("Expected skill to be enabled")
	}
}

func TestSearchSkillDisabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	skill := NewSearchSkill(cfg)

	if skill.IsEnabled() {
		t.Error("Expected skill to be disabled")
	}
}

func TestProviderAvailability(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{"Brave with key", "test_key", true},
		{"Brave without key", "", false},
		{"Serper with key", "test_key", true},
		{"Serper without key", "", false},
		{"DuckDuckGo", "", true}, // DuckDuckGo doesn't need API key
		{"Google with key", "test_key", true},
		{"Google without key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider Provider
			switch {
			case tt.name == "Brave with key" || tt.name == "Brave without key":
				provider = NewBraveProvider(tt.apiKey)
			case tt.name == "Serper with key" || tt.name == "Serper without key":
				provider = NewSerperProvider(tt.apiKey)
			case tt.name == "DuckDuckGo":
				provider = NewDuckDuckGoProvider()
			case tt.name == "Google with key" || tt.name == "Google without key":
				provider = NewGoogleProvider(tt.apiKey)
			}

			if provider == nil {
				t.Fatal("Provider not created")
			}

			if provider.IsAvailable() != tt.expected {
				t.Errorf("Expected IsAvailable() = %v, got %v", tt.expected, provider.IsAvailable())
			}
		})
	}
}

func TestSearchSkillTools(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Provider:    "brave",
		APIKey:      "test_key",
		MaxResults:  5,
		TimeoutSecs: 30,
	}

	skill := NewSearchSkill(cfg)
	tools := skill.Tools()

	expectedTools := []string{"web_search", "get_search_providers"}
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool '%s' not found", expected)
		}
	}
}

func TestGetSearchProviders(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Provider:    "brave",
		APIKey:      "test_key",
		MaxResults:  5,
		TimeoutSecs: 30,
	}

	skill := NewSearchSkill(cfg)

	// Get the get_search_providers tool
	var handler func(context.Context, map[string]interface{}) (interface{}, error)
	for _, tool := range skill.Tools() {
		if tool.Name == "get_search_providers" {
			handler = tool.Handler
			break
		}
	}

	if handler == nil {
		t.Fatal("get_search_providers tool not found")
	}

	result, err := handler(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if resultMap["default_provider"] != "brave" {
		t.Errorf("Expected default_provider 'brave', got '%v'", resultMap["default_provider"])
	}

	providers, ok := resultMap["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("providers is not a slice of maps")
	}

	if len(providers) == 0 {
		t.Error("Expected at least one provider")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := Config{
		Enabled: true,
	}

	skill := NewSearchSkill(cfg)

	// Check that defaults are applied
	if skill.config.MaxResults != 5 {
		t.Errorf("Expected default MaxResults 5, got %d", skill.config.MaxResults)
	}

	if skill.config.TimeoutSecs != 30 {
		t.Errorf("Expected default TimeoutSecs 30, got %d", skill.config.TimeoutSecs)
	}
}

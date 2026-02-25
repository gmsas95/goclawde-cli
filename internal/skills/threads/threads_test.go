package threads

import (
	"testing"
)

func TestNewThreadsSkill(t *testing.T) {
	cfg := Config{
		Enabled:       true,
		AccessToken:   "test_token",
		TimeoutSecs:   30,
		MaxTextLength: 500,
	}

	skill := NewThreadsSkill(cfg)

	if skill.Name() != "threads" {
		t.Errorf("Expected name 'threads', got '%s'", skill.Name())
	}

	if !skill.IsEnabled() {
		t.Error("Expected skill to be enabled")
	}

	if len(skill.Tools()) != 4 {
		t.Errorf("Expected 4 tools, got %d", len(skill.Tools()))
	}
}

func TestNewThreadsSkillDefaults(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		AccessToken: "test_token",
	}

	skill := NewThreadsSkill(cfg)

	if skill.config.BaseURL != "https://graph.threads.net/v1.0" {
		t.Errorf("Expected default base URL, got '%s'", skill.config.BaseURL)
	}

	if skill.config.TimeoutSecs != 30 {
		t.Errorf("Expected default timeout 30, got %d", skill.config.TimeoutSecs)
	}

	if skill.config.MaxTextLength != 500 {
		t.Errorf("Expected default max text length 500, got %d", skill.config.MaxTextLength)
	}
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		token    string
		expected bool
	}{
		{"enabled with token", true, "test_token", true},
		{"enabled without token", true, "", false},
		{"disabled with token", false, "test_token", false},
		{"disabled without token", false, "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := Config{
				Enabled:     test.enabled,
				AccessToken: test.token,
			}
			skill := NewThreadsSkill(cfg)
			if skill.IsEnabled() != test.expected {
				t.Errorf("Expected IsEnabled() = %v, got %v", test.expected, skill.IsEnabled())
			}
		})
	}
}

func TestTools(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		AccessToken: "test_token",
	}

	skill := NewThreadsSkill(cfg)
	tools := skill.Tools()

	expectedTools := []string{
		"threads_create_post",
		"threads_create_media_post",
		"threads_get_user",
		"threads_list_posts",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	for i, expected := range expectedTools {
		if tools[i].Name != expected {
			t.Errorf("Expected tool %d to be '%s', got '%s'", i, expected, tools[i].Name)
		}
	}
}

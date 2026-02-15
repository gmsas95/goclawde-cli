package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	content := `# Test env file
KEY1=value1
KEY2="quoted value"
KEY3='single quoted'
# Comment
KEY4=value4
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("KEY1")
	os.Unsetenv("KEY2")
	os.Unsetenv("KEY3")
	os.Unsetenv("KEY4")

	if err := loadEnvFile(envFile); err != nil {
		t.Fatalf("loadEnvFile failed: %v", err)
	}

	if os.Getenv("KEY1") != "value1" {
		t.Errorf("KEY1 not set correctly: %s", os.Getenv("KEY1"))
	}
	if os.Getenv("KEY2") != "quoted value" {
		t.Errorf("KEY2 not set correctly: %s", os.Getenv("KEY2"))
	}
	if os.Getenv("KEY3") != "single quoted" {
		t.Errorf("KEY3 not set correctly: %s", os.Getenv("KEY3"))
	}
	if os.Getenv("KEY4") != "value4" {
		t.Errorf("KEY4 not set correctly: %s", os.Getenv("KEY4"))
	}
}

func TestLoadEnvFile_DoesNotOverride(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	content := `EXISTING_KEY=new_value`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("EXISTING_KEY", "original_value")
	defer os.Unsetenv("EXISTING_KEY")

	if err := loadEnvFile(envFile); err != nil {
		t.Fatalf("loadEnvFile failed: %v", err)
	}

	if os.Getenv("EXISTING_KEY") != "original_value" {
		t.Error("loadEnvFile should not override existing env vars")
	}
}

func TestGetEnvWithFallback(t *testing.T) {
	os.Unsetenv("FALLBACK_KEY1")
	os.Unsetenv("FALLBACK_KEY2")

	result := GetEnvWithFallback("FALLBACK_KEY1", "FALLBACK_KEY2")
	if result != "" {
		t.Error("Expected empty string when no keys set")
	}

	os.Setenv("FALLBACK_KEY2", "value2")
	defer os.Unsetenv("FALLBACK_KEY2")

	result = GetEnvWithFallback("FALLBACK_KEY1", "FALLBACK_KEY2")
	if result != "value2" {
		t.Errorf("Expected value2, got %s", result)
	}

	os.Setenv("FALLBACK_KEY1", "value1")
	defer os.Unsetenv("FALLBACK_KEY1")

	result = GetEnvWithFallback("FALLBACK_KEY1", "FALLBACK_KEY2")
	if result != "value1" {
		t.Errorf("Expected value1 (first priority), got %s", result)
	}
}

func TestGetEnvDefault(t *testing.T) {
	os.Unsetenv("DEFAULT_KEY")

	result := GetEnvDefault("DEFAULT_KEY", "fallback")
	if result != "fallback" {
		t.Errorf("Expected fallback, got %s", result)
	}

	os.Setenv("DEFAULT_KEY", "actual")
	defer os.Unsetenv("DEFAULT_KEY")

	result = GetEnvDefault("DEFAULT_KEY", "fallback")
	if result != "actual" {
		t.Errorf("Expected actual, got %s", result)
	}
}

func TestResolveEnvWithAliases(t *testing.T) {
	os.Unsetenv("GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY")
	os.Unsetenv("KIMI_API_KEY")
	os.Unsetenv("MOONSHOT_API_KEY")

	result := ResolveEnvWithAliases("GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY")
	if result != "" {
		t.Error("Expected empty when no keys set")
	}

	os.Setenv("MOONSHOT_API_KEY", "moonshot_value")
	defer os.Unsetenv("MOONSHOT_API_KEY")

	result = ResolveEnvWithAliases("GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY")
	if result != "moonshot_value" {
		t.Errorf("Expected moonshot_value from alias, got %s", result)
	}

	os.Setenv("KIMI_API_KEY", "kimi_value")
	defer os.Unsetenv("KIMI_API_KEY")

	result = ResolveEnvWithAliases("GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY")
	if result != "kimi_value" {
		t.Errorf("Expected kimi_value from first alias, got %s", result)
	}

	os.Setenv("GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY", "canonical_value")
	defer os.Unsetenv("GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY")

	result = ResolveEnvWithAliases("GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY")
	if result != "canonical_value" {
		t.Errorf("Expected canonical_value, got %s", result)
	}
}

func TestMissingEnvError(t *testing.T) {
	err := &MissingEnvError{Key: "TEST_KEY"}

	if err.Error() != "required environment variable not set: TEST_KEY" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestGetRequiredEnv(t *testing.T) {
	os.Unsetenv("REQUIRED_TEST_KEY")

	_, err := GetRequiredEnv("REQUIRED_TEST_KEY")
	if err == nil {
		t.Error("Expected error for missing required env var")
	}

	os.Setenv("REQUIRED_TEST_KEY", "required_value")
	defer os.Unsetenv("REQUIRED_TEST_KEY")

	val, err := GetRequiredEnv("REQUIRED_TEST_KEY")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != "required_value" {
		t.Errorf("Expected required_value, got %s", val)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, test := range tests {
		result := expandPath(test.input)
		if result != test.expected {
			t.Errorf("expandPath(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestEnvAliases_Exist(t *testing.T) {
	requiredAliases := map[string][]string{
		"GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY":     {"KIMI_API_KEY"},
		"GOCLAWDE_LLM_PROVIDERS_OPENAI_API_KEY":   {"OPENAI_API_KEY"},
		"GOCLAWDE_LLM_PROVIDERS_ANTHROPIC_API_KEY": {"ANTHROPIC_API_KEY"},
		"GOCLAWDE_CHANNELS_TELEGRAM_BOT_TOKEN":    {"TELEGRAM_BOT_TOKEN"},
		"GOCLAWDE_SKILLS_GITHUB_TOKEN":            {"GITHUB_TOKEN"},
	}

	for canonical, aliases := range requiredAliases {
		for _, alias := range aliases {
			found := false
			if envAliases[canonical] != nil {
				for _, a := range envAliases[canonical] {
					if a == alias {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Missing alias %s for %s", alias, canonical)
			}
		}
	}
}

func BenchmarkLoadEnvFile(b *testing.B) {
	tmpDir := b.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	content := `KEY1=value1
KEY2=value2
KEY3=value3
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loadEnvFile(envFile)
	}
}

func BenchmarkGetEnvWithFallback(b *testing.B) {
	os.Setenv("BENCH_KEY", "value")
	defer os.Unsetenv("BENCH_KEY")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetEnvWithFallback("BENCH_KEY", "FALLBACK")
	}
}

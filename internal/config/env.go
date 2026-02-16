package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type EnvVar struct {
	Key   string
	Value string
}

func LoadEnvFiles() error {
	envPaths := []string{
		"./.env",
	}

	if home, err := os.UserHomeDir(); err == nil {
		envPaths = append(envPaths,
			filepath.Join(home, ".myrai", ".env"),
			filepath.Join(home, ".config", "myrai", ".env"),
		)
	}

	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			if err := loadEnvFile(path); err != nil {
				return err
			}
		}
	}

	return nil
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = strings.Trim(value, `"`)
		} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, `'`)
		}

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func GetEnvWithFallback(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}

func GetEnvDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

var envAliases = map[string][]string{
	"MYRAI_LLM_PROVIDERS_KIMI_API_KEY": {"KIMI_API_KEY", "MOONSHOT_API_KEY"},
	"MYRAI_LLM_PROVIDERS_OPENAI_API_KEY": {"OPENAI_API_KEY"},
	"MYRAI_LLM_PROVIDERS_ANTHROPIC_API_KEY": {"ANTHROPIC_API_KEY"},
	"MYRAI_LLM_PROVIDERS_GOOGLE_API_KEY": {"GOOGLE_API_KEY", "GEMINI_API_KEY"},
	"MYRAI_LLM_PROVIDERS_OPENROUTER_API_KEY": {"OPENROUTER_API_KEY"},
	"MYRAI_LLM_PROVIDERS_DEEPSEEK_API_KEY": {"DEEPSEEK_API_KEY"},
	"MYRAI_CHANNELS_TELEGRAM_BOT_TOKEN": {"TELEGRAM_BOT_TOKEN"},
	"MYRAI_CHANNELS_DISCORD_TOKEN": {"DISCORD_BOT_TOKEN", "DISCORD_TOKEN"},
	"MYRAI_SKILLS_GITHUB_TOKEN": {"GITHUB_TOKEN"},
	"MYRAI_SKILLS_WEATHER_API_KEY": {"WEATHER_API_KEY"},
	"MYRAI_SECURITY_JWT_SECRET": {"MYRAI_JWT_SECRET"},
	"MYRAI_SECURITY_ADMIN_PASSWORD": {"MYRAI_ADMIN_PASSWORD"},
}

func ResolveEnvWithAliases(canonicalKey string) string {
	if val := os.Getenv(canonicalKey); val != "" {
		return val
	}

	if aliases, ok := envAliases[canonicalKey]; ok {
		for _, alias := range aliases {
			if val := os.Getenv(alias); val != "" {
				return val
			}
		}
	}

	return ""
}

func GetRequiredEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", &MissingEnvError{Key: key}
	}
	return val, nil
}

type MissingEnvError struct {
	Key string
}

func (e *MissingEnvError) Error() string {
	return "required environment variable not set: " + e.Key
}

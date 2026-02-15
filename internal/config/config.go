package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for Jimmy.ai
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Channels ChannelsConfig `mapstructure:"channels"`
	Tools    ToolsConfig    `mapstructure:"tools"`
	Security SecurityConfig `mapstructure:"security"`
	Skills   SkillsConfig   `mapstructure:"skills"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Address      string `mapstructure:"address"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// LLMConfig holds language model settings
type LLMConfig struct {
	DefaultProvider string            `mapstructure:"default_provider"`
	Providers       map[string]Provider `mapstructure:"providers"`
}

// Provider holds individual LLM provider configuration
type Provider struct {
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
	Model    string `mapstructure:"model"`
	Timeout  int    `mapstructure:"timeout"`
	MaxTokens int   `mapstructure:"max_tokens"`
}

// StorageConfig holds database settings
type StorageConfig struct {
	DataDir     string `mapstructure:"data_dir"`
	SQLitePath  string `mapstructure:"sqlite_path"`
	BadgerPath  string `mapstructure:"badger_path"`
}

// ChannelsConfig holds integration settings
type ChannelsConfig struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	WhatsApp WhatsAppConfig `mapstructure:"whatsapp"`
	Discord  DiscordConfig  `mapstructure:"discord"`
}

// TelegramConfig holds Telegram bot settings
type TelegramConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	BotToken  string  `mapstructure:"bot_token"`
	Webhook   string  `mapstructure:"webhook"`
	AllowList []int64 `mapstructure:"allow_list"`
}

// WhatsAppConfig holds WhatsApp settings
type WhatsAppConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// DiscordConfig holds Discord bot settings
type DiscordConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"`
}

// ToolsConfig holds tool system settings
type ToolsConfig struct {
	Enabled    []string          `mapstructure:"enabled"`
	AllowedCmds []string         `mapstructure:"allowed_commands"`
	Sandbox    bool              `mapstructure:"sandbox"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	JWTSecret     string   `mapstructure:"jwt_secret"`
	AdminPassword string   `mapstructure:"admin_password"`
	AllowOrigins  []string `mapstructure:"allow_origins"`
}

// SkillsConfig holds skills configuration
type SkillsConfig struct {
	GitHub  GitHubSkillConfig  `mapstructure:"github"`
	Weather WeatherSkillConfig `mapstructure:"weather"`
}

// GitHubSkillConfig holds GitHub skill settings
type GitHubSkillConfig struct {
	Token string `mapstructure:"token"`
}

// WeatherSkillConfig holds weather skill settings
type WeatherSkillConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// Load loads configuration from file, env, and defaults
func Load(configPath, dataDir string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Determine data directory
	if dataDir == "" {
		dataDir = getDefaultDataDir()
	}
	
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	v.Set("storage.data_dir", dataDir)
	v.Set("storage.sqlite_path", filepath.Join(dataDir, "jimmy.db"))
	v.Set("storage.badger_path", filepath.Join(dataDir, "badger"))

	// Config file path
	if configPath == "" {
		configPath = filepath.Join(dataDir, "jimmy.yaml")
	}

	// If config file exists, load it
	if _, err := os.Stat(configPath); err == nil {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// Environment variables (JIMMY_SERVER_PORT, JIMMY_LLM_API_KEY, etc.)
	v.SetEnvPrefix("JIMMY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal to struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Load API keys from environment (Viper doesn't handle nested maps well with env vars)
	loadEnvOverrides(&cfg)

	// Validate
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.address", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)

	// LLM defaults
	v.SetDefault("llm.default_provider", "kimi")
	v.SetDefault("llm.providers.kimi.base_url", "https://api.moonshot.cn/v1")
	v.SetDefault("llm.providers.kimi.model", "kimi-k2.5")
	v.SetDefault("llm.providers.kimi.timeout", 60)
	v.SetDefault("llm.providers.kimi.max_tokens", 4096)

	// Tools defaults
	v.SetDefault("tools.enabled", []string{"read_file", "write_file", "list_dir", "exec_command", "web_search"})
	v.SetDefault("tools.sandbox", true)

	// Security defaults
	v.SetDefault("security.allow_origins", []string{"*"})
}

func getDefaultDataDir() string {
	// Try XDG_DATA_HOME first
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "jimmy")
	}

	// Fall back to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "./data"
	}

	return filepath.Join(home, ".local", "share", "jimmy")
}

// loadEnvOverrides loads specific env vars that Viper doesn't handle well with nested maps
func loadEnvOverrides(cfg *Config) {
	// Helper to get env var
	getEnv := func(key, fallback string) string {
		if val := os.Getenv(key); val != "" {
			return val
		}
		return fallback
	}

	// LLM Provider settings
	cfg.LLM.DefaultProvider = getEnv("JIMMY_LLM_DEFAULT_PROVIDER", cfg.LLM.DefaultProvider)

	// Initialize providers map if nil
	if cfg.LLM.Providers == nil {
		cfg.LLM.Providers = make(map[string]Provider)
	}

	// Kimi provider
	if apiKey := os.Getenv("JIMMY_LLM_PROVIDERS_KIMI_API_KEY"); apiKey != "" {
		kimi := cfg.LLM.Providers["kimi"]
		kimi.APIKey = apiKey
		kimi.BaseURL = getEnv("JIMMY_LLM_PROVIDERS_KIMI_BASE_URL", kimi.BaseURL)
		kimi.Model = getEnv("JIMMY_LLM_PROVIDERS_KIMI_MODEL", kimi.Model)
		cfg.LLM.Providers["kimi"] = kimi
	}

	// OpenRouter provider
	if apiKey := os.Getenv("JIMMY_LLM_PROVIDERS_OPENROUTER_API_KEY"); apiKey != "" {
		or := cfg.LLM.Providers["openrouter"]
		or.APIKey = apiKey
		or.BaseURL = getEnv("JIMMY_LLM_PROVIDERS_OPENROUTER_BASE_URL", or.BaseURL)
		or.Model = getEnv("JIMMY_LLM_PROVIDERS_OPENROUTER_MODEL", or.Model)
		cfg.LLM.Providers["openrouter"] = or
	}

	// Server settings
	cfg.Server.Address = getEnv("JIMMY_SERVER_ADDRESS", cfg.Server.Address)
	if port := os.Getenv("JIMMY_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	// Storage settings
	cfg.Storage.DataDir = getEnv("JIMMY_STORAGE_DATA_DIR", cfg.Storage.DataDir)

	// Security settings
	cfg.Security.JWTSecret = getEnv("JIMMY_SECURITY_JWT_SECRET", cfg.Security.JWTSecret)
	cfg.Security.AdminPassword = getEnv("JIMMY_SECURITY_ADMIN_PASSWORD", cfg.Security.AdminPassword)

	// Skills settings
	cfg.Skills.GitHub.Token = getEnv("JIMMY_SKILLS_GITHUB_TOKEN", cfg.Skills.GitHub.Token)
	cfg.Skills.Weather.APIKey = getEnv("JIMMY_SKILLS_WEATHER_API_KEY", cfg.Skills.Weather.APIKey)
}

func validate(cfg *Config) error {
	// Check for required LLM configuration
	if cfg.LLM.DefaultProvider == "" {
		return fmt.Errorf("llm.default_provider is required")
	}

	provider, ok := cfg.LLM.Providers[cfg.LLM.DefaultProvider]
	if !ok {
		return fmt.Errorf("provider %s not configured", cfg.LLM.DefaultProvider)
	}

	if provider.APIKey == "" {
		return fmt.Errorf("llm.providers.%s.api_key is required", cfg.LLM.DefaultProvider)
	}

	// Generate JWT secret if not provided
	if cfg.Security.JWTSecret == "" {
		cfg.Security.JWTSecret = generateRandomString(32)
	}

	return nil
}

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

// GetProvider returns the provider configuration by name
func (c *Config) GetProvider(name string) (Provider, bool) {
	p, ok := c.LLM.Providers[name]
	return p, ok
}

// DefaultProvider returns the default provider configuration
func (c *Config) DefaultProvider() (Provider, error) {
	p, ok := c.LLM.Providers[c.LLM.DefaultProvider]
	if !ok {
		return Provider{}, fmt.Errorf("default provider %s not found", c.LLM.DefaultProvider)
	}
	return p, nil
}

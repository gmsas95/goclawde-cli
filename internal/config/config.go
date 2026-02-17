package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Channels ChannelsConfig `mapstructure:"channels"`
	Tools    ToolsConfig    `mapstructure:"tools"`
	Security SecurityConfig `mapstructure:"security"`
	Skills   SkillsConfig   `mapstructure:"skills"`
	MCP      MCPConfig      `mapstructure:"mcp"`
	Cron     CronConfig     `mapstructure:"cron"`
	Vector   VectorConfig   `mapstructure:"vector"`
}

type ServerConfig struct {
	Address      string `mapstructure:"address"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

type LLMConfig struct {
	DefaultProvider string              `mapstructure:"default_provider"`
	Providers       map[string]Provider `mapstructure:"providers"`
}

type Provider struct {
	APIKey    string `mapstructure:"api_key"`
	BaseURL   string `mapstructure:"base_url"`
	Model     string `mapstructure:"model"`
	Timeout   int    `mapstructure:"timeout"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

type StorageConfig struct {
	DataDir    string `mapstructure:"data_dir"`
	SQLitePath string `mapstructure:"sqlite_path"`
	BadgerPath string `mapstructure:"badger_path"`
}

type ChannelsConfig struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	WhatsApp WhatsAppConfig `mapstructure:"whatsapp"`
	Discord  DiscordConfig  `mapstructure:"discord"`
	Slack    SlackConfig    `mapstructure:"slack"`
}

type TelegramConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	BotToken  string  `mapstructure:"bot_token"`
	Webhook   string  `mapstructure:"webhook"`
	AllowList []int64 `mapstructure:"allow_list"`
}

type WhatsAppConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type DiscordConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"`
}

type SlackConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BotToken string `mapstructure:"bot_token"`
	AppToken string `mapstructure:"app_token"`
}

type ToolsConfig struct {
	Enabled     []string `mapstructure:"enabled"`
	AllowedCmds []string `mapstructure:"allowed_commands"`
	Sandbox     bool     `mapstructure:"sandbox"`
}

type SecurityConfig struct {
	JWTSecret     string   `mapstructure:"jwt_secret"`
	AdminPassword string   `mapstructure:"admin_password"`
	AllowOrigins  []string `mapstructure:"allow_origins"`
	GatewayToken  string   `mapstructure:"gateway_token"`
}

type SkillsConfig struct {
	GitHub  GitHubSkillConfig  `mapstructure:"github"`
	Weather WeatherSkillConfig `mapstructure:"weather"`
	Browser BrowserSkillConfig `mapstructure:"browser"`
	Brave   BraveSkillConfig   `mapstructure:"brave"`
	Search  SearchSkillConfig  `mapstructure:"search"`
	Vision  VisionSkillConfig  `mapstructure:"vision"`
}

type GitHubSkillConfig struct {
	Token string `mapstructure:"token"`
}

type WeatherSkillConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type BrowserSkillConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Headless       bool   `mapstructure:"headless"`
	ExecutablePath string `mapstructure:"executable_path"`
}

type BraveSkillConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type SearchSkillConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Provider    string `mapstructure:"provider"` // brave, serper, google, duckduckgo
	APIKey      string `mapstructure:"api_key"`
	MaxResults  int    `mapstructure:"max_results"`
	TimeoutSecs int    `mapstructure:"timeout_seconds"`
}

type VisionSkillConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	VisionModel string `mapstructure:"vision_model"` // gpt-4o, claude-3-opus, gemini-pro-vision
}

// MCPConfig holds MCP server configuration
type MCPConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// CronConfig holds cron runner configuration
type CronConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	IntervalMinutes int  `mapstructure:"interval_minutes"`
	MaxConcurrent   int  `mapstructure:"max_concurrent"`
}

// VectorConfig holds vector search configuration
type VectorConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Provider       string `mapstructure:"provider"` // "local", "openai", "ollama"
	EmbeddingModel string `mapstructure:"embedding_model"`
	Dimension      int    `mapstructure:"dimension"`
	OpenAIAPIKey   string `mapstructure:"openai_api_key"`
	OllamaHost     string `mapstructure:"ollama_host"`
}

// Load loads configuration from file, env, and defaults
func Load(configPath, dataDir string) (*Config, error) {
	if err := LoadEnvFiles(); err != nil {
		// Log but don't fail - .env files are optional
		fmt.Fprintf(os.Stderr, "Warning: error loading .env files: %v\n", err)
	}

	v := viper.New()

	setDefaults(v)

	if dataDir == "" {
		dataDir = getDefaultDataDir()
	}

	dataDir = expandPath(dataDir)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	v.Set("storage.data_dir", dataDir)
	v.Set("storage.sqlite_path", filepath.Join(dataDir, "myrai.db"))
	v.Set("storage.badger_path", filepath.Join(dataDir, "badger"))

	if configPath == "" {
		configPath = filepath.Join(dataDir, "myrai.yaml")
	}

	configPath = expandPath(configPath)

	if _, err := os.Stat(configPath); err == nil {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	v.SetEnvPrefix("MYRAI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	loadEnvOverrides(&cfg)
	loadStandardEnvVars(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func loadStandardEnvVars(cfg *Config) {
	if cfg.LLM.Providers == nil {
		cfg.LLM.Providers = make(map[string]Provider)
	}

	// Major LLM providers (OpenAI-compatible)
	loadProviderFromEnv(cfg, "openai", "OPENAI_API_KEY", "https://api.openai.com/v1", "gpt-5.2")
	loadProviderFromEnv(cfg, "anthropic", "ANTHROPIC_API_KEY", "https://api.anthropic.com/v1", "claude-opus-4.6")
	loadProviderFromEnv(cfg, "google", "GOOGLE_API_KEY", "https://generativelanguage.googleapis.com/v1beta", "gemini-3-pro-preview")
	loadProviderFromEnv(cfg, "openrouter", "OPENROUTER_API_KEY", "https://openrouter.ai/api/v1", "anthropic/claude-opus-4.6")
	loadProviderFromEnv(cfg, "kimi", "KIMI_API_KEY", "https://api.moonshot.cn/v1", "kimi-k2.5")
	loadProviderFromEnv(cfg, "deepseek", "DEEPSEEK_API_KEY", "https://api.deepseek.com/v1", "deepseek-v3.2")

	// Additional OpenAI-compatible providers
	loadProviderFromEnv(cfg, "groq", "GROQ_API_KEY", "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile")
	loadProviderFromEnv(cfg, "mistral", "MISTRAL_API_KEY", "https://api.mistral.ai/v1", "mistral-large-latest")
	loadProviderFromEnv(cfg, "together", "TOGETHER_API_KEY", "https://api.together.xyz/v1", "meta-llama/Llama-3.3-70B-Instruct-Turbo")
	loadProviderFromEnv(cfg, "cerebras", "CEREBRAS_API_KEY", "https://api.cerebras.ai/v1", "llama-3.3-70b")
	loadProviderFromEnv(cfg, "xai", "XAI_API_KEY", "https://api.x.ai/v1", "grok-2-latest")
	loadProviderFromEnv(cfg, "perplexity", "PERPLEXITY_API_KEY", "https://api.perplexity.ai/v1", "sonar-reasoning-pro")
	loadProviderFromEnv(cfg, "fireworks", "FIREWORKS_API_KEY", "https://api.fireworks.ai/inference/v1", "accounts/fireworks/models/llama-v3p3-70b-instruct")
	loadProviderFromEnv(cfg, "novita", "NOVITA_API_KEY", "https://api.novita.ai/v3/openai", "meta-llama/llama-3.3-70b-instruct")
	loadProviderFromEnv(cfg, "siliconflow", "SILICONFLOW_API_KEY", "https://api.siliconflow.cn/v1", "deepseek-ai/DeepSeek-V3")
	loadProviderFromEnv(cfg, "zhipu", "ZHIPU_API_KEY", "https://open.bigmodel.cn/api/paas/v4", "glm-4-plus")
	loadProviderFromEnv(cfg, "moonshot", "MOONSHOT_API_KEY", "https://api.moonshot.cn/v1", "moonshot-v1-128k")

	// Local/self-hosted providers
	loadProviderFromEnv(cfg, "ollama", "OLLAMA_API_KEY", "http://localhost:11434/v1", "llama3.2")
	loadProviderFromEnv(cfg, "localai", "LOCALAI_API_KEY", "http://localhost:8080/v1", "llama3")
	loadProviderFromEnv(cfg, "vllm", "VLLM_API_KEY", "http://localhost:8000/v1", "llama3")

	// Azure OpenAI (special handling)
	if apiKey := os.Getenv("AZURE_OPENAI_API_KEY"); apiKey != "" {
		baseURL := GetEnvDefault("AZURE_OPENAI_ENDPOINT", "https://your-resource.openai.azure.com")
		baseURL = strings.TrimSuffix(baseURL, "/") + "/openai/deployments/" + GetEnvDefault("AZURE_OPENAI_DEPLOYMENT", "gpt-4o")
		provider := cfg.LLM.Providers["azure"]
		provider.APIKey = apiKey
		provider.BaseURL = baseURL
		provider.Model = GetEnvDefault("AZURE_OPENAI_MODEL", "gpt-4o")
		provider.Timeout = 60
		provider.MaxTokens = 4096
		cfg.LLM.Providers["azure"] = provider
	}

	if token := GetEnvWithFallback("MYRAI_GATEWAY_TOKEN", "GATEWAY_TOKEN"); token != "" {
		cfg.Security.GatewayToken = token
	}

	if token := ResolveEnvWithAliases("MYRAI_CHANNELS_TELEGRAM_BOT_TOKEN"); token != "" {
		cfg.Channels.Telegram.BotToken = token
		cfg.Channels.Telegram.Enabled = true
	}

	if token := ResolveEnvWithAliases("MYRAI_CHANNELS_DISCORD_TOKEN"); token != "" {
		cfg.Channels.Discord.Token = token
		cfg.Channels.Discord.Enabled = true
	}

	if token := GetEnvWithFallback("SLACK_BOT_TOKEN"); token != "" {
		cfg.Channels.Slack.BotToken = token
		cfg.Channels.Slack.Enabled = true
	}

	if token := GetEnvWithFallback("SLACK_APP_TOKEN"); token != "" {
		cfg.Channels.Slack.AppToken = token
	}

	if token := ResolveEnvWithAliases("MYRAI_SKILLS_GITHUB_TOKEN"); token != "" {
		cfg.Skills.GitHub.Token = token
	}

	if key := ResolveEnvWithAliases("MYRAI_SKILLS_BRAVE_API_KEY"); key != "" {
		cfg.Skills.Brave.APIKey = key
		cfg.Skills.Search.APIKey = key
	}

	if key := ResolveEnvWithAliases("MYRAI_SKILLS_SEARCH_API_KEY"); key != "" {
		cfg.Skills.Search.APIKey = key
	}

	if provider := ResolveEnvWithAliases("MYRAI_SKILLS_SEARCH_PROVIDER"); provider != "" {
		cfg.Skills.Search.Provider = provider
	}
}

func loadProviderFromEnv(cfg *Config, name, envKey, defaultBaseURL, defaultModel string) {
	apiKey := os.Getenv(envKey)
	if apiKey == "" {
		return
	}

	provider := cfg.LLM.Providers[name]
	provider.APIKey = apiKey

	if provider.BaseURL == "" {
		provider.BaseURL = GetEnvDefault(strings.ToUpper(name)+"_BASE_URL", defaultBaseURL)
	}
	if provider.Model == "" {
		provider.Model = GetEnvDefault(strings.ToUpper(name)+"_MODEL", defaultModel)
	}
	if provider.Timeout == 0 {
		provider.Timeout = 60
	}
	if provider.MaxTokens == 0 {
		provider.MaxTokens = 4096
	}

	cfg.LLM.Providers[name] = provider
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

	// MCP defaults
	v.SetDefault("mcp.enabled", false)
	v.SetDefault("mcp.host", "0.0.0.0")
	v.SetDefault("mcp.port", 8081)

	// Cron defaults
	v.SetDefault("cron.enabled", true)
	v.SetDefault("cron.interval_minutes", 1)
	v.SetDefault("cron.max_concurrent", 3)

	// Vector defaults
	v.SetDefault("vector.enabled", false)
	v.SetDefault("vector.provider", "local")
	v.SetDefault("vector.embedding_model", "all-MiniLM-L6-v2")
	v.SetDefault("vector.dimension", 384)
	v.SetDefault("vector.ollama_host", "http://localhost:11434")

	// Search defaults
	v.SetDefault("search.enabled", true)
	v.SetDefault("search.provider", "brave")
	v.SetDefault("search.max_results", 5)
	v.SetDefault("search.timeout_seconds", 30)

	// Vision defaults
	v.SetDefault("vision.enabled", false)
	v.SetDefault("vision.vision_model", "gpt-4o")
}

func getDefaultDataDir() string {
	// Try XDG_DATA_HOME first
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "myrai")
	}

	// Fall back to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "./data"
	}

	return filepath.Join(home, ".local", "share", "myrai")
}

func loadEnvOverrides(cfg *Config) {
	cfg.LLM.DefaultProvider = GetEnvDefault("MYRAI_LLM_DEFAULT_PROVIDER", cfg.LLM.DefaultProvider)

	if cfg.LLM.Providers == nil {
		cfg.LLM.Providers = make(map[string]Provider)
	}

	if apiKey := ResolveEnvWithAliases("MYRAI_LLM_PROVIDERS_KIMI_API_KEY"); apiKey != "" {
		kimi := cfg.LLM.Providers["kimi"]
		kimi.APIKey = apiKey
		kimi.BaseURL = GetEnvDefault("MYRAI_LLM_PROVIDERS_KIMI_BASE_URL", kimi.BaseURL)
		kimi.Model = GetEnvDefault("MYRAI_LLM_PROVIDERS_KIMI_MODEL", kimi.Model)
		cfg.LLM.Providers["kimi"] = kimi
	}

	if apiKey := ResolveEnvWithAliases("MYRAI_LLM_PROVIDERS_OPENROUTER_API_KEY"); apiKey != "" {
		or := cfg.LLM.Providers["openrouter"]
		or.APIKey = apiKey
		or.BaseURL = GetEnvDefault("MYRAI_LLM_PROVIDERS_OPENROUTER_BASE_URL", or.BaseURL)
		or.Model = GetEnvDefault("MYRAI_LLM_PROVIDERS_OPENROUTER_MODEL", or.Model)
		cfg.LLM.Providers["openrouter"] = or
	}

	cfg.Server.Address = GetEnvDefault("MYRAI_SERVER_ADDRESS", cfg.Server.Address)
	if port := os.Getenv("MYRAI_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	cfg.Storage.DataDir = GetEnvDefault("MYRAI_STORAGE_DATA_DIR", cfg.Storage.DataDir)

	cfg.Security.JWTSecret = ResolveEnvWithAliases("MYRAI_SECURITY_JWT_SECRET")
	cfg.Security.AdminPassword = ResolveEnvWithAliases("MYRAI_SECURITY_ADMIN_PASSWORD")

	cfg.Skills.GitHub.Token = ResolveEnvWithAliases("MYRAI_SKILLS_GITHUB_TOKEN")
	cfg.Skills.Weather.APIKey = ResolveEnvWithAliases("MYRAI_SKILLS_WEATHER_API_KEY")

	cfg.Channels.Telegram.BotToken = ResolveEnvWithAliases("MYRAI_CHANNELS_TELEGRAM_BOT_TOKEN")
	cfg.Channels.Discord.Token = ResolveEnvWithAliases("MYRAI_CHANNELS_DISCORD_TOKEN")
}

func validate(cfg *Config) error {
	if cfg.LLM.DefaultProvider == "" {
		cfg.LLM.DefaultProvider = "kimi"
	}

	provider, ok := cfg.LLM.Providers[cfg.LLM.DefaultProvider]
	if !ok || provider.APIKey == "" {
		hasAnyProvider := false
		for name, p := range cfg.LLM.Providers {
			if p.APIKey != "" {
				cfg.LLM.DefaultProvider = name
				hasAnyProvider = true
				break
			}
		}

		if !hasAnyProvider {
			return fmt.Errorf("no LLM provider configured. Set an API key via environment variable (e.g., OPENAI_API_KEY, ANTHROPIC_API_KEY, KIMI_API_KEY) or in myrai.yaml")
		}
	}

	if cfg.Security.JWTSecret == "" {
		cfg.Security.JWTSecret = generateRandomString(32)
	}

	if cfg.Security.GatewayToken == "" {
		cfg.Security.GatewayToken = generateRandomString(32)
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

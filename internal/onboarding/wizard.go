package onboarding

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/persona"
	"go.uber.org/zap"
)

// Wizard handles the interactive setup process
type Wizard struct {
	reader    *bufio.Reader
	logger    *zap.Logger
	workspace string
	configDir string
	config    *WizardConfig
	spinner   chan bool
}

// WizardConfig holds the configuration collected during setup
type WizardConfig struct {
	UserName           string
	CommunicationStyle string
	Expertise          []string
	Goals              []string
	LLMProvider        string
	APIKey             string
	BaseURL            string
	DefaultModel       string
	EnableTelegram     bool
	TelegramToken      string
	SearchAPIKey       string
	SearchProvider     string
	EnableVision       bool
	VisionModel        string
}

// NewWizard creates a new setup wizard
func NewWizard(logger *zap.Logger) *Wizard {
	return &Wizard{
		reader: bufio.NewReader(os.Stdin),
		logger: logger,
		config: &WizardConfig{},
	}
}

// Run runs the interactive setup wizard
func (w *Wizard) Run() error {
	// Clear screen and show welcome
	w.clearScreen()
	fmt.Print(SetupWizardWelcome)
	w.waitForEnter()

	// Step 1: Workspace setup
	if err := w.setupWorkspace(); err != nil {
		return fmt.Errorf("workspace setup failed: %w", err)
	}

	// Step 2: User profile
	if err := w.setupUserProfile(); err != nil {
		return fmt.Errorf("user profile setup failed: %w", err)
	}

	// Step 3: AI Configuration
	if err := w.setupAIConfiguration(); err != nil {
		return fmt.Errorf("AI configuration failed: %w", err)
	}

	// Step 4: Optional integrations
	if err := w.setupIntegrations(); err != nil {
		return fmt.Errorf("integrations setup failed: %w", err)
	}

	// Step 5: Create configuration
	spinner := w.startSpinner("⚙️  Creating configuration files...")
	if err := w.createConfiguration(); err != nil {
		w.stopSpinner(spinner)
		return fmt.Errorf("configuration creation failed: %w", err)
	}
	w.stopSpinner(spinner)

	// Step 6: Create persona files
	spinner = w.startSpinner("📝 Creating persona files...")
	if err := w.createPersonaFiles(); err != nil {
		w.stopSpinner(spinner)
		return fmt.Errorf("persona creation failed: %w", err)
	}
	w.stopSpinner(spinner)

	// Show completion message
	w.showCompletion()

	// Verify files were created
	fmt.Println()
	fmt.Println("Verifying installation...")

	configPath := filepath.Join(w.configDir, "myrai.yaml")
	envPath := filepath.Join(w.configDir, ".env")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("⚠️  Warning: Config file was not created!")
	} else {
		fmt.Println("✓ Config file verified:", configPath)
	}

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		fmt.Println("⚠️  Warning: .env file was not created!")
	} else {
		fmt.Println("✓ Secrets file verified:", envPath)
	}

	fmt.Println()
	fmt.Print("Press Enter to exit...")
	w.reader.ReadString('\n')

	return nil
}

func (w *Wizard) setupWorkspace() error {
	w.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Step 1: Workspace Setup                                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get default workspace path (data directory)
	// Try XDG_DATA_HOME first, then fall back to ~/.local/share/myrai
	defaultWorkspace := os.Getenv("XDG_DATA_HOME")
	if defaultWorkspace == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		defaultWorkspace = filepath.Join(home, ".local", "share", "myrai")
	} else {
		defaultWorkspace = filepath.Join(defaultWorkspace, "myrai")
	}

	// Get config directory for storing config and .env files
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		configDir = filepath.Join(home, ".config", "myrai")
	} else {
		configDir = filepath.Join(configDir, "myrai")
	}

	// Store configDir in wizard for later use
	w.configDir = configDir

	fmt.Printf("Where should Myrai store its data? [default: %s]: ", defaultWorkspace)
	workspace, _ := w.reader.ReadString('\n')
	workspace = strings.TrimSpace(workspace)

	if workspace == "" {
		workspace = defaultWorkspace
	}

	w.workspace = workspace

	// Create workspace
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"projects", "diary", "memory"}
	for _, dir := range subdirs {
		if err := os.MkdirAll(filepath.Join(workspace, dir), 0755); err != nil {
			return err
		}
	}

	fmt.Println("✓ Workspace created")
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (w *Wizard) setupUserProfile() error {
	w.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Step 2: Your Profile                                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Name
	fmt.Print("What's your name? ")
	name, _ := w.reader.ReadString('\n')
	w.config.UserName = strings.TrimSpace(name)
	if w.config.UserName == "" {
		w.config.UserName = "User"
	}

	// Communication style
	fmt.Println()
	fmt.Println("How would you prefer Myrai to communicate?")
	for i, style := range CommunicationStyles {
		fmt.Printf("  %d. %s\n", i+1, style)
	}
	fmt.Print("\nSelect (1-5) or describe your own: ")
	styleInput, _ := w.reader.ReadString('\n')
	styleInput = strings.TrimSpace(styleInput)

	if choice, err := parseInt(styleInput); err == nil && choice >= 1 && choice <= 5 {
		w.config.CommunicationStyle = CommunicationStyles[choice-1]
	} else if styleInput != "" {
		w.config.CommunicationStyle = styleInput
	} else {
		w.config.CommunicationStyle = CommunicationStyles[0]
	}

	// Expertise
	fmt.Println()
	fmt.Println("What are your areas of expertise? (comma-separated, e.g., 'Go, Python, Kubernetes')")
	fmt.Print("> ")
	expertise, _ := w.reader.ReadString('\n')
	expertise = strings.TrimSpace(expertise)
	if expertise != "" {
		w.config.Expertise = splitAndTrim(expertise, ",")
	}

	// Goals
	fmt.Println()
	fmt.Println("What are your main goals for using Myrai? (comma-separated)")
	fmt.Print("> ")
	goals, _ := w.reader.ReadString('\n')
	goals = strings.TrimSpace(goals)
	if goals != "" {
		w.config.Goals = splitAndTrim(goals, ",")
	}

	fmt.Println("\n✓ Profile configured")
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (w *Wizard) setupAIConfiguration() error {
	w.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Step 3: AI Configuration                                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("Select your LLM provider:")
	fmt.Println()
	fmt.Println("  === Cloud Providers (Recommended) ===")
	fmt.Println("  1. OpenAI        - GPT-5.2, GPT-5.2 Codex, GPT-5.1")
	fmt.Println("  2. Anthropic     - Claude Opus 4.6, Claude Sonnet 4.5")
	fmt.Println("  3. Google        - Gemini 3 Pro, Gemini 2.5 Pro")
	fmt.Println("  4. Kimi/Moonshot - Recommended for coding, Chinese support")
	fmt.Println()
	fmt.Println("  === Fast & Affordable ===")
	fmt.Println("  5. Groq          - Ultra-fast inference, Llama models")
	fmt.Println("  6. DeepSeek      - Great for coding, very affordable")
	fmt.Println("  7. Together AI   - Many open-source models")
	fmt.Println("  8. Cerebras      - Fastest inference speed")
	fmt.Println()
	fmt.Println("  === Model Aggregators ===")
	fmt.Println("  9. OpenRouter    - Access to 100+ models via one API")
	fmt.Println("  10. Fireworks    - Fast serverless inference")
	fmt.Println()
	fmt.Println("  === Chinese Providers ===")
	fmt.Println("  11. Zhipu (智谱) - GLM-4 models")
	fmt.Println("  12. SiliconFlow  - Qwen, DeepSeek, more")
	fmt.Println("  13. Novita       - Affordable, many models")
	fmt.Println()
	fmt.Println("  === Local/Self-Hosted ===")
	fmt.Println("  14. Ollama       - Run models locally (no API key)")
	fmt.Println("  15. LocalAI      - OpenAI-compatible local server")
	fmt.Println("  16. vLLM         - High-performance local inference")
	fmt.Println()
	fmt.Println("  === Other ===")
	fmt.Println("  17. Mistral      - Mistral Large, Codestral")
	fmt.Println("  18. xAI (Grok)   - Grok models by xAI")
	fmt.Println("  19. Perplexity   - Sonar models with web search")
	fmt.Println("  20. Azure OpenAI - Enterprise OpenAI")
	fmt.Println()
	fmt.Print("Select (1-20) [default: 4]: ")
	providerChoice, _ := w.reader.ReadString('\n')
	providerChoice = strings.TrimSpace(providerChoice)

	var err error
	switch providerChoice {
	case "1":
		w.config.LLMProvider = "openai"
		err = w.configureOpenAI()
	case "2":
		w.config.LLMProvider = "anthropic"
		err = w.configureAnthropic()
	case "3":
		w.config.LLMProvider = "google"
		err = w.configureGoogle()
	case "5":
		w.config.LLMProvider = "groq"
		err = w.configureGroq()
	case "6":
		w.config.LLMProvider = "deepseek"
		err = w.configureDeepSeek()
	case "7":
		w.config.LLMProvider = "together"
		err = w.configureTogether()
	case "8":
		w.config.LLMProvider = "cerebras"
		err = w.configureCerebras()
	case "9":
		w.config.LLMProvider = "openrouter"
		err = w.configureOpenRouter()
	case "10":
		w.config.LLMProvider = "fireworks"
		err = w.configureFireworks()
	case "11":
		w.config.LLMProvider = "zhipu"
		err = w.configureZhipu()
	case "12":
		w.config.LLMProvider = "siliconflow"
		err = w.configureSiliconFlow()
	case "13":
		w.config.LLMProvider = "novita"
		err = w.configureNovita()
	case "14":
		w.config.LLMProvider = "ollama"
		err = w.configureOllama()
	case "15":
		w.config.LLMProvider = "localai"
		err = w.configureLocalAI()
	case "16":
		w.config.LLMProvider = "vllm"
		err = w.configureVLLM()
	case "17":
		w.config.LLMProvider = "mistral"
		err = w.configureMistral()
	case "18":
		w.config.LLMProvider = "xai"
		err = w.configureXAI()
	case "19":
		w.config.LLMProvider = "perplexity"
		err = w.configurePerplexity()
	case "20":
		w.config.LLMProvider = "azure"
		err = w.configureAzure()
	default:
		w.config.LLMProvider = "kimi"
		err = w.configureKimi()
	}

	if err != nil {
		return err
	}

	fmt.Println("\n✓ AI configured")
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (w *Wizard) configureKimi() error {
	fmt.Println()
	fmt.Println("Kimi (Moonshot) Configuration")
	fmt.Println("Get your API key from: https://platform.moonshot.cn")
	fmt.Println()
	fmt.Print("Enter your Kimi API Key (starts with 'sk-'): ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" || !strings.HasPrefix(w.config.APIKey, "sk-") {
		fmt.Println("❌ Invalid API key. Please enter a valid Kimi API key.")
		fmt.Print("Enter your Kimi API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	// Model selection
	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. kimi-k2.5 (default, balanced)")
	fmt.Println("  2. kimi-k2.5-long (longer context)")
	fmt.Print("\nSelect (1-2) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "kimi-k2.5-long"
	default:
		w.config.DefaultModel = "kimi-k2.5"
	}

	return nil
}

func (w *Wizard) configureOpenAI() error {
	fmt.Println()
	fmt.Println("OpenAI Configuration")
	fmt.Println("Get your API key from: https://platform.openai.com")
	fmt.Println()
	fmt.Print("Enter your OpenAI API Key (starts with 'sk-'): ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" || !strings.HasPrefix(w.config.APIKey, "sk-") {
		fmt.Println("❌ Invalid API key. Please enter a valid OpenAI API key.")
		fmt.Print("Enter your OpenAI API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	// Model selection
	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. gpt-5.2 (latest, best overall)")
	fmt.Println("  2. gpt-5.2-codex (best for coding)")
	fmt.Println("  3. gpt-5.1 (previous generation)")
	fmt.Println("  4. gpt-5.1-codex-max (agentic coding)")
	fmt.Print("\nSelect (1-4) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "gpt-5.2-codex"
	case "3":
		w.config.DefaultModel = "gpt-5.1"
	case "4":
		w.config.DefaultModel = "gpt-5.1-codex-max"
	default:
		w.config.DefaultModel = "gpt-5.2"
	}

	return nil
}

func (w *Wizard) configureAnthropic() error {
	fmt.Println()
	fmt.Println("Anthropic Configuration")
	fmt.Println("Get your API key from: https://console.anthropic.com")
	fmt.Println()
	fmt.Print("Enter your Anthropic API Key (starts with 'sk-ant-'): ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" || !strings.HasPrefix(w.config.APIKey, "sk-ant-") {
		fmt.Println("❌ Invalid API key. Please enter a valid Anthropic API key.")
		fmt.Print("Enter your Anthropic API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	// Model selection
	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. claude-opus-4.6 (latest, best overall)")
	fmt.Println("  2. claude-opus-4.5 (powerful)")
	fmt.Println("  3. claude-sonnet-4.5 (balanced)")
	fmt.Println("  4. claude-sonnet-4 (fast)")
	fmt.Print("\nSelect (1-4) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "claude-opus-4.5"
	case "3":
		w.config.DefaultModel = "claude-sonnet-4.5"
	case "4":
		w.config.DefaultModel = "claude-sonnet-4"
	default:
		w.config.DefaultModel = "claude-opus-4.6"
	}

	return nil
}

func (w *Wizard) configureOllama() error {
	fmt.Println()
	fmt.Println("Ollama Configuration")
	fmt.Println("Make sure Ollama is running locally (http://localhost:11434)")
	fmt.Println()

	w.config.APIKey = "ollama"

	fmt.Println("Select your preferred local model:")
	fmt.Println("  1. llama3.3 (default, latest, 70B SOTA)")
	fmt.Println("  2. deepseek-r1 (reasoning model)")
	fmt.Println("  3. gemma3 (vision-capable)")
	fmt.Println("  4. qwen3 (latest Qwen)")
	fmt.Println("  5. qwen2.5-coder (best for coding)")
	fmt.Println("  6. llama3.2 (efficient)")
	fmt.Println("  7. phi4 (Microsoft 14B)")
	fmt.Println("  8. Other (specify)")
	fmt.Print("\nSelect (1-8) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "deepseek-r1"
	case "3":
		w.config.DefaultModel = "gemma3:27b"
	case "4":
		w.config.DefaultModel = "qwen3:30b"
	case "5":
		w.config.DefaultModel = "qwen2.5-coder:32b"
	case "6":
		w.config.DefaultModel = "llama3.2"
	case "7":
		w.config.DefaultModel = "phi4"
	case "8":
		fmt.Print("Enter model name: ")
		customModel, _ := w.reader.ReadString('\n')
		w.config.DefaultModel = strings.TrimSpace(customModel)
		if w.config.DefaultModel == "" {
			w.config.DefaultModel = "llama3.3"
		}
	default:
		w.config.DefaultModel = "llama3.3"
	}

	fmt.Println()
	fmt.Printf("Make sure to run: ollama pull %s\n", w.config.DefaultModel)

	return nil
}

func (w *Wizard) configureGoogle() error {
	fmt.Println()
	fmt.Println("Google AI Configuration")
	fmt.Println("Get your API key from: https://aistudio.google.com/apikey")
	fmt.Println()
	fmt.Print("Enter your Google API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Google API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. gemini-3-pro-preview (latest, frontier)")
	fmt.Println("  2. gemini-3-flash-preview (fast)")
	fmt.Println("  3. gemini-2.5-pro (workhorse)")
	fmt.Println("  4. gemini-2.5-flash (efficient)")
	fmt.Print("\nSelect (1-4) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "gemini-3-flash-preview"
	case "3":
		w.config.DefaultModel = "gemini-2.5-pro"
	case "4":
		w.config.DefaultModel = "gemini-2.5-flash"
	default:
		w.config.DefaultModel = "gemini-3-pro-preview"
	}

	return nil
}

func (w *Wizard) configureGroq() error {
	fmt.Println()
	fmt.Println("Groq Configuration")
	fmt.Println("Get your API key from: https://console.groq.com/keys")
	fmt.Println()
	fmt.Print("Enter your Groq API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Groq API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. llama-3.3-70b-versatile (default, best balance)")
	fmt.Println("  2. llama-3.1-8b-instant (fastest)")
	fmt.Println("  3. mixtral-8x7b-32768 (good for long context)")
	fmt.Println("  4. deepseek-r1-distill-llama-70b (reasoning)")
	fmt.Print("\nSelect (1-4) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "llama-3.1-8b-instant"
	case "3":
		w.config.DefaultModel = "mixtral-8x7b-32768"
	case "4":
		w.config.DefaultModel = "deepseek-r1-distill-llama-70b"
	default:
		w.config.DefaultModel = "llama-3.3-70b-versatile"
	}

	return nil
}

func (w *Wizard) configureDeepSeek() error {
	fmt.Println()
	fmt.Println("DeepSeek Configuration")
	fmt.Println("Get your API key from: https://platform.deepseek.com/api_keys")
	fmt.Println()
	fmt.Print("Enter your DeepSeek API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your DeepSeek API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. deepseek-chat (default, general purpose)")
	fmt.Println("  2. deepseek-reasoner (reasoning tasks)")
	fmt.Print("\nSelect (1-2) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "deepseek-reasoner"
	default:
		w.config.DefaultModel = "deepseek-chat"
	}

	return nil
}

func (w *Wizard) configureTogether() error {
	fmt.Println()
	fmt.Println("Together AI Configuration")
	fmt.Println("Get your API key from: https://api.together.xyz/settings/api-keys")
	fmt.Println()
	fmt.Print("Enter your Together API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Together API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. meta-llama/Llama-3.3-70B-Instruct-Turbo (default)")
	fmt.Println("  2. meta-llama/Llama-3.1-8B-Instruct-Turbo (fast)")
	fmt.Println("  3. mistralai/Mixtral-8x7B-Instruct-v0.1")
	fmt.Println("  4. Qwen/Qwen2.5-72B-Instruct-Turbo")
	fmt.Print("\nSelect (1-4) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "meta-llama/Llama-3.1-8B-Instruct-Turbo"
	case "3":
		w.config.DefaultModel = "mistralai/Mixtral-8x7B-Instruct-v0.1"
	case "4":
		w.config.DefaultModel = "Qwen/Qwen2.5-72B-Instruct-Turbo"
	default:
		w.config.DefaultModel = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	}

	return nil
}

func (w *Wizard) configureCerebras() error {
	fmt.Println()
	fmt.Println("Cerebras Configuration")
	fmt.Println("Get your API key from: https://cloud.cerebras.ai")
	fmt.Println()
	fmt.Print("Enter your Cerebras API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Cerebras API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. llama3.1-8b (default, fast)")
	fmt.Println("  2. llama3.1-70b (more capable)")
	fmt.Print("\nSelect (1-2) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "llama3.1-70b"
	default:
		w.config.DefaultModel = "llama3.1-8b"
	}

	return nil
}

func (w *Wizard) configureOpenRouter() error {
	fmt.Println()
	fmt.Println("OpenRouter Configuration")
	fmt.Println("Get your API key from: https://openrouter.ai/keys")
	fmt.Println()
	fmt.Print("Enter your OpenRouter API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your OpenRouter API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. anthropic/claude-3.5-sonnet (default)")
	fmt.Println("  2. openai/gpt-4o")
	fmt.Println("  3. google/gemini-2.0-flash-exp:free")
	fmt.Println("  4. meta-llama/llama-3.3-70b-instruct")
	fmt.Println("  5. deepseek/deepseek-chat")
	fmt.Println("  6. Other (specify model ID)")
	fmt.Print("\nSelect (1-6) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "openai/gpt-4o"
	case "3":
		w.config.DefaultModel = "google/gemini-2.0-flash-exp:free"
	case "4":
		w.config.DefaultModel = "meta-llama/llama-3.3-70b-instruct"
	case "5":
		w.config.DefaultModel = "deepseek/deepseek-chat"
	case "6":
		fmt.Print("Enter model ID (e.g., anthropic/claude-3-opus): ")
		customModel, _ := w.reader.ReadString('\n')
		w.config.DefaultModel = strings.TrimSpace(customModel)
		if w.config.DefaultModel == "" {
			w.config.DefaultModel = "anthropic/claude-3.5-sonnet"
		}
	default:
		w.config.DefaultModel = "anthropic/claude-3.5-sonnet"
	}

	return nil
}

func (w *Wizard) configureFireworks() error {
	fmt.Println()
	fmt.Println("Fireworks AI Configuration")
	fmt.Println("Get your API key from: https://fireworks.ai/api-keys")
	fmt.Println()
	fmt.Print("Enter your Fireworks API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Fireworks API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. accounts/fireworks/models/llama-v3-70b-instruct (default)")
	fmt.Println("  2. accounts/fireworks/models/qwen2p5-72b-instruct")
	fmt.Print("\nSelect (1-2) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "accounts/fireworks/models/qwen2p5-72b-instruct"
	default:
		w.config.DefaultModel = "accounts/fireworks/models/llama-v3-70b-instruct"
	}

	return nil
}

func (w *Wizard) configureZhipu() error {
	fmt.Println()
	fmt.Println("智谱 AI (Zhipu) Configuration")
	fmt.Println("Get your API key from: https://open.bigmodel.cn/api-keys")
	fmt.Println()
	fmt.Print("Enter your Zhipu API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Zhipu API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. glm-4-plus (default, most capable)")
	fmt.Println("  2. glm-4-air (faster, cheaper)")
	fmt.Println("  3. glm-4-flash (fastest)")
	fmt.Print("\nSelect (1-3) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "glm-4-air"
	case "3":
		w.config.DefaultModel = "glm-4-flash"
	default:
		w.config.DefaultModel = "glm-4-plus"
	}

	return nil
}

func (w *Wizard) configureSiliconFlow() error {
	fmt.Println()
	fmt.Println("SiliconFlow Configuration")
	fmt.Println("Get your API key from: https://cloud.siliconflow.cn/account/ak")
	fmt.Println()
	fmt.Print("Enter your SiliconFlow API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your SiliconFlow API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. Qwen/Qwen2.5-72B-Instruct (default)")
	fmt.Println("  2. deepseek-ai/DeepSeek-V3")
	fmt.Println("  3. meta-llama/Meta-Llama-3.1-70B-Instruct")
	fmt.Print("\nSelect (1-3) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "deepseek-ai/DeepSeek-V3"
	case "3":
		w.config.DefaultModel = "meta-llama/Meta-Llama-3.1-70B-Instruct"
	default:
		w.config.DefaultModel = "Qwen/Qwen2.5-72B-Instruct"
	}

	return nil
}

func (w *Wizard) configureNovita() error {
	fmt.Println()
	fmt.Println("Novita AI Configuration")
	fmt.Println("Get your API key from: https://novita.ai/settings/key-management")
	fmt.Println()
	fmt.Print("Enter your Novita API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Novita API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. meta-llama/llama-3.1-70b-instruct (default)")
	fmt.Println("  2. meta-llama/llama-3.1-8b-instruct")
	fmt.Print("\nSelect (1-2) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "meta-llama/llama-3.1-8b-instruct"
	default:
		w.config.DefaultModel = "meta-llama/llama-3.1-70b-instruct"
	}

	return nil
}

func (w *Wizard) configureLocalAI() error {
	fmt.Println()
	fmt.Println("LocalAI Configuration")
	fmt.Println("Make sure LocalAI is running locally")
	fmt.Println()
	fmt.Print("Enter LocalAI server URL [default: http://localhost:8080]: ")
	baseURL, _ := w.reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	w.config.APIKey = "localai"

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. llama3 (default)")
	fmt.Println("  2. mistral")
	fmt.Println("  3. Other (specify)")
	fmt.Print("\nSelect (1-3) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "mistral"
	case "3":
		fmt.Print("Enter model name: ")
		customModel, _ := w.reader.ReadString('\n')
		w.config.DefaultModel = strings.TrimSpace(customModel)
		if w.config.DefaultModel == "" {
			w.config.DefaultModel = "llama3"
		}
	default:
		w.config.DefaultModel = "llama3"
	}

	return nil
}

func (w *Wizard) configureVLLM() error {
	fmt.Println()
	fmt.Println("vLLM Configuration")
	fmt.Println("Make sure vLLM server is running locally")
	fmt.Println()
	fmt.Print("Enter vLLM server URL [default: http://localhost:8000]: ")
	baseURL, _ := w.reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	w.config.APIKey = "vllm"

	fmt.Println()
	fmt.Print("Enter model name [default: llama3]: ")
	modelName, _ := w.reader.ReadString('\n')
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		w.config.DefaultModel = "llama3"
	} else {
		w.config.TelegramToken = ""
	}

	// Web Search
	fmt.Println()
	fmt.Println("─" + strings.Repeat("─", 60))
	fmt.Println()
	fmt.Println("🌐 Web Search Integration")
	fmt.Println()
	fmt.Println("Enable web search to get real-time information from the internet.")
	fmt.Println("This allows Myrai to answer questions about:")
	fmt.Println("  • Current news and events")
	fmt.Println("  • Weather, stock prices, sports scores")
	fmt.Println("  • Recent developments beyond training data")
	fmt.Println()
	fmt.Println("Available providers:")
	fmt.Println("  • Brave Search (Recommended) - https://api.search.brave.com")
	fmt.Println("    Free tier: 2,000 queries/month")
	fmt.Println("  • Serper (Google) - https://serper.dev")
	fmt.Println("    Free tier: 2,500 queries")
	fmt.Println("  • DuckDuckGo - No API key needed (less reliable)")
	fmt.Println()
	fmt.Print("Enable web search? (y/n) [default: y]: ")
	enableSearch, _ := w.reader.ReadString('\n')
	enableSearch = strings.ToLower(strings.TrimSpace(enableSearch))

	if enableSearch != "n" && enableSearch != "no" {
		fmt.Println()
		fmt.Println("Select search provider:")
		fmt.Println("  1. Brave Search (recommended)")
		fmt.Println("  2. Serper (Google)")
		fmt.Println("  3. DuckDuckGo (no API key)")
		fmt.Print("Choice [1-3] [default: 1]: ")
		providerChoice, _ := w.reader.ReadString('\n')
		providerChoice = strings.TrimSpace(providerChoice)

		switch providerChoice {
		case "2":
			w.config.SearchProvider = "serper"
		case "3":
			w.config.SearchProvider = "duckduckgo"
		default:
			w.config.SearchProvider = "brave"
		}

		if w.config.SearchProvider != "duckduckgo" {
			fmt.Println()
			fmt.Printf("To use %s:\n", w.config.SearchProvider)
			if w.config.SearchProvider == "brave" {
				fmt.Println("1. Go to https://api.search.brave.com")
				fmt.Println("2. Sign up and get your API key")
			} else {
				fmt.Println("1. Go to https://serper.dev")
				fmt.Println("2. Sign up and get your API key")
			}
			fmt.Println()
			fmt.Print("Enter your API key (press Enter to skip): ")
			apiKey, _ := w.reader.ReadString('\n')
			w.config.SearchAPIKey = strings.TrimSpace(apiKey)
		} else {
			w.config.SearchAPIKey = ""
		}
	} else {
		w.config.SearchProvider = ""
		w.config.SearchAPIKey = ""
	}

	fmt.Println("\n✓ Integrations configured")
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (w *Wizard) configureMistral() error {
	fmt.Println()
	fmt.Println("Mistral AI Configuration")
	fmt.Println("Get your API key from: https://console.mistral.ai/api-keys")
	fmt.Println()
	fmt.Print("Enter your Mistral API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Mistral API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. mistral-large-latest (default, most capable)")
	fmt.Println("  2. mistral-small-latest (faster)")
	fmt.Println("  3. codestral-latest (code-focused)")
	fmt.Print("\nSelect (1-3) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "mistral-small-latest"
	case "3":
		w.config.DefaultModel = "codestral-latest"
	default:
		w.config.DefaultModel = "mistral-large-latest"
	}

	return nil
}

func (w *Wizard) configureXAI() error {
	fmt.Println()
	fmt.Println("xAI (Grok) Configuration")
	fmt.Println("Get your API key from: https://console.x.ai")
	fmt.Println()
	fmt.Print("Enter your xAI API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your xAI API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. grok-beta (default)")
	fmt.Print("\nSelect [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	w.config.DefaultModel = "grok-beta"

	return nil
}

func (w *Wizard) configurePerplexity() error {
	fmt.Println()
	fmt.Println("Perplexity AI Configuration")
	fmt.Println("Get your API key from: https://www.perplexity.ai/settings/api")
	fmt.Println()
	fmt.Print("Enter your Perplexity API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Perplexity API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Println("Select your preferred model:")
	fmt.Println("  1. llama-3.1-sonar-large-128k-online (default, with web search)")
	fmt.Println("  2. llama-3.1-sonar-small-128k-online (faster)")
	fmt.Print("\nSelect (1-2) [default: 1]: ")
	modelChoice, _ := w.reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "2":
		w.config.DefaultModel = "llama-3.1-sonar-small-128k-online"
	default:
		w.config.DefaultModel = "llama-3.1-sonar-large-128k-online"
	}

	return nil
}

func (w *Wizard) configureAzure() error {
	fmt.Println()
	fmt.Println("Azure OpenAI Configuration")
	fmt.Println("Get your credentials from: https://portal.azure.com")
	fmt.Println()
	fmt.Print("Enter your Azure OpenAI API Key: ")
	apiKey, _ := w.reader.ReadString('\n')
	w.config.APIKey = strings.TrimSpace(apiKey)

	for w.config.APIKey == "" {
		fmt.Println("❌ API key cannot be empty.")
		fmt.Print("Enter your Azure OpenAI API Key: ")
		apiKey, _ := w.reader.ReadString('\n')
		w.config.APIKey = strings.TrimSpace(apiKey)
	}

	fmt.Println()
	fmt.Print("Enter your Azure endpoint (e.g., https://your-resource.openai.azure.com): ")
	endpoint, _ := w.reader.ReadString('\n')
	endpoint = strings.TrimSpace(endpoint)

	fmt.Print("Enter your deployment name: ")
	deployment, _ := w.reader.ReadString('\n')
	deployment = strings.TrimSpace(deployment)

	w.config.DefaultModel = deployment
	w.config.BaseURL = strings.TrimSuffix(endpoint, "/") + "/openai/deployments/" + deployment

	return nil
}

func (w *Wizard) setupIntegrations() error {
	w.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Step 4: Optional Integrations                                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Telegram
	fmt.Println("Would you like to enable Telegram integration?")
	fmt.Println("This allows you to chat with Myrai via Telegram.")
	fmt.Print("Enable Telegram? (y/n) [default: n]: ")
	enableTelegram, _ := w.reader.ReadString('\n')
	enableTelegram = strings.ToLower(strings.TrimSpace(enableTelegram))

	if enableTelegram == "y" || enableTelegram == "yes" {
		w.config.EnableTelegram = true
		fmt.Println()
		fmt.Println("To set up Telegram:")
		fmt.Println("1. Message @BotFather on Telegram")
		fmt.Println("2. Create a new bot with /newbot")
		fmt.Println("3. Copy the bot token")
		fmt.Println()
		fmt.Print("Enter your Telegram Bot Token: ")
		token, _ := w.reader.ReadString('\n')
		w.config.TelegramToken = strings.TrimSpace(token)
	}

	// Vision Capabilities
	fmt.Println()
	fmt.Println("─" + strings.Repeat("─", 60))
	fmt.Println()
	fmt.Println("👁️  Vision & Image Analysis")
	fmt.Println()
	fmt.Println("Enable vision capabilities to:")
	fmt.Println("  • Capture and analyze photos from your camera")
	fmt.Println("  • Analyze images and screenshots")
	fmt.Println("  • Describe visual content")
	fmt.Println()
	fmt.Println("⚠️  Requires a vision-capable LLM:")
	fmt.Println("  • OpenAI: GPT-4o, GPT-4 Turbo")
	fmt.Println("  • Anthropic: Claude 3 Opus/Sonnet")
	fmt.Println("  • Google: Gemini Pro Vision")
	fmt.Println("  • OpenRouter: Any vision model")
	fmt.Println()
	fmt.Print("Enable vision capabilities? (y/n) [default: n]: ")
	enableVision, _ := w.reader.ReadString('\n')
	enableVision = strings.ToLower(strings.TrimSpace(enableVision))

	if enableVision == "y" || enableVision == "yes" {
		w.config.EnableVision = true
		fmt.Println()
		fmt.Println("Select vision model:")
		fmt.Println("  1. GPT-4o (OpenAI) - Best overall vision")
		fmt.Println("  2. Claude 3 Sonnet (Anthropic) - Great detail")
		fmt.Println("  3. Gemini Pro Vision (Google) - Good for charts")
		fmt.Println("  4. Use your current LLM (if vision-capable)")
		fmt.Print("Choice [1-4] [default: 1]: ")
		visionChoice, _ := w.reader.ReadString('\n')
		visionChoice = strings.TrimSpace(visionChoice)

		switch visionChoice {
		case "2":
			w.config.VisionModel = "claude-3-sonnet-20240229"
		case "3":
			w.config.VisionModel = "gemini-pro-vision"
		case "4":
			w.config.VisionModel = w.config.DefaultModel
		default:
			w.config.VisionModel = "gpt-4o"
		}
	}

	fmt.Println("\n✓ Integrations configured")
	time.Sleep(300 * time.Millisecond)

	return nil
}

func (w *Wizard) createConfiguration() error {
	// Create config directory
	if err := os.MkdirAll(w.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(w.configDir, "myrai.yaml")

	// Provider configurations
	providerConfigs := map[string]struct {
		baseURL string
		timeout int
	}{
		"openai":      {"https://api.openai.com/v1", 60},
		"anthropic":   {"https://api.anthropic.com/v1", 60},
		"google":      {"https://generativelanguage.googleapis.com/v1beta", 60},
		"kimi":        {"https://api.moonshot.cn/v1", 60},
		"deepseek":    {"https://api.deepseek.com/v1", 60},
		"groq":        {"https://api.groq.com/openai/v1", 60},
		"mistral":     {"https://api.mistral.ai/v1", 60},
		"together":    {"https://api.together.xyz/v1", 60},
		"cerebras":    {"https://api.cerebras.ai/v1", 60},
		"xai":         {"https://api.x.ai/v1", 60},
		"perplexity":  {"https://api.perplexity.ai/v1", 60},
		"fireworks":   {"https://api.fireworks.ai/inference/v1", 60},
		"novita":      {"https://api.novita.ai/v3/openai", 60},
		"siliconflow": {"https://api.siliconflow.cn/v1", 60},
		"zhipu":       {"https://open.bigmodel.cn/api/paas/v4", 60},
		"moonshot":    {"https://api.moonshot.cn/v1", 60},
		"openrouter":  {"https://openrouter.ai/api/v1", 60},
		"ollama":      {"http://localhost:11434/v1", 120},
		"localai":     {"http://localhost:8080/v1", 120},
		"vllm":        {"http://localhost:8000/v1", 120},
		"azure":       {"", 60},
	}

	cfg, ok := providerConfigs[w.config.LLMProvider]
	if !ok {
		cfg = providerConfigs["kimi"]
	}

	baseURL := cfg.baseURL
	if w.config.BaseURL != "" {
		baseURL = w.config.BaseURL
	}

	providerConfig := fmt.Sprintf(`    %s:
      api_key: ""  # Loaded from environment variable (see .env file)
      model: "%s"
      base_url: "%s"
      timeout: %d
      max_tokens: 4096`, w.config.LLMProvider, w.config.DefaultModel, baseURL, cfg.timeout)

	configContent := fmt.Sprintf(`# Myrai Configuration
# Generated on %s

server:
  address: 0.0.0.0
  port: 8080

llm:
  default_provider: %s
  providers:
%s

storage:
  data_dir: "%s"

channels:
  telegram:
    enabled: %v
    bot_token: ""  # Loaded from environment variable TELEGRAM_BOT_TOKEN (see .env file)
    allow_list: []

search:
  enabled: %v
  provider: "%s"
  api_key: ""  # Loaded from environment variable MYRAI_SEARCH_API_KEY (see .env file)
  max_results: 5
  timeout_seconds: 30

vision:
  enabled: %v
  vision_model: "%s"

tools:
  enabled:
    - read_file
    - write_file
    - list_dir
    - exec_command
    - web_search
  sandbox: true

security:
  allow_origins:
    - "*"
`, time.Now().Format("2006-01-02"), w.config.LLMProvider, providerConfig, w.workspace, w.config.EnableTelegram, w.config.SearchProvider != "", w.config.SearchProvider, w.config.EnableVision, w.config.VisionModel)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Create .env file in config directory
	envPath := filepath.Join(w.configDir, ".env")

	envKeyMap := map[string]string{
		"openai":      "OPENAI_API_KEY",
		"anthropic":   "ANTHROPIC_API_KEY",
		"google":      "GOOGLE_API_KEY",
		"kimi":        "KIMI_API_KEY",
		"deepseek":    "DEEPSEEK_API_KEY",
		"groq":        "GROQ_API_KEY",
		"mistral":     "MISTRAL_API_KEY",
		"together":    "TOGETHER_API_KEY",
		"cerebras":    "CEREBRAS_API_KEY",
		"xai":         "XAI_API_KEY",
		"perplexity":  "PERPLEXITY_API_KEY",
		"fireworks":   "FIREWORKS_API_KEY",
		"novita":      "NOVITA_API_KEY",
		"siliconflow": "SILICONFLOW_API_KEY",
		"zhipu":       "ZHIPU_API_KEY",
		"moonshot":    "MOONSHOT_API_KEY",
		"openrouter":  "OPENROUTER_API_KEY",
		"ollama":      "OLLAMA_API_KEY",
		"localai":     "LOCALAI_API_KEY",
		"vllm":        "VLLM_API_KEY",
		"azure":       "AZURE_OPENAI_API_KEY",
	}

	envKey, ok := envKeyMap[w.config.LLMProvider]
	if !ok {
		envKey = "KIMI_API_KEY"
	}

	envContent := fmt.Sprintf(`# Myrai Environment Variables
# Generated on %s

%s=%s
`, time.Now().Format("2006-01-02"), envKey, w.config.APIKey)

	if w.config.EnableTelegram && w.config.TelegramToken != "" {
		envContent += fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s\n", w.config.TelegramToken)
	}

	if w.config.SearchProvider != "" && w.config.SearchAPIKey != "" {
		envContent += fmt.Sprintf("MYRAI_SEARCH_API_KEY=%s\n", w.config.SearchAPIKey)
		if w.config.SearchProvider != "brave" {
			envContent += fmt.Sprintf("MYRAI_SEARCH_PROVIDER=%s\n", w.config.SearchProvider)
		}
	}

	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	return nil
}

func (w *Wizard) createPersonaFiles() error {
	// Ensure workspace directory exists
	if err := os.MkdirAll(w.workspace, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create persona manager
	pm, err := persona.NewPersonaManager(w.workspace, w.logger)
	if err != nil {
		return err
	}

	// Set identity from template
	identity := &persona.Identity{
		Name:        "Myrai",
		Personality: "Friendly, professional, and helpful AI assistant",
		Voice:       "Clear, approachable, and conversational",
		Values:      []string{"Privacy", "Transparency", "Efficiency"},
		Expertise:   []string{"Software development", "Writing", "Analysis"},
	}
	if err := pm.SetIdentity(identity); err != nil {
		return err
	}

	// Set user profile
	user := &persona.UserProfile{
		Name:               w.config.UserName,
		CommunicationStyle: w.config.CommunicationStyle,
		Expertise:          w.config.Expertise,
		Goals:              w.config.Goals,
		Preferences:        make(map[string]string),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Save user profile manually since it's not exposed directly
	userPath := filepath.Join(w.workspace, "USER.md")
	if err := os.WriteFile(userPath, []byte(user.String()), 0644); err != nil {
		return err
	}

	// Create TOOLS.md
	toolsPath := filepath.Join(w.workspace, "TOOLS.md")
	if err := os.WriteFile(toolsPath, []byte(DefaultToolsTemplate), 0644); err != nil {
		return err
	}

	// Create AGENTS.md
	agentsPath := filepath.Join(w.workspace, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte(DefaultAgentsTemplate), 0644); err != nil {
		return err
	}

	return nil
}

func (w *Wizard) showCompletion() {
	// Don't clear screen - just add some spacing
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	// Prepare template data
	data := struct {
		WorkspacePath string
		ConfigPath    string
		ConfigDir     string
	}{
		WorkspacePath: w.workspace,
		ConfigPath:    filepath.Join(w.configDir, "myrai.yaml"),
		ConfigDir:     w.configDir,
	}

	// Simple template replacement
	message := SetupCompleteMessage
	message = strings.ReplaceAll(message, "{{.WorkspacePath}}", data.WorkspacePath)
	message = strings.ReplaceAll(message, "{{.ConfigPath}}", data.ConfigPath)
	message = strings.ReplaceAll(message, "{{.ConfigDir}}", data.ConfigDir)

	fmt.Print(message)
	fmt.Println()
	fmt.Println()
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Println("✅ SETUP COMPLETE! Your AI assistant is ready to use.")
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Print("Press Enter to exit...")
	w.reader.ReadString('\n')
}

func (w *Wizard) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (w *Wizard) waitForEnter() {
	w.reader.ReadString('\n')
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// startSpinner starts an animated spinner
func (w *Wizard) startSpinner(message string) *chan bool {
	stop := make(chan bool)
	go func() {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-stop:
				fmt.Printf("\r%s Done!          \n", message)
				return
			default:
				fmt.Printf("\r%s %s", message, spinner[i%len(spinner)])
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
	return &stop
}

// stopSpinner stops the spinner
func (w *Wizard) stopSpinner(stopChan *chan bool) {
	if stopChan != nil {
		close(*stopChan)
		time.Sleep(150 * time.Millisecond) // Wait for spinner to finish printing
	}
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// CheckFirstRun checks if this is the first run (no config exists)
func CheckFirstRun() bool {
	configPath := GetConfigPath()

	_, err := os.Stat(configPath)
	return os.IsNotExist(err)
}

// GetWorkspacePath returns the default workspace (data) path
func GetWorkspacePath() string {
	// Try XDG_DATA_HOME first, then fall back to ~/.local/share/myrai
	workspace := os.Getenv("XDG_DATA_HOME")
	if workspace == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		workspace = filepath.Join(home, ".local", "share", "myrai")
	} else {
		workspace = filepath.Join(workspace, "myrai")
	}
	return workspace
}

// GetConfigPath returns the default config path
func GetConfigPath() string {
	// Try XDG_CONFIG_HOME first, then fall back to ~/.config/myrai
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		configDir = filepath.Join(home, ".config", "myrai")
	} else {
		configDir = filepath.Join(configDir, "myrai")
	}
	return filepath.Join(configDir, "myrai.yaml")
}

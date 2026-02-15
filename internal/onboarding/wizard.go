package onboarding

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/persona"
	"go.uber.org/zap"
)

// Wizard handles the interactive setup process
type Wizard struct {
	reader    *bufio.Reader
	logger    *zap.Logger
	workspace string
	config    *WizardConfig
}

// WizardConfig holds the configuration collected during setup
type WizardConfig struct {
	UserName           string
	CommunicationStyle string
	Expertise          []string
	Goals              []string
	APIKey             string
	DefaultModel       string
	EnableTelegram     bool
	TelegramToken      string
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
	if err := w.createConfiguration(); err != nil {
		return fmt.Errorf("configuration creation failed: %w", err)
	}

	// Step 6: Create persona files
	if err := w.createPersonaFiles(); err != nil {
		return fmt.Errorf("persona creation failed: %w", err)
	}

	// Show completion message
	w.showCompletion()

	return nil
}

func (w *Wizard) setupWorkspace() error {
	w.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Step 1: Workspace Setup                                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get default workspace path
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	defaultWorkspace := filepath.Join(home, ".goclawde")

	fmt.Printf("Where should GoClawde store its data? [default: %s]: ", defaultWorkspace)
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
	fmt.Println("How would you prefer GoClawde to communicate?")
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
	fmt.Println("What are your main goals for using GoClawde? (comma-separated)")
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

	// API Key
	fmt.Println("GoClawde uses the Kimi API by default.")
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

	fmt.Println("\n✓ AI configured")
	time.Sleep(500 * time.Millisecond)

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
	fmt.Println("This allows you to chat with GoClawde via Telegram.")
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

	fmt.Println("\n✓ Integrations configured")
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (w *Wizard) createConfiguration() error {
	// Create config.yaml
	configPath := filepath.Join(w.workspace, "goclawde.yaml")

	configContent := fmt.Sprintf(`# GoClawde Configuration
# Generated on %s

server:
  address: 0.0.0.0
  port: 8080

llm:
  default_provider: kimi
  providers:
    kimi:
      api_key: "%s"
      model: "%s"
      base_url: "https://api.moonshot.cn/v1"
      timeout: 60
      max_tokens: 4096

storage:
  data_dir: "%s"

channels:
  telegram:
    enabled: %v
    bot_token: "%s"
    allow_list: []

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
`, time.Now().Format("2006-01-02"), w.config.APIKey, w.config.DefaultModel, w.workspace, w.config.EnableTelegram, w.config.TelegramToken)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Create .env file
	envPath := filepath.Join(w.workspace, ".env")
	envContent := fmt.Sprintf(`# GoClawde Environment Variables
# Generated on %s

GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY=%s
`, time.Now().Format("2006-01-02"), w.config.APIKey)

	if w.config.EnableTelegram && w.config.TelegramToken != "" {
		envContent += fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s\n", w.config.TelegramToken)
	}

	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	return nil
}

func (w *Wizard) createPersonaFiles() error {
	// Create persona manager
	pm, err := persona.NewPersonaManager(w.workspace, w.logger)
	if err != nil {
		return err
	}

	// Set identity from template
	identity := &persona.Identity{
		Name:        "GoClawde",
		Personality: "Friendly, professional, and helpful AI assistant",
		Voice:       "Clear, approachable, and conversational",
		Values:      []string{"Privacy", "Transparency", "Efficiency"},
		Expertise:   []string{"Software development", "Writing", "Analysis"},
	}
	pm.SetIdentity(identity)

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

	// Save IDENTITY.md
	if err := pm.Save(); err != nil {
		return err
	}

	return nil
}

func (w *Wizard) showCompletion() {
	w.clearScreen()

	// Prepare template data
	data := struct {
		WorkspacePath string
		ConfigPath    string
	}{
		WorkspacePath: w.workspace,
		ConfigPath:    filepath.Join(w.workspace, "goclawde.yaml"),
	}

	// Simple template replacement
	message := SetupCompleteMessage
	message = strings.ReplaceAll(message, "{{.WorkspacePath}}", data.WorkspacePath)
	message = strings.ReplaceAll(message, "{{.ConfigPath}}", data.ConfigPath)

	fmt.Print(message)
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
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	workspace := filepath.Join(home, ".goclawde")
	configPath := filepath.Join(workspace, "goclawde.yaml")

	_, err = os.Stat(configPath)
	return os.IsNotExist(err)
}

// GetWorkspacePath returns the default workspace path
func GetWorkspacePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".goclawde")
}

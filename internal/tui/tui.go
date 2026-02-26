// Package tui provides a beautiful Bubble Tea interface for Myrai
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/gmsas95/myrai-cli/internal/agent"
)

// Message represents a chat message
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// Model represents the TUI state
type Model struct {
	agent          *agent.Agent
	viewport       viewport.Model
	textInput      textinput.Model
	spinner        spinner.Model
	messages       []Message
	isLoading      bool
	width          int
	height         int
	renderer       *glamour.TermRenderer
	showHelp       bool
	conversationID string
}

// Styles
type Styles struct {
	Header       lipgloss.Style
	UserStyle    lipgloss.Style
	AIStyle      lipgloss.Style
	SystemStyle  lipgloss.Style
	HelpStyle    lipgloss.Style
	InputStyle   lipgloss.Style
	SpinnerStyle lipgloss.Style
}

func NewStyles() Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00BFFF")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1).
			Width(100),

		UserStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF7F")),

		AIStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00BFFF")),

		SystemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true),

		HelpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true),

		InputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00BFFF")).
			Padding(0, 1),

		SpinnerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF")),
	}
}

var styles = NewStyles()

// NewModel creates a new TUI model
func NewModel(agentInstance *agent.Agent) (*Model, error) {
	// Initialize glamour renderer
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 80

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.SpinnerStyle

	// Initialize viewport
	vp := viewport.New(100, 20)
	vp.SetContent("")

	return &Model{
		agent:     agentInstance,
		viewport:  vp,
		textInput: ti,
		spinner:   sp,
		messages:  []Message{},
		renderer:  renderer,
		showHelp:  true,
	}, nil
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 8
		m.textInput.Width = msg.Width - 10
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			input := m.textInput.Value()
			if input == "" {
				return m, nil
			}

			// Handle slash commands
			if strings.HasPrefix(input, "/") {
				return m.handleSlashCommand(input)
			}

			// Add user message
			m.messages = append(m.messages, Message{
				Role:      "user",
				Content:   input,
				Timestamp: time.Now(),
			})

			// Clear input
			m.textInput.SetValue("")

			// Start loading
			m.isLoading = true

			// Update viewport
			m.updateViewport()

			// Send to agent
			return m, m.sendToAgent(input)

		case tea.KeyPgUp:
			m.viewport.LineUp(5)
			return m, nil

		case tea.KeyPgDown:
			m.viewport.LineDown(5)
			return m, nil
		}

	case responseMsg:
		m.isLoading = false
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   msg.content,
			Timestamp: time.Now(),
		})
		m.conversationID = msg.conversationID
		m.updateViewport()
		return m, nil

	case errMsg:
		m.isLoading = false
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   fmt.Sprintf("Error: %v", msg.err),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	// Update components
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	if m.isLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Header
	header := styles.Header.Render("  🤖 Myrai (未来) - Your Personal AI Assistant  ")
	sb.WriteString(header)
	sb.WriteString("\n\n")

	// Messages viewport
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Loading indicator
	if m.isLoading {
		sb.WriteString(fmt.Sprintf("\n  %s Thinking...\n", m.spinner.View()))
	}

	// Input area
	sb.WriteString("\n")
	inputBox := styles.InputStyle.Render(m.textInput.View())
	sb.WriteString(inputBox)

	// Help text
	if m.showHelp {
		sb.WriteString("\n")
		helpText := styles.HelpStyle.Render("  Enter: Send | /skills: List skills | Ctrl+C: Exit | PgUp/PgDn: Scroll")
		sb.WriteString(helpText)
	}

	return sb.String()
}

// updateViewport updates the viewport content
func (m *Model) updateViewport() {
	var sb strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(styles.UserStyle.Render("👤 You:"))
			sb.WriteString(" ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")

		case "assistant":
			sb.WriteString(styles.AIStyle.Render("🤖 Myrai:"))
			sb.WriteString(" ")
			// Render markdown for AI responses
			rendered, err := m.renderer.Render(msg.Content)
			if err != nil {
				sb.WriteString(msg.Content)
			} else {
				sb.WriteString(rendered)
			}
			sb.WriteString("\n\n")

		case "system":
			sb.WriteString(styles.SystemStyle.Render(msg.Content))
			sb.WriteString("\n\n")
		}
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

// handleSlashCommand handles slash commands
func (m Model) handleSlashCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return m, nil
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/skills":
		return m.handleSkillsCommand()
	case "/help":
		return m.handleHelpCommand()
	case "/new":
		m.conversationID = ""
		m.messages = []Message{}
		m.updateViewport()
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "🆕 New conversation started!",
			Timestamp: time.Now(),
		})
		m.updateViewport()
	case "/clear":
		m.messages = []Message{}
		m.updateViewport()
	default:
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   fmt.Sprintf("❓ Unknown command: %s. Type /help for available commands.", command),
			Timestamp: time.Now(),
		})
		m.updateViewport()
	}

	m.textInput.SetValue("")
	return m, nil
}

// handleSkillsCommand shows all skills
func (m Model) handleSkillsCommand() (tea.Model, tea.Cmd) {
	if m.agent == nil {
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "❌ Agent not initialized",
			Timestamp: time.Now(),
		})
		m.updateViewport()
		m.textInput.SetValue("")
		return m, nil
	}

	registry := m.agent.GetSkillsRegistry()
	if registry == nil {
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "❌ Skills registry not available",
			Timestamp: time.Now(),
		})
		m.updateViewport()
		m.textInput.SetValue("")
		return m, nil
	}

	skills := registry.ListSkills()
	if len(skills) == 0 {
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "📭 No skills registered",
			Timestamp: time.Now(),
		})
		m.updateViewport()
		m.textInput.SetValue("")
		return m, nil
	}

	var sb strings.Builder
	sb.WriteString("## 🛠️ Available Skills\n\n")

	for _, skill := range skills {
		status := "✅"
		if !skill.IsEnabled() {
			status = "❌"
		}
		sb.WriteString(fmt.Sprintf("%s **%s**\n", status, skill.Name()))
		sb.WriteString(fmt.Sprintf("   %s\n", skill.Description()))

		tools := skill.Tools()
		if len(tools) > 0 {
			sb.WriteString(fmt.Sprintf("   *Tools: %d*\n", len(tools)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("**Total: %d skills**", len(skills)))

	m.messages = append(m.messages, Message{
		Role:      "assistant",
		Content:   sb.String(),
		Timestamp: time.Now(),
	})
	m.updateViewport()
	m.textInput.SetValue("")
	return m, nil
}

// handleHelpCommand shows help
func (m Model) handleHelpCommand() (tea.Model, tea.Cmd) {
	help := `## 🆘 Help

### Slash Commands:
- **/skills** - List all available skills
- **/new** - Start a new conversation
- **/clear** - Clear the chat history
- **/help** - Show this help

### Keyboard Shortcuts:
- **Enter** - Send message
- **PgUp/PgDn** - Scroll through chat
- **Ctrl+C** or **Esc** - Exit

### Tips:
- Just type naturally to chat with Myrai
- Use skills by asking naturally (e.g., "Search GitHub for...")
- The AI can execute commands, read files, and more!`

	m.messages = append(m.messages, Message{
		Role:      "assistant",
		Content:   help,
		Timestamp: time.Now(),
	})
	m.updateViewport()
	m.textInput.SetValue("")
	return m, nil
}

// sendToAgent sends message to agent and returns response
func (m Model) sendToAgent(message string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.agent.Chat(ctx, agent.ChatRequest{
			Message:        message,
			ConversationID: m.conversationID,
			Stream:         false,
		})
		if err != nil {
			return errMsg{err: err}
		}
		return responseMsg{
			content:        resp.Content,
			conversationID: resp.ConversationID,
		}
	}
}

// Messages for tea.Cmd
type responseMsg struct {
	content        string
	conversationID string
}

type errMsg struct {
	err error
}

// Run starts the TUI
func Run(agentInstance *agent.Agent) error {
	model, err := NewModel(agentInstance)
	if err != nil {
		return fmt.Errorf("failed to create TUI model: %w", err)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

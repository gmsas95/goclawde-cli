// Package discord provides Discord bot integration
package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/agent"
	"github.com/gmsas95/goclawde-cli/internal/store"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Config holds Discord bot configuration
type Config struct {
	Token     string
	Enabled   bool
	GuildID   string          // Optional: restrict to specific server
	Channels  []string        // Optional: whitelist channels
	AllowDM   bool            // Allow direct messages
}

// Bot represents a Discord bot instance
type Bot struct {
	session *discordgo.Session
	agent   *agent.Agent
	store   *store.Store
	config  Config
	logger  *zap.Logger
}

// NewBot creates a new Discord bot
func NewBot(cfg Config, agentInstance *agent.Agent, st *store.Store, logger *zap.Logger) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("discord token is required")
	}
	
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}
	
	bot := &Bot{
		session: session,
		agent:   agentInstance,
		store:   st,
		config:  cfg,
		logger:  logger,
	}
	
	// Register handlers
	session.AddHandler(bot.messageCreate)
	session.AddHandler(bot.ready)
	
	// Set intents
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	
	return bot, nil
}

// Start starts the Discord bot
func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open discord connection: %w", err)
	}
	
	b.logger.Info("Discord bot started",
		zap.String("username", b.session.State.User.Username),
	)
	
	return nil
}

// Stop stops the Discord bot
func (b *Bot) Stop() error {
	return b.session.Close()
}

// ready is called when the bot is ready
func (b *Bot) ready(s *discordgo.Session, event *discordgo.Ready) {
	b.logger.Info("Discord bot ready",
		zap.String("username", s.State.User.Username),
		zap.Int("guilds", len(event.Guilds)),
	)
}

// messageCreate handles incoming messages
func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}
	
	// Check if DM is allowed
	if m.GuildID == "" && !b.config.AllowDM {
		return
	}
	
	// Check guild restriction
	if b.config.GuildID != "" && m.GuildID != b.config.GuildID {
		return
	}
	
	// Check channel whitelist
	if len(b.config.Channels) > 0 {
		allowed := false
		for _, ch := range b.config.Channels {
			if m.ChannelID == ch {
				allowed = true
				break
			}
		}
		if !allowed {
			return
		}
	}
	
	// Check if bot is mentioned or DM
	isDM := m.GuildID == ""
	isMentioned := false
	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			isMentioned = true
			break
		}
	}
	
	// In guilds, only respond to mentions
	if !isDM && !isMentioned {
		return
	}
	
	// Clean message content
	content := m.Content
	if isMentioned {
		// Remove bot mention
		content = strings.ReplaceAll(content, "<@"+s.State.User.ID+">", "")
		content = strings.ReplaceAll(content, "<@!"+s.State.User.ID+">", "")
		content = strings.TrimSpace(content)
	}
	
	if content == "" {
		return
	}
	
	// Handle commands
	if strings.HasPrefix(content, "/") {
		b.handleCommand(s, m, content)
		return
	}
	
	// Process with agent
	ctx := context.Background()
	
	// Show typing indicator
	s.ChannelTyping(m.ChannelID)
	
	resp, err := b.agent.Chat(ctx, agent.ChatRequest{
		Message: content,
		Stream:  false,
	})
	
	if err != nil {
		b.logger.Error("Agent error", zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "❌ Error: "+err.Error())
		return
	}
	
	// Send response (split if too long)
	if len(resp.Content) > 2000 {
		// Discord has 2000 char limit
		parts := splitMessage(resp.Content, 2000)
		for _, part := range parts {
			s.ChannelMessageSend(m.ChannelID, part)
			time.Sleep(100 * time.Millisecond) // Rate limit
		}
	} else {
		s.ChannelMessageSend(m.ChannelID, resp.Content)
	}
}

// handleCommand handles bot commands
func (b *Bot) handleCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}
	
	command := parts[0]
	
	switch command {
	case "/help":
		help := `**GoClawde Discord Bot**

Commands:
• "/help" - Show this help
• "/new" - Start new conversation
• "/status" - Check bot status
• "/ping" - Test latency

Or just mention me (@GoClawde) and ask anything!`
		s.ChannelMessageSend(m.ChannelID, help)
		
	case "/new":
		s.ChannelMessageSend(m.ChannelID, "🆕 New conversation started!")
		
	case "/status":
		status := fmt.Sprintf("🟢 Online | Latency: %dms", s.HeartbeatLatency().Milliseconds())
		s.ChannelMessageSend(m.ChannelID, status)
		
	case "/ping":
		start := time.Now()
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		latency := time.Since(start).Milliseconds()
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Latency: %dms", latency))
		
	default:
		// Unknown command, treat as normal message
		// Re-process without the command prefix
		content := strings.TrimPrefix(cmd, command)
		content = strings.TrimSpace(content)
		if content != "" {
			m.Content = content
			b.messageCreate(s, &discordgo.MessageCreate{Message: m.Message})
		}
	}
}

// splitMessage splits a message into chunks under max length
func splitMessage(text string, maxLen int) []string {
	var parts []string
	lines := strings.Split(text, "\n")
	var current strings.Builder
	
	for _, line := range lines {
		if current.Len()+len(line)+1 > maxLen {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}
	
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	
	return parts
}

package telegram

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gmsas95/myrai-cli/internal/agent"
	"github.com/gmsas95/myrai-cli/internal/store"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Bot represents a Telegram bot integration
type Bot struct {
	api       *tgbotapi.BotAPI
	agent     *agent.Agent
	store     *store.Store
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	enabled   bool
	allowList map[int64]bool // Allowed user IDs
	// Track conversations per chat
	conversations map[int64]string // chatID -> conversationID
	convMu        sync.RWMutex
}

// Config holds Telegram bot configuration
type Config struct {
	Token      string
	Enabled    bool
	AllowList  []int64 // List of allowed user IDs (empty = allow all)
	WebhookURL string  // Optional webhook URL (empty = use polling)
}

// NewBot creates a new Telegram bot
func NewBot(cfg Config, agent *agent.Agent, store *store.Store, logger *zap.Logger) (*Bot, error) {
	if !cfg.Enabled || cfg.Token == "" {
		return &Bot{enabled: false}, nil
	}

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	api.Debug = false
	log.Printf("Authorized on account %s", api.Self.UserName)

	ctx, cancel := context.WithCancel(context.Background())

	allowList := make(map[int64]bool)
	for _, id := range cfg.AllowList {
		allowList[id] = true
	}

	return &Bot{
		api:           api,
		agent:         agent,
		store:         store,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		enabled:       true,
		allowList:     allowList,
		conversations: make(map[int64]string),
	}, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	if !b.enabled {
		return nil
	}

	b.wg.Add(1)
	go b.run()

	return nil
}

// Stop stops the bot
func (b *Bot) Stop() {
	if !b.enabled {
		return
	}

	b.cancel()
	b.wg.Wait()
}

func (b *Bot) run() {
	defer b.wg.Done()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-b.ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if err := b.handleUpdate(update); err != nil {
				b.logger.Error("Failed to handle update", zap.Error(err))
			}
		}
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) error {
	// Handle messages
	if update.Message == nil {
		return nil
	}

	msg := update.Message
	userID := msg.From.ID

	// Check allowlist
	if len(b.allowList) > 0 && !b.allowList[userID] {
		b.sendMessage(msg.Chat.ID, "⛔ You are not authorized to use this bot.")
		return nil
	}

	// Handle commands
	if msg.IsCommand() {
		return b.handleCommand(msg)
	}

	// Handle text messages
	if msg.Text != "" {
		return b.handleMessage(msg)
	}

	// Handle voice messages
	if msg.Voice != nil {
		return b.handleVoiceMessage(msg)
	}

	// Handle photos
	if msg.Photo != nil && len(msg.Photo) > 0 {
		return b.handlePhoto(msg)
	}

	// Handle documents (PDFs, etc.)
	if msg.Document != nil {
		return b.handleDocument(msg)
	}

	return nil
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) error {
	chatID := msg.Chat.ID

	switch msg.Command() {
	case "start":
		_, err := b.sendMessage(chatID, `🤖 *Myrai Bot*

Welcome! I'm your personal AI assistant. I can help you with:

• Answering questions
• File operations
• System commands
• GitHub integration
• Note-taking
• Weather info

Just send me a message!`)
		return err

	case "help":
		_, err := b.sendMessage(chatID, `*Available Commands:*

/start - Start the bot
/help - Show this help
/new - Start new conversation
/status - Show bot status

*Features:*
Just chat naturally or ask me to:
• Read/write files
• Execute commands
• Search GitHub
• Take notes
• Check weather`)
		return err

	case "new":
		// Clear conversation context for this chat
		b.convMu.Lock()
		delete(b.conversations, chatID)
		b.convMu.Unlock()
		_, err := b.sendMessage(chatID, "🆕 Starting new conversation! Context cleared.")
		return err

	case "status":
		_, err := b.sendMessage(chatID, "✅ Bot is running and ready!")
		return err

	default:
		_, err := b.sendMessage(chatID, "❓ Unknown command. Use /help for available commands.")
		return err
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) error {
	chatID := msg.Chat.ID
	text := msg.Text

	// Get or create conversation for this chat
	b.convMu.RLock()
	convID := b.conversations[chatID]
	b.convMu.RUnlock()

	// Show typing indicator
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(typing)

	// Process through agent
	ctx, cancel := context.WithTimeout(b.ctx, 60*time.Second)
	defer cancel()

	var responseText strings.Builder

	resp, err := b.agent.Chat(ctx, agent.ChatRequest{
		ConversationID: convID,
		Message:        text,
		Stream:         false, // Non-streaming for Telegram
	})

	if err != nil {
		b.logger.Error("Agent error", zap.Error(err))
		_, sendErr := b.sendMessage(chatID, fmt.Sprintf("❌ Error: %v", err))
		return sendErr
	}

	// Save conversation ID for future messages in this chat
	if resp.ConversationID != "" {
		b.convMu.Lock()
		b.conversations[chatID] = resp.ConversationID
		b.convMu.Unlock()
	}

	responseText.WriteString(resp.Content)

	// Format response for Telegram (respecting message limits)
	response := responseText.String()
	if len(response) > 4096 {
		response = response[:4093] + "..."
	}

	// Send response
	_, err = b.sendMessage(chatID, response)
	return err
}

func (b *Bot) handleVoiceMessage(msg *tgbotapi.Message) error {
	chatID := msg.Chat.ID

	// Show typing indicator
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(typing)

	// For now, just acknowledge voice message
	// Full voice processing will be implemented in next iteration
	_, err := b.sendMessage(chatID, "🎙️ Voice message received! Voice processing coming soon to Myrai (未来).")
	return err
}

// handlePhoto handles photo messages
func (b *Bot) handlePhoto(msg *tgbotapi.Message) error {
	chatID := msg.Chat.ID

	// Show typing indicator
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(typing)

	// Get the largest photo (best quality)
	photos := msg.Photo
	if len(photos) == 0 {
		_, err := b.sendMessage(chatID, "❌ No photo found in message.")
		return err
	}
	photo := photos[len(photos)-1]

	// Download the photo
	b.sendMessage(chatID, "📸 Downloading and analyzing image...")

	filePath, err := b.downloadFile(photo.FileID, "image")
	if err != nil {
		b.logger.Error("Failed to download photo", zap.Error(err))
		_, sendErr := b.sendMessage(chatID, fmt.Sprintf("❌ Failed to download image: %v", err))
		return sendErr
	}
	defer os.Remove(filePath) // Clean up after processing

	// Process through agent with the image
	ctx, cancel := context.WithTimeout(b.ctx, 60*time.Second)
	defer cancel()

	prompt := "Analyze this image and describe what you see."
	if msg.Caption != "" {
		prompt = msg.Caption
	}

	resp, err := b.agent.Chat(ctx, agent.ChatRequest{
		ConversationID: b.getConversationID(chatID),
		Message:        fmt.Sprintf("[Image attached: %s]\n\n%s", filePath, prompt),
		Stream:         false,
	})

	if err != nil {
		b.logger.Error("Agent error", zap.Error(err))
		_, sendErr := b.sendMessage(chatID, fmt.Sprintf("❌ Error analyzing image: %v", err))
		return sendErr
	}

	// Save conversation ID
	b.setConversationID(chatID, resp.ConversationID)

	_, err = b.sendMessage(chatID, resp.Content)
	return err
}

// handleDocument handles document/file messages (PDFs, etc.)
func (b *Bot) handleDocument(msg *tgbotapi.Message) error {
	chatID := msg.Chat.ID
	doc := msg.Document

	// Show typing indicator
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(typing)

	// Check file size (limit to 20MB)
	if doc.FileSize > 20*1024*1024 {
		_, err := b.sendMessage(chatID, "❌ File too large. Maximum size is 20MB.")
		return err
	}

	// Download the document
	b.sendMessage(chatID, fmt.Sprintf("📄 Downloading file: %s...", doc.FileName))

	filePath, err := b.downloadFile(doc.FileID, "document")
	if err != nil {
		b.logger.Error("Failed to download document", zap.Error(err))
		_, sendErr := b.sendMessage(chatID, fmt.Sprintf("❌ Failed to download file: %v", err))
		return sendErr
	}
	defer os.Remove(filePath) // Clean up after processing

	// Process through agent with the document
	ctx, cancel := context.WithTimeout(b.ctx, 120*time.Second)
	defer cancel()

	prompt := fmt.Sprintf("Please analyze this document: %s", filePath)
	if msg.Caption != "" {
		prompt = fmt.Sprintf("%s\n\nUser request: %s", prompt, msg.Caption)
	}

	resp, err := b.agent.Chat(ctx, agent.ChatRequest{
		ConversationID: b.getConversationID(chatID),
		Message:        prompt,
		Stream:         false,
	})

	if err != nil {
		b.logger.Error("Agent error", zap.Error(err))
		_, sendErr := b.sendMessage(chatID, fmt.Sprintf("❌ Error processing document: %v", err))
		return sendErr
	}

	// Save conversation ID
	b.setConversationID(chatID, resp.ConversationID)

	_, err = b.sendMessage(chatID, resp.Content)
	return err
}

// downloadFile downloads a file from Telegram and returns the local path
func (b *Bot) downloadFile(fileID string, fileType string) (string, error) {
	// Get file info from Telegram
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := b.api.GetFile(fileConfig)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Create temp directory if it doesn't exist
	tempDir := filepath.Join(os.TempDir(), "myrai-telegram")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Generate local filename
	ext := "bin"
	if fileType == "image" {
		ext = "jpg"
	} else if fileType == "document" {
		ext = "pdf"
	}
	localPath := filepath.Join(tempDir, fmt.Sprintf("%s-%d.%s", fileID, time.Now().Unix(), ext))

	// Download file from Telegram
	fileURL := file.Link(b.api.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Save to local file
	out, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return localPath, nil
}

// getConversationID returns the conversation ID for a chat
func (b *Bot) getConversationID(chatID int64) string {
	b.convMu.RLock()
	defer b.convMu.RUnlock()
	return b.conversations[chatID]
}

// setConversationID sets the conversation ID for a chat
func (b *Bot) setConversationID(chatID int64, convID string) {
	if convID != "" {
		b.convMu.Lock()
		b.conversations[chatID] = convID
		b.convMu.Unlock()
	}
}

func (b *Bot) sendMessage(chatID int64, text string) (int, error) {
	// Escape special characters for Markdown
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	sent, err := b.api.Send(msg)
	if err != nil {
		// Try without markdown if it fails
		msg.ParseMode = ""
		sent, err = b.api.Send(msg)
		if err != nil {
			return 0, err
		}
	}

	return sent.MessageID, nil
}

// GetBotInfo returns bot information
func (b *Bot) GetBotInfo() map[string]interface{} {
	if !b.enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	return map[string]interface{}{
		"enabled":   true,
		"username":  b.api.Self.UserName,
		"firstName": b.api.Self.FirstName,
	}
}

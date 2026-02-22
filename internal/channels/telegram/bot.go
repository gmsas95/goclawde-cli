package telegram

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
/history - Show conversation history
/resume <number> - Resume a previous conversation
/documents - Show all uploaded documents
/status - Show bot status

*Features:*
Just chat naturally or ask me to:
• Read/write files
• Execute commands
• Search GitHub
• Take notes
• Check weather
• Analyze documents and images`)
		return err

	case "new":
		// Clear conversation context for this chat
		b.clearConversationID(chatID)
		_, err := b.sendMessage(chatID, "🆕 Starting new conversation! Context cleared.")
		return err

	case "history":
		return b.handleHistoryCommand(chatID)

	case "resume":
		return b.handleResumeCommand(msg)

	case "status":
		_, err := b.sendMessage(chatID, "✅ Bot is running and ready!")
		return err

	case "restart":
		// Clear all conversations
		b.convMu.Lock()
		b.conversations = make(map[int64]string)
		b.convMu.Unlock()

		_, err := b.sendMessage(chatID, "🔄 Restarting...\n\nConversations cleared. Bot will restart shortly.")
		if err != nil {
			return err
		}

		// Signal restart by exiting - relies on external process manager to restart
		b.logger.Info("Restart requested via Telegram")
		b.cancel() // Cancel context to trigger shutdown
		return nil

	case "documents":
		return b.handleDocumentsCommand(chatID)

	default:
		_, err := b.sendMessage(chatID, "❓ Unknown command. Use /help for available commands.")
		return err
	}
}

// handleHistoryCommand shows conversation history for the chat
func (b *Bot) handleHistoryCommand(chatID int64) error {
	if b.store == nil {
		_, err := b.sendMessage(chatID, "❌ History not available - database not connected.")
		return err
	}

	mappings, err := b.store.GetChatConversationHistory(chatID, "telegram", 10)
	if err != nil {
		b.logger.Error("Failed to get conversation history", zap.Error(err))
		_, err := b.sendMessage(chatID, "❌ Failed to retrieve conversation history.")
		return err
	}

	if len(mappings) == 0 {
		_, err := b.sendMessage(chatID, "📭 No conversation history found.\n\nStart chatting to create a conversation!")
		return err
	}

	var sb strings.Builder
	sb.WriteString("📜 *Conversation History*\n\n")

	for i, mapping := range mappings {
		// Get conversation details
		conv, err := b.store.GetConversation(mapping.ConversationID)
		if err != nil {
			continue
		}

		status := ""
		if mapping.IsActive {
			status = " ✅ *Active*"
		}

		sb.WriteString(fmt.Sprintf("%d. *%s*%s\n", i+1, conv.Title, status))
		sb.WriteString(fmt.Sprintf("   🕐 %s\n", conv.UpdatedAt.Format("Jan 2, 3:04 PM")))
		sb.WriteString(fmt.Sprintf("   💬 %d messages\n\n", conv.MessageCount))
	}

	sb.WriteString("Use `/resume <number>` to continue a conversation.")

	_, err = b.sendMessage(chatID, sb.String())
	return err
}

// handleResumeCommand resumes a previous conversation
func (b *Bot) handleResumeCommand(msg *tgbotapi.Message) error {
	chatID := msg.Chat.ID

	// Parse the command argument
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 1 {
		_, err := b.sendMessage(chatID, "❌ Please specify a conversation number.\n\nExample: `/resume 1`\n\nUse `/history` to see available conversations.")
		return err
	}

	// Parse the number
	num, err := strconv.Atoi(args[0])
	if err != nil || num < 1 {
		_, err := b.sendMessage(chatID, "❌ Invalid conversation number. Please use a number from 1-10.")
		return err
	}

	if b.store == nil {
		_, err := b.sendMessage(chatID, "❌ Resume not available - database not connected.")
		return err
	}

	// Get conversation history
	mappings, err := b.store.GetChatConversationHistory(chatID, "telegram", 10)
	if err != nil {
		b.logger.Error("Failed to get conversation history", zap.Error(err))
		_, err := b.sendMessage(chatID, "❌ Failed to retrieve conversation history.")
		return err
	}

	if num > len(mappings) {
		_, err := b.sendMessage(chatID, fmt.Sprintf("❌ Conversation %d not found. Only %d conversations available.\n\nUse `/history` to see available conversations.", num, len(mappings)))
		return err
	}

	// Get the selected conversation
	selectedMapping := mappings[num-1]

	// Set this as the active conversation
	b.setConversationID(chatID, selectedMapping.ConversationID)

	// Get conversation details
	conv, err := b.store.GetConversation(selectedMapping.ConversationID)
	if err != nil {
		b.logger.Error("Failed to get conversation", zap.Error(err))
		_, err := b.sendMessage(chatID, "❌ Failed to resume conversation.")
		return err
	}

	_, err = b.sendMessage(chatID, fmt.Sprintf("✅ Resumed conversation: *%s*\n\n📊 %d messages\n🕐 Last updated: %s\n\nYou can now continue chatting!",
		conv.Title,
		conv.MessageCount,
		conv.UpdatedAt.Format("Jan 2, 3:04 PM")))
	return err
}

// handleDocumentsCommand shows all uploaded documents
func (b *Bot) handleDocumentsCommand(chatID int64) error {
	if b.store == nil {
		_, err := b.sendMessage(chatID, "❌ Document storage not available - database not connected.")
		return err
	}

	files, err := b.store.ListAllFiles(20, 0)
	if err != nil {
		b.logger.Error("Failed to list documents", zap.Error(err))
		_, err := b.sendMessage(chatID, "❌ Failed to retrieve documents.")
		return err
	}

	if len(files) == 0 {
		_, err := b.sendMessage(chatID, "📭 No documents found.\n\nUpload PDFs, images, or other files to access them here!")
		return err
	}

	var sb strings.Builder
	sb.WriteString("📁 *Your Documents*\n\n")

	for i, file := range files {
		sizeStr := formatFileSize(file.SizeBytes)
		sb.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, file.Filename))
		sb.WriteString(fmt.Sprintf("   📄 %s | 📦 %s\n", file.MimeType, sizeStr))
		sb.WriteString(fmt.Sprintf("   🕐 %s\n\n", file.CreatedAt.Format("Jan 2, 3:04 PM")))
	}

	_, err = b.sendMessage(chatID, sb.String())
	return err
}

// formatFileSize converts bytes to human readable format
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) error {
	chatID := msg.Chat.ID
	text := msg.Text

	// Get or create conversation for this chat
	convID := b.getConversationID(chatID)

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

	// Save conversation ID for future messages in this chat (persisted to database)
	if resp.ConversationID != "" {
		b.setConversationID(chatID, resp.ConversationID)
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

	// Process image through skills registry first
	ctx, cancel := context.WithTimeout(b.ctx, 60*time.Second)
	defer cancel()

	prompt := "Analyze this image and describe what you see."
	if msg.Caption != "" {
		prompt = msg.Caption
	}

	// Try to process image using the process_image skill if available
	var imageAnalysis string
	if result, err := b.agent.ExecuteTool(ctx, "process_image", map[string]interface{}{
		"file_path": filePath,
		"query":     prompt,
	}); err == nil {
		// Successfully processed image, extract the description
		if resultMap, ok := result.(map[string]interface{}); ok {
			if desc, ok := resultMap["description"].(string); ok && desc != "" {
				imageAnalysis = desc
			} else if text, ok := resultMap["text"].(string); ok && text != "" {
				imageAnalysis = text
			}
		}
		b.logger.Info("Image processed via vision API", zap.String("file", filePath))
	} else {
		b.logger.Warn("Image processing via skills failed, falling back to LLM tool calling",
			zap.Error(err))
	}

	// Save image to database for global access
	var fileRecord *store.File
	if b.store != nil {
		convID := b.getConversationID(chatID)
		mimeType := "image/jpeg"
		fileSize := int64(photo.FileSize)
		fileRecord = &store.File{
			Filename:    fmt.Sprintf("photo_%d.jpg", time.Now().Unix()),
			MimeType:    mimeType,
			SizeBytes:   fileSize,
			StoragePath: filePath,
			ConversationID: func() *string {
				if convID != "" {
					return &convID
				} else {
					return nil
				}
			}(),
			SourceChatID: &chatID,
		}
		if err := b.store.CreateFile(fileRecord); err != nil {
			b.logger.Warn("Failed to save image record", zap.Error(err))
			// Continue anyway, not critical
		} else {
			b.logger.Info("Image saved to database", zap.String("file_id", fileRecord.ID))
		}
	}

	// Build message - include image analysis if we got it, otherwise just the path
	var message string
	if imageAnalysis != "" {
		message = fmt.Sprintf("Image Analysis:\n%s\n\nUser question: %s", imageAnalysis, prompt)
	} else {
		message = fmt.Sprintf("[Image attached: %s]\n\n%s", filePath, prompt)
	}

	resp, err := b.agent.Chat(ctx, agent.ChatRequest{
		ConversationID: b.getConversationID(chatID),
		Message:        message,
		Stream:         false,
	})

	if err != nil {
		b.logger.Error("Agent error", zap.Error(err))
		_, sendErr := b.sendMessage(chatID, fmt.Sprintf("❌ Error analyzing image: %v", err))
		return sendErr
	}

	// Save conversation ID
	b.setConversationID(chatID, resp.ConversationID)

	// Update file record with processed text if successful
	if fileRecord != nil && b.store != nil {
		b.store.UpdateFileProcessedText(fileRecord.ID, resp.Content)
	}

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

	// Save file to database for global access
	var fileRecord *store.File
	if b.store != nil {
		convID := b.getConversationID(chatID)
		fileRecord = &store.File{
			Filename:    doc.FileName,
			MimeType:    doc.MimeType,
			SizeBytes:   int64(doc.FileSize),
			StoragePath: filePath,
			ConversationID: func() *string {
				if convID != "" {
					return &convID
				} else {
					return nil
				}
			}(),
			SourceChatID: &chatID,
		}
		if err := b.store.CreateFile(fileRecord); err != nil {
			b.logger.Warn("Failed to save file record", zap.Error(err))
			// Continue anyway, not critical
		} else {
			b.logger.Info("File saved to database", zap.String("file_id", fileRecord.ID), zap.String("filename", doc.FileName))
		}
	}

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

	// Update file record with processed text if successful
	if fileRecord != nil && b.store != nil {
		b.store.UpdateFileProcessedText(fileRecord.ID, resp.Content)
	}

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

// getConversationID returns the conversation ID for a chat (checks memory first, then database)
func (b *Bot) getConversationID(chatID int64) string {
	// Check in-memory cache first
	b.convMu.RLock()
	convID := b.conversations[chatID]
	b.convMu.RUnlock()

	if convID != "" {
		return convID
	}

	// Try to load from database
	if b.store != nil {
		mapping, err := b.store.GetChatMapping(chatID, "telegram")
		if err == nil && mapping != nil {
			b.convMu.Lock()
			b.conversations[chatID] = mapping.ConversationID
			b.convMu.Unlock()
			return mapping.ConversationID
		}
	}

	return ""
}

// setConversationID sets the conversation ID for a chat and persists it
func (b *Bot) setConversationID(chatID int64, convID string) {
	if convID == "" {
		return
	}

	// Update in-memory cache
	b.convMu.Lock()
	b.conversations[chatID] = convID
	b.convMu.Unlock()

	// Persist to database
	if b.store != nil {
		if err := b.store.SetChatMapping(chatID, "telegram", convID); err != nil {
			b.logger.Warn("Failed to persist conversation mapping",
				zap.Error(err),
				zap.Int64("chat_id", chatID),
				zap.String("conversation_id", convID))
		} else {
			b.logger.Debug("Conversation mapping persisted",
				zap.Int64("chat_id", chatID),
				zap.String("conversation_id", convID))
		}
	}
}

// clearConversationID clears the conversation mapping for a chat
func (b *Bot) clearConversationID(chatID int64) {
	// Clear in-memory cache
	b.convMu.Lock()
	delete(b.conversations, chatID)
	b.convMu.Unlock()

	// Deactivate in database
	if b.store != nil {
		if err := b.store.DeactivateChatMapping(chatID, "telegram"); err != nil {
			b.logger.Warn("Failed to deactivate conversation mapping",
				zap.Error(err),
				zap.Int64("chat_id", chatID))
		}
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

// Package voice provides Telegram bot voice message handling
package voice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramVoiceHandler handles voice messages in Telegram
type TelegramVoiceHandler struct {
	voiceSkill *VoiceSkill
	bot        *tgbotapi.BotAPI
}

// NewTelegramVoiceHandler creates a new voice handler for Telegram
func NewTelegramVoiceHandler(voiceSkill *VoiceSkill, bot *tgbotapi.BotAPI) *TelegramVoiceHandler {
	return &TelegramVoiceHandler{
		voiceSkill: voiceSkill,
		bot:        bot,
	}
}

// HandleVoiceMessage processes a voice message
func (tvh *TelegramVoiceHandler) HandleVoiceMessage(ctx context.Context, message *tgbotapi.Message) (string, error) {
	if message.Voice == nil {
		return "", fmt.Errorf("not a voice message")
	}
	
	// Download voice file
	voiceFile, err := tvh.downloadVoiceFile(message.Voice.FileID)
	if err != nil {
		return "", fmt.Errorf("failed to download voice: %w", err)
	}
	defer os.Remove(voiceFile)
	
	// Convert to WAV if needed (Telegram sends OGG)
	wavFile, err := tvh.convertToWav(voiceFile)
	if err != nil {
		return "", fmt.Errorf("failed to convert audio: %w", err)
	}
	defer os.Remove(wavFile)
	
	// Transcribe
	text, err := tvh.voiceSkill.Transcribe(ctx, wavFile)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}
	
	return text, nil
}

// SendVoiceResponse sends a voice message response
func (tvh *TelegramVoiceHandler) SendVoiceResponse(ctx context.Context, chatID int64, text string) error {
	// Synthesize speech
	audioPath, err := tvh.voiceSkill.Speak(ctx, text)
	if err != nil {
		return fmt.Errorf("speech synthesis failed: %w", err)
	}
	defer os.Remove(audioPath)
	
	// Convert to OGG for Telegram
	oggPath, err := tvh.convertToOgg(audioPath)
	if err != nil {
		return fmt.Errorf("failed to convert to OGG: %w", err)
	}
	defer os.Remove(oggPath)
	
	// Send voice message
	voice := tgbotapi.NewVoice(chatID, tgbotapi.FilePath(oggPath))
	_, err = tvh.bot.Send(voice)
	return err
}

// downloadVoiceFile downloads a voice file from Telegram
func (tvh *TelegramVoiceHandler) downloadVoiceFile(fileID string) (string, error) {
	// Get file URL
	file, err := tvh.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", err
	}
	
	// Download file
	url := file.Link(tvh.bot.Token)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	// Create temp file
	tempFile, err := os.CreateTemp("", "myrai_voice_*.oga")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	
	// Copy content
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}
	
	return tempFile.Name(), nil
}

// convertToWav converts audio to WAV format
func (tvh *TelegramVoiceHandler) convertToWav(inputPath string) (string, error) {
	ext := filepath.Ext(inputPath)
	if ext == ".wav" {
		return inputPath, nil
	}
	
	outputPath := inputPath + ".wav"
	
	// Use ffmpeg to convert
	// Telegram voice messages are OGG Opus
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-y",
		outputPath,
	)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w (output: %s)", err, string(output))
	}
	
	return outputPath, nil
}

// convertToOgg converts audio to OGG format for Telegram
func (tvh *TelegramVoiceHandler) convertToOgg(inputPath string) (string, error) {
	ext := filepath.Ext(inputPath)
	if ext == ".ogg" || ext == ".oga" {
		return inputPath, nil
	}
	
	outputPath := inputPath + ".ogg"
	
	// Use ffmpeg to convert
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:a", "libopus",
		"-b:a", "24k",
		"-y",
		outputPath,
	)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w (output: %s)", err, string(output))
	}
	
	return outputPath, nil
}

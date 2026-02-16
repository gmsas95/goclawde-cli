// Package voice provides speech-to-text and text-to-speech capabilities
// for Myrai (未来) - enabling natural voice interactions
package voice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/gmsas95/goclawde-cli/internal/skills"
)

// VoiceSkill provides voice processing capabilities
type VoiceSkill struct {
	*skills.BaseSkill
	
	// Configuration
	config Config
	
	// STT (Speech-to-Text)
	stt STTProvider
	
	// TTS (Text-to-Speech)
	tts TTSProvider
	
	// Audio recorder
	recorder *AudioRecorder
	
	// Audio player
	player *AudioPlayer
	
	// State
	mu       sync.RWMutex
	isReady  bool
}

// Config holds voice skill configuration
type Config struct {
	// Model paths
	WhisperModelPath string
	PiperModelPath   string
	PiperConfigPath  string
	
	// Audio settings
	SampleRate    int
	Channels      int
	BufferSize    int
	
	// STT settings
	STTLanguage   string // "auto" for auto-detect, or "en", "ja", etc.
	STTTranslate  bool   // Translate to English
	
	// TTS settings
	TTSSpeakerID  int    // Speaker ID for multi-speaker models
	TTSLengthScale float64 // Speed (1.0 = normal, <1.0 = faster, >1.0 = slower)
	
	// Voice Activity Detection
	EnableVAD     bool
	VADThreshold  float64
	VADMinSilence int // milliseconds
}

// DefaultConfig returns default voice configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".myrai")
	
	return Config{
		WhisperModelPath: filepath.Join(dataDir, "models", "whisper", "ggml-base.en.bin"),
		PiperModelPath:   filepath.Join(dataDir, "models", "piper", "en_US-lessac-medium.onnx"),
		PiperConfigPath:  filepath.Join(dataDir, "models", "piper", "en_US-lessac-medium.onnx.json"),
		SampleRate:       16000,
		Channels:         1,
		BufferSize:       4096,
		STTLanguage:      "auto",
		STTTranslate:     false,
		TTSSpeakerID:     0,
		TTSLengthScale:   1.0,
		EnableVAD:        true,
		VADThreshold:     0.5,
		VADMinSilence:    500,
	}
}

// NewVoiceSkill creates a new voice skill
func NewVoiceSkill(config Config) *VoiceSkill {
	vs := &VoiceSkill{
		BaseSkill: skills.NewBaseSkill("voice", "Voice processing with STT and TTS", "1.0.0"),
		config:    config,
	}
	vs.registerTools()
	return vs
}

// Initialize sets up the voice skill
func (vs *VoiceSkill) Initialize() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	
	if vs.isReady {
		return nil
	}
	
	// Initialize STT
	sttConfig := STTConfig{
		ModelPath:  vs.config.WhisperModelPath,
		Language:   vs.config.STTLanguage,
		Translate:  vs.config.STTTranslate,
	}
	
	stt, err := NewWhisperSTT(sttConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize STT: %w", err)
	}
	vs.stt = stt
	
	// Initialize TTS
	ttsConfig := TTSConfig{
		ModelPath:   vs.config.PiperModelPath,
		ConfigPath:  vs.config.PiperConfigPath,
		SpeakerID:   vs.config.TTSSpeakerID,
		LengthScale: vs.config.TTSLengthScale,
	}
	
	tts, err := NewPiperTTS(ttsConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize TTS: %w", err)
	}
	vs.tts = tts
	
	// Initialize audio recorder
	vs.recorder = NewAudioRecorder(vs.config.SampleRate, vs.config.Channels, vs.config.BufferSize)
	
	// Initialize audio player
	vs.player = NewAudioPlayer()
	
	vs.isReady = true
	return nil
}

// IsReady returns true if voice skill is initialized
func (vs *VoiceSkill) IsReady() bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.isReady
}

// Transcribe transcribes audio file to text
func (vs *VoiceSkill) Transcribe(ctx context.Context, audioPath string) (string, error) {
	if !vs.IsReady() {
		if err := vs.Initialize(); err != nil {
			return "", err
		}
	}
	
	return vs.stt.Transcribe(ctx, audioPath)
}

// TranscribeBytes transcribes audio bytes to text
func (vs *VoiceSkill) TranscribeBytes(ctx context.Context, audioData []byte) (string, error) {
	if !vs.IsReady() {
		if err := vs.Initialize(); err != nil {
			return "", err
		}
	}
	
	return vs.stt.TranscribeBytes(ctx, audioData)
}

// Speak converts text to speech and returns audio file path
func (vs *VoiceSkill) Speak(ctx context.Context, text string) (string, error) {
	if !vs.IsReady() {
		if err := vs.Initialize(); err != nil {
			return "", err
		}
	}
	
	return vs.tts.Synthesize(ctx, text)
}

// SpeakBytes converts text to speech and returns audio bytes
func (vs *VoiceSkill) SpeakBytes(ctx context.Context, text string) ([]byte, error) {
	if !vs.IsReady() {
		if err := vs.Initialize(); err != nil {
			return nil, err
		}
	}
	
	return vs.tts.SynthesizeBytes(ctx, text)
}

// PlayAudio plays audio file
func (vs *VoiceSkill) PlayAudio(audioPath string) error {
	if vs.player == nil {
		return fmt.Errorf("audio player not initialized")
	}
	return vs.player.PlayFile(audioPath)
}

// PlayAudioBytes plays audio bytes
func (vs *VoiceSkill) PlayAudioBytes(audioData []byte) error {
	if vs.player == nil {
		return fmt.Errorf("audio player not initialized")
	}
	return vs.player.PlayBytes(audioData)
}

// StartRecording starts recording audio
func (vs *VoiceSkill) StartRecording(outputPath string) error {
	if vs.recorder == nil {
		return fmt.Errorf("audio recorder not initialized")
	}
	return vs.recorder.Start(outputPath)
}

// StopRecording stops recording and returns path
func (vs *VoiceSkill) StopRecording() (string, error) {
	if vs.recorder == nil {
		return "", fmt.Errorf("audio recorder not initialized")
	}
	return vs.recorder.Stop()
}

// IsRecording returns true if currently recording
func (vs *VoiceSkill) IsRecording() bool {
	if vs.recorder == nil {
		return false
	}
	return vs.recorder.IsRecording()
}

// GetInfo returns voice skill info
func (vs *VoiceSkill) GetInfo() map[string]interface{} {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	info := map[string]interface{}{
		"name":     "voice",
		"version":  "1.0.0",
		"ready":    vs.isReady,
		"platform": runtime.GOOS,
	}
	
	if vs.stt != nil {
		info["stt_provider"] = vs.stt.Name()
		info["stt_ready"] = vs.stt.IsReady()
	}
	
	if vs.tts != nil {
		info["tts_provider"] = vs.tts.Name()
		info["tts_ready"] = vs.tts.IsReady()
	}
	
	return info
}

// registerTools registers voice-related tools
func (vs *VoiceSkill) registerTools() {
	// Transcribe audio
	vs.AddTool(skills.Tool{
		Name:        "transcribe_audio",
		Description: "Transcribe audio file to text",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"audio_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to audio file (wav, mp3, ogg)",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Language code (auto, en, ja, etc.)",
				},
			},
			"required": []string{"audio_path"},
		},
		Handler: vs.handleTranscribe,
	})
	
	// Text to speech
	vs.AddTool(skills.Tool{
		Name:        "text_to_speech",
		Description: "Convert text to speech audio",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to speak",
				},
				"speed": map[string]interface{}{
					"type":        "number",
					"description": "Speech speed (0.5-2.0, 1.0=normal)",
				},
			},
			"required": []string{"text"},
		},
		Handler: vs.handleTextToSpeech,
	})
	
	// Get voice info
	vs.AddTool(skills.Tool{
		Name:        "voice_info",
		Description: "Get voice processing status and info",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: vs.handleVoiceInfo,
	})
}

// handleTranscribe handles transcribe_audio tool
func (vs *VoiceSkill) handleTranscribe(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	audioPath, _ := args["audio_path"].(string)
	if audioPath == "" {
		return nil, fmt.Errorf("audio_path is required")
	}
	
	// Override language if provided
	if lang, ok := args["language"].(string); ok && lang != "" {
		// TODO: Implement per-request language override
		_ = lang
	}
	
	text, err := vs.Transcribe(ctx, audioPath)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}
	
	return map[string]string{
		"text": text,
	}, nil
}

// handleTextToSpeech handles text_to_speech tool
func (vs *VoiceSkill) handleTextToSpeech(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	text, _ := args["text"].(string)
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	
	// Get speed if provided
	if speed, ok := args["speed"].(float64); ok {
		vs.tts.SetSpeed(speed)
	}
	
	audioPath, err := vs.Speak(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("speech synthesis failed: %w", err)
	}
	
	return map[string]string{
		"audio_path": audioPath,
	}, nil
}

// handleVoiceInfo handles voice_info tool
func (vs *VoiceSkill) handleVoiceInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return vs.GetInfo(), nil
}

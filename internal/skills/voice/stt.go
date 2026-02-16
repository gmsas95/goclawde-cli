// Package voice provides STT (Speech-to-Text) capabilities
package voice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// STTProvider defines the interface for speech-to-text providers
type STTProvider interface {
	Name() string
	IsReady() bool
	Transcribe(ctx context.Context, audioPath string) (string, error)
	TranscribeBytes(ctx context.Context, audioData []byte) (string, error)
}

// STTConfig holds STT configuration
type STTConfig struct {
	ModelPath string
	Language  string // "auto" for auto-detect
	Translate bool   // Translate to English
}

// WhisperSTT implements STT using whisper.cpp
type WhisperSTT struct {
	config    STTConfig
	ready     bool
	modelDir  string
}

// NewWhisperSTT creates a new Whisper STT provider
func NewWhisperSTT(config STTConfig) (*WhisperSTT, error) {
	w := &WhisperSTT{
		config:   config,
		modelDir: filepath.Dir(config.ModelPath),
	}
	
	// Check if model exists
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		// Model doesn't exist, we'll mark as not ready
		// User needs to download it
		w.ready = false
	} else {
		w.ready = true
	}
	
	return w, nil
}

// Name returns the provider name
func (w *WhisperSTT) Name() string {
	return "whisper.cpp"
}

// IsReady returns true if the provider is ready
func (w *WhisperSTT) IsReady() bool {
	return w.ready
}

// Transcribe transcribes an audio file to text
func (w *WhisperSTT) Transcribe(ctx context.Context, audioPath string) (string, error) {
	if !w.ready {
		return "", fmt.Errorf("whisper model not found at %s. Run: myrai models download whisper", w.config.ModelPath)
	}
	
	// Check if audio file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return "", fmt.Errorf("audio file not found: %s", audioPath)
	}
	
	// Build whisper.cpp command
	args := []string{
		"-m", w.config.ModelPath,
		"-f", audioPath,
		"--output-txt",
		"--no-timestamps",
	}
	
	// Add language if specified
	if w.config.Language != "auto" && w.config.Language != "" {
		args = append(args, "-l", w.config.Language)
	}
	
	// Add translate flag if needed
	if w.config.Translate {
		args = append(args, "--translate")
	}
	
	// Find whisper executable
	whisperBin := w.findWhisperBinary()
	if whisperBin == "" {
		return "", fmt.Errorf("whisper binary not found. Please install whisper.cpp")
	}
	
	// Run whisper
	cmd := exec.CommandContext(ctx, whisperBin, args...)
	cmd.Dir = w.modelDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper failed: %w (output: %s)", err, string(output))
	}
	
	// Read the output file
	outputPath := audioPath + ".txt"
	content, err := os.ReadFile(outputPath)
	if err != nil {
		// Try to parse from stdout
		return strings.TrimSpace(string(output)), nil
	}
	
	// Clean up output file
	os.Remove(outputPath)
	
	return strings.TrimSpace(string(content)), nil
}

// TranscribeBytes transcribes audio bytes to text
func (w *WhisperSTT) TranscribeBytes(ctx context.Context, audioData []byte) (string, error) {
	// Write to temp file
	tempFile, err := os.CreateTemp("", "myrai_stt_*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	
	if _, err := tempFile.Write(audioData); err != nil {
		tempFile.Close()
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tempFile.Close()
	
	return w.Transcribe(ctx, tempFile.Name())
}

// findWhisperBinary finds the whisper.cpp executable
func (w *WhisperSTT) findWhisperBinary() string {
	// Check common locations
	candidates := []string{
		"whisper-cli",
		"whisper",
		"./whisper-cli",
		"./whisper",
		"/usr/local/bin/whisper-cli",
		"/usr/bin/whisper-cli",
	}
	
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	
	return ""
}

// DownloadModel downloads the whisper model
func (w *WhisperSTT) DownloadModel(modelName string) error {
	// Model URLs from Hugging Face
	modelURLs := map[string]string{
		"tiny":     "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
		"tiny.en":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin",
		"base":     "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
		"base.en":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin",
		"small":    "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin",
		"small.en": "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin",
		"medium":   "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin",
		"large":    "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large.bin",
	}
	
	url, ok := modelURLs[modelName]
	if !ok {
		return fmt.Errorf("unknown model: %s (available: %v)", modelName, getMapKeys(modelURLs))
	}
	
	// Create model directory
	if err := os.MkdirAll(w.modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}
	
	// Download model
	modelPath := filepath.Join(w.modelDir, fmt.Sprintf("ggml-%s.bin", modelName))
	
	fmt.Printf("Downloading whisper model: %s\n", modelName)
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Destination: %s\n", modelPath)
	
	cmd := exec.Command("curl", "-L", "-o", modelPath, url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	
	// Update config
	w.config.ModelPath = modelPath
	w.ready = true
	
	fmt.Printf("✓ Model downloaded to: %s\n", modelPath)
	return nil
}

// MockSTT is a mock STT provider for testing
type MockSTT struct {
	ready bool
}

// NewMockSTT creates a mock STT provider
func NewMockSTT() *MockSTT {
	return &MockSTT{ready: true}
}

// Name returns the provider name
func (m *MockSTT) Name() string {
	return "mock"
}

// IsReady returns true
func (m *MockSTT) IsReady() bool {
	return m.ready
}

// Transcribe returns mock transcription
func (m *MockSTT) Transcribe(ctx context.Context, audioPath string) (string, error) {
	return "This is a mock transcription. In production, this would be actual speech-to-text output.", nil
}

// TranscribeBytes returns mock transcription
func (m *MockSTT) TranscribeBytes(ctx context.Context, audioData []byte) (string, error) {
	return m.Transcribe(ctx, "")
}

func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

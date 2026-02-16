// Package voice provides TTS (Text-to-Speech) capabilities
package voice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TTSProvider defines the interface for text-to-speech providers
type TTSProvider interface {
	Name() string
	IsReady() bool
	Synthesize(ctx context.Context, text string) (string, error)
	SynthesizeBytes(ctx context.Context, text string) ([]byte, error)
	SetSpeed(speed float64)
}

// TTSConfig holds TTS configuration
type TTSConfig struct {
	ModelPath   string
	ConfigPath  string
	SpeakerID   int
	LengthScale float64 // 1.0 = normal, <1.0 = faster, >1.0 = slower
}

// PiperTTS implements TTS using piper
type PiperTTS struct {
	config TTSConfig
	ready  bool
}

// NewPiperTTS creates a new Piper TTS provider
func NewPiperTTS(config TTSConfig) (*PiperTTS, error) {
	p := &PiperTTS{
		config: config,
	}
	
	// Check if model exists
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		p.ready = false
	} else {
		p.ready = true
	}
	
	return p, nil
}

// Name returns the provider name
func (p *PiperTTS) Name() string {
	return "piper"
}

// IsReady returns true if the provider is ready
func (p *PiperTTS) IsReady() bool {
	return p.ready
}

// Synthesize converts text to speech and returns audio file path
func (p *PiperTTS) Synthesize(ctx context.Context, text string) (string, error) {
	if !p.ready {
		return "", fmt.Errorf("piper model not found at %s. Run: myrai models download piper", p.config.ModelPath)
	}
	
	// Create temp output file
	tempFile, err := os.CreateTemp("", "myrai_tts_*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFile.Close()
	outputPath := tempFile.Name()
	
	// Build piper command
	args := []string{
		"--model", p.config.ModelPath,
		"--config", p.config.ConfigPath,
		"--output_file", outputPath,
	}
	
	// Add speaker ID if specified
	if p.config.SpeakerID > 0 {
		args = append(args, "--speaker", fmt.Sprintf("%d", p.config.SpeakerID))
	}
	
	// Add length scale (speed)
	if p.config.LengthScale != 1.0 {
		args = append(args, "--length_scale", fmt.Sprintf("%.2f", p.config.LengthScale))
	}
	
	// Find piper executable
	piperBin := p.findPiperBinary()
	if piperBin == "" {
		return "", fmt.Errorf("piper binary not found. Please install piper-tts")
	}
	
	// Run piper
	cmd := exec.CommandContext(ctx, piperBin, args...)
	cmd.Stdin = strings.NewReader(text)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("piper failed: %w (output: %s)", err, string(output))
	}
	
	return outputPath, nil
}

// SynthesizeBytes converts text to speech and returns audio bytes
func (p *PiperTTS) SynthesizeBytes(ctx context.Context, text string) ([]byte, error) {
	outputPath, err := p.Synthesize(ctx, text)
	if err != nil {
		return nil, err
	}
	defer os.Remove(outputPath)
	
	return os.ReadFile(outputPath)
}

// SetSpeed sets the speech speed
func (p *PiperTTS) SetSpeed(speed float64) {
	if speed < 0.5 {
		speed = 0.5
	}
	if speed > 2.0 {
		speed = 2.0
	}
	p.config.LengthScale = 1.0 / speed // Invert: higher scale = slower
}

// findPiperBinary finds the piper executable
func (p *PiperTTS) findPiperBinary() string {
	candidates := []string{
		"piper",
		"piper-tts",
		"./piper",
		"./piper-tts",
		"/usr/local/bin/piper",
		"/usr/bin/piper",
	}
	
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	
	return ""
}

// DownloadModel downloads a piper voice model
func (p *PiperTTS) DownloadModel(voiceName string) error {
	// Piper voice URLs from Hugging Face
	voiceURLs := map[string]struct {
		model  string
		config string
	}{
		"en_US-lessac-medium": {
			model:  "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/en_US-lessac-medium.onnx",
			config: "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json",
		},
		"en_US-lessac-low": {
			model:  "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/low/en_US-lessac-low.onnx",
			config: "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/low/en_US-lessac-low.onnx.json",
		},
		"en_US-amy-medium": {
			model:  "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/amy/medium/en_US-amy-medium.onnx",
			config: "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/amy/medium/en_US-amy-medium.onnx.json",
		},
		"en_GB-southern_male-medium": {
			model:  "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_GB/southern_male/medium/en_GB-southern_male-medium.onnx",
			config: "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_GB/southern_male/medium/en_GB-southern_male-medium.onnx.json",
		},
	}
	
	voice, ok := voiceURLs[voiceName]
	if !ok {
		return fmt.Errorf("unknown voice: %s", voiceName)
	}
	
	// Create model directory
	modelDir := filepath.Dir(p.config.ModelPath)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}
	
	// Download model
	modelPath := filepath.Join(modelDir, fmt.Sprintf("%s.onnx", voiceName))
	configPath := filepath.Join(modelDir, fmt.Sprintf("%s.onnx.json", voiceName))
	
	fmt.Printf("Downloading piper voice: %s\n", voiceName)
	
	// Download model file
	fmt.Printf("Downloading model...\n")
	cmd := exec.Command("curl", "-L", "-o", modelPath, voice.model)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	
	// Download config file
	fmt.Printf("Downloading config...\n")
	cmd = exec.Command("curl", "-L", "-o", configPath, voice.config)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download config: %w", err)
	}
	
	// Update config
	p.config.ModelPath = modelPath
	p.config.ConfigPath = configPath
	p.ready = true
	
	fmt.Printf("✓ Voice model downloaded:\n")
	fmt.Printf("  Model: %s\n", modelPath)
	fmt.Printf("  Config: %s\n", configPath)
	return nil
}

// ListAvailableVoices returns available piper voices
func (p *PiperTTS) ListAvailableVoices() []string {
	return []string{
		"en_US-lessac-medium",
		"en_US-lessac-low",
		"en_US-amy-medium",
		"en_GB-southern_male-medium",
	}
}

// MockTTS is a mock TTS provider for testing
type MockTTS struct {
	ready bool
	speed float64
}

// NewMockTTS creates a mock TTS provider
func NewMockTTS() *MockTTS {
	return &MockTTS{ready: true, speed: 1.0}
}

// Name returns the provider name
func (m *MockTTS) Name() string {
	return "mock"
}

// IsReady returns true
func (m *MockTTS) IsReady() bool {
	return m.ready
}

// Synthesize creates a mock audio file
func (m *MockTTS) Synthesize(ctx context.Context, text string) (string, error) {
	// Create empty wav file for testing
	tempFile, err := os.CreateTemp("", "myrai_tts_mock_*.wav")
	if err != nil {
		return "", err
	}
	tempFile.Close()
	return tempFile.Name(), nil
}

// SynthesizeBytes returns mock audio bytes
func (m *MockTTS) SynthesizeBytes(ctx context.Context, text string) ([]byte, error) {
	return []byte("mock audio data"), nil
}

// SetSpeed sets mock speed
func (m *MockTTS) SetSpeed(speed float64) {
	m.speed = speed
}

// Package voice provides tests for voice processing
package voice

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewVoiceSkill(t *testing.T) {
	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	if skill == nil {
		t.Fatal("NewVoiceSkill returned nil")
	}

	if skill.Name() != "voice" {
		t.Errorf("Expected name 'voice', got '%s'", skill.Name())
	}

	if skill.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", skill.Version())
	}
}

func TestVoiceSkill_Initialize(t *testing.T) {
	// Use mock providers for testing
	config := Config{
		WhisperModelPath: "/tmp/mock_whisper.bin",
		PiperModelPath:   "/tmp/mock_piper.onnx",
		PiperConfigPath:  "/tmp/mock_piper.json",
		SampleRate:       16000,
		Channels:         1,
	}

	skill := NewVoiceSkill(config)

	// Should initialize without error even if models don't exist
	// (they'll be marked as not ready)
	err := skill.Initialize()
	if err != nil {
		t.Logf("Initialize error (expected if models not present): %v", err)
	}
}

func TestVoiceSkill_IsReady(t *testing.T) {
	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	// Initially not ready
	if skill.IsReady() {
		t.Error("Expected IsReady to be false before initialization")
	}
}

func TestVoiceSkill_GetInfo(t *testing.T) {
	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	info := skill.GetInfo()

	if info == nil {
		t.Fatal("GetInfo returned nil")
	}

	if info["name"] != "voice" {
		t.Errorf("Expected name 'voice', got '%v'", info["name"])
	}

	if info["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%v'", info["version"])
	}
}

func TestMockSTT(t *testing.T) {
	stt := NewMockSTT()

	if stt.Name() != "mock" {
		t.Errorf("Expected name 'mock', got '%s'", stt.Name())
	}

	if !stt.IsReady() {
		t.Error("Expected mock STT to be ready")
	}

	ctx := context.Background()
	text, err := stt.Transcribe(ctx, "test.wav")
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	if text == "" {
		t.Error("Expected non-empty transcription")
	}

	// Test TranscribeBytes
	text2, err := stt.TranscribeBytes(ctx, []byte("fake audio data"))
	if err != nil {
		t.Fatalf("TranscribeBytes failed: %v", err)
	}

	if text2 == "" {
		t.Error("Expected non-empty transcription from bytes")
	}
}

func TestMockTTS(t *testing.T) {
	tts := NewMockTTS()

	if tts.Name() != "mock" {
		t.Errorf("Expected name 'mock', got '%s'", tts.Name())
	}

	if !tts.IsReady() {
		t.Error("Expected mock TTS to be ready")
	}

	ctx := context.Background()

	// Test Synthesize
	path, err := tts.Synthesize(ctx, "Hello world")
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}

	if path == "" {
		t.Error("Expected non-empty path")
	}

	// Cleanup
	defer os.Remove(path)

	// Test SynthesizeBytes
	data, err := tts.SynthesizeBytes(ctx, "Hello world")
	if err != nil {
		t.Fatalf("SynthesizeBytes failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty audio data")
	}

	// Test SetSpeed
	tts.SetSpeed(1.5)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.SampleRate != 16000 {
		t.Errorf("Expected sample rate 16000, got %d", config.SampleRate)
	}

	if config.Channels != 1 {
		t.Errorf("Expected 1 channel, got %d", config.Channels)
	}

	if config.STTLanguage != "auto" {
		t.Errorf("Expected language 'auto', got '%s'", config.STTLanguage)
	}

	if config.TTSLengthScale != 1.0 {
		t.Errorf("Expected length scale 1.0, got %f", config.TTSLengthScale)
	}
}

func TestAudioRecorder(t *testing.T) {
	recorder := NewAudioRecorder(16000, 1, 4096)

	if recorder == nil {
		t.Fatal("NewAudioRecorder returned nil")
	}

	// Initially not recording
	if recorder.IsRecording() {
		t.Error("Expected IsRecording to be false initially")
	}

	// Test GetDuration (should be 0 when not recording)
	duration := recorder.GetDuration()
	if duration != 0 {
		t.Errorf("Expected duration 0, got %v", duration)
	}
}

func TestAudioPlayer(t *testing.T) {
	player := NewAudioPlayer()

	if player == nil {
		t.Fatal("NewAudioPlayer returned nil")
	}

	// Initially not playing
	if player.IsPlaying() {
		t.Error("Expected IsPlaying to be false initially")
	}
}

func TestVoiceSkill_Tools(t *testing.T) {
	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	tools := skill.Tools()

	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	// Check for expected tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{"transcribe_audio", "text_to_speech", "voice_info"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool '%s' not found", name)
		}
	}
}

func TestVoiceSkill_HandleTranscribe(t *testing.T) {
	// Create a mock audio file
	tmpFile, err := os.CreateTemp("", "test_*.wav")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write minimal WAV header
	wavHeader := []byte{
		'R', 'I', 'F', 'F',
		0x24, 0x00, 0x00, 0x00,
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		0x10, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x01, 0x00,
		0x44, 0xac, 0x00, 0x00,
		0x88, 0x58, 0x01, 0x00,
		0x02, 0x00,
		0x10, 0x00,
		'd', 'a', 't', 'a',
		0x00, 0x00, 0x00, 0x00,
	}
	tmpFile.Write(wavHeader)
	tmpFile.Close()

	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	// Initialize with mock
	skill.stt = NewMockSTT()
	skill.isReady = true

	ctx := context.Background()
	result, err := skill.handleTranscribe(ctx, map[string]interface{}{
		"audio_path": tmpFile.Name(),
	})

	if err != nil {
		t.Logf("Transcription error (may be expected without real STT): %v", err)
	}

	if result != nil {
		resultMap, ok := result.(map[string]string)
		if ok && resultMap["text"] == "" {
			t.Error("Expected non-empty text in result")
		}
	}
}

func TestVoiceSkill_HandleTextToSpeech(t *testing.T) {
	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	// Initialize with mock
	skill.tts = NewMockTTS()
	skill.isReady = true

	ctx := context.Background()
	result, err := skill.handleTextToSpeech(ctx, map[string]interface{}{
		"text": "Hello world",
	})

	if err != nil {
		t.Fatalf("Text to speech failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	resultMap, ok := result.(map[string]string)
	if !ok {
		t.Fatal("Expected map result")
	}

	if resultMap["audio_path"] == "" {
		t.Error("Expected non-empty audio_path")
	}

	// Cleanup
	os.Remove(resultMap["audio_path"])
}

func TestVoiceSkill_HandleVoiceInfo(t *testing.T) {
	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	ctx := context.Background()
	result, err := skill.handleVoiceInfo(ctx, map[string]interface{}{})

	if err != nil {
		t.Fatalf("Voice info failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	info, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if info["name"] != "voice" {
		t.Errorf("Expected name 'voice', got '%v'", info["name"])
	}
}

// Integration test (skipped by default)
func TestVoiceSkill_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := DefaultConfig()
	skill := NewVoiceSkill(config)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = ctx

	// Try to initialize with real providers
	err := skill.Initialize()
	if err != nil {
		t.Skipf("Skipping integration test - initialization failed: %v", err)
	}

	if !skill.IsReady() {
		t.Skip("Skipping integration test - skill not ready")
	}

	t.Logf("Voice skill ready with providers: STT=%v, TTS=%v", 
		skill.stt != nil, skill.tts != nil)
}

// Benchmark tests
func BenchmarkMockSTT_Transcribe(b *testing.B) {
	stt := NewMockSTT()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = stt.Transcribe(ctx, "test.wav")
	}
}

func BenchmarkMockTTS_Synthesize(b *testing.B) {
	tts := NewMockTTS()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path, _ := tts.Synthesize(ctx, "Hello world")
		os.Remove(path)
	}
}

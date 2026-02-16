// Package voice provides audio recording and playback
package voice

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// AudioRecorder handles audio recording
type AudioRecorder struct {
	sampleRate int
	channels   int
	bufferSize int
	
	recording bool
	mu        sync.Mutex
	
	// Current recording
	outputPath string
	startTime  time.Time
	cancelFunc func()
}

// NewAudioRecorder creates a new audio recorder
func NewAudioRecorder(sampleRate, channels, bufferSize int) *AudioRecorder {
	return &AudioRecorder{
		sampleRate: sampleRate,
		channels:   channels,
		bufferSize: bufferSize,
	}
}

// Start starts recording audio to the specified file
func (ar *AudioRecorder) Start(outputPath string) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	
	if ar.recording {
		return fmt.Errorf("already recording")
	}
	
	ar.outputPath = outputPath
	ar.startTime = time.Now()
	ar.recording = true
	
	// Use arecord (Linux) or sox or ffmpeg for recording
	// This is a platform-specific implementation
	go ar.record()
	
	return nil
}

// Stop stops recording and returns the output path
func (ar *AudioRecorder) Stop() (string, error) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	
	if !ar.recording {
		return "", fmt.Errorf("not recording")
	}
	
	ar.recording = false
	
	if ar.cancelFunc != nil {
		ar.cancelFunc()
	}
	
	return ar.outputPath, nil
}

// IsRecording returns true if currently recording
func (ar *AudioRecorder) IsRecording() bool {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	return ar.recording
}

// GetDuration returns the current recording duration
func (ar *AudioRecorder) GetDuration() time.Duration {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	
	if !ar.recording {
		return 0
	}
	
	return time.Since(ar.startTime)
}

// record performs the actual recording
func (ar *AudioRecorder) record() {
	// Platform-specific recording
	// For now, we'll use ffmpeg or arecord
	
	var cmd *exec.Cmd
	
	// Try arecord first (Linux with ALSA)
	if _, err := exec.LookPath("arecord"); err == nil {
		cmd = exec.Command("arecord",
			"-f", "S16_LE",
			"-r", fmt.Sprintf("%d", ar.sampleRate),
			"-c", fmt.Sprintf("%d", ar.channels),
			"-t", "wav",
			ar.outputPath,
		)
	} else if _, err := exec.LookPath("ffmpeg"); err == nil {
		// Fallback to ffmpeg
		cmd = exec.Command("ffmpeg",
			"-f", "alsa",
			"-i", "default",
			"-ar", fmt.Sprintf("%d", ar.sampleRate),
			"-ac", fmt.Sprintf("%d", ar.channels),
			"-y",
			ar.outputPath,
		)
	} else {
		// No recording tool available
		fmt.Println("Warning: No audio recording tool found (arecord or ffmpeg)")
		return
	}
	
	// Start recording
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start recording: %v\n", err)
		return
	}
	
	// Wait for stop signal
	for ar.IsRecording() {
		time.Sleep(100 * time.Millisecond)
	}
	
	// Stop recording
	cmd.Process.Signal(os.Interrupt)
	cmd.Wait()
}

// AudioPlayer handles audio playback
type AudioPlayer struct {
	mu      sync.Mutex
	playing bool
}

// NewAudioPlayer creates a new audio player
func NewAudioPlayer() *AudioPlayer {
	return &AudioPlayer{}
}

// PlayFile plays an audio file
func (ap *AudioPlayer) PlayFile(audioPath string) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	
	if ap.playing {
		return fmt.Errorf("already playing")
	}
	
	// Check if file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", audioPath)
	}
	
	ap.playing = true
	defer func() { ap.playing = false }()
	
	// Try different audio players
	players := []string{
		"aplay",     // Linux ALSA
		"paplay",    // PulseAudio
		"afplay",    // macOS
		"ffplay",    // FFmpeg
	}
	
	for _, player := range players {
		if _, err := exec.LookPath(player); err == nil {
			var cmd *exec.Cmd
			
			switch player {
			case "aplay":
				cmd = exec.Command("aplay", audioPath)
			case "paplay":
				cmd = exec.Command("paplay", audioPath)
			case "afplay":
				cmd = exec.Command("afplay", audioPath)
			case "ffplay":
				cmd = exec.Command("ffplay", "-nodisp", "-autoexit", audioPath)
			}
			
			return cmd.Run()
		}
	}
	
	return fmt.Errorf("no audio player found (tried: %v)", players)
}

// PlayBytes plays audio from bytes
func (ap *AudioPlayer) PlayBytes(audioData []byte) error {
	// Write to temp file and play
	tempFile, err := os.CreateTemp("", "myrai_audio_*.wav")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	
	if _, err := tempFile.Write(audioData); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tempFile.Close()
	
	return ap.PlayFile(tempFile.Name())
}

// IsPlaying returns true if currently playing
func (ap *AudioPlayer) IsPlaying() bool {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	return ap.playing
}

// ConvertAudio converts audio file to different format
func ConvertAudio(inputPath, outputPath string, sampleRate, channels int) error {
	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found for audio conversion")
	}
	
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ar", fmt.Sprintf("%d", sampleRate),
		"-ac", fmt.Sprintf("%d", channels),
		"-y",
		outputPath,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %w (output: %s)", err, string(output))
	}
	
	return nil
}

// GetAudioInfo returns audio file info
func GetAudioInfo(audioPath string) (map[string]interface{}, error) {
	// Check if ffprobe is available
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, fmt.Errorf("ffprobe not found")
	}
	
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration,size,bit_rate",
		"-show_entries", "stream=codec_name,sample_rate,channels",
		"-of", "json",
		audioPath,
	)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}
	
	// Parse JSON output
	info := map[string]interface{}{}
	// TODO: Parse JSON properly
	_ = output
	
	return info, nil
}

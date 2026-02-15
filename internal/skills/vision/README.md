# Vision & Audio Skill

Camera, screenshot, and audio capabilities for GoClawde - bringing Jarvis-like vision and hearing to your AI assistant.

## Capabilities

### Camera (`capture_photo`)
- Capture photos from system camera
- Automatic analysis with vision models
- Cross-platform support (macOS, Linux, Windows)

**Requirements:** ffmpeg or platform-specific tools (imagesnap on macOS, fswebcam on Linux)

### Screenshot (`capture_screenshot`)
- Capture full screen
- Optional AI analysis of screen content
- Useful for "what do you see on my screen?" queries

**Requirements:** Platform screenshot tools (screencapture on macOS, gnome-screenshot/ImageMagick on Linux)

### Image Analysis (`analyze_image`, `describe_image`)
- Analyze existing image files
- Multiple detail levels (low/medium/high)
- Extract text (OCR), objects, context

**Requirements:** Vision-capable LLM (GPT-4V, Claude 3, etc.)

### Audio (`listen`)
- Record audio from microphone
- Transcribe to text using Whisper
- Duration limits (max 60 seconds)

**Requirements:** ffmpeg, Whisper API or local model

## Configuration

```yaml
vision:
  vision_model: "gpt-4-vision-preview"  # or "claude-3-opus-20240229"
  data_dir: "~/.goclawde"
```

## Platform Notes

### macOS
```bash
# Install imagesnap for camera
brew install imagesnap

# ffmpeg is recommended for all features
brew install ffmpeg
```

### Linux
```bash
# Ubuntu/Debian
sudo apt install ffmpeg fswebcam

# Or use v4l2 directly with ffmpeg
```

### Windows
- Install ffmpeg and add to PATH
- Camera/audio support via dshow

## Usage Examples

```go
// Capture and analyze
capture, err := vision.NewCapture(llmClient, "~/.goclawde")
result, err := capture.CaptureSnapshot(ctx, vision.CaptureOptions{})
analysis, err := capture.AnalyzeImage(ctx, result, "What's on my desk?")

// Record voice command
text, err := capture.ProcessVoiceCommand(ctx, 5*time.Second)
// Returns: "turn on the lights"
```

## Future Enhancements

- [ ] Real-time video streaming
- [ ] Continuous audio monitoring (wake word)
- [ ] Screen recording for demos/tutorials
- [ ] Barcode/QR code reading
- [ ] Face detection/recognition
- [ ] Object tracking in video

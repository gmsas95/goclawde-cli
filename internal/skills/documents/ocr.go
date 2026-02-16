// Package documents provides OCR capabilities
package documents

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// OCRProcessor defines the interface for OCR
type OCRProcessor interface {
	ExtractText(imagePath string) (string, error)
	ExtractTextWithLang(imagePath string, lang string) (string, error)
	IsAvailable() bool
}

// tesseractOCR implements OCR using tesseract
type tesseractOCR struct {
	binaryPath string
}

// NewTesseractOCR creates a new tesseract OCR processor
func NewTesseractOCR(binaryPath string) OCRProcessor {
	if binaryPath == "" {
		binaryPath = "tesseract"
	}
	
	return &tesseractOCR{
		binaryPath: binaryPath,
	}
}

// IsAvailable checks if tesseract is installed
func (t *tesseractOCR) IsAvailable() bool {
	_, err := exec.LookPath(t.binaryPath)
	return err == nil
}

// ExtractText extracts text from image using tesseract
func (t *tesseractOCR) ExtractText(imagePath string) (string, error) {
	return t.ExtractTextWithLang(imagePath, "eng")
}

// ExtractTextWithLang extracts text with specified language
func (t *tesseractOCR) ExtractTextWithLang(imagePath string, lang string) (string, error) {
	// Check file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return "", fmt.Errorf("image file not found: %s", imagePath)
	}
	
	// Create temp output file
	tempOutput := imagePath + "_ocr"
	
	// Build tesseract command
	args := []string{
		imagePath,
		tempOutput,
		"-l", lang,
		"--psm", "6", // Assume single uniform block of text
	}
	
	cmd := exec.Command(t.binaryPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tesseract failed: %w (output: %s)", err, string(output))
	}
	
	// Read output file
	outputFile := tempOutput + ".txt"
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read OCR output: %w", err)
	}
	
	// Clean up
	os.Remove(outputFile)
	
	return strings.TrimSpace(string(content)), nil
}

// ListLanguages lists available tesseract languages
func (t *tesseractOCR) ListLanguages() ([]string, error) {
	cmd := exec.Command(t.binaryPath, "--list-langs")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list languages: %w", err)
	}
	
	lines := strings.Split(string(output), "\n")
	var langs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "List") {
			langs = append(langs, line)
		}
	}
	
	return langs, nil
}

// InstallLanguage installs a tesseract language pack
func (t *tesseractOCR) InstallLanguage(lang string) error {
	// This is platform-specific
	// On Ubuntu/Debian: apt-get install tesseract-ocr-<lang>
	return fmt.Errorf("language installation is platform-specific. Install tesseract-ocr-%s manually", lang)
}

// MockOCR is a mock OCR for testing
type MockOCR struct{}

// NewMockOCR creates a mock OCR
func NewMockOCR() OCRProcessor {
	return &MockOCR{}
}

// IsAvailable always returns true
func (m *MockOCR) IsAvailable() bool {
	return true
}

// ExtractText returns mock text
func (m *MockOCR) ExtractText(imagePath string) (string, error) {
	return "This is mock OCR text. In production, this would be the actual text extracted from the image.", nil
}

// ExtractTextWithLang returns mock text
func (m *MockOCR) ExtractTextWithLang(imagePath string, lang string) (string, error) {
	return m.ExtractText(imagePath)
}

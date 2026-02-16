// Package documents provides image processing capabilities
package documents

import (
	"fmt"
	"image"
	_ "image/gif"  // Register GIF format
	_ "image/jpeg" // Register JPEG format
	_ "image/png"  // Register PNG format
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// ImageProcessor defines the interface for image processing
type ImageProcessor interface {
	GetInfo(imagePath string) (*ImageInfo, error)
	Resize(imagePath string, width, height int) (string, error)
	ConvertToJPEG(imagePath string, quality int) (string, error)
	Thumbnail(imagePath string, maxDim int) (string, error)
}

// imageProcessor implements ImageProcessor
type imageProcessor struct {
	tempDir string
}

// NewImageProcessor creates a new image processor
func NewImageProcessor() ImageProcessor {
	return &imageProcessor{
		tempDir: os.TempDir(),
	}
}

// GetInfo returns image information
func (ip *imageProcessor) GetInfo(imagePath string) (*ImageInfo, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	// Decode image
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	
	bounds := img.Bounds()
	
	return &ImageInfo{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Format: format,
		Size:   stat.Size(),
	}, nil
}

// Resize resizes an image using ImageMagick or ffmpeg
func (ip *imageProcessor) Resize(imagePath string, width, height int) (string, error) {
	outputPath := filepath.Join(ip.tempDir, fmt.Sprintf("resized_%dx%d_%s", width, height, filepath.Base(imagePath)))
	
	// Try ImageMagick convert
	if _, err := exec.LookPath("convert"); err == nil {
		cmd := exec.Command("convert", imagePath, "-resize", fmt.Sprintf("%dx%d", width, height), outputPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("convert failed: %w", err)
		}
		return outputPath, nil
	}
	
	// Try ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-i", imagePath, "-vf", fmt.Sprintf("scale=%d:%d", width, height), "-y", outputPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("ffmpeg failed: %w", err)
		}
		return outputPath, nil
	}
	
	return "", fmt.Errorf("no image processing tool found (install ImageMagick or ffmpeg)")
}

// ConvertToJPEG converts image to JPEG format
func (ip *imageProcessor) ConvertToJPEG(imagePath string, quality int) (string, error) {
	if quality < 1 || quality > 100 {
		quality = 85
	}
	
	outputPath := filepath.Join(ip.tempDir, fmt.Sprintf("converted_%d_%s.jpg", quality, filepath.Base(imagePath)))
	
	// Try ImageMagick
	if _, err := exec.LookPath("convert"); err == nil {
		cmd := exec.Command("convert", imagePath, "-quality", strconv.Itoa(quality), outputPath)
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return outputPath, nil
	}
	
	// Try ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-i", imagePath, "-q:v", strconv.Itoa(quality/10), "-y", outputPath)
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return outputPath, nil
	}
	
	return "", fmt.Errorf("no image conversion tool found")
}

// Thumbnail creates a thumbnail of the image
func (ip *imageProcessor) Thumbnail(imagePath string, maxDim int) (string, error) {
	info, err := ip.GetInfo(imagePath)
	if err != nil {
		return "", err
	}
	
	// Calculate new dimensions maintaining aspect ratio
	width, height := info.Width, info.Height
	if width > height {
		if width > maxDim {
			height = height * maxDim / width
			width = maxDim
		}
	} else {
		if height > maxDim {
			width = width * maxDim / height
			height = maxDim
		}
	}
	
	return ip.Resize(imagePath, width, height)
}

// OptimizeForAPI optimizes image for API upload (resize + compress)
func OptimizeForAPI(imagePath string, maxDim, quality int) (string, error) {
	ip := NewImageProcessor()
	
	// Get current info
	info, err := ip.GetInfo(imagePath)
	if err != nil {
		return "", err
	}
	
	// Resize if too large
	if info.Width > maxDim || info.Height > maxDim {
		return ip.Thumbnail(imagePath, maxDim)
	}
	
	// Convert to JPEG for compression
	if info.Format != "jpeg" {
		return ip.ConvertToJPEG(imagePath, quality)
	}
	
	return imagePath, nil
}

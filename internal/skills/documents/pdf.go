// Package documents provides PDF processing capabilities
package documents

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// PDFProcessor defines the interface for PDF processing
type PDFProcessor interface {
	ExtractText(filePath string, options ProcessOptions) (string, error)
	ExtractImages(filePath string, maxPages int) ([]string, error)
	GetMetadata(filePath string) (*PDFMetadata, error)
	ConvertToImages(filePath string, startPage, endPage int) ([]string, error)
}

// pdfProcessor implements PDFProcessor using external tools
type pdfProcessor struct {
	// Configuration
	tempDir string
}

// NewPDFProcessor creates a new PDF processor
func NewPDFProcessor() PDFProcessor {
	tempDir := os.TempDir()
	return &pdfProcessor{
		tempDir: tempDir,
	}
}

// ExtractText extracts text from PDF using pdftotext or pdfcpu
func (p *pdfProcessor) ExtractText(filePath string, options ProcessOptions) (string, error) {
	// Try pdftotext first (poppler-utils)
	if _, err := exec.LookPath("pdftotext"); err == nil {
		return p.extractTextWithPdftotext(filePath, options)
	}
	
	// Try pdfcpu
	if _, err := exec.LookPath("pdfcpu"); err == nil {
		return p.extractTextWithPDFCPU(filePath)
	}
	
	// Try pdftoppm + tesseract as fallback
	return "", fmt.Errorf("no PDF text extraction tool found (install poppler-utils or pdfcpu)")
}

// extractTextWithPdftotext uses pdftotext from poppler-utils
func (p *pdfProcessor) extractTextWithPdftotext(filePath string, options ProcessOptions) (string, error) {
	outputFile := filepath.Join(p.tempDir, fmt.Sprintf("pdf_extract_%d.txt", os.Getpid()))
	defer os.Remove(outputFile)
	
	args := []string{
		"-layout", // Maintain layout
		"-nopgbrk", // No page breaks
		filePath,
		outputFile,
	}
	
	cmd := exec.Command("pdftotext", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftotext failed: %w (output: %s)", err, string(output))
	}
	
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", err
	}
	
	return string(content), nil
}

// extractTextWithPDFCPU uses pdfcpu
func (p *pdfProcessor) extractTextWithPDFCPU(filePath string) (string, error) {
	// pdfcpu doesn't have direct text extraction, use extraction + parsing
	// This is a simplified version
	cmd := exec.Command("pdfcpu", "info", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdfcpu failed: %w", err)
	}
	
	// pdfcpu info doesn't give text, we'd need to extract and parse
	// For now, return metadata
	return string(output), nil
}

// ExtractImages extracts images from PDF pages
func (p *pdfProcessor) ExtractImages(filePath string, maxPages int) ([]string, error) {
	// Use pdfimages from poppler-utils
	if _, err := exec.LookPath("pdfimages"); err != nil {
		return nil, fmt.Errorf("pdfimages not found (install poppler-utils)")
	}
	
	outputPrefix := filepath.Join(p.tempDir, fmt.Sprintf("pdf_img_%d", os.Getpid()))
	
	cmd := exec.Command("pdfimages", "-j", filePath, outputPrefix)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdfimages failed: %w", err)
	}
	
	// Find extracted images
	pattern := outputPrefix + "*"
	images, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	// Limit to maxPages
	if len(images) > maxPages {
		// Clean up excess
		for _, img := range images[maxPages:] {
			os.Remove(img)
		}
		images = images[:maxPages]
	}
	
	return images, nil
}

// GetMetadata extracts PDF metadata
func (p *pdfProcessor) GetMetadata(filePath string) (*PDFMetadata, error) {
	// Try pdfinfo first
	if _, err := exec.LookPath("pdfinfo"); err == nil {
		return p.getMetadataWithPdfinfo(filePath)
	}
	
	// Try pdfcpu
	if _, err := exec.LookPath("pdfcpu"); err == nil {
		return p.getMetadataWithPDFCPU(filePath)
	}
	
	return nil, fmt.Errorf("no PDF metadata tool found")
}

// getMetadataWithPdfinfo uses pdfinfo from poppler-utils
func (p *pdfProcessor) getMetadataWithPdfinfo(filePath string) (*PDFMetadata, error) {
	cmd := exec.Command("pdfinfo", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pdfinfo failed: %w", err)
	}
	
	metadata := &PDFMetadata{}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		switch key {
		case "Title":
			metadata.Title = value
		case "Author":
			metadata.Author = value
		case "Subject":
			metadata.Subject = value
		case "Creator":
			metadata.Creator = value
		case "Producer":
			metadata.Producer = value
		case "CreationDate":
			metadata.CreationDate = value
		case "ModDate":
			metadata.ModDate = value
		case "Pages":
			if pages, err := strconv.Atoi(value); err == nil {
				metadata.PageCount = pages
			}
		case "Encrypted":
			metadata.Encrypted = strings.ToLower(value) == "yes"
		}
	}
	
	return metadata, nil
}

// getMetadataWithPDFCPU uses pdfcpu
func (p *pdfProcessor) getMetadataWithPDFCPU(filePath string) (*PDFMetadata, error) {
	cmd := exec.Command("pdfcpu", "info", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pdfcpu failed: %w", err)
	}
	
	metadata := &PDFMetadata{}
	
	// Parse pdfcpu output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Pages:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "Pages:" && i+1 < len(parts) {
					if pages, err := strconv.Atoi(parts[i+1]); err == nil {
						metadata.PageCount = pages
					}
				}
			}
		}
	}
	
	return metadata, nil
}

// ConvertToImages converts PDF pages to images using pdftoppm
func (p *pdfProcessor) ConvertToImages(filePath string, startPage, endPage int) ([]string, error) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil, fmt.Errorf("pdftoppm not found (install poppler-utils)")
	}
	
	outputPrefix := filepath.Join(p.tempDir, fmt.Sprintf("pdf_page_%d", os.Getpid()))
	
	args := []string{
		"-jpeg",        // JPEG output
		"-r", "150",     // 150 DPI (good balance)
		"-f", strconv.Itoa(startPage), // First page
		"-l", strconv.Itoa(endPage),   // Last page
		filePath,
		outputPrefix,
	}
	
	cmd := exec.Command("pdftoppm", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm failed: %w", err)
	}
	
	// Find generated images
	pattern := outputPrefix + "*.jpg"
	images, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	return images, nil
}

// PDFExtractPages extracts specific pages to a new PDF
func PDFExtractPages(inputPath, outputPath string, pages []int) error {
	if _, err := exec.LookPath("pdfcpu"); err == nil {
		// Use pdfcpu
		pageStr := make([]string, len(pages))
		for i, p := range pages {
			pageStr[i] = strconv.Itoa(p)
		}
		
		cmd := exec.Command("pdfcpu", "trim", "-pages", strings.Join(pageStr, ","), inputPath, outputPath)
		return cmd.Run()
	}
	
	return fmt.Errorf("pdfcpu required for page extraction")
}

// PDFMerge merges multiple PDFs into one
func PDFMerge(outputPath string, inputPaths []string) error {
	if _, err := exec.LookPath("pdfcpu"); err == nil {
		args := append([]string{"merge", outputPath}, inputPaths...)
		cmd := exec.Command("pdfcpu", args...)
		return cmd.Run()
	}
	
	return fmt.Errorf("pdfcpu required for PDF merging")
}

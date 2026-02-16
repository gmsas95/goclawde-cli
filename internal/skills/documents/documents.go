// Package documents provides document processing capabilities
// for Myrai (未来) - handling PDFs, images, OCR, and multimodal AI
package documents

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gmsas95/myrai-cli/internal/skills"
)

// DocumentSkill provides document processing capabilities
type DocumentSkill struct {
	*skills.BaseSkill
	config Config
	
	// Processors
	pdfProcessor    PDFProcessor
	imageProcessor  ImageProcessor
	ocrProcessor    OCRProcessor
	visionProcessor VisionProcessor
	
	mu      sync.RWMutex
	isReady bool
}

// Config holds document skill configuration
type Config struct {
	// Processing mode
	Mode string // "local", "api", "hybrid" (default: hybrid)
	
	// Local processing
	EnableLocalPDF   bool
	EnableLocalOCR   bool
	EnableLocalVision bool
	
	// API processing
	EnableAPIVision bool
	APIProvider     string // "gemini", "openai", "anthropic"
	APIKey          string
	
	// Model paths
	TesseractPath   string
	MoondreamPath   string // Local vision model
	
	// Processing limits
	MaxFileSize     int64  // bytes
	MaxPages        int    // for PDFs
	MaxImageSize    int    // max dimension in pixels
}

// DefaultConfig returns default document configuration
func DefaultConfig() Config {
	return Config{
		Mode:              "hybrid", // Use local when possible, API for complex tasks
		EnableLocalPDF:    true,
		EnableLocalOCR:    true,
		EnableLocalVision: false, // Moondream is optional
		EnableAPIVision:   true,
		APIProvider:       "gemini", // Default to Gemini (good multimodal)
		MaxFileSize:       50 * 1024 * 1024, // 50MB
		MaxPages:          100,
		MaxImageSize:      4096,
	}
}

// NewDocumentSkill creates a new document skill
func NewDocumentSkill(config Config) *DocumentSkill {
	ds := &DocumentSkill{
		BaseSkill: skills.NewBaseSkill("documents", "Document processing with PDF, OCR, and vision", "1.0.0"),
		config:    config,
	}
	ds.registerTools()
	return ds
}

// Initialize sets up the document skill
func (ds *DocumentSkill) Initialize() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	if ds.isReady {
		return nil
	}
	
	// Initialize PDF processor
	if ds.config.EnableLocalPDF {
		ds.pdfProcessor = NewPDFProcessor()
	}
	
	// Initialize OCR processor
	if ds.config.EnableLocalOCR {
		ds.ocrProcessor = NewTesseractOCR(ds.config.TesseractPath)
	}
	
	// Initialize image processor
	ds.imageProcessor = NewImageProcessor()
	
	// Initialize vision processor (API-based)
	if ds.config.EnableAPIVision {
		ds.visionProcessor = NewAPIVisionProcessor(
			ds.config.APIProvider,
			ds.config.APIKey,
		)
	}
	
	ds.isReady = true
	return nil
}

// ProcessPDF extracts text and information from PDF
func (ds *DocumentSkill) ProcessPDF(ctx context.Context, filePath string, options ProcessOptions) (*DocumentResult, error) {
	if !ds.isReady {
		if err := ds.Initialize(); err != nil {
			return nil, err
		}
	}
	
	// Check file exists and size
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	if info.Size() > ds.config.MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), ds.config.MaxFileSize)
	}
	
	result := &DocumentResult{
		FilePath: filePath,
		FileType: "pdf",
		FileSize: info.Size(),
	}
	
	// Strategy based on mode
	switch ds.config.Mode {
	case "local":
		return ds.processPDFLocal(ctx, filePath, options, result)
	case "api":
		return ds.processPDFViaAPI(ctx, filePath, options, result)
	case "hybrid":
		// Try local first, fall back to API for complex docs
		localResult, err := ds.processPDFLocal(ctx, filePath, options, result)
		if err != nil || len(localResult.Text) < 100 {
			return ds.processPDFViaAPI(ctx, filePath, options, result)
		}
		return localResult, nil
	default:
		return ds.processPDFLocal(ctx, filePath, options, result)
	}
}

// ProcessImage analyzes an image using OCR and/or vision
func (ds *DocumentSkill) ProcessImage(ctx context.Context, filePath string, query string) (*ImageResult, error) {
	if !ds.isReady {
		if err := ds.Initialize(); err != nil {
			return nil, err
		}
	}
	
	// Check file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	
	result := &ImageResult{
		FilePath: filePath,
		FileSize: info.Size(),
	}
	
	// Get image info
	imgInfo, err := ds.imageProcessor.GetInfo(filePath)
	if err == nil {
		result.Width = imgInfo.Width
		result.Height = imgInfo.Height
		result.Format = imgInfo.Format
	}
	
	// Strategy based on mode and query
	switch ds.config.Mode {
	case "local":
		// OCR only
		if ds.ocrProcessor != nil {
			text, err := ds.ocrProcessor.ExtractText(filePath)
			if err == nil {
				result.Text = text
				result.OCRText = text
			}
		}
		
	case "api", "hybrid":
		// Use multimodal AI for understanding
		if ds.visionProcessor != nil {
			visionResult, err := ds.visionProcessor.Analyze(ctx, filePath, query)
			if err == nil {
				result.Description = visionResult.Description
				result.Text = visionResult.Text
				result.Entities = visionResult.Entities
				result.IsReceipt = visionResult.IsReceipt
				result.IsDocument = visionResult.IsDocument
				
				// Extract receipt data if applicable
				if visionResult.IsReceipt && visionResult.ReceiptData != nil {
					result.ReceiptData = visionResult.ReceiptData
				}
			}
		}
		
		// Also do OCR as fallback/supplement
		if ds.ocrProcessor != nil && result.Text == "" {
			text, err := ds.ocrProcessor.ExtractText(filePath)
			if err == nil {
				result.OCRText = text
				if result.Text == "" {
					result.Text = text
				}
			}
		}
	}
	
	return result, nil
}

// ExtractReceipt specifically handles receipt extraction
func (ds *DocumentSkill) ExtractReceipt(ctx context.Context, filePath string) (*ReceiptData, error) {
	result, err := ds.ProcessImage(ctx, filePath, "Extract receipt data: merchant, date, items, total, tax")
	if err != nil {
		return nil, err
	}
	
	if result.ReceiptData != nil {
		return result.ReceiptData, nil
	}
	
	// Fallback: try to parse from text
	return ds.parseReceiptFromText(result.Text), nil
}

// processPDFLocal uses local PDF processing
func (ds *DocumentSkill) processPDFLocal(ctx context.Context, filePath string, options ProcessOptions, result *DocumentResult) (*DocumentResult, error) {
	if ds.pdfProcessor == nil {
		return nil, fmt.Errorf("PDF processor not available")
	}
	
	// Extract text
	text, err := ds.pdfProcessor.ExtractText(filePath, options)
	if err != nil {
		return nil, fmt.Errorf("PDF text extraction failed: %w", err)
	}
	result.Text = text
	
	// Get metadata
	metadata, err := ds.pdfProcessor.GetMetadata(filePath)
	if err == nil {
		result.Metadata = metadata
		result.PageCount = metadata.PageCount
	}
	
	// Extract images for OCR if needed
	if options.ExtractImages && ds.ocrProcessor != nil {
		images, err := ds.pdfProcessor.ExtractImages(filePath, ds.config.MaxPages)
		if err == nil {
			var ocrTexts []string
			for _, img := range images {
				text, _ := ds.ocrProcessor.ExtractText(img)
				if text != "" {
					ocrTexts = append(ocrTexts, text)
				}
				os.Remove(img) // Clean up temp image
			}
			result.OCRText = strings.Join(ocrTexts, "\n")
		}
	}
	
	// Classify document type
	result.DocumentType = ds.classifyDocument(result.Text)
	
	return result, nil
}

// processPDFViaAPI converts PDF to images and uses multimodal AI
func (ds *DocumentSkill) processPDFViaAPI(ctx context.Context, filePath string, options ProcessOptions, result *DocumentResult) (*DocumentResult, error) {
	if ds.visionProcessor == nil {
		return nil, fmt.Errorf("vision processor not available")
	}
	
	// Convert PDF pages to images
	images, err := ds.pdfProcessor.ConvertToImages(filePath, 1, min(options.MaxPages, ds.config.MaxPages))
	if err != nil {
		return nil, fmt.Errorf("PDF conversion failed: %w", err)
	}
	defer func() {
		// Clean up temp images
		for _, img := range images {
			os.Remove(img)
		}
	}()
	
	result.PageCount = len(images)
	
	// Analyze each page with vision API
	var descriptions []string
	var allText []string
	
	for i, img := range images {
		query := "Extract all text and describe the content. If this is a receipt or invoice, extract merchant, date, items, and total."
		if i == 0 {
			query = "Describe this document and extract all text. What type of document is this?"
		}
		
		visionResult, err := ds.visionProcessor.Analyze(ctx, img, query)
		if err != nil {
			continue
		}
		
		descriptions = append(descriptions, visionResult.Description)
		allText = append(allText, visionResult.Text)
		
		// Extract receipt data from first page
		if i == 0 && visionResult.IsReceipt && visionResult.ReceiptData != nil {
			result.ReceiptData = visionResult.ReceiptData
		}
		
		// Store entities
		if len(visionResult.Entities) > 0 {
			result.Entities = append(result.Entities, visionResult.Entities...)
		}
	}
	
	result.Description = strings.Join(descriptions, "\n")
	result.Text = strings.Join(allText, "\n")
	result.DocumentType = ds.classifyDocument(result.Text)
	
	return result, nil
}

// classifyDocument determines document type from content
func (ds *DocumentSkill) classifyDocument(text string) string {
	text = strings.ToLower(text)
	
	// Receipt indicators
	if strings.Contains(text, "receipt") || 
	   strings.Contains(text, "total") && strings.Contains(text, "tax") ||
	   strings.Contains(text, "invoice #") {
		return "receipt"
	}
	
	// Contract indicators
	if strings.Contains(text, "agreement") ||
	   strings.Contains(text, "contract") ||
	   strings.Contains(text, "terms and conditions") {
		return "contract"
	}
	
	// Resume indicators
	if strings.Contains(text, "experience") && strings.Contains(text, "education") ||
	   strings.Contains(text, "curriculum vitae") ||
	   strings.Contains(text, "resume") {
		return "resume"
	}
	
	// Financial
	if strings.Contains(text, "statement") ||
	   strings.Contains(text, "balance") ||
	   strings.Contains(text, "account") {
		return "financial"
	}
	
	return "document"
}

// parseReceiptFromText attempts to extract receipt data from raw text
func (ds *DocumentSkill) parseReceiptFromText(text string) *ReceiptData {
	// Simple heuristic parsing - can be improved with regex
	receipt := &ReceiptData{
		Items: []ReceiptItem{},
	}
	
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Look for total
		if strings.Contains(strings.ToLower(line), "total") {
			// Try to extract amount
			receipt.Total = ds.extractAmount(line)
		}
		
		// Look for merchant (first non-empty line often has merchant)
		if receipt.Merchant == "" && len(line) > 2 && len(line) < 50 {
			receipt.Merchant = line
		}
	}
	
	return receipt
}

// extractAmount tries to find a dollar amount in text
func (ds *DocumentSkill) extractAmount(text string) string {
	// Simple regex-like parsing
	words := strings.Fields(text)
	for _, word := range words {
		// Look for $X.XX pattern
		if strings.HasPrefix(word, "$") {
			return word
		}
		// Look for X.XX with dollar context
		if strings.Contains(word, ".") {
			// Check if it's a number
			if _, err := fmt.Sscanf(word, "%f", new(float64)); err == nil {
				return "$" + word
			}
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetInfo returns document skill info
func (ds *DocumentSkill) GetInfo() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	info := map[string]interface{}{
		"name":        "documents",
		"version":     "1.0.0",
		"ready":       ds.isReady,
		"mode":        ds.config.Mode,
		"max_file_size": ds.config.MaxFileSize,
	}
	
	if ds.pdfProcessor != nil {
		info["pdf_processor"] = "enabled"
	}
	if ds.ocrProcessor != nil {
		info["ocr_processor"] = "enabled"
	}
	if ds.visionProcessor != nil {
		info["vision_processor"] = ds.config.APIProvider
	}
	
	return info
}

// registerTools registers document-related tools
func (ds *DocumentSkill) registerTools() {
	// Process PDF
	ds.AddTool(skills.Tool{
		Name:        "process_pdf",
		Description: "Extract text and information from PDF documents",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the PDF file",
				},
				"extract_images": map[string]interface{}{
					"type":        "boolean",
					"description": "Also extract images for OCR (default: false)",
				},
				"max_pages": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum pages to process (default: 50)",
				},
			},
			"required": []string{"file_path"},
		},
		Handler: ds.handleProcessPDF,
	})
	
	// Process Image
	ds.AddTool(skills.Tool{
		Name:        "process_image",
		Description: "Analyze images using OCR and vision AI. Can read text, describe scenes, extract receipt data",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the image file",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "What to look for in the image (e.g., 'extract receipt data', 'describe this scene')",
				},
			},
			"required": []string{"file_path"},
		},
		Handler: ds.handleProcessImage,
	})
	
	// Extract Receipt
	ds.AddTool(skills.Tool{
		Name:        "extract_receipt",
		Description: "Specifically extract receipt/invoice data: merchant, date, items, total, tax",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to receipt image or PDF",
				},
			},
			"required": []string{"file_path"},
		},
		Handler: ds.handleExtractReceipt,
	})
	
	// Document Info
	ds.AddTool(skills.Tool{
		Name:        "document_info",
		Description: "Get document processing status and available capabilities",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: ds.handleDocumentInfo,
	})
}

// handleProcessPDF handles process_pdf tool
func (ds *DocumentSkill) handleProcessPDF(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	
	options := ProcessOptions{
		MaxPages: 50,
	}
	
	if ei, ok := args["extract_images"].(bool); ok {
		options.ExtractImages = ei
	}
	if mp, ok := args["max_pages"].(float64); ok {
		options.MaxPages = int(mp)
	}
	
	result, err := ds.ProcessPDF(ctx, filePath, options)
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// handleProcessImage handles process_image tool
func (ds *DocumentSkill) handleProcessImage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	
	query, _ := args["query"].(string)
	if query == "" {
		query = "Describe this image and extract any text"
	}
	
	result, err := ds.ProcessImage(ctx, filePath, query)
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// handleExtractReceipt handles extract_receipt tool
func (ds *DocumentSkill) handleExtractReceipt(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	
	receipt, err := ds.ExtractReceipt(ctx, filePath)
	if err != nil {
		return nil, err
	}
	
	return receipt, nil
}

// handleDocumentInfo handles document_info tool
func (ds *DocumentSkill) handleDocumentInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return ds.GetInfo(), nil
}

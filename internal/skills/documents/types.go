// Package documents provides document types and structures
package documents

// ProcessOptions holds options for document processing
type ProcessOptions struct {
	ExtractImages bool
	ExtractTables bool
	MaxPages      int
	Language      string
}

// DocumentResult holds the result of document processing
type DocumentResult struct {
	FilePath     string                 `json:"file_path"`
	FileType     string                 `json:"file_type"`
	FileSize     int64                  `json:"file_size"`
	PageCount    int                    `json:"page_count"`
	Text         string                 `json:"text"`
	OCRText      string                 `json:"ocr_text,omitempty"`
	Description  string                 `json:"description,omitempty"`
	DocumentType string                 `json:"document_type"`
	Metadata     *PDFMetadata           `json:"metadata,omitempty"`
	ReceiptData  *ReceiptData           `json:"receipt_data,omitempty"`
	Entities     []Entity               `json:"entities,omitempty"`
}

// ImageResult holds the result of image processing
type ImageResult struct {
	FilePath    string      `json:"file_path"`
	FileSize    int64       `json:"file_size"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`
	Format      string      `json:"format"`
	Text        string      `json:"text"`
	OCRText     string      `json:"ocr_text,omitempty"`
	Description string      `json:"description,omitempty"`
	IsReceipt   bool        `json:"is_receipt"`
	IsDocument  bool        `json:"is_document"`
	ReceiptData *ReceiptData `json:"receipt_data,omitempty"`
	Entities    []Entity    `json:"entities,omitempty"`
}

// ReceiptData holds extracted receipt information
type ReceiptData struct {
	Merchant string        `json:"merchant"`
	Date     string        `json:"date"`
	Time     string        `json:"time,omitempty"`
	Items    []ReceiptItem `json:"items"`
	Subtotal string        `json:"subtotal,omitempty"`
	Tax      string        `json:"tax,omitempty"`
	Tip      string        `json:"tip,omitempty"`
	Total    string        `json:"total"`
	Payment  string        `json:"payment_method,omitempty"`
	Category string        `json:"category,omitempty"`
}

// ReceiptItem represents a single item on a receipt
type ReceiptItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    string `json:"price"`
	Total    string `json:"total,omitempty"`
}

// Entity represents an extracted entity
type Entity struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

// PDFMetadata holds PDF metadata
type PDFMetadata struct {
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Creator     string `json:"creator,omitempty"`
	Producer    string `json:"producer,omitempty"`
	CreationDate string `json:"creation_date,omitempty"`
	ModDate     string `json:"mod_date,omitempty"`
	PageCount   int    `json:"page_count"`
	Encrypted   bool   `json:"encrypted"`
}

// ImageInfo holds image information
type ImageInfo struct {
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Format  string `json:"format"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode,omitempty"` // RGB, RGBA, etc.
}

// VisionResult holds vision API result
type VisionResult struct {
	Description string       `json:"description"`
	Text        string       `json:"text"`
	IsReceipt   bool         `json:"is_receipt"`
	IsDocument  bool         `json:"is_document"`
	ReceiptData *ReceiptData `json:"receipt_data,omitempty"`
	Entities    []Entity     `json:"entities,omitempty"`
}

// DocumentType represents document classification
type DocumentType string

const (
	DocTypeReceipt   DocumentType = "receipt"
	DocTypeInvoice   DocumentType = "invoice"
	DocTypeContract  DocumentType = "contract"
	DocTypeResume    DocumentType = "resume"
	DocTypeFinancial DocumentType = "financial"
	DocTypeForm      DocumentType = "form"
	DocTypeManual    DocumentType = "manual"
	DocTypeUnknown   DocumentType = "document"
)

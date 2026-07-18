---
type: guide
title: Myrai Document Processing Guide
resource: goclawde-cli
description: "Myrai's document skill provides comprehensive document processing capabilities including:"
tags: []
updated: 2026-06-18
---

# Myrai Document Processing Guide

## Overview

Myrai's document skill provides comprehensive document processing capabilities including:
- **PDF Processing**: Text extraction, OCR, metadata
- **Image Analysis**: OCR, scene description, object detection
- **Receipt Extraction**: Structured data extraction
- **Multimodal AI**: Vision APIs for complex understanding

---

## Processing Modes

### 1. Local Mode (Privacy-First)
Uses only local tools, no data sent to cloud.

**Requirements**:
- `pdftotext` (poppler-utils) - PDF text extraction
- `tesseract-ocr` - OCR for images
- `pdftoppm` - PDF to image conversion

**Pros**:
- ✅ Complete privacy
- ✅ Works offline
- ✅ No API costs

**Cons**:
- ❌ Limited accuracy on complex layouts
- ❌ No semantic understanding
- ❌ Requires tool installation

### 2. API Mode (Accuracy-First)
Uses multimodal AI APIs for best results.

**Supported APIs**:
- Google Gemini 1.5 Flash/Pro
- OpenAI GPT-4 Vision
- Anthropic Claude 3

**Pros**:
- ✅ High accuracy
- ✅ Semantic understanding
- ✅ Complex layout handling
- ✅ Receipt extraction

**Cons**:
- ❌ Requires internet
- ❌ API costs
- ❌ Data sent to cloud

### 3. Hybrid Mode (Default)
Best of both worlds. Tries local first, falls back to API when needed.

**Logic**:
1. Try local extraction
2. If result is poor (<100 chars), use API
3. Use API for scanned documents

---

## Capabilities Matrix

| Feature | Local | API | Hybrid |
|---------|-------|-----|--------|
| PDF text extraction | ✅ | ✅ | ✅ |
| PDF OCR (scanned) | ✅ | ✅ | ✅ |
| Image OCR | ✅ | ✅ | ✅ |
| Scene description | ❌ | ✅ | ✅ |
| Receipt extraction | ⚠️ | ✅ | ✅ |
| Document classification | ⚠️ | ✅ | ✅ |
| Entity extraction | ❌ | ✅ | ✅ |
| Multi-page PDFs | ✅ | ✅ | ✅ |
| Handwriting | ❌ | ✅ | ✅ |

---

## File Size Limits

| Type | Max Size | Max Pages/Dimensions |
|------|----------|---------------------|
| PDF | 50MB | 100 pages |
| Image | 20MB | 4096x4096px |

---

## API Providers Comparison

| Provider | Speed | Accuracy | Cost | Best For |
|----------|-------|----------|------|----------|
| Gemini Flash | ⚡ Fast | Good | 💰 Low | Quick OCR, receipts |
| Gemini Pro | 🚀 Medium | Excellent | 💰💰 Medium | Complex docs |
| GPT-4 Vision | 🐢 Slow | Excellent | 💰💰💰 High | Detailed analysis |
| Claude 3 | 🚀 Medium | Excellent | 💰💰 Medium | Long documents |

---

## Setup

### Install Local Tools (Ubuntu/Debian)
```bash
# PDF tools
sudo apt-get install poppler-utils

# OCR
sudo apt-get install tesseract-ocr

# Additional languages (optional)
sudo apt-get install tesseract-ocr-jpn tesseract-ocr-chi-sim
```

### Configure API Keys

**Config file** (`~/.myrai/config.yaml`):
```yaml
llm:
  providers:
    google:
      api_key: "your-gemini-api-key"
    openai:
      api_key: "your-openai-api-key"
    anthropic:
      api_key: "your-anthropic-api-key"

documents:
  mode: "hybrid"  # local, api, or hybrid
  api_provider: "gemini"
```

**Environment variables**:
```bash
export GOOGLE_API_KEY="your-gemini-key"
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
```

---

## Usage Examples

### Process a PDF
```bash
# Basic extraction
./myrai tools process_pdf file_path=document.pdf

# With OCR for scanned PDFs
./myrai tools process_pdf file_path=scanned.pdf extract_images=true

# Limit pages
./myrai tools process_pdf file_path=large.pdf max_pages=10
```

### Process an Image
```bash
# Basic OCR
./myrai tools process_image file_path=receipt.jpg

# With specific query
./myrai tools process_image file_path=document.jpg \
  query="What type of document is this?"

# Extract text only
./myrai tools process_image file_path=menu.jpg \
  query="Extract all menu items and prices"
```

### Extract Receipt
```bash
./myrai tools extract_receipt file_path=receipt.jpg

# Returns structured data:
# {
#   "merchant": "Whole Foods",
#   "date": "2024-02-16",
#   "items": [...],
#   "total": "$45.50"
# }
```

---

## Output Format

### PDF Result
```json
{
  "file_path": "/path/to/doc.pdf",
  "file_type": "pdf",
  "file_size": 1024000,
  "page_count": 5,
  "text": "Full extracted text...",
  "ocr_text": "OCR text from images...",
  "description": "API-generated description",
  "document_type": "contract",
  "metadata": {
    "title": "Contract",
    "author": "John Doe",
    "page_count": 5
  },
  "receipt_data": null,
  "entities": [
    {"type": "date", "value": "2024-01-15"},
    {"type": "email", "value": "john@example.com"}
  ]
}
```

### Image Result
```json
{
  "file_path": "/path/to/image.jpg",
  "file_size": 204800,
  "width": 1920,
  "height": 1080,
  "format": "jpeg",
  "text": "OCR extracted text...",
  "description": "A receipt from Whole Foods...",
  "is_receipt": true,
  "is_document": false,
  "receipt_data": {
    "merchant": "Whole Foods",
    "total": "$45.50"
  },
  "entities": [...]
}
```

---

## Constraints & Limitations

### Local Processing
- Requires tool installation
- Lower accuracy on complex layouts
- No semantic understanding
- Limited language support (depends on tesseract)

### API Processing
- Requires internet connection
- API rate limits apply
- Costs per request
- Privacy concerns (data sent to cloud)

### General
- Max file sizes enforced
- Large PDFs may take time
- Handwriting accuracy varies
- Complex tables may not parse perfectly

---

## Troubleshooting

### "No PDF tool found"
Install poppler-utils:
```bash
sudo apt-get install poppler-utils
```

### "Tesseract not found"
Install tesseract:
```bash
sudo apt-get install tesseract-ocr
```

### "API key not configured"
Set API key in config or environment variable.

### "File too large"
Split PDF into smaller chunks or reduce image size.

---

## Privacy Recommendations

**For sensitive documents**:
1. Use `mode: local` in config
2. Install local tools
3. Avoid API mode for confidential data

**For general use**:
1. Hybrid mode (default) provides good balance
2. Gemini Flash is cost-effective
3. Claude 3 for sensitive but complex docs

---

## Future Enhancements

- [ ] Support for more document types
- [ ] Table extraction
- [ ] Handwriting improvement
- [ ] Local vision models (Moondream, LLaVA)
- [ ] Document comparison
- [ ] Batch processing
- [ ] Document search/indexing

---

## API Costs (Approximate)

| Provider | Cost per 1K pages |
|----------|-------------------|
| Gemini Flash | ~$0.075 |
| Gemini Pro | ~$0.50 |
| GPT-4 Vision | ~$1.00 |
| Claude 3 | ~$0.80 |

*Costs vary by document complexity and region*

---

**Myrai: Read the world, understand your documents.** 📄👁️

package expenses

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ExpenseParser parses natural language expense descriptions
type ExpenseParser struct {
	referenceTime time.Time
}

// NewExpenseParser creates a new expense parser
func NewExpenseParser() *ExpenseParser {
	return &ExpenseParser{
		referenceTime: time.Now(),
	}
}

// WithReference sets the reference time
func (p *ExpenseParser) WithReference(t time.Time) *ExpenseParser {
	p.referenceTime = t
	return p
}

// ParseExpense parses natural language expense description
func (p *ExpenseParser) ParseExpense(text string) (*ParseResult, error) {
	text = strings.TrimSpace(text)
	
	result := &ParseResult{
		Currency:   "USD",
		Date:       p.referenceTime,
		Confidence: 0.5,
	}
	
	// Extract amount
	amount, currency := p.extractAmount(text)
	if amount > 0 {
		result.Amount = amount
		result.Currency = currency
		result.Confidence += 0.2
	}
	
	// Extract merchant/description
	merchant, description := p.extractMerchant(text)
	if merchant != "" {
		result.Merchant = merchant
	}
	if description != "" {
		result.Description = description
	}
	
	// Infer category
	category := p.inferCategory(text, merchant)
	if category != "" {
		result.Category = category
		result.Confidence += 0.1
	}
	
	// Extract date
	date := p.extractDate(text)
	if !date.IsZero() {
		result.Date = date
	}
	
	return result, nil
}

// extractAmount extracts the monetary amount from text
func (p *ExpenseParser) extractAmount(text string) (float64, string) {
	currency := "USD"
	
	// Currency symbols/prefixes
	currencyPatterns := []struct {
		pattern  string
		symbol   string
		currency string
	}{
		{`\$([0-9,]+\.?\d*)`, "$", "USD"},
		{`€([0-9,]+\.?\d*)`, "€", "EUR"},
		{`£([0-9,]+\.?\d*)`, "£", "GBP"},
		{`¥([0-9,]+\.?\d*)`, "¥", "JPY"},
		{`([0-9,]+\.?\d*)\s*USD?`, "", "USD"},
		{`([0-9,]+\.?\d*)\s*EUR?`, "", "EUR"},
	}
	
	for _, cp := range currencyPatterns {
		re := regexp.MustCompile(`(?i)` + cp.pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			amountStr := strings.ReplaceAll(matches[1], ",", "")
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err == nil && amount > 0 {
				return amount, cp.currency
			}
		}
	}
	
	return 0, currency
}

// extractMerchant extracts merchant name from text
func (p *ExpenseParser) extractMerchant(text string) (merchant, description string) {
	// Common patterns
	patterns := []string{
		`(?i)(?:at|from|with)\s+([A-Z][a-zA-Z\s&]+)`,
		`(?i)([A-Z][a-zA-Z\s&]+)(?:\s+(?:store|shop|market|restaurant|cafe|gas|pharmacy))`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			merchant = strings.TrimSpace(matches[1])
			break
		}
	}
	
	// Description is the whole text without amount
	description = text
	// Remove amount patterns
	amountRe := regexp.MustCompile(`(?i)[\$€£¥]?[\d,]+\.?\d*\s*(?:USD|EUR|GBP|JPY)?`)
	description = amountRe.ReplaceAllString(description, "")
	description = strings.TrimSpace(description)
	
	return
}

// inferCategory infers category from merchant and description
func (p *ExpenseParser) inferCategory(text, merchant string) string {
	textLower := strings.ToLower(text)
	merchantLower := strings.ToLower(merchant)
	
	// Category keywords
	categories := map[string][]string{
		string(CategoryFood):        {"restaurant", "food", "lunch", "dinner", "breakfast", "cafe", "coffee", "pizza", "burger", "sushi"},
		string(CategoryGroceries):   {"grocery", "supermarket", "walmart", "target", "costco", "trader joe", "whole foods", "safeway"},
		string(CategoryTransport):   {"gas", "fuel", "uber", "lyft", "taxi", "train", "bus", "subway", "metro", "parking"},
		string(CategoryHousing):     {"rent", "mortgage", "apartment", "housing"},
		string(CategoryUtilities):   {"electric", "water", "gas bill", "internet", "phone", "utility"},
		string(CategoryHealth):      {"doctor", "pharmacy", "medical", "health", "dentist", "hospital", "drug"},
		string(CategoryEntertainment): {"movie", "netflix", "spotify", "game", "theater", "concert", "bar", "entertainment"},
		string(CategoryShopping):    {"amazon", "shopping", "clothes", "shoes", "electronics", "mall"},
		string(CategoryPersonal):    {"haircut", "salon", "spa", "gym", "fitness"},
		string(CategoryEducation):   {"book", "course", "tuition", "school", "university", "education"},
		string(CategoryTravel):      {"hotel", "flight", "airline", "travel", "booking", "airbnb"},
		string(CategoryBills):       {"bill", "payment", "subscription", "insurance"},
	}
	
	// Check merchant first
	for cat, keywords := range categories {
		for _, kw := range keywords {
			if strings.Contains(merchantLower, kw) {
				return cat
			}
		}
	}
	
	// Check text
	for cat, keywords := range categories {
		for _, kw := range keywords {
			if strings.Contains(textLower, kw) {
				return cat
			}
		}
	}
	
	return string(CategoryOther)
}

// extractDate extracts date from text
func (p *ExpenseParser) extractDate(text string) time.Time {
	now := p.referenceTime
	
	// Today
	if regexp.MustCompile(`(?i)\btoday\b`).MatchString(text) {
		return now
	}
	
	// Yesterday
	if regexp.MustCompile(`(?i)\byesterday\b`).MatchString(text) {
		return now.AddDate(0, 0, -1)
	}
	
	// This week
	if regexp.MustCompile(`(?i)\bthis week\b`).MatchString(text) {
		return now.AddDate(0, 0, -int(now.Weekday()))
	}
	
	// Last week
	if regexp.MustCompile(`(?i)\blast week\b`).MatchString(text) {
		return now.AddDate(0, 0, -int(now.Weekday())-7)
	}
	
	// Day of week
	days := map[string]int{
		"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3,
		"thursday": 4, "friday": 5, "saturday": 6,
	}
	
	for day, num := range days {
		pattern := regexp.MustCompile(`(?i)\b` + day + `\b`)
		if pattern.MatchString(text) {
			currentDay := int(now.Weekday())
			daysBack := currentDay - num
			if daysBack <= 0 {
				daysBack += 7
			}
			return now.AddDate(0, 0, -daysBack)
		}
	}
	
	return now
}

// ParseReceipt parses OCR text from a receipt
func (p *ExpenseParser) ParseReceipt(ocrText string) (*ReceiptData, error) {
	receipt := &ReceiptData{
		Items:      []ReceiptItem{},
		Currency:   "USD",
		Confidence: 0.5,
		Date:       p.referenceTime,
	}
	
	lines := strings.Split(ocrText, "\n")
	
	// Look for merchant name (usually at top)
	for i, line := range lines {
		if i < 5 && len(line) > 2 && len(line) < 50 {
			// Clean up
			merchant := strings.TrimSpace(line)
			if merchant != "" && !regexp.MustCompile(`(?i)(total|subtotal|tax|date)`).MatchString(merchant) {
				receipt.Merchant = merchant
				break
			}
		}
	}
	
	// Look for total amount
	totalPatterns := []string{
		`(?i)total[:\s]+[$€£¥]?([0-9,]+\.\d{2})`,
		`(?i)amount[:\s]+[$€£¥]?([0-9,]+\.\d{2})`,
		`(?i)balance[:\s]+[$€£¥]?([0-9,]+\.\d{2})`,
		`[$€£¥]([0-9,]+\.\d{2})\s*(?:total)?`,
	}
	
	for _, pattern := range totalPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(ocrText)
		if len(matches) > 1 {
			amountStr := strings.ReplaceAll(matches[1], ",", "")
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err == nil && amount > 0 {
				receipt.Total = amount
				receipt.Confidence += 0.2
				break
			}
		}
	}
	
	// Look for date
	datePatterns := []string{
		`(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`,
		`(\d{4}[/-]\d{1,2}[/-]\d{1,2})`,
	}
	
	for _, pattern := range datePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(ocrText)
		if len(matches) > 1 {
			// Try to parse date
			dateFormats := []string{"1/2/2006", "01/02/2006", "2006-01-02", "1-2-06"}
			for _, format := range dateFormats {
				if d, err := time.Parse(format, matches[1]); err == nil {
					receipt.Date = d
					break
				}
			}
			break
		}
	}
	
	// Look for items
	for _, line := range lines {
		// Simple item pattern: description + price
		itemRe := regexp.MustCompile(`(?i)^(.+?)\s+[$€£¥]?([0-9,]+\.\d{2})`)
		matches := itemRe.FindStringSubmatch(line)
		if len(matches) > 2 {
			priceStr := strings.ReplaceAll(matches[2], ",", "")
			price, err := strconv.ParseFloat(priceStr, 64)
			if err == nil && price > 0 && price < receipt.Total {
				item := ReceiptItem{
					Name:  strings.TrimSpace(matches[1]),
					Price: price,
					Total: price,
				}
				receipt.Items = append(receipt.Items, item)
			}
		}
	}
	
	return receipt, nil
}

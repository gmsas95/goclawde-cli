package shopping

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedItem represents a single parsed item from natural language
type ParsedItem struct {
	Name           string
	Quantity       string
	Unit           string
	Category       string
	Priority       string
	StoreAisle     string
	EstimatedPrice float64
	Notes          string
}

// ParsedShoppingInput represents the full parsed input
type ParsedShoppingInput struct {
	Name        string
	Description string
	Category    string
	StoreName   string
	Tags        []string
	Items       []ParsedItem
}

// Parser handles natural language parsing for shopping lists
type Parser struct {
	// Keywords for item categories
	categoryKeywords map[string][]string
	// Common units
	unitPatterns []string
	// Priority keywords
	priorityKeywords map[string]string
}

// NewParser creates a new shopping parser
func NewParser() *Parser {
	return &Parser{
		categoryKeywords: map[string][]string{
			"produce":    {"apple", "banana", "orange", "fruit", "vegetable", "lettuce", "tomato", "onion", "garlic", "carrot", "potato"},
			"dairy":      {"milk", "cheese", "butter", "yogurt", "cream", "egg", "eggs"},
			"meat":       {"chicken", "beef", "pork", "steak", "fish", "salmon", "ground meat", "sausage"},
			"bakery":     {"bread", "bagel", "croissant", "muffin", "bun", "roll", "tortilla"},
			"pantry":     {"rice", "pasta", "noodle", "flour", "sugar", "oil", "vinegar", "sauce", "spice", "salt", "pepper"},
			"beverages":  {"water", "juice", "soda", "coffee", "tea", "wine", "beer", "soda"},
			"frozen":     {"frozen", "ice cream", "pizza"},
			"snacks":     {"chip", "cracker", "cookie", "candy", "snack", "chocolate", "nut"},
			"household":  {"toilet paper", "paper towel", "soap", "shampoo", "toothpaste", "cleaner"},
			"personal":   {"toothbrush", "deodorant", "razor", "shaving", "lotion", "skincare"},
		},
		unitPatterns: []string{
			`\d+\s*(?:gallon|gal)s?`,
			`\d+\s*(?:liter|litre|l)s?`,
			`\d+\s*(?:pound|lb)s?`,
			`\d+\s*(?:ounce|oz)s?`,
			`\d+\s*(?:gram|g)s?`,
			`\d+\s*(?:kilogram|kg)s?`,
			`\d+\s*(?:cup|tbsp|tsp)s?`,
			`\d+\s*(?:pack|box|bag|can|bottle|jar|loaf|dozen)s?`,
		},
		priorityKeywords: map[string]string{
			"urgent":    "high",
			"asap":      "high",
			"important": "high",
			"needed":    "high",
			"low":       "low",
			"whenever":  "low",
			"optional":  "low",
			"maybe":     "low",
		},
	}
}

// ParseShoppingInput parses a natural language shopping request
func (p *Parser) ParseShoppingInput(text string) *ParsedShoppingInput {
	result := &ParsedShoppingInput{
		Items: []ParsedItem{},
		Tags:  []string{},
	}
	
	text = strings.ToLower(strings.TrimSpace(text))
	
	// Extract list name if present (e.g., "Create a list called Groceries")
	result.Name = p.extractListName(text)
	
	// Extract store name
	result.StoreName = p.extractStoreName(text)
	
	// Extract category hint
	result.Category = p.extractCategoryHint(text)
	
	// Parse individual items
	result.Items = p.ParseItems(text)
	
	return result
}

// ParseItems extracts individual items from text
func (p *Parser) ParseItems(text string) []ParsedItem {
	var items []ParsedItem
	
	// Split by common delimiters
	delimiters := []string{
		", and ",
		" and ",
		", ",
		"; ",
		"\n",
	}
	
	parts := []string{text}
	for _, delim := range delimiters {
		var newParts []string
		for _, part := range parts {
			split := strings.Split(part, delim)
			for _, s := range split {
				s = strings.TrimSpace(s)
				if s != "" {
					newParts = append(newParts, s)
				}
			}
		}
		parts = newParts
	}
	
	for _, part := range parts {
		if item := p.parseSingleItem(part); item.Name != "" {
			items = append(items, *item)
		}
	}
	
	return items
}

func (p *Parser) parseSingleItem(text string) *ParsedItem {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return &ParsedItem{}
	}
	
	item := &ParsedItem{}
	
	// Extract priority
	item.Priority = p.extractPriority(text)
	
	// Extract quantity and unit
	quantity, unit, remaining := p.extractQuantityAndUnit(text)
	item.Quantity = quantity
	item.Unit = unit
	
	// Extract price
	item.EstimatedPrice = p.extractPrice(text)
	
	// The rest is the item name
	item.Name = p.cleanItemName(remaining)
	
	// Auto-categorize
	item.Category = p.categorizeItem(item.Name)
	
	// Extract notes in parentheses
	item.Notes = p.extractNotes(text)
	
	return item
}

func (p *Parser) extractQuantityAndUnit(text string) (quantity, unit, remaining string) {
	// Try to match number + unit patterns
	patterns := []struct {
		regex string
		unit  string
	}{
		{`(\d+(?:\.\d+)?)\s*(?:gallon|gal)s?\b`, "gal"},
		{`(\d+(?:\.\d+)?)\s*(?:liter|litre|litre|liter)s?\b`, "L"},
		{`(\d+(?:\.\d+)?)\s*(?:pound|lb)s?\b`, "lb"},
		{`(\d+(?:\.\d+)?)\s*(?:ounce|oz)s?\b`, "oz"},
		{`(\d+(?:\.\d+)?)\s*(?:gram|g)s?\b`, "g"},
		{`(\d+(?:\.\d+)?)\s*(?:kilogram|kg)s?\b`, "kg"},
		{`(\d+(?:\.\d+)?)\s*(?:cup|tbsp|tsp)s?\b`, "cup"},
		{`(\d+(?:\.\d+)?)\s*pack\b`, "pack"},
		{`(\d+(?:\.\d+)?)\s*box\b`, "box"},
		{`(\d+(?:\.\d+)?)\s*bag\b`, "bag"},
		{`(\d+(?:\.\d+)?)\s*can\b`, "can"},
		{`(\d+(?:\.\d+)?)\s*bottle\b`, "bottle"},
		{`(\d+(?:\.\d+)?)\s*jar\b`, "jar"},
		{`(\d+(?:\.\d+)?)\s*loaf\b`, "loaf"},
		{`(\d+)\s*dozen\b`, "dozen"},
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern.regex)
		matches := re.FindStringSubmatchIndex(text)
		if len(matches) >= 4 {
			_ = text[matches[0]:matches[1]] // full match, not used
			numMatch := text[matches[2]:matches[3]]
			
			// Remove the matched part from text
			remaining = strings.TrimSpace(text[:matches[0]] + text[matches[1]:])
			
			return numMatch, pattern.unit, remaining
		}
	}
	
	// Try to match just a number at the beginning
	re := regexp.MustCompile(`^(\d+)\s+`)
	matches := re.FindStringSubmatchIndex(text)
	if len(matches) >= 4 {
		numMatch := text[matches[2]:matches[3]]
		remaining = text[matches[1]:]
		return numMatch, "", remaining
	}
	
	// Try to match "a " or "an " as quantity 1
	if strings.HasPrefix(text, "a ") || strings.HasPrefix(text, "an ") {
		remaining = text[2:]
		if strings.HasPrefix(text, "an ") {
			remaining = text[3:]
		}
		return "1", "", remaining
	}
	
	return "1", "", text
}

func (p *Parser) extractPriority(text string) string {
	for keyword, priority := range p.priorityKeywords {
		if strings.Contains(text, keyword) {
			return priority
		}
	}
	return "medium"
}

func (p *Parser) extractPrice(text string) float64 {
	// Match price patterns like $5.99, 5.99 dollars, etc.
	patterns := []string{
		`\$\s*(\d+(?:\.\d{2})?)`,
		`(\d+(?:\.\d{2})?)\s*dollars?`,
		`(\d+(?:\.\d{2})?)\s*usd`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) >= 2 {
			if price, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return price
			}
		}
	}
	
	return 0
}

func (p *Parser) extractListName(text string) string {
	patterns := []string{
		`(?:called|named)\s+["']?([^"',.]+)["']?`,
		`list\s+(?:for|of)\s+["']?([^"',.]+)["']?`,
		`create\s+(?:a\s+)?(?:shopping\s+)?list\s+(?:for\s+)?["']?([^"',.]+)["']?`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) >= 2 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

func (p *Parser) extractStoreName(text string) string {
	patterns := []string{
		`(?:at|from)\s+([A-Za-z\s]+(?:market|store|shop|mart|grocery|supermarket|whole\s+foods|target|walmart|costco|trader\s*joe|safeway))`,
		`for\s+([A-Za-z\s]+(?:market|store|shop|mart))`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) >= 2 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

func (p *Parser) extractCategoryHint(text string) string {
	// Check for explicit category mention
	for category := range p.categoryKeywords {
		pattern := `(?:for|category)\s+` + category
		re := regexp.MustCompile(`(?i)` + pattern)
		if re.MatchString(text) {
			return category
		}
	}
	return ""
}

func (p *Parser) extractNotes(text string) string {
	re := regexp.MustCompile(`\(([^)]+)\)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func (p *Parser) cleanItemName(text string) string {
	// Remove common filler words
	fillers := []string{
		"some ", "a few ", "few ", "couple of ", "bunch of ",
		"get ", "buy ", "need ", "pick up ", "grab ",
		"please ", "can you ", "i need ", "add ",
		"urgent ", "asap ", "important ", "needed ",
		`\$?\d+\.?\d*\s*dollars?`,
		`\$\d+\.?\d*`,
		`\$`,
		`\d+\.?\d*\s*dollars?`,
	}
	
	result := strings.ToLower(text)
	for _, filler := range fillers {
		re := regexp.MustCompile(`(?i)^` + filler)
		result = re.ReplaceAllString(result, "")
	}
	
	// Remove notes in parentheses
	re := regexp.MustCompile(`\s*\([^)]+\)`)
	result = re.ReplaceAllString(result, "")
	
	// Clean up
	result = strings.TrimSpace(result)
	
	// Capitalize first letter
	if len(result) > 0 {
		result = strings.ToUpper(result[:1]) + result[1:]
	}
	
	return result
}

func (p *Parser) categorizeItem(name string) string {
	nameLower := strings.ToLower(name)
	
	for category, keywords := range p.categoryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(nameLower, keyword) {
				return category
			}
		}
	}
	
	return "other"
}

// SuggestCategory suggests a category for an item based on name
func (p *Parser) SuggestCategory(itemName string) string {
	return p.categorizeItem(itemName)
}

// ExtractTags extracts hashtags or tags from text
func (p *Parser) ExtractTags(text string) []string {
	re := regexp.MustCompile(`#(\w+)`)
	matches := re.FindAllStringSubmatch(text, -1)
	
	var tags []string
	for _, match := range matches {
		if len(match) >= 2 {
			tags = append(tags, match[1])
		}
	}
	
	return tags
}

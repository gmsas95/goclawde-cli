package expenses

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ExpensesSkill provides expense tracking capabilities
type ExpensesSkill struct {
	*skills.BaseSkill
	store      *Store
	parser     *ExpenseParser
	logger     *zap.Logger
	httpClient *http.Client
}

// NewExpensesSkill creates a new expenses skill
func NewExpensesSkill(db *gorm.DB, logger *zap.Logger) (*ExpensesSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense store: %w", err)
	}

	skill := &ExpensesSkill{
		BaseSkill:  skills.NewBaseSkill("expenses", "Expense Tracking", "1.0.0"),
		store:      store,
		parser:     NewExpenseParser(),
		logger:     logger,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	skill.registerTools()
	return skill, nil
}

// registerTools registers all expense tracking tools
func (e *ExpensesSkill) registerTools() {
	tools := []skills.Tool{
		{
			Name:        "add_expense",
			Description: "Add an expense with natural language. Examples: 'Spent $45 at Whole Foods', 'Coffee $5.50', 'Gas $40 yesterday'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of the expense (what, where, how much)",
					},
					"amount": map[string]interface{}{
						"type":        "number",
						"description": "Amount spent (optional, will be extracted from description)",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category (optional, will be auto-detected)",
					},
					"date": map[string]interface{}{
						"type":        "string",
						"description": "Date (today, yesterday, or YYYY-MM-DD)",
					},
				},
				"required": []string{"description"},
			},
		},
		{
			Name:        "list_expenses",
			Description: "List your expenses for a period",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"period": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"today", "yesterday", "week", "month", "last_month", "all"},
						"default":     "month",
						"description": "Time period",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"default":     20,
						"description": "Maximum expenses to show",
					},
				},
			},
		},
		{
			Name:        "get_expense_summary",
			Description: "Get spending summary and insights for a period",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"period": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"today", "week", "month", "last_month", "year"},
						"default":     "month",
						"description": "Time period for summary",
					},
				},
			},
		},
		{
			Name:        "add_budget",
			Description: "Set a spending budget for a category",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category to budget (or 'overall' for total)",
					},
					"amount": map[string]interface{}{
						"type":        "number",
						"description": "Budget amount",
					},
					"period": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"daily", "weekly", "monthly", "yearly"},
						"default":     "monthly",
						"description": "Budget period",
					},
				},
				"required": []string{"category", "amount"},
			},
		},
		{
			Name:        "check_budget",
			Description: "Check your budget status and spending",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Specific category to check (optional)",
					},
				},
			},
		},
		{
			Name:        "delete_expense",
			Description: "Delete an expense",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expense_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the expense to delete",
					},
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"default":     false,
						"description": "Confirm deletion",
					},
				},
				"required": []string{"expense_id"},
			},
		},
		{
			Name:        "process_receipt",
			Description: "Process a receipt image to extract expense data",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"image_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to receipt image",
					},
					"ocr_text": map[string]interface{}{
						"type":        "string",
						"description": "OCR text from receipt (if already processed)",
					},
				},
				"required": []string{"image_path"},
			},
		},
	}

	for _, tool := range tools {
		tool.Handler = e.handleTool(tool.Name)
		e.AddTool(tool)
	}
}

// handleTool handles tool calls
func (e *ExpensesSkill) handleTool(name string) skills.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "add_expense":
			return e.handleAddExpense(ctx, args)
		case "list_expenses":
			return e.handleListExpenses(ctx, args)
		case "get_expense_summary":
			return e.handleGetSummary(ctx, args)
		case "add_budget":
			return e.handleAddBudget(ctx, args)
		case "check_budget":
			return e.handleCheckBudget(ctx, args)
		case "delete_expense":
			return e.handleDeleteExpense(ctx, args)
		case "process_receipt":
			return e.handleProcessReceipt(ctx, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	}
}

// handleAddExpense adds a new expense
func (e *ExpensesSkill) handleAddExpense(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	description, _ := args["description"].(string)

	if description == "" {
		return nil, fmt.Errorf("description is required")
	}

	// Parse the expense
	parseResult, err := e.parser.ParseExpense(description)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expense: %w", err)
	}

	// Allow overrides
	if amount, ok := args["amount"].(float64); ok && amount > 0 {
		parseResult.Amount = amount
	}
	if category, ok := args["category"].(string); ok && category != "" {
		parseResult.Category = category
	}
	if dateStr, ok := args["date"].(string); ok && dateStr != "" {
		if d := e.parseDate(dateStr); !d.IsZero() {
			parseResult.Date = d
		}
	}

	if parseResult.Amount <= 0 {
		return nil, fmt.Errorf("could not determine expense amount")
	}

	userID := e.getUserID(ctx)

	expense := &Expense{
		UserID:      userID,
		Amount:      parseResult.Amount,
		Currency:    parseResult.Currency,
		Description: parseResult.Description,
		Category:    parseResult.Category,
		Merchant:    parseResult.Merchant,
		Date:        parseResult.Date,
		Source:      "manual",
	}

	if err := e.store.CreateExpense(expense); err != nil {
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	e.logger.Info("Expense added",
		zap.Float64("amount", expense.Amount),
		zap.String("category", expense.Category),
	)

	return map[string]interface{}{
		"expense_id": expense.ID,
		"amount":     expense.FormatAmount(),
		"category":   expense.Category,
		"merchant":   expense.Merchant,
		"date":       expense.Date.Format("Jan 2, 2006"),
		"added":      true,
	}, nil
}

// handleListExpenses lists expenses
func (e *ExpensesSkill) handleListExpenses(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	period, _ := args["period"].(string)
	category, _ := args["category"].(string)
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	userID := e.getUserID(ctx)

	// Build filters
	filters := ExpenseFilters{
		Category: category,
		Limit:    limit,
	}

	// Set date range based on period
	now := time.Now()
	switch period {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		filters.StartDate = &start
		filters.EndDate = &now
	case "yesterday":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		start := end.AddDate(0, 0, -1)
		filters.StartDate = &start
		filters.EndDate = &end
	case "week":
		start := now.AddDate(0, 0, -7)
		filters.StartDate = &start
		filters.EndDate = &now
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		filters.StartDate = &start
		filters.EndDate = &now
	case "last_month":
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		start := end.AddDate(0, -1, 0)
		filters.StartDate = &start
		filters.EndDate = &end
	}

	list, err := e.store.ListExpenses(userID, filters)
	if err != nil {
		return nil, err
	}

	// Format expenses
	formatted := make([]map[string]interface{}, len(list.Expenses))
	for i, exp := range list.Expenses {
		formatted[i] = map[string]interface{}{
			"id":          exp.ID,
			"amount":      exp.FormatAmount(),
			"description": exp.Description,
			"category":    exp.Category,
			"merchant":    exp.Merchant,
			"date":        exp.Date.Format("Jan 2"),
		}
	}

	return map[string]interface{}{
		"expenses": formatted,
		"total":    list.Total,
		"period":   period,
	}, nil
}

// handleGetSummary gets expense summary
func (e *ExpensesSkill) handleGetSummary(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	period, _ := args["period"].(string)
	if period == "" {
		period = "month"
	}

	userID := e.getUserID(ctx)
	now := time.Now()

	var start, end time.Time
	switch period {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = now
	case "week":
		start = now.AddDate(0, 0, -7)
		end = now
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_month":
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		start = end.AddDate(0, -1, 0)
	case "year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = now
	}

	summary, err := e.store.GetSummary(userID, start, end)
	if err != nil {
		return nil, err
	}

	// Format category breakdown
	categories := []map[string]interface{}{}
	for cat, amount := range summary.ByCategory {
		categories = append(categories, map[string]interface{}{
			"category": cat,
			"amount":   fmt.Sprintf("%.2f", amount),
		})
	}

	return map[string]interface{}{
		"period":       summary.Period,
		"total_spent":  fmt.Sprintf("%.2f", summary.TotalSpent),
		"total_income": fmt.Sprintf("%.2f", summary.TotalIncome),
		"net":          fmt.Sprintf("%.2f", summary.NetAmount),
		"categories":   categories,
	}, nil
}

// handleAddBudget adds a budget
func (e *ExpensesSkill) handleAddBudget(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	category, _ := args["category"].(string)
	amount, _ := args["amount"].(float64)
	period, _ := args["period"].(string)

	if category == "" || amount <= 0 {
		return nil, fmt.Errorf("category and amount are required")
	}

	if period == "" {
		period = string(PeriodMonthly)
	}

	userID := e.getUserID(ctx)

	budget := &Budget{
		UserID:   userID,
		Name:     fmt.Sprintf("%s Budget", category),
		Category: category,
		Amount:   amount,
		Period:   BudgetPeriod(period),
	}

	if err := e.store.CreateBudget(budget); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"budget_id": budget.ID,
		"category":  category,
		"amount":    amount,
		"period":    period,
		"created":   true,
	}, nil
}

// handleCheckBudget checks budget status
func (e *ExpensesSkill) handleCheckBudget(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	category, _ := args["category"].(string)
	userID := e.getUserID(ctx)

	budgets, err := e.store.GetBudgets(userID, true)
	if err != nil {
		return nil, err
	}

	statuses := []map[string]interface{}{}
	for _, budget := range budgets {
		if category != "" && budget.Category != category {
			continue
		}

		status, err := e.store.GetBudgetStatus(&budget)
		if err != nil {
			continue
		}

		statuses = append(statuses, map[string]interface{}{
			"name":            status.Name,
			"category":        status.Category,
			"budget":          status.BudgetAmount,
			"spent":           status.SpentAmount,
			"remaining":       status.RemainingAmount,
			"percent_used":    fmt.Sprintf("%.1f%%", status.PercentUsed),
			"alert_triggered": status.AlertTriggered,
			"is_over_budget":  status.IsOverBudget,
		})
	}

	return map[string]interface{}{
		"budgets": statuses,
	}, nil
}

// handleDeleteExpense deletes an expense
func (e *ExpensesSkill) handleDeleteExpense(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	expenseID, _ := args["expense_id"].(string)
	confirm, _ := args["confirm"].(bool)

	if expenseID == "" {
		return nil, fmt.Errorf("expense_id is required")
	}

	if !confirm {
		return map[string]interface{}{
			"confirm_required": true,
			"message":          "Set confirm=true to delete this expense",
		}, nil
	}

	if err := e.store.DeleteExpense(expenseID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"expense_id": expenseID,
		"deleted":    true,
	}, nil
}

// handleProcessReceipt processes a receipt
func (e *ExpensesSkill) handleProcessReceipt(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	imagePath, _ := args["image_path"].(string)
	ocrText, _ := args["ocr_text"].(string)

	if imagePath == "" {
		return nil, fmt.Errorf("image_path is required")
	}

	// If OCR text not provided, extract it from the image
	if ocrText == "" {
		extractedText, err := e.extractTextFromImage(ctx, imagePath)
		if err != nil {
			e.logger.Warn("OCR extraction failed, falling back to manual input", zap.Error(err))
			return map[string]interface{}{
				"message": "Could not extract text from receipt automatically. Please provide the text manually via ocr_text parameter.",
				"image":   imagePath,
				"error":   err.Error(),
			}, nil
		}
		ocrText = extractedText
	}

	receipt, err := e.parser.ParseReceipt(ocrText)
	if err != nil {
		return nil, err
	}

	// Create expense from receipt
	userID := e.getUserID(ctx)
	expense := &Expense{
		UserID:      userID,
		Amount:      receipt.Total,
		Currency:    receipt.Currency,
		Description: fmt.Sprintf("Receipt from %s", receipt.Merchant),
		Category:    e.parser.inferCategory("", receipt.Merchant),
		Merchant:    receipt.Merchant,
		Date:        receipt.Date,
		HasReceipt:  true,
		Source:      "receipt",
	}

	if err := e.store.CreateExpense(expense); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"expense_id": expense.ID,
		"merchant":   receipt.Merchant,
		"total":      receipt.Total,
		"date":       receipt.Date.Format("Jan 2, 2006"),
		"items":      len(receipt.Items),
		"confidence": receipt.Confidence,
	}, nil
}

// Helper methods

func (e *ExpensesSkill) getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "default_user"
}

func (e *ExpensesSkill) parseDate(dateStr string) time.Time {
	switch strings.ToLower(dateStr) {
	case "today":
		return time.Now()
	case "yesterday":
		return time.Now().AddDate(0, 0, -1)
	default:
		if d, err := time.Parse("2006-01-02", dateStr); err == nil {
			return d
		}
	}
	return time.Time{}
}

// extractTextFromImage performs OCR on an image file
// Supports multiple OCR providers: OpenAI Vision, Google Vision API, or Tesseract (local)
func (e *ExpensesSkill) extractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	// Read image file
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	// Try OpenAI Vision API first (if available)
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		text, err := e.ocrWithOpenAI(ctx, imageData, apiKey)
		if err == nil && text != "" {
			return text, nil
		}
		e.logger.Warn("OpenAI OCR failed, trying fallback", zap.Error(err))
	}

	// Try Google Vision API
	if apiKey := os.Getenv("GOOGLE_VISION_API_KEY"); apiKey != "" {
		text, err := e.ocrWithGoogleVision(ctx, imageData, apiKey)
		if err == nil && text != "" {
			return text, nil
		}
		e.logger.Warn("Google Vision OCR failed, trying fallback", zap.Error(err))
	}

	// Fallback to Tesseract (local OCR)
	text, err := e.ocrWithTesseract(ctx, imagePath)
	if err == nil {
		return text, nil
	}

	return "", fmt.Errorf("all OCR methods failed: %w", err)
}

// ocrWithOpenAI uses OpenAI's GPT-4 Vision API for OCR
func (e *ExpensesSkill) ocrWithOpenAI(ctx context.Context, imageData []byte, apiKey string) (string, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := map[string]interface{}{
		"model": "gpt-4-vision-preview",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Extract all text from this receipt image. Return ONLY the raw text content, no formatting or commentary.",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url":    fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
							"detail": "high",
						},
					},
				},
			},
		},
		"max_tokens": 1000,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return result.Choices[0].Message.Content, nil
}

// ocrWithGoogleVision uses Google Cloud Vision API for OCR
func (e *ExpensesSkill) ocrWithGoogleVision(ctx context.Context, imageData []byte, apiKey string) (string, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"image": map[string]string{
					"content": base64Image,
				},
				"features": []map[string]interface{}{
					{
						"type": "TEXT_DETECTION",
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Responses []struct {
			FullTextAnnotation struct {
				Text string `json:"text"`
			} `json:"fullTextAnnotation"`
		} `json:"responses"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Responses) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return result.Responses[0].FullTextAnnotation.Text, nil
}

// ocrWithTesseract uses local Tesseract OCR
func (e *ExpensesSkill) ocrWithTesseract(ctx context.Context, imagePath string) (string, error) {
	// Check if tesseract is available
	_, err := exec.LookPath("tesseract")
	if err != nil {
		return "", fmt.Errorf("tesseract not installed")
	}

	// Create temp output file
	tempFile := imagePath + ".ocr"
	defer os.Remove(tempFile + ".txt") // Tesseract adds .txt extension

	// Run tesseract
	cmd := exec.CommandContext(ctx, "tesseract", imagePath, tempFile, "-l", "eng")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w (output: %s)", err, string(output))
	}

	// Read result
	resultBytes, err := os.ReadFile(tempFile + ".txt")
	if err != nil {
		return "", fmt.Errorf("failed to read OCR result: %w", err)
	}

	return string(resultBytes), nil
}

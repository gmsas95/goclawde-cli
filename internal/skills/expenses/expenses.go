package expenses

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ExpensesSkill provides expense tracking capabilities
type ExpensesSkill struct {
	*skills.BaseSkill
	store  *Store
	parser *ExpenseParser
	logger *zap.Logger
}

// NewExpensesSkill creates a new expenses skill
func NewExpensesSkill(db *gorm.DB, logger *zap.Logger) (*ExpensesSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense store: %w", err)
	}
	
	skill := &ExpensesSkill{
		BaseSkill: skills.NewBaseSkill("expenses", "Expense Tracking", "1.0.0"),
		store:     store,
		parser:    NewExpenseParser(),
		logger:    logger,
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
			"name":             status.Name,
			"category":         status.Category,
			"budget":           status.BudgetAmount,
			"spent":            status.SpentAmount,
			"remaining":        status.RemainingAmount,
			"percent_used":     fmt.Sprintf("%.1f%%", status.PercentUsed),
			"alert_triggered":  status.AlertTriggered,
			"is_over_budget":   status.IsOverBudget,
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
	
	// In production, this would run OCR on the image
	// For now, use provided OCR text or mock
	if ocrText == "" {
		return map[string]interface{}{
			"message": "Receipt processing requires OCR text. Please provide the text extracted from the receipt.",
			"image":   imagePath,
		}, nil
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

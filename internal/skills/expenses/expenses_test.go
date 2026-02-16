package expenses

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupExpenseTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func setupExpensesSkill(t *testing.T) (*ExpensesSkill, *gorm.DB) {
	db := setupExpenseTestDB(t)
	logger, _ := zap.NewDevelopment()
	
	skill, err := NewExpensesSkill(db, logger)
	require.NoError(t, err)
	
	return skill, db
}

// Store Tests

func TestStore_CreateAndGetExpense(t *testing.T) {
	db := setupExpenseTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	expense := &Expense{
		UserID:      "user1",
		Amount:      45.50,
		Currency:    "USD",
		Description: "Grocery shopping",
		Category:    string(CategoryGroceries),
		Merchant:    "Whole Foods",
		Date:        time.Now(),
	}
	
	err = store.CreateExpense(expense)
	require.NoError(t, err)
	assert.NotEmpty(t, expense.ID)
	
	// Retrieve
	retrieved, err := store.GetExpense(expense.ID)
	require.NoError(t, err)
	assert.Equal(t, 45.50, retrieved.Amount)
	assert.Equal(t, "Whole Foods", retrieved.Merchant)
}

func TestStore_ListExpenses(t *testing.T) {
	db := setupExpenseTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	now := time.Now()
	
	// Create expenses
	expenses := []*Expense{
		{UserID: "user1", Amount: 10, Category: string(CategoryFood), Date: now},
		{UserID: "user1", Amount: 20, Category: string(CategoryGroceries), Date: now},
		{UserID: "user1", Amount: 30, Category: string(CategoryTransport), Date: now.AddDate(0, 0, -10)},
	}
	
	for _, e := range expenses {
		err := store.CreateExpense(e)
		require.NoError(t, err)
	}
	
	// List all
	filters := ExpenseFilters{}
	list, err := store.ListExpenses("user1", filters)
	require.NoError(t, err)
	assert.Len(t, list.Expenses, 3)
	
	// Filter by category
	filters = ExpenseFilters{Category: string(CategoryFood)}
	list, err = store.ListExpenses("user1", filters)
	require.NoError(t, err)
	assert.Len(t, list.Expenses, 1)
}

func TestStore_CategoryTotals(t *testing.T) {
	db := setupExpenseTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	now := time.Now()
	
	// Create expenses
	expenses := []*Expense{
		{UserID: "user1", Amount: 10, Category: string(CategoryFood), Date: now},
		{UserID: "user1", Amount: 20, Category: string(CategoryFood), Date: now},
		{UserID: "user1", Amount: 30, Category: string(CategoryGroceries), Date: now},
	}
	
	for _, e := range expenses {
		err := store.CreateExpense(e)
		require.NoError(t, err)
	}
	
	// Get totals
	start := now.AddDate(0, 0, -1)
	end := now.AddDate(0, 0, 1)
	totals, err := store.GetCategoryTotals("user1", start, end)
	require.NoError(t, err)
	
	assert.Equal(t, 30.0, totals[string(CategoryFood)])
	assert.Equal(t, 30.0, totals[string(CategoryGroceries)])
}

func TestStore_BudgetOperations(t *testing.T) {
	db := setupExpenseTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create budget
	budget := &Budget{
		UserID:   "user1",
		Name:     "Food Budget",
		Category: string(CategoryFood),
		Amount:   500,
		Period:   PeriodMonthly,
	}
	
	err = store.CreateBudget(budget)
	require.NoError(t, err)
	assert.NotEmpty(t, budget.ID)
	
	// Create some expenses
	now := time.Now()
	expenses := []*Expense{
		{UserID: "user1", Amount: 50, Category: string(CategoryFood), Date: now},
		{UserID: "user1", Amount: 30, Category: string(CategoryFood), Date: now},
	}
	for _, e := range expenses {
		store.CreateExpense(e)
	}
	
	// Check status
	status, err := store.GetBudgetStatus(budget)
	require.NoError(t, err)
	
	assert.Equal(t, 500.0, status.BudgetAmount)
	assert.Equal(t, 80.0, status.SpentAmount)
	assert.Equal(t, 420.0, status.RemainingAmount)
	assert.Equal(t, 16.0, status.PercentUsed)
}

// Parser Tests

func TestExpenseParser_ParseExpense(t *testing.T) {
	parser := NewExpenseParser()
	
	tests := []struct {
		input          string
		expectedAmount float64
		validCategories []string
	}{
		{"Spent $45 at Whole Foods", 45, []string{string(CategoryGroceries), string(CategoryFood)}},
		{"Coffee $5.50 at Starbucks", 5.50, []string{string(CategoryFood)}},
		{"Gas $40 at Shell", 40, []string{string(CategoryTransport)}},
		{"Lunch $15", 15, []string{string(CategoryFood)}},
	}
	
	for _, test := range tests {
		result, err := parser.ParseExpense(test.input)
		require.NoError(t, err)
		assert.InDelta(t, test.expectedAmount, result.Amount, 0.01, "Amount for: %s", test.input)
		assert.Contains(t, test.validCategories, result.Category, "Category for: %s", test.input)
	}
}

func TestExpenseParser_ExtractAmount(t *testing.T) {
	parser := NewExpenseParser()
	
	tests := []struct {
		input            string
		expectedAmount   float64
		expectedCurrency string
	}{
		{"$45.50", 45.50, "USD"},
		{"€30", 30, "EUR"},
		{"£25.99", 25.99, "GBP"},
		{"100 USD", 100, "USD"},
		{"no amount here", 0, "USD"},
	}
	
	for _, test := range tests {
		amount, currency := parser.extractAmount(test.input)
		assert.InDelta(t, test.expectedAmount, amount, 0.01, "Input: %s", test.input)
		assert.Equal(t, test.expectedCurrency, currency, "Input: %s", test.input)
	}
}

func TestExpenseParser_InferCategory(t *testing.T) {
	parser := NewExpenseParser()
	
	tests := []struct {
		text            string
		merchant        string
		validCategories []string
	}{
		{"lunch at restaurant", "", []string{string(CategoryFood)}},
		{"", "Whole Foods", []string{string(CategoryGroceries), string(CategoryFood)}},
		{"gas station", "", []string{string(CategoryTransport)}},
		{"", "Uber", []string{string(CategoryTransport)}},
		{"movie tickets", "", []string{string(CategoryEntertainment)}},
	}
	
	for _, test := range tests {
		category := parser.inferCategory(test.text, test.merchant)
		assert.Contains(t, test.validCategories, category, "Text: %s, Merchant: %s", test.text, test.merchant)
	}
}

// Expense Skill Tests

func TestExpensesSkill_AddExpense(t *testing.T) {
	skill, _ := setupExpensesSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	result, err := skill.handleAddExpense(ctx, map[string]interface{}{
		"description": "Spent $45 at Whole Foods for groceries",
	})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.NotEmpty(t, resultMap["expense_id"])
	assert.Equal(t, true, resultMap["added"])
	// Category can be groceries or food depending on parser
	assert.NotEmpty(t, resultMap["category"])
}

func TestExpensesSkill_ListExpenses(t *testing.T) {
	skill, _ := setupExpensesSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add expenses
	skill.handleAddExpense(ctx, map[string]interface{}{
		"description": "Coffee $5",
	})
	skill.handleAddExpense(ctx, map[string]interface{}{
		"description": "Lunch $15",
	})
	
	// List
	result, err := skill.handleListExpenses(ctx, map[string]interface{}{
		"period": "month",
		"limit":  float64(10),
	})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	expenses := resultMap["expenses"].([]map[string]interface{})
	assert.GreaterOrEqual(t, len(expenses), 2)
}

func TestExpensesSkill_GetSummary(t *testing.T) {
	skill, _ := setupExpensesSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add expenses
	skill.handleAddExpense(ctx, map[string]interface{}{
		"description": "Groceries $100",
	})
	skill.handleAddExpense(ctx, map[string]interface{}{
		"description": "Gas $50",
	})
	
	result, err := skill.handleGetSummary(ctx, map[string]interface{}{
		"period": "month",
	})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.NotNil(t, resultMap["total_spent"])
	assert.NotNil(t, resultMap["categories"])
}

func TestExpensesSkill_AddAndCheckBudget(t *testing.T) {
	skill, _ := setupExpensesSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add budget
	result, err := skill.handleAddBudget(ctx, map[string]interface{}{
		"category": "food",
		"amount":   float64(500),
		"period":   "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, true, result.(map[string]interface{})["created"])
	
	// Add expenses
	skill.handleAddExpense(ctx, map[string]interface{}{
		"description": "Lunch $30",
		"category":    "food",
	})
	
	// Check budget
	result, err = skill.handleCheckBudget(ctx, map[string]interface{}{
		"category": "food",
	})
	require.NoError(t, err)
	
	budgets := result.(map[string]interface{})["budgets"].([]map[string]interface{})
	assert.GreaterOrEqual(t, len(budgets), 1)
}

// Expense Helper Tests

func TestExpense_IsIncome(t *testing.T) {
	tests := []struct {
		amount   float64
		category string
		expected bool
	}{
		{100, string(CategoryFood), false},
		{-100, string(CategoryIncome), true},
		{100, string(CategoryIncome), true},
	}
	
	for _, test := range tests {
		e := &Expense{Amount: test.amount, Category: test.category}
		assert.Equal(t, test.expected, e.IsIncome())
	}
}

func TestExpense_FormatAmount(t *testing.T) {
	tests := []struct {
		amount   float64
		expected string
	}{
		{45.50, "45.50"},
		{-100, "+100.00"},
		{0, "0.00"},
	}
	
	for _, test := range tests {
		e := &Expense{Amount: test.amount}
		assert.Equal(t, test.expected, e.FormatAmount())
	}
}

func TestExpense_Tags(t *testing.T) {
	e := &Expense{}
	
	// Add tags
	e.AddTag("urgent")
	e.AddTag("reimbursable")
	
	tags := e.GetTags()
	assert.Len(t, tags, 2)
	assert.Contains(t, tags, "urgent")
	assert.Contains(t, tags, "reimbursable")
}

func TestBudget_GetPeriodRange(t *testing.T) {
	tests := []struct {
		period BudgetPeriod
	}{
		{PeriodDaily},
		{PeriodWeekly},
		{PeriodMonthly},
		{PeriodYearly},
	}
	
	for _, test := range tests {
		budget := &Budget{Period: test.period}
		start, end := budget.GetPeriodRange()
		
		assert.False(t, start.IsZero())
		assert.False(t, end.IsZero())
		assert.True(t, end.After(start) || end.Equal(start))
	}
}

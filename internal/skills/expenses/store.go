package expenses

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles expense persistence
type Store struct {
	db *gorm.DB
}

// NewStore creates a new expense store
func NewStore(db *gorm.DB) (*Store, error) {
	store := &Store{db: db}
	
	// Auto-migrate schemas
	if err := db.AutoMigrate(&Expense{}, &Budget{}); err != nil {
		return nil, fmt.Errorf("failed to migrate expense schemas: %w", err)
	}
	
	// Create indexes
	store.createIndexes()
	
	return store, nil
}

// createIndexes creates database indexes
func (s *Store) createIndexes() {
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_expenses_user_date ON expenses(user_id, date)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_expenses_user_category ON expenses(user_id, category)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_budgets_user ON budgets(user_id)")
}

// generateID generates a unique ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "exp_" + hex.EncodeToString(bytes)
}

// Expense operations

// CreateExpense creates a new expense
func (s *Store) CreateExpense(expense *Expense) error {
	if expense.ID == "" {
		expense.ID = generateID()
	}
	if expense.Currency == "" {
		expense.Currency = "USD"
	}
	if expense.Date.IsZero() {
		expense.Date = time.Now()
	}
	expense.CreatedAt = time.Now()
	expense.UpdatedAt = time.Now()
	
	return s.db.Create(expense).Error
}

// GetExpense retrieves an expense by ID
func (s *Store) GetExpense(expenseID string) (*Expense, error) {
	var expense Expense
	err := s.db.Where("id = ?", expenseID).First(&expense).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &expense, err
}

// UpdateExpense updates an expense
func (s *Store) UpdateExpense(expense *Expense) error {
	expense.UpdatedAt = time.Now()
	return s.db.Save(expense).Error
}

// DeleteExpense deletes an expense
func (s *Store) DeleteExpense(expenseID string) error {
	return s.db.Where("id = ?", expenseID).Delete(&Expense{}).Error
}

// ListExpenses lists expenses with filters
func (s *Store) ListExpenses(userID string, filters ExpenseFilters) (*ExpenseList, error) {
	query := s.db.Where("user_id = ?", userID)
	
	// Apply filters
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.StartDate != nil {
		query = query.Where("date >= ?", filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("date <= ?", filters.EndDate)
	}
	if filters.Merchant != "" {
		query = query.Where("merchant LIKE ?", "%"+filters.Merchant+"%")
	}
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where(
			"description LIKE ? OR merchant LIKE ? OR notes LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}
	if len(filters.Tags) > 0 {
		for _, tag := range filters.Tags {
			query = query.Where("tags LIKE ?", "%"+tag+"%")
		}
	}
	if filters.MinAmount > 0 {
		query = query.Where("amount >= ?", filters.MinAmount)
	}
	if filters.MaxAmount > 0 {
		query = query.Where("amount <= ?", filters.MaxAmount)
	}
	
	// Order by date desc
	query = query.Order("date DESC, created_at DESC")
	
	// Pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}
	
	var expenses []Expense
	if err := query.Find(&expenses).Error; err != nil {
		return nil, err
	}
	
	// Calculate total amount
	var totalAmount float64
	s.db.Model(&Expense{}).Where("user_id = ?", userID).Select("COALESCE(SUM(amount), 0)").Scan(&totalAmount)
	
	return &ExpenseList{
		Expenses:    expenses,
		Total:       len(expenses),
		TotalAmount: totalAmount,
	}, nil
}

// ExpenseFilters contains filters for listing expenses
type ExpenseFilters struct {
	Category  string
	StartDate *time.Time
	EndDate   *time.Time
	Merchant  string
	Search    string
	Tags      []string
	MinAmount float64
	MaxAmount float64
	Limit     int
	Offset    int
}

// GetExpensesByDateRange gets expenses within a date range
func (s *Store) GetExpensesByDateRange(userID string, start, end time.Time) ([]Expense, error) {
	var expenses []Expense
	err := s.db.Where(
		"user_id = ? AND date >= ? AND date <= ?",
		userID, start, end,
	).Order("date DESC").Find(&expenses).Error
	return expenses, err
}

// GetCategoryTotals gets total spending by category for a period
func (s *Store) GetCategoryTotals(userID string, start, end time.Time) (map[string]float64, error) {
	var results []struct {
		Category string
		Total    float64
	}
	
	err := s.db.Model(&Expense{}).
		Select("category, COALESCE(SUM(amount), 0) as total").
		Where("user_id = ? AND date >= ? AND date <= ? AND amount > 0", userID, start, end).
		Group("category").
		Scan(&results).Error
	
	if err != nil {
		return nil, err
	}
	
	totals := make(map[string]float64)
	for _, r := range results {
		totals[r.Category] = r.Total
	}
	
	return totals, nil
}

// Budget operations

// CreateBudget creates a new budget
func (s *Store) CreateBudget(budget *Budget) error {
	if budget.ID == "" {
		budget.ID = generateID()
	}
	if budget.Currency == "" {
		budget.Currency = "USD"
	}
	if budget.Period == "" {
		budget.Period = PeriodMonthly
	}
	budget.CreatedAt = time.Now()
	budget.UpdatedAt = time.Now()
	
	return s.db.Create(budget).Error
}

// GetBudget retrieves a budget by ID
func (s *Store) GetBudget(budgetID string) (*Budget, error) {
	var budget Budget
	err := s.db.Where("id = ?", budgetID).First(&budget).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &budget, err
}

// GetBudgets gets all budgets for a user
func (s *Store) GetBudgets(userID string, activeOnly bool) ([]Budget, error) {
	query := s.db.Where("user_id = ?", userID)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	
	var budgets []Budget
	err := query.Order("category ASC").Find(&budgets).Error
	return budgets, err
}

// UpdateBudget updates a budget
func (s *Store) UpdateBudget(budget *Budget) error {
	budget.UpdatedAt = time.Now()
	return s.db.Save(budget).Error
}

// DeleteBudget deletes a budget
func (s *Store) DeleteBudget(budgetID string) error {
	return s.db.Where("id = ?", budgetID).Delete(&Budget{}).Error
}

// GetBudgetStatus calculates the current status of a budget
func (s *Store) GetBudgetStatus(budget *Budget) (*BudgetStatus, error) {
	start, end := budget.GetPeriodRange()
	
	var spent float64
	err := s.db.Model(&Expense{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND date >= ? AND date <= ? AND amount > 0", budget.UserID, start, end).
		Scan(&spent).Error
	
	if err != nil {
		return nil, err
	}
	
	// Filter by category if specified
	if budget.Category != "" {
		err = s.db.Model(&Expense{}).
			Select("COALESCE(SUM(amount), 0)").
			Where("user_id = ? AND category = ? AND date >= ? AND date <= ? AND amount > 0",
				budget.UserID, budget.Category, start, end).
			Scan(&spent).Error
		if err != nil {
			return nil, err
		}
	}
	
	remaining := budget.Amount - spent
	percentUsed := 0.0
	if budget.Amount > 0 {
		percentUsed = (spent / budget.Amount) * 100
	}
	
	daysRemaining := int(end.Sub(time.Now()).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	
	return &BudgetStatus{
		BudgetID:        budget.ID,
		Name:            budget.Name,
		Category:        budget.Category,
		BudgetAmount:    budget.Amount,
		SpentAmount:     spent,
		RemainingAmount: remaining,
		PercentUsed:     percentUsed,
		DaysRemaining:   daysRemaining,
		IsOverBudget:    spent > budget.Amount,
		AlertTriggered:  budget.AlertEnabled && percentUsed >= (budget.AlertThreshold*100),
	}, nil
}

// GetAllBudgetStatuses gets status for all active budgets
func (s *Store) GetAllBudgetStatuses(userID string) ([]BudgetStatus, error) {
	budgets, err := s.GetBudgets(userID, true)
	if err != nil {
		return nil, err
	}
	
	statuses := make([]BudgetStatus, len(budgets))
	for i, budget := range budgets {
		status, err := s.GetBudgetStatus(&budget)
		if err != nil {
			continue
		}
		statuses[i] = *status
	}
	
	return statuses, nil
}

// Statistics

// GetSummary gets expense summary for a period
func (s *Store) GetSummary(userID string, start, end time.Time) (*ExpenseSummary, error) {
	summary := &ExpenseSummary{
		Period:     fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		StartDate:  start,
		EndDate:    end,
		ByCategory: make(map[string]float64),
		ByDay:      make(map[string]float64),
	}
	
	// Get all expenses in range
	expenses, err := s.GetExpensesByDateRange(userID, start, end)
	if err != nil {
		return nil, err
	}
	
	// Calculate totals
	for _, e := range expenses {
		if e.Amount > 0 {
			summary.TotalSpent += e.Amount
			summary.ByCategory[e.Category] += e.Amount
			day := e.Date.Format("2006-01-02")
			summary.ByDay[day] += e.Amount
		} else {
			summary.TotalIncome += -e.Amount
		}
	}
	
	summary.NetAmount = summary.TotalIncome - summary.TotalSpent
	
	// Get top 5 expenses
	if len(expenses) > 0 {
		topExpenses := make([]Expense, 0, 5)
		for i, e := range expenses {
			if i >= 5 {
				break
			}
			topExpenses = append(topExpenses, e)
		}
		summary.TopExpenses = topExpenses
	}
	
	// Get budget statuses
	budgets, err := s.GetAllBudgetStatuses(userID)
	if err == nil {
		summary.Budgets = budgets
	}
	
	return summary, nil
}

// GetMonthlySummary gets summary for a specific month
func (s *Store) GetMonthlySummary(userID string, year, month int) (*ExpenseSummary, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	return s.GetSummary(userID, start, end)
}

// UpsertExpense creates or updates an expense
func (s *Store) UpsertExpense(expense *Expense) error {
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(expense).Error
}

package expenses

import (
	"fmt"
	"time"
)

// Expense represents a single expense transaction
type Expense struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Transaction details
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency" gorm:"default:USD"`
	Description string    `json:"description"`
	Category    string    `json:"category" gorm:"index"`
	SubCategory string    `json:"sub_category,omitempty"`
	
	// Receipt/OCR
	HasReceipt      bool   `json:"has_receipt"`
	ReceiptImageID  string `json:"receipt_image_id,omitempty"`
	Merchant        string `json:"merchant,omitempty"`
	MerchantAddress string `json:"merchant_address,omitempty"`
	
	// Timing
	Date        time.Time `json:"date" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Source
	Source      string    `json:"source"` // manual, receipt, voice, import
	SourceID    string    `json:"source_id,omitempty"`
	
	// Location (optional)
	Location    string    `json:"location,omitempty"`
	Latitude    *float64  `json:"latitude,omitempty"`
	Longitude   *float64  `json:"longitude,omitempty"`
	
	// Tags
	Tags        string    `json:"tags"` // Comma-separated
	
	// Payment method
	PaymentMethod string `json:"payment_method,omitempty"` // cash, credit, debit, etc.
	
	// Notes
	Notes       string    `json:"notes,omitempty"`
}

// ExpenseCategory represents expense categories
type ExpenseCategory string

const (
	CategoryFood        ExpenseCategory = "food"
	CategoryGroceries   ExpenseCategory = "groceries"
	CategoryTransport   ExpenseCategory = "transport"
	CategoryHousing     ExpenseCategory = "housing"
	CategoryUtilities   ExpenseCategory = "utilities"
	CategoryHealth      ExpenseCategory = "health"
	CategoryEntertainment ExpenseCategory = "entertainment"
	CategoryShopping    ExpenseCategory = "shopping"
	CategoryPersonal    ExpenseCategory = "personal"
	CategoryEducation   ExpenseCategory = "education"
	CategoryTravel      ExpenseCategory = "travel"
	CategoryBills       ExpenseCategory = "bills"
	CategoryIncome      ExpenseCategory = "income"
	CategoryOther       ExpenseCategory = "other"
)

// Budget represents a spending budget
type Budget struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Budget details
	Name        string    `json:"name"`
	Category    string    `json:"category"` // empty for overall budget
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency" gorm:"default:USD"`
	
	// Period
	Period      BudgetPeriod `json:"period"` // daily, weekly, monthly, yearly
	StartDate   time.Time    `json:"start_date"`
	EndDate     *time.Time   `json:"end_date,omitempty"`
	
	// Alerts
	AlertThreshold float64 `json:"alert_threshold" gorm:"default:0.8"` // Alert at 80%
	AlertEnabled   bool    `json:"alert_enabled" gorm:"default:true"`
	
	// Status
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	
	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BudgetPeriod represents budget period
type BudgetPeriod string

const (
	PeriodDaily   BudgetPeriod = "daily"
	PeriodWeekly  BudgetPeriod = "weekly"
	PeriodMonthly BudgetPeriod = "monthly"
	PeriodYearly  BudgetPeriod = "yearly"
)

// BudgetStatus represents current budget status
type BudgetStatus struct {
	BudgetID        string    `json:"budget_id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	BudgetAmount    float64   `json:"budget_amount"`
	SpentAmount     float64   `json:"spent_amount"`
	RemainingAmount float64   `json:"remaining_amount"`
	PercentUsed     float64   `json:"percent_used"`
	DaysRemaining   int       `json:"days_remaining"`
	IsOverBudget    bool      `json:"is_over_budget"`
	AlertTriggered  bool      `json:"alert_triggered"`
}

// ExpenseSummary represents expense summary for a period
type ExpenseSummary struct {
	Period      string                 `json:"period"`
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	TotalSpent  float64                `json:"total_spent"`
	TotalIncome float64                `json:"total_income"`
	NetAmount   float64                `json:"net_amount"`
	Currency    string                 `json:"currency"`
	ByCategory  map[string]float64     `json:"by_category"`
	ByDay       map[string]float64     `json:"by_day"`
	TopExpenses []Expense              `json:"top_expenses"`
	Budgets     []BudgetStatus         `json:"budgets"`
}

// ReceiptData represents extracted receipt data
type ReceiptData struct {
	Merchant        string    `json:"merchant"`
	Date            time.Time `json:"date"`
	Total           float64   `json:"total"`
	Subtotal        float64   `json:"subtotal"`
	Tax             float64   `json:"tax"`
	Tip             float64   `json:"tip"`
	Currency        string    `json:"currency"`
	Items           []ReceiptItem `json:"items"`
	PaymentMethod   string    `json:"payment_method"`
	Address         string    `json:"address"`
	Confidence      float64   `json:"confidence"`
}

// ReceiptItem represents an item on a receipt
type ReceiptItem struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Total    float64 `json:"total"`
	Category string  `json:"category,omitempty"`
}

// ExpenseList represents a list of expenses
type ExpenseList struct {
	Expenses    []Expense `json:"expenses"`
	Total       int       `json:"total"`
	TotalAmount float64   `json:"total_amount"`
}

// ParseResult represents the result of parsing an expense
type ParseResult struct {
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Merchant    string    `json:"merchant"`
	Date        time.Time `json:"date"`
	Confidence  float64   `json:"confidence"`
}

// CategoryStats represents statistics for a category
type CategoryStats struct {
	Category      string  `json:"category"`
	TotalSpent    float64 `json:"total_spent"`
	TransactionCount int  `json:"transaction_count"`
	AverageAmount float64 `json:"average_amount"`
	PercentOfTotal float64 `json:"percent_of_total"`
}

// Helper methods

// IsIncome returns true if this is an income transaction
func (e *Expense) IsIncome() bool {
	return e.Category == string(CategoryIncome) || e.Amount < 0
}

// AbsoluteAmount returns the absolute value of the amount
func (e *Expense) AbsoluteAmount() float64 {
	if e.Amount < 0 {
		return -e.Amount
	}
	return e.Amount
}

// FormatAmount formats the amount with currency
func (e *Expense) FormatAmount() string {
	if e.Amount < 0 {
		// Income shown as positive
		return fmt.Sprintf("+%.2f", -e.Amount)
	}
	return fmt.Sprintf("%.2f", e.Amount)
}

// GetTags returns the list of tags
func (e *Expense) GetTags() []string {
	if e.Tags == "" {
		return []string{}
	}
	return splitAndTrim(e.Tags, ",")
}

// AddTag adds a tag to the expense
func (e *Expense) AddTag(tag string) {
	if e.Tags == "" {
		e.Tags = tag
	} else {
		e.Tags = e.Tags + "," + tag
	}
}

// IsInCurrentPeriod checks if the expense is in the budget's current period
func (b *Budget) IsInCurrentPeriod(t time.Time) bool {
	now := time.Now()
	
	switch b.Period {
	case PeriodDaily:
		return t.Year() == now.Year() && t.YearDay() == now.YearDay()
	case PeriodWeekly:
		_, week1 := t.ISOWeek()
		_, week2 := now.ISOWeek()
		return t.Year() == now.Year() && week1 == week2
	case PeriodMonthly:
		return t.Year() == now.Year() && t.Month() == now.Month()
	case PeriodYearly:
		return t.Year() == now.Year()
	}
	return false
}

// GetPeriodRange returns the start and end dates for the current period
func (b *Budget) GetPeriodRange() (start, end time.Time) {
	now := time.Now()
	
	switch b.Period {
	case PeriodDaily:
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1).Add(-time.Second)
	case PeriodWeekly:
		weekday := int(now.Weekday())
		start = time.Date(now.Year(), now.Month(), now.Day()-weekday, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 7).Add(-time.Second)
	case PeriodMonthly:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0).Add(-time.Second)
	case PeriodYearly:
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(1, 0, 0).Add(-time.Second)
	}
	
	return
}

// Helper functions
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	parts := []string{}
	for _, p := range splitString(s, sep) {
		trimmed := trimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i < len(s)-len(sep)+1 && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

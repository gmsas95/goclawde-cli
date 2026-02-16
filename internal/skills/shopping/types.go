package shopping

import (
	"time"
)

// ShoppingList represents a shopping list
type ShoppingList struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// List details
	Name        string    `json:"name"`
	Description string    `json:"description"`
	
	// Status
	IsActive    bool      `json:"is_active"`
	IsTemplate  bool      `json:"is_template" gorm:"default:false"`
	
	// Location-based reminders
	LocationName string   `json:"location_name,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	RadiusMeters int      `json:"radius_meters,omitempty"` // Geofence radius
	
	// Schedule
	DueDate     *time.Time `json:"due_date,omitempty"`
	RemindAt    *time.Time `json:"remind_at,omitempty"`
	
	// Recurrence for recurring lists (e.g., weekly grocery run)
	IsRecurring    bool   `json:"is_recurring" gorm:"default:false"`
	RecurrenceRule string `json:"recurrence_rule,omitempty"`
	
	// Metadata
	Category    string    `json:"category,omitempty"` // groceries, household, personal, etc.
	StoreName   string    `json:"store_name,omitempty"`
	Tags        string    `json:"tags"` // Comma-separated
	
	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ShoppingItem represents an item in a shopping list
type ShoppingItem struct {
	ID       string `json:"id" gorm:"primaryKey"`
	ListID   string `json:"list_id" gorm:"index"`
	UserID   string `json:"user_id" gorm:"index"`
	
	// Item details
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Quantity    string  `json:"quantity"`
	Unit        string  `json:"unit,omitempty"` // pcs, kg, lbs, oz, etc.
	
	// Status
	IsChecked   bool       `json:"is_checked" gorm:"default:false"`
	CheckedAt   *time.Time `json:"checked_at,omitempty"`
	
	// Category/location in store
	Category    string `json:"category,omitempty"` // produce, dairy, meat, etc.
	StoreAisle  string `json:"store_aisle,omitempty"`
	
	// Pricing
	EstimatedPrice float64 `json:"estimated_price,omitempty"`
	ActualPrice    float64 `json:"actual_price,omitempty"`
	Currency       string  `json:"currency,omitempty"`
	
	// Priority
	Priority    string `json:"priority"` // low, medium, high
	
	// Barcode/SKU
	Barcode     string `json:"barcode,omitempty"`
	SKU         string `json:"sku,omitempty"`
	
	// Source
	AddedFrom   string `json:"added_from,omitempty"` // voice, text, scan, recipe
	
	// Notes
	Notes       string `json:"notes,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListWithItems includes list and its items
type ListWithItems struct {
	List  ShoppingList   `json:"list"`
	Items []ShoppingItem `json:"items"`
}

// ShoppingStats represents shopping statistics
type ShoppingStats struct {
	TotalLists        int                `json:"total_lists"`
	ActiveLists       int                `json:"active_lists"`
	CompletedLists    int                `json:"completed_lists"`
	TotalItems        int                `json:"total_items"`
	CheckedItems      int                `json:"checked_items"`
	UncheckedItems    int                `json:"unchecked_items"`
	CompletionRate    float64            `json:"completion_rate"`
	ByCategory        map[string]int     `json:"by_category"`
	EstimatedTotal    float64            `json:"estimated_total"`
	Currency          string             `json:"currency"`
}

// StoreLocation represents a store location for geofencing
type StoreLocation struct {
	ID       string  `json:"id" gorm:"primaryKey"`
	UserID   string  `json:"user_id" gorm:"index"`
	Name     string  `json:"name"`
	Address  string  `json:"address,omitempty"`
	Latitude float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius   int     `json:"radius" gorm:"default:100"` // meters
}

// Helper methods

// IsOverdue returns true if the list is overdue
func (l *ShoppingList) IsOverdue() bool {
	if l.DueDate == nil || l.CompletedAt != nil {
		return false
	}
	return time.Now().After(*l.DueDate)
}

// CompletionPercentage returns completion percentage
func (l *ShoppingList) CompletionPercentage(items []ShoppingItem) float64 {
	if len(items) == 0 {
		return 0
	}
	checked := 0
	for _, item := range items {
		if item.IsChecked {
			checked++
		}
	}
	return float64(checked) / float64(len(items)) * 100
}

// TotalChecked returns number of checked items
func (l *ShoppingList) TotalChecked(items []ShoppingItem) int {
	count := 0
	for _, item := range items {
		if item.IsChecked {
			count++
		}
	}
	return count
}

// FormatQuantity formats item quantity with unit
func (i *ShoppingItem) FormatQuantity() string {
	if i.Unit == "" {
		return i.Quantity
	}
	return i.Quantity + " " + i.Unit
}

// ToggleChecked toggles the checked status
func (i *ShoppingItem) ToggleChecked() {
	i.IsChecked = !i.IsChecked
	if i.IsChecked {
		now := time.Now()
		i.CheckedAt = &now
	} else {
		i.CheckedAt = nil
	}
}



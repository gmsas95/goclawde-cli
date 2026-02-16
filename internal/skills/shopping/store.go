package shopping

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Store handles shopping list persistence
type Store struct {
	db *gorm.DB
}

// NewStore creates a new shopping store
func NewStore(db *gorm.DB) (*Store, error) {
	store := &Store{db: db}
	
	if err := db.AutoMigrate(&ShoppingList{}, &ShoppingItem{}, &StoreLocation{}); err != nil {
		return nil, fmt.Errorf("failed to migrate shopping schemas: %w", err)
	}
	
	return store, nil
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "shop_" + hex.EncodeToString(bytes)
}

// List operations

func (s *Store) CreateList(list *ShoppingList) error {
	if list.ID == "" {
		list.ID = generateID()
	}
	// Default IsActive to true if not explicitly set
	// Note: There's no way to distinguish between "explicitly set to false" and "not set"
	// So we always set it to true unless the CreatedAt is already set (indicating it's being re-saved)
	if list.CreatedAt.IsZero() {
		list.IsActive = true
	}
	list.CreatedAt = time.Now()
	list.UpdatedAt = time.Now()
	return s.db.Create(list).Error
}

func (s *Store) GetList(listID string) (*ShoppingList, error) {
	var list ShoppingList
	err := s.db.Where("id = ?", listID).First(&list).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &list, err
}

func (s *Store) UpdateList(list *ShoppingList) error {
	list.UpdatedAt = time.Now()
	return s.db.Save(list).Error
}

func (s *Store) DeleteList(listID string) error {
	// Delete items first
	s.db.Where("list_id = ?", listID).Delete(&ShoppingItem{})
	return s.db.Where("id = ?", listID).Delete(&ShoppingList{}).Error
}

func (s *Store) ListLists(userID string, activeOnly bool) ([]ShoppingList, error) {
	query := s.db.Where("user_id = ?", userID)
	if activeOnly {
		query = query.Where("is_active = ? AND completed_at IS NULL", true)
	}
	
	var lists []ShoppingList
	err := query.Order("created_at DESC").Find(&lists).Error
	return lists, err
}

// Item operations

func (s *Store) CreateItem(item *ShoppingItem) error {
	if item.ID == "" {
		item.ID = generateID()
	}
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	return s.db.Create(item).Error
}

func (s *Store) GetItem(itemID string) (*ShoppingItem, error) {
	var item ShoppingItem
	err := s.db.Where("id = ?", itemID).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &item, err
}

func (s *Store) UpdateItem(item *ShoppingItem) error {
	item.UpdatedAt = time.Now()
	return s.db.Save(item).Error
}

func (s *Store) DeleteItem(itemID string) error {
	return s.db.Where("id = ?", itemID).Delete(&ShoppingItem{}).Error
}

func (s *Store) GetItemsByList(listID string) ([]ShoppingItem, error) {
	var items []ShoppingItem
	err := s.db.Where("list_id = ?", listID).Order("category ASC, created_at ASC").Find(&items).Error
	return items, err
}

func (s *Store) GetUncheckedItems(userID string) ([]ShoppingItem, error) {
	var items []ShoppingItem
	err := s.db.Where("user_id = ? AND is_checked = ?", userID, false).
		Order("created_at DESC").Find(&items).Error
	return items, err
}

func (s *Store) CheckItem(itemID string) error {
	now := time.Now()
	return s.db.Model(&ShoppingItem{}).Where("id = ?", itemID).Updates(map[string]interface{}{
		"is_checked": true,
		"checked_at": &now,
		"updated_at": now,
	}).Error
}

func (s *Store) UncheckItem(itemID string) error {
	return s.db.Model(&ShoppingItem{}).Where("id = ?", itemID).Updates(map[string]interface{}{
		"is_checked": false,
		"checked_at": nil,
		"updated_at": time.Now(),
	}).Error
}

func (s *Store) ClearCheckedItems(listID string) error {
	return s.db.Where("list_id = ? AND is_checked = ?", listID, true).Delete(&ShoppingItem{}).Error
}

// Statistics

func (s *Store) GetStats(userID string) (*ShoppingStats, error) {
	stats := &ShoppingStats{
		ByCategory: make(map[string]int),
	}
	
	var count int64
	
	// Count lists
	s.db.Model(&ShoppingList{}).Where("user_id = ?", userID).Count(&count)
	stats.TotalLists = int(count)
	s.db.Model(&ShoppingList{}).Where("user_id = ? AND is_active = ? AND completed_at IS NULL", userID, true).Count(&count)
	stats.ActiveLists = int(count)
	s.db.Model(&ShoppingList{}).Where("user_id = ? AND completed_at IS NOT NULL", userID).Count(&count)
	stats.CompletedLists = int(count)
	
	// Count items
	s.db.Model(&ShoppingItem{}).Where("user_id = ?", userID).Count(&count)
	stats.TotalItems = int(count)
	s.db.Model(&ShoppingItem{}).Where("user_id = ? AND is_checked = ?", userID, true).Count(&count)
	stats.CheckedItems = int(count)
	s.db.Model(&ShoppingItem{}).Where("user_id = ? AND is_checked = ?", userID, false).Count(&count)
	stats.UncheckedItems = int(count)
	
	// Completion rate
	if stats.TotalItems > 0 {
		stats.CompletionRate = float64(stats.CheckedItems) / float64(stats.TotalItems) * 100
	}
	
	// Items by category
	var catCounts []struct {
		Category string
		Count    int
	}
	s.db.Model(&ShoppingItem{}).Select("category, COUNT(*) as count").Where("user_id = ?", userID).Group("category").Scan(&catCounts)
	for _, c := range catCounts {
		stats.ByCategory[c.Category] = c.Count
	}
	
	return stats, nil
}

// Complete list

func (s *Store) CompleteList(listID string) error {
	now := time.Now()
	return s.db.Model(&ShoppingList{}).Where("id = ?", listID).Updates(map[string]interface{}{
		"is_active":    false,
		"completed_at": &now,
		"updated_at":   now,
	}).Error
}

// Copy list template

func (s *Store) CopyList(sourceListID, newName string) (*ShoppingList, error) {
	source, err := s.GetList(sourceListID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("source list not found")
	}
	
	// Create new list
	newList := &ShoppingList{
		UserID:      source.UserID,
		Name:        newName,
		Description: source.Description,
		Category:    source.Category,
		Tags:        source.Tags,
		IsActive:    true,
	}
	
	if err := s.CreateList(newList); err != nil {
		return nil, err
	}
	
	// Copy items
	items, err := s.GetItemsByList(sourceListID)
	if err != nil {
		return nil, err
	}
	
	for _, item := range items {
		newItem := &ShoppingItem{
			ListID:         newList.ID,
			UserID:         item.UserID,
			Name:           item.Name,
			Description:    item.Description,
			Quantity:       item.Quantity,
			Unit:           item.Unit,
			Category:       item.Category,
			StoreAisle:     item.StoreAisle,
			EstimatedPrice: item.EstimatedPrice,
			Currency:       item.Currency,
			Priority:       item.Priority,
			Barcode:        item.Barcode,
			SKU:            item.SKU,
			Notes:          item.Notes,
		}
		s.CreateItem(newItem)
	}
	
	return newList, nil
}

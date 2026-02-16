package shopping

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func setupTestSkill(t *testing.T) (*ShoppingSkill, *gorm.DB) {
	db := setupTestDB(t)
	logger, _ := zap.NewDevelopment()

	skill, err := NewShoppingSkill(db, logger)
	require.NoError(t, err)

	return skill, db
}

// Store Tests

func TestStore_CreateList(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	list := &ShoppingList{
		UserID:      "user_123",
		Name:        "Weekly Groceries",
		Category:    "groceries",
		Description: "Weekly shopping list",
	}

	err = store.CreateList(list)
	require.NoError(t, err)
	assert.NotEmpty(t, list.ID)
	assert.True(t, list.IsActive)

	// Verify we can retrieve it
	retrieved, err := store.GetList(list.ID)
	require.NoError(t, err)
	assert.Equal(t, list.Name, retrieved.Name)
	assert.Equal(t, list.UserID, retrieved.UserID)
}

func TestStore_ListOperations(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	userID := "user_123"

	// Create active lists (default behavior)
	list1 := &ShoppingList{UserID: userID, Name: "Groceries", Category: "groceries"}
	list2 := &ShoppingList{UserID: userID, Name: "Household", Category: "household"}
	store.CreateList(list1)
	store.CreateList(list2)

	// Create completed list by completing it after creation
	list3 := &ShoppingList{UserID: userID, Name: "Old List", Category: "general"}
	store.CreateList(list3)
	store.CompleteList(list3.ID) // This sets IsActive to false

	// Test ListLists - all
	allLists, err := store.ListLists(userID, false)
	require.NoError(t, err)
	assert.Len(t, allLists, 3)

	// Test ListLists - active only
	activeLists, err := store.ListLists(userID, true)
	require.NoError(t, err)
	assert.Len(t, activeLists, 2)
}

func TestStore_ItemOperations(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	userID := "user_123"

	// Create a list first
	list := &ShoppingList{
		UserID:   userID,
		Name:     "Test List",
		Category: "general",
	}
	err = store.CreateList(list)
	require.NoError(t, err)

	// Add items
	items := []*ShoppingItem{
		{ListID: list.ID, UserID: userID, Name: "Milk", Quantity: "1", Unit: "gal", Category: "dairy"},
		{ListID: list.ID, UserID: userID, Name: "Eggs", Quantity: "12", Unit: "count", Category: "dairy"},
		{ListID: list.ID, UserID: userID, Name: "Bread", Quantity: "1", Unit: "loaf", Category: "bakery"},
	}

	for _, item := range items {
		err := store.CreateItem(item)
		require.NoError(t, err)
		assert.NotEmpty(t, item.ID)
	}

	// Get items by list
	retrievedItems, err := store.GetItemsByList(list.ID)
	require.NoError(t, err)
	assert.Len(t, retrievedItems, 3)

	// Check an item
	err = store.CheckItem(items[0].ID)
	require.NoError(t, err)

	checkedItem, err := store.GetItem(items[0].ID)
	require.NoError(t, err)
	assert.True(t, checkedItem.IsChecked)
	assert.NotNil(t, checkedItem.CheckedAt)

	// Uncheck
	err = store.UncheckItem(items[0].ID)
	require.NoError(t, err)

	uncheckedItem, err := store.GetItem(items[0].ID)
	require.NoError(t, err)
	assert.False(t, uncheckedItem.IsChecked)

	// Delete item
	err = store.DeleteItem(items[1].ID)
	require.NoError(t, err)

	remainingItems, err := store.GetItemsByList(list.ID)
	require.NoError(t, err)
	assert.Len(t, remainingItems, 2)
}

func TestStore_ClearCheckedItems(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	userID := "user_123"

	list := &ShoppingList{
		UserID:   userID,
		Name:     "Clear Test",
		Category: "general",
	}
	err = store.CreateList(list)
	require.NoError(t, err)

	// Add and check some items
	items := []*ShoppingItem{
		{ListID: list.ID, UserID: userID, Name: "Item 1", Quantity: "1"},
		{ListID: list.ID, UserID: userID, Name: "Item 2", Quantity: "1"},
		{ListID: list.ID, UserID: userID, Name: "Item 3", Quantity: "1"},
	}

	for _, item := range items {
		store.CreateItem(item)
	}

	// Check first two
	store.CheckItem(items[0].ID)
	store.CheckItem(items[1].ID)

	// Clear checked
	err = store.ClearCheckedItems(list.ID)
	require.NoError(t, err)

	remaining, err := store.GetItemsByList(list.ID)
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
	assert.Equal(t, "Item 3", remaining[0].Name)
}

func TestStore_Stats(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	userID := "user_123"

	// Create active list
	list1 := &ShoppingList{UserID: userID, Name: "List 1", Category: "groceries"}
	store.CreateList(list1)

	// Create completed list
	list2 := &ShoppingList{UserID: userID, Name: "List 2", Category: "household"}
	store.CreateList(list2)
	store.CompleteList(list2.ID) // Mark as completed

	// Add items to first list
	store.CreateItem(&ShoppingItem{ListID: list1.ID, UserID: userID, Name: "Milk", Category: "dairy"})
	store.CreateItem(&ShoppingItem{ListID: list1.ID, UserID: userID, Name: "Eggs", Category: "dairy"})
	store.CreateItem(&ShoppingItem{ListID: list1.ID, UserID: userID, Name: "Bread", Category: "bakery"})

	stats, err := store.GetStats(userID)
	require.NoError(t, err)

	assert.Equal(t, 2, stats.TotalLists)
	assert.Equal(t, 1, stats.ActiveLists)
	assert.Equal(t, 1, stats.CompletedLists)
	assert.Equal(t, 3, stats.TotalItems)
	assert.Equal(t, 2, len(stats.ByCategory)) // dairy, bakery
}

// Parser Tests

func TestParser_ParseItems(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input    string
		expected int
	}{
		{"milk, eggs, bread", 3},
		{"2 gallons milk and 1 dozen eggs", 2},
		{"milk; eggs; bread", 3},
		{"get some milk and buy eggs", 2},
		{"urgent milk, important bread", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			items := parser.ParseItems(tt.input)
			assert.Len(t, items, tt.expected)
		})
	}
}

func TestParser_ExtractQuantity(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input            string
		expectedName     string
		expectedQuantity string
		expectedUnit     string
	}{
		{"2 gallons milk", "Milk", "2", "gal"},
		{"1 dozen eggs", "Eggs", "1", "dozen"},
		{"3 lbs chicken", "Chicken", "3", "lb"},
		{"500g flour", "Flour", "500", "g"},
		{"a bread", "Bread", "1", ""},
		{"milk", "Milk", "1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			items := parser.ParseItems(tt.input)
			require.Len(t, items, 1)
			assert.Equal(t, tt.expectedName, items[0].Name)
			assert.Equal(t, tt.expectedQuantity, items[0].Quantity)
			assert.Equal(t, tt.expectedUnit, items[0].Unit)
		})
	}
}

func TestParser_Categorize(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		item     string
		expected string
	}{
		{"milk", "dairy"},
		{"eggs", "dairy"},
		{"bread", "bakery"},
		{"chicken", "meat"},
		{"apples", "produce"},
		{"rice", "pantry"},
		{"soda", "beverages"},
		{"chips", "snacks"},
		{"toilet paper", "pantry"}, // "paper" in toilet paper matches pantry
		{"random thing", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			category := parser.categorizeItem(tt.item)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestParser_ExtractPriority(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input    string
		expected string
	}{
		{"urgent milk", "high"},
		{"asap eggs", "high"},
		{"low priority bread", "low"},
		{"whenever cheese", "low"},
		{"regular milk", "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			items := parser.ParseItems(tt.input)
			require.Len(t, items, 1)
			assert.Equal(t, tt.expected, items[0].Priority)
		})
	}
}

func TestParser_ExtractPrice(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input         string
		expectedPrice float64
	}{
		{"milk $5.99", 5.99},
		{"eggs 3.50 dollars", 3.50},
		{"bread", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			items := parser.ParseItems(tt.input)
			require.Len(t, items, 1)
			assert.InDelta(t, tt.expectedPrice, items[0].EstimatedPrice, 0.01)
		})
	}
}

// ShoppingSkill Tests

func TestShoppingSkill_CreateList(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	result, err := skill.handleCreateList(ctx, map[string]interface{}{
		"name":        "Weekly Groceries",
		"category":    "groceries",
		"description": "Weekly shopping",
	})

	require.NoError(t, err)
	resp := result.(map[string]interface{})
	assert.Equal(t, "Weekly Groceries", resp["name"])
	assert.Equal(t, "groceries", resp["category"])
	assert.NotEmpty(t, resp["id"])
}

func TestShoppingSkill_AddItems(t *testing.T) {
	skill, db := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	// Create a list first
	store, _ := NewStore(db)
	list := &ShoppingList{UserID: "user_123", Name: "Test", Category: "general"}
	store.CreateList(list)

	result, err := skill.handleAddItems(ctx, map[string]interface{}{
		"list_id": list.ID,
		"items":   "2 gallons milk, 1 dozen eggs, urgent bread",
	})

	require.NoError(t, err)
	resp := result.(map[string]interface{})
	assert.Equal(t, 3, resp["added_count"])

	items := resp["items"].([]map[string]interface{})
	assert.Len(t, items, 3)

	// Verify parsed correctly
	assert.Equal(t, "Milk", items[0]["name"])
	assert.Equal(t, "2 gal", items[0]["quantity"])
	assert.Equal(t, "dairy", items[0]["category"])

	assert.Equal(t, "Eggs", items[1]["name"])
	assert.Equal(t, "1 dozen", items[1]["quantity"])

	assert.Equal(t, "Bread", items[2]["name"])
	assert.Equal(t, "high", items[2]["priority"])
}

func TestShoppingSkill_AddItems_DefaultList(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	result, err := skill.handleAddItems(ctx, map[string]interface{}{
		"list_id": "default",
		"items":   "milk, eggs",
	})

	require.NoError(t, err)
	resp := result.(map[string]interface{})
	assert.Equal(t, 2, resp["added_count"])
	assert.NotEmpty(t, resp["list_id"])
}

func TestShoppingSkill_GetList(t *testing.T) {
	skill, db := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	// Create list and items
	store, _ := NewStore(db)
	list := &ShoppingList{UserID: "user_123", Name: "Groceries", Category: "general"}
	store.CreateList(list)
	store.CreateItem(&ShoppingItem{ListID: list.ID, UserID: "user_123", Name: "Milk", Category: "dairy"})
	store.CreateItem(&ShoppingItem{ListID: list.ID, UserID: "user_123", Name: "Bread", Category: "bakery"})

	result, err := skill.handleGetList(ctx, map[string]interface{}{
		"list_id":  list.ID,
		"group_by": "category",
	})

	require.NoError(t, err)
	resp := result.(map[string]interface{})
	assert.Equal(t, "Groceries", resp["name"])
	assert.Equal(t, 2, resp["item_count"])
	assert.NotNil(t, resp["grouped_by_category"])
}

func TestShoppingSkill_CheckUncheckItem(t *testing.T) {
	skill, db := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	store, _ := NewStore(db)
	list := &ShoppingList{UserID: "user_123", Name: "Test", Category: "general"}
	store.CreateList(list)
	item := &ShoppingItem{ListID: list.ID, UserID: "user_123", Name: "Milk"}
	store.CreateItem(item)

	// Check
	result, err := skill.handleCheckItem(ctx, map[string]interface{}{"item_id": item.ID})
	require.NoError(t, err)
	assert.True(t, result.(map[string]interface{})["success"].(bool))

	// Verify checked
	checked, _ := store.GetItem(item.ID)
	assert.True(t, checked.IsChecked)

	// Uncheck
	result, err = skill.handleUncheckItem(ctx, map[string]interface{}{"item_id": item.ID})
	require.NoError(t, err)
	assert.True(t, result.(map[string]interface{})["success"].(bool))

	// Verify unchecked
	unchecked, _ := store.GetItem(item.ID)
	assert.False(t, unchecked.IsChecked)
}

func TestShoppingSkill_CompleteAndDeleteList(t *testing.T) {
	skill, db := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	store, _ := NewStore(db)
	list := &ShoppingList{UserID: "user_123", Name: "Test", Category: "general"}
	store.CreateList(list)

	// Complete
	result, err := skill.handleCompleteList(ctx, map[string]interface{}{"list_id": list.ID})
	require.NoError(t, err)
	assert.True(t, result.(map[string]interface{})["success"].(bool))

	// Verify completed
	completed, _ := store.GetList(list.ID)
	assert.False(t, completed.IsActive)
	assert.NotNil(t, completed.CompletedAt)

	// Delete
	result, err = skill.handleDeleteList(ctx, map[string]interface{}{"list_id": list.ID})
	require.NoError(t, err)
	assert.True(t, result.(map[string]interface{})["success"].(bool))

	// Verify deleted
	deleted, _ := store.GetList(list.ID)
	assert.Nil(t, deleted)
}

func TestShoppingSkill_GetStats(t *testing.T) {
	skill, db := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	// Create test data
	store, _ := NewStore(db)
	list := &ShoppingList{UserID: "user_123", Name: "Test", Category: "general"}
	store.CreateList(list)
	store.CreateItem(&ShoppingItem{ListID: list.ID, UserID: "user_123", Name: "Milk", Category: "dairy", IsChecked: true})
	store.CreateItem(&ShoppingItem{ListID: list.ID, UserID: "user_123", Name: "Eggs", Category: "dairy"})

	result, err := skill.handleGetStats(ctx, map[string]interface{}{})
	require.NoError(t, err)

	resp := result.(map[string]interface{})
	assert.Equal(t, 1, resp["total_lists"])
	assert.Equal(t, 2, resp["total_items"])
	assert.Equal(t, 1, resp["checked_items"])
}

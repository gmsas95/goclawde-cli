package shopping

import (
	"context"
	"fmt"

	"github.com/gmsas95/goclawde-cli/internal/skills"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ShoppingSkill provides shopping list management
type ShoppingSkill struct {
	*skills.BaseSkill
	store  *Store
	parser *Parser
	logger *zap.Logger
}

// NewShoppingSkill creates a new shopping skill
func NewShoppingSkill(db *gorm.DB, logger *zap.Logger) (*ShoppingSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create shopping store: %w", err)
	}

	skill := &ShoppingSkill{
		BaseSkill: skills.NewBaseSkill("shopping", "Shopping Lists", "1.0.0"),
		store:     store,
		parser:    NewParser(),
		logger:    logger,
	}

	skill.registerTools()
	return skill, nil
}

func (s *ShoppingSkill) registerTools() {
	tools := []skills.Tool{
		{
			Name:        "create_shopping_list",
			Description: "Create a new shopping list with a name and optional category",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the shopping list (e.g., 'Weekly Groceries', 'Party Supplies')",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category of the list (e.g., 'groceries', 'household', 'personal')",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Optional description or notes",
					},
					"store_name": map[string]interface{}{
						"type":        "string",
						"description": "Preferred store for this list",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "add_shopping_items",
			Description: "Add items to a shopping list. Can parse natural language like '2 gallons milk, bread, urgent eggs'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"list_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the shopping list (or 'default' for user's default list)",
					},
					"items": map[string]interface{}{
						"type":        "string",
						"description": "Items to add, comma or newline separated. Can include quantities, units, priorities (e.g., '2 gallons milk, urgent bread, 3 lbs chicken')",
					},
				},
				"required": []string{"items"},
			},
		},
		{
			Name:        "get_shopping_list",
			Description: "Get shopping list details including all items",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"list_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the shopping list",
					},
					"group_by": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"category", "aisle", "none"},
						"description": "Group items by category, aisle, or don't group",
					},
				},
				"required": []string{"list_id"},
			},
		},
		{
			Name:        "get_shopping_lists",
			Description: "Get all shopping lists for the user",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active_only": map[string]interface{}{
						"type":        "boolean",
						"description": "Only return active (incomplete) lists",
					},
				},
			},
		},
		{
			Name:        "check_shopping_item",
			Description: "Mark a shopping item as checked/purchased",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"item_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the item to check off",
					},
				},
				"required": []string{"item_id"},
			},
		},
		{
			Name:        "uncheck_shopping_item",
			Description: "Mark a checked item as unchecked",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"item_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the item to uncheck",
					},
				},
				"required": []string{"item_id"},
			},
		},
		{
			Name:        "remove_shopping_item",
			Description: "Remove an item from the list",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"item_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the item to remove",
					},
				},
				"required": []string{"item_id"},
			},
		},
		{
			Name:        "clear_checked_items",
			Description: "Remove all checked items from a list",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"list_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the shopping list",
					},
				},
				"required": []string{"list_id"},
			},
		},
		{
			Name:        "complete_shopping_list",
			Description: "Mark a shopping list as completed",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"list_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the shopping list to complete",
					},
				},
				"required": []string{"list_id"},
			},
		},
		{
			Name:        "delete_shopping_list",
			Description: "Delete a shopping list and all its items",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"list_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the shopping list to delete",
					},
				},
				"required": []string{"list_id"},
			},
		},
		{
			Name:        "get_shopping_suggestions",
			Description: "Get smart suggestions for items based on shopping patterns",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category filter (e.g., 'produce', 'dairy')",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of suggestions",
					},
				},
			},
		},
		{
			Name:        "get_shopping_stats",
			Description: "Get shopping statistics and insights",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	for _, tool := range tools {
		tool.Handler = s.handleTool(tool.Name)
		s.AddTool(tool)
	}
}

func (s *ShoppingSkill) handleTool(name string) skills.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "create_shopping_list":
			return s.handleCreateList(ctx, args)
		case "add_shopping_items":
			return s.handleAddItems(ctx, args)
		case "get_shopping_list":
			return s.handleGetList(ctx, args)
		case "get_shopping_lists":
			return s.handleGetLists(ctx, args)
		case "check_shopping_item":
			return s.handleCheckItem(ctx, args)
		case "uncheck_shopping_item":
			return s.handleUncheckItem(ctx, args)
		case "remove_shopping_item":
			return s.handleRemoveItem(ctx, args)
		case "clear_checked_items":
			return s.handleClearChecked(ctx, args)
		case "complete_shopping_list":
			return s.handleCompleteList(ctx, args)
		case "delete_shopping_list":
			return s.handleDeleteList(ctx, args)
		case "get_shopping_suggestions":
			return s.handleGetSuggestions(ctx, args)
		case "get_shopping_stats":
			return s.handleGetStats(ctx, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	}
}

func getStringArg(args map[string]interface{}, key string, defaultVal string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return defaultVal
}

func getBoolArg(args map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultVal
}

func (s *ShoppingSkill) getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "default_user"
}

func (s *ShoppingSkill) handleCreateList(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)

	name := getStringArg(args, "name", "Shopping List")
	category := getStringArg(args, "category", "general")
	description := getStringArg(args, "description", "")
	storeName := getStringArg(args, "store_name", "")

	list := &ShoppingList{
		UserID:      userID,
		Name:        name,
		Category:    category,
		Description: description,
		StoreName:   storeName,
		IsActive:    true,
	}

	if err := s.store.CreateList(list); err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	s.logger.Info("Shopping list created",
		zap.String("list_id", list.ID),
		zap.String("name", list.Name),
	)

	return map[string]interface{}{
		"id":          list.ID,
		"name":        list.Name,
		"category":    list.Category,
		"description": list.Description,
		"store_name":  list.StoreName,
		"message":     fmt.Sprintf("Created shopping list '%s'", list.Name),
	}, nil
}

func (s *ShoppingSkill) handleAddItems(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)

	listID := getStringArg(args, "list_id", "default")
	itemsText := getStringArg(args, "items", "")

	if itemsText == "" {
		return nil, fmt.Errorf("no items provided")
	}

	// Parse items from natural language
	parsedItems := s.parser.ParseItems(itemsText)

	if len(parsedItems) == 0 {
		return nil, fmt.Errorf("could not parse any items from input")
	}

	// Create default list if needed
	if listID == "default" {
		lists, err := s.store.ListLists(userID, true)
		if err != nil {
			return nil, err
		}

		if len(lists) == 0 {
			// Create default list
			list := &ShoppingList{
				UserID:   userID,
				Name:     "My Shopping List",
				Category: "general",
				IsActive: true,
			}
			if err := s.store.CreateList(list); err != nil {
				return nil, err
			}
			listID = list.ID
		} else {
			listID = lists[0].ID
		}
	}

	// Verify list exists and belongs to user
	list, err := s.store.GetList(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}
	if list == nil {
		return nil, fmt.Errorf("list not found")
	}
	if list.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// Add items
	var addedItems []map[string]interface{}
	for _, parsed := range parsedItems {
		item := &ShoppingItem{
			ListID:         listID,
			UserID:         userID,
			Name:           parsed.Name,
			Quantity:       parsed.Quantity,
			Unit:           parsed.Unit,
			Category:       parsed.Category,
			Priority:       parsed.Priority,
			StoreAisle:     parsed.StoreAisle,
			EstimatedPrice: parsed.EstimatedPrice,
			Notes:          parsed.Notes,
			IsChecked:      false,
		}

		if err := s.store.CreateItem(item); err != nil {
			return nil, fmt.Errorf("failed to add item %s: %w", item.Name, err)
		}

		addedItems = append(addedItems, map[string]interface{}{
			"id":       item.ID,
			"name":     item.Name,
			"quantity": fmt.Sprintf("%s %s", item.Quantity, item.Unit),
			"category": item.Category,
			"priority": item.Priority,
		})
	}

	s.logger.Info("Items added to shopping list",
		zap.String("list_id", listID),
		zap.Int("count", len(addedItems)),
	)

	return map[string]interface{}{
		"list_id":     listID,
		"list_name":   list.Name,
		"added_count": len(addedItems),
		"items":       addedItems,
		"message":     fmt.Sprintf("Added %d item(s) to '%s'", len(addedItems), list.Name),
	}, nil
}

func (s *ShoppingSkill) handleGetList(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)

	listID := getStringArg(args, "list_id", "")
	groupBy := getStringArg(args, "group_by", "category")

	if listID == "" {
		return nil, fmt.Errorf("list_id is required")
	}

	list, err := s.store.GetList(listID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, fmt.Errorf("list not found")
	}
	if list.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	items, err := s.store.GetItemsByList(listID)
	if err != nil {
		return nil, err
	}

	// Format response
	result := map[string]interface{}{
		"id":          list.ID,
		"name":        list.Name,
		"category":    list.Category,
		"description": list.Description,
		"store_name":  list.StoreName,
		"is_active":   list.IsActive,
		"item_count":  len(items),
		"items":       items,
	}

	// Add grouped view if requested
	if groupBy == "category" && len(items) > 0 {
		grouped := make(map[string][]ShoppingItem)
		for _, item := range items {
			grouped[item.Category] = append(grouped[item.Category], item)
		}
		result["grouped_by_category"] = grouped
	}

	return result, nil
}

func (s *ShoppingSkill) handleGetLists(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)
	activeOnly := getBoolArg(args, "active_only", false)

	lists, err := s.store.ListLists(userID, activeOnly)
	if err != nil {
		return nil, err
	}

	var summaries []map[string]interface{}
	for _, list := range lists {
		items, _ := s.store.GetItemsByList(list.ID)
		checkedCount := 0
		for _, item := range items {
			if item.IsChecked {
				checkedCount++
			}
		}

		summaries = append(summaries, map[string]interface{}{
			"id":            list.ID,
			"name":          list.Name,
			"category":      list.Category,
			"is_active":     list.IsActive,
			"total_items":   len(items),
			"checked_items": checkedCount,
			"progress":      fmt.Sprintf("%d/%d", checkedCount, len(items)),
		})
	}

	return map[string]interface{}{
		"count": len(summaries),
		"lists": summaries,
	}, nil
}

func (s *ShoppingSkill) handleCheckItem(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)
	itemID := getStringArg(args, "item_id", "")

	if itemID == "" {
		return nil, fmt.Errorf("item_id is required")
	}

	item, err := s.store.GetItem(itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("item not found")
	}
	if item.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := s.store.CheckItem(itemID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"item":    item.Name,
		"message": fmt.Sprintf("Checked off '%s'", item.Name),
	}, nil
}

func (s *ShoppingSkill) handleUncheckItem(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)
	itemID := getStringArg(args, "item_id", "")

	if itemID == "" {
		return nil, fmt.Errorf("item_id is required")
	}

	item, err := s.store.GetItem(itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("item not found")
	}
	if item.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := s.store.UncheckItem(itemID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"item":    item.Name,
		"message": fmt.Sprintf("Unchecked '%s'", item.Name),
	}, nil
}

func (s *ShoppingSkill) handleRemoveItem(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)
	itemID := getStringArg(args, "item_id", "")

	if itemID == "" {
		return nil, fmt.Errorf("item_id is required")
	}

	item, err := s.store.GetItem(itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("item not found")
	}
	if item.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := s.store.DeleteItem(itemID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"item":    item.Name,
		"message": fmt.Sprintf("Removed '%s'", item.Name),
	}, nil
}

func (s *ShoppingSkill) handleClearChecked(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)
	listID := getStringArg(args, "list_id", "")

	if listID == "" {
		return nil, fmt.Errorf("list_id is required")
	}

	list, err := s.store.GetList(listID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, fmt.Errorf("list not found")
	}
	if list.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := s.store.ClearCheckedItems(listID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "Cleared all checked items",
	}, nil
}

func (s *ShoppingSkill) handleCompleteList(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)
	listID := getStringArg(args, "list_id", "")

	if listID == "" {
		return nil, fmt.Errorf("list_id is required")
	}

	list, err := s.store.GetList(listID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, fmt.Errorf("list not found")
	}
	if list.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := s.store.CompleteList(listID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"list":    list.Name,
		"message": fmt.Sprintf("Completed shopping list '%s'", list.Name),
	}, nil
}

func (s *ShoppingSkill) handleDeleteList(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)
	listID := getStringArg(args, "list_id", "")

	if listID == "" {
		return nil, fmt.Errorf("list_id is required")
	}

	list, err := s.store.GetList(listID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, fmt.Errorf("list not found")
	}
	if list.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	listName := list.Name

	if err := s.store.DeleteList(listID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"list":    listName,
		"message": fmt.Sprintf("Deleted shopping list '%s'", listName),
	}, nil
}

func (s *ShoppingSkill) handleGetSuggestions(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)

	// Get user's purchase history patterns from checked items
	stats, err := s.store.GetStats(userID)
	if err != nil {
		return nil, err
	}

	suggestions := []map[string]interface{}{}

	// Based on frequency analysis of past items
	if stats.ByCategory != nil {
		for cat, count := range stats.ByCategory {
			suggestions = append(suggestions, map[string]interface{}{
				"category": cat,
				"count":    count,
				"reason":   fmt.Sprintf("You've added %d %s items before", count, cat),
			})
		}
	}

	// Common household items
	commonItems := []string{
		"Milk", "Bread", "Eggs", "Butter",
		"Bananas", "Apples", "Rice", "Pasta",
		"Chicken", "Ground Beef", "Toilet Paper", "Dish Soap",
	}

	return map[string]interface{}{
		"suggestions":  suggestions,
		"common_items": commonItems,
		"message":      "Based on your shopping history",
	}, nil
}

func (s *ShoppingSkill) handleGetStats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := s.getUserID(ctx)

	stats, err := s.store.GetStats(userID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_lists":     stats.TotalLists,
		"active_lists":    stats.ActiveLists,
		"completed_lists": stats.CompletedLists,
		"total_items":     stats.TotalItems,
		"checked_items":   stats.CheckedItems,
		"unchecked_items": stats.UncheckedItems,
		"completion_rate": fmt.Sprintf("%.1f%%", stats.CompletionRate),
		"by_category":     stats.ByCategory,
	}, nil
}

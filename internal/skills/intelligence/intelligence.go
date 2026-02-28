package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/skills/calendar"
	"github.com/gmsas95/myrai-cli/internal/skills/expenses"
	"github.com/gmsas95/myrai-cli/internal/skills/health"
	"github.com/gmsas95/myrai-cli/internal/skills/shopping"
	"github.com/gmsas95/myrai-cli/internal/skills/tasks"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// IntelligenceSkill provides AI-powered insights and automation
type IntelligenceSkill struct {
	*skills.BaseSkill
	store            *Store
	analyzer         *PatternAnalyzer
	suggestionEngine *SuggestionEngine
	logger           *zap.Logger
	// External skill stores for dashboard data
	healthStore   *health.Store
	tasksStore    *tasks.Store
	shoppingStore *shopping.Store
	expensesStore *expenses.Store
	calendarStore *calendar.Store
}

// NewIntelligenceSkill creates a new intelligence skill with optional external stores
func NewIntelligenceSkill(db *gorm.DB, logger *zap.Logger, deps ...interface{}) (*IntelligenceSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create intelligence store: %w", err)
	}

	skill := &IntelligenceSkill{
		BaseSkill:        skills.NewBaseSkill("intelligence", "AI Intelligence & Insights", "1.0.0"),
		store:            store,
		analyzer:         NewPatternAnalyzer(store),
		suggestionEngine: NewSuggestionEngine(store),
		logger:           logger,
	}

	// Extract optional dependencies
	for _, dep := range deps {
		switch d := dep.(type) {
		case *health.Store:
			skill.healthStore = d
		case *tasks.Store:
			skill.tasksStore = d
		case *shopping.Store:
			skill.shoppingStore = d
		case *expenses.Store:
			skill.expensesStore = d
		case *calendar.Store:
			skill.calendarStore = d
		}
	}

	skill.registerTools()
	return skill, nil
}

func (i *IntelligenceSkill) registerTools() {
	tools := []skills.Tool{
		{
			Name:        "get_life_dashboard",
			Description: "Get a comprehensive overview of your life including health, productivity, finances, and upcoming items",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"include_insights": map[string]interface{}{
						"type":        "boolean",
						"description": "Include AI-generated insights",
					},
				},
			},
		},
		{
			Name:        "get_suggestions",
			Description: "Get personalized AI suggestions based on your patterns and current context",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"context": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"morning", "evening", "shopping", "health", "work", "finance", "general"},
						"description": "Current context for relevant suggestions",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of suggestions",
					},
				},
			},
		},
		{
			Name:        "dismiss_suggestion",
			Description: "Dismiss a suggestion that isn't helpful",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"suggestion_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the suggestion to dismiss",
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Optional reason for dismissing",
					},
				},
				"required": []string{"suggestion_id"},
			},
		},
		{
			Name:        "analyze_patterns",
			Description: "Analyze your behavior patterns to discover insights about your habits and routines",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"all", "health", "shopping", "productivity", "finance"},
						"description": "Category to analyze",
					},
				},
			},
		},
		{
			Name:        "list_patterns",
			Description: "View your detected behavior patterns",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category",
					},
				},
			},
		},
		{
			Name:        "create_workflow",
			Description: "Create an automated workflow. Example: 'Every morning at 8am, remind me to take medication and check my calendar'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name for this workflow",
					},
					"trigger": map[string]interface{}{
						"type":        "string",
						"description": "When to trigger (e.g., 'every morning at 8am', 'when I arrive home')",
					},
					"actions": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of actions to perform",
					},
				},
				"required": []string{"name", "trigger", "actions"},
			},
		},
		{
			Name:        "list_workflows",
			Description: "List your automated workflows",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active_only": map[string]interface{}{
						"type":        "boolean",
						"description": "Only show active workflows",
					},
				},
			},
		},
		{
			Name:        "toggle_workflow",
			Description: "Enable or disable a workflow",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workflow_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the workflow",
					},
					"enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable or disable",
					},
				},
				"required": []string{"workflow_id", "enabled"},
			},
		},
		{
			Name:        "delete_workflow",
			Description: "Delete an automated workflow",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workflow_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the workflow to delete",
					},
				},
				"required": []string{"workflow_id"},
			},
		},
		{
			Name:        "track_event",
			Description: "Track a behavior event for pattern analysis (internal use)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of event",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category",
					},
					"data": map[string]interface{}{
						"type":        "object",
						"description": "Event data",
					},
				},
				"required": []string{"event_type", "category"},
			},
		},
	}

	for _, tool := range tools {
		tool.Handler = i.handleTool(tool.Name)
		i.AddTool(tool)
	}
}

func (i *IntelligenceSkill) handleTool(name string) skills.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "get_life_dashboard":
			return i.handleGetLifeDashboard(ctx, args)
		case "get_suggestions":
			return i.handleGetSuggestions(ctx, args)
		case "dismiss_suggestion":
			return i.handleDismissSuggestion(ctx, args)
		case "analyze_patterns":
			return i.handleAnalyzePatterns(ctx, args)
		case "list_patterns":
			return i.handleListPatterns(ctx, args)
		case "create_workflow":
			return i.handleCreateWorkflow(ctx, args)
		case "list_workflows":
			return i.handleListWorkflows(ctx, args)
		case "toggle_workflow":
			return i.handleToggleWorkflow(ctx, args)
		case "delete_workflow":
			return i.handleDeleteWorkflow(ctx, args)
		case "track_event":
			return i.handleTrackEvent(ctx, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	}
}

func (i *IntelligenceSkill) getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "default_user"
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

func (i *IntelligenceSkill) handleGetLifeDashboard(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)
	includeInsights := getBoolArg(args, "include_insights", true)

	// Build dashboard
	dashboard := &LifeDashboard{
		UserID:      userID,
		GeneratedAt: time.Now(),
	}

	// Fetch real data from each skill store
	dashboard.Health = i.fetchHealthData(userID)
	dashboard.Productivity = i.fetchProductivityData(userID)
	dashboard.Finance = i.fetchFinanceData(userID)
	dashboard.Shopping = i.fetchShoppingData(userID)
	dashboard.Social = i.fetchSocialData(userID)

	// Generate insights
	if includeInsights {
		insights, _ := i.analyzer.GenerateInsights(userID)
		dashboard.Insights = insights
	}

	// Fetch upcoming items from calendar and tasks
	dashboard.Upcoming = i.fetchUpcomingItems(userID)

	return dashboard, nil
}

// fetchHealthData gets real health data from health store
func (i *IntelligenceSkill) fetchHealthData(userID string) DashboardHealth {
	health := DashboardHealth{}

	if i.healthStore == nil {
		return health
	}

	// Get medications
	medications, err := i.healthStore.ListMedications(userID, true)
	if err != nil {
		i.logger.Warn("Failed to fetch medications", zap.Error(err))
		return health
	}

	// Count doses for today
	dosesToday := 0
	dosesRemaining := 0

	for _, med := range medications {
		// Count expected doses based on frequency
		expectedDoses := len(med.Times)
		dosesRemaining += expectedDoses
	}

	// Get today's logs
	todayLogs, err := i.healthStore.GetTodayLogs(userID)
	if err == nil {
		for _, log := range todayLogs {
			if log.Status == "taken" {
				dosesToday++
				dosesRemaining--
			}
		}
	}

	health.DosesToday = dosesToday
	health.DosesRemaining = dosesRemaining

	// Calculate adherence (simplified)
	if len(medications) > 0 && dosesRemaining > 0 {
		expectedTotal := dosesToday + dosesRemaining
		if expectedTotal > 0 {
			health.MedicationAdherence = float64(dosesToday) / float64(expectedTotal) * 100
		}
	} else if dosesToday > 0 {
		health.MedicationAdherence = 100.0
	}

	// Get health score from metrics
	metric, err := i.healthStore.GetLatestMetric(userID, "weight")
	if err == nil && metric != nil {
		// Simple health score based on having metrics tracked
		health.HealthScore = 75
	}

	return health
}

// fetchProductivityData gets real productivity data from tasks store
func (i *IntelligenceSkill) fetchProductivityData(userID string) DashboardProductivity {
	prod := DashboardProductivity{}

	if i.tasksStore == nil {
		return prod
	}

	// Get task stats
	stats, err := i.tasksStore.GetStats(userID)
	if err != nil {
		i.logger.Warn("Failed to fetch task stats", zap.Error(err))
		return prod
	}

	prod.TasksToday = stats.TotalCreated
	prod.TasksCompleted = stats.Completed

	if stats.TotalCreated > 0 {
		prod.CompletionRate = float64(stats.Completed) / float64(stats.TotalCreated) * 100
	}

	// Calculate focus score based on completion rate
	prod.FocusScore = int(prod.CompletionRate)

	return prod
}

// fetchFinanceData gets real finance data from expenses store
func (i *IntelligenceSkill) fetchFinanceData(userID string) DashboardFinance {
	finance := DashboardFinance{}

	if i.expensesStore == nil {
		return finance
	}

	// Get today's expenses
	today := time.Now()
	todayStart := today.Truncate(24 * time.Hour)
	todayEnd := todayStart.Add(24 * time.Hour)

	todayExpenses, err := i.expensesStore.GetExpensesByDateRange(userID, todayStart, todayEnd)
	if err == nil {
		var totalToday float64
		for _, e := range todayExpenses {
			totalToday += e.Amount
		}
		finance.SpentToday = totalToday
	}

	// Get this month's expenses
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	monthExpenses, err := i.expensesStore.GetExpensesByDateRange(userID, monthStart, todayEnd)
	if err == nil {
		var totalMonth float64
		categoryTotals := make(map[string]float64)

		for _, e := range monthExpenses {
			totalMonth += e.Amount
			categoryTotals[e.Category] += e.Amount
		}

		finance.SpentThisMonth = totalMonth

		// Find top category
		var topCategory string
		var topAmount float64
		for cat, amount := range categoryTotals {
			if amount > topAmount {
				topAmount = amount
				topCategory = cat
			}
		}
		finance.TopCategory = topCategory
	}

	// Get budget status
	budgets, err := i.expensesStore.GetBudgets(userID, true)
	if err == nil && len(budgets) > 0 {
		var totalBudget, totalSpent float64
		for _, budget := range budgets {
			status, err := i.expensesStore.GetBudgetStatus(&budget)
			if err == nil {
				totalBudget += status.BudgetAmount
				totalSpent += status.SpentAmount
			}
		}

		if totalBudget > 0 {
			finance.BudgetPercent = (totalSpent / totalBudget) * 100
		}
	}

	return finance
}

// fetchShoppingData gets real shopping data from shopping store
func (i *IntelligenceSkill) fetchShoppingData(userID string) DashboardShopping {
	shopping := DashboardShopping{}

	if i.shoppingStore == nil {
		return shopping
	}

	// Get all lists
	lists, err := i.shoppingStore.ListLists(userID, false)
	if err != nil {
		i.logger.Warn("Failed to fetch shopping lists", zap.Error(err))
		return shopping
	}

	shopping.ActiveLists = len(lists)

	// Count items
	var totalItems, checkedItems int
	for _, list := range lists {
		items, err := i.shoppingStore.GetItemsByList(list.ID)
		if err == nil {
			for _, item := range items {
				totalItems++
				if item.IsChecked {
					checkedItems++
				}
			}
		}
	}

	shopping.ItemsNeeded = totalItems - checkedItems
	shopping.ItemsChecked = checkedItems

	if totalItems > 0 {
		shopping.CompletionRate = float64(checkedItems) / float64(totalItems) * 100
	}

	return shopping
}

// fetchSocialData gets real social data (currently limited)
func (i *IntelligenceSkill) fetchSocialData(userID string) DashboardSocial {
	social := DashboardSocial{}

	// Social data would come from messaging integrations
	// For now, we can count upcoming calendar events as social activity
	if i.calendarStore != nil {
		events, err := i.calendarStore.GetUpcomingEvents(userID, 10)
		if err == nil {
			social.UpcomingEvents = len(events)
		}
	}

	return social
}

// fetchUpcomingItems gets upcoming tasks and events
func (i *IntelligenceSkill) fetchUpcomingItems(userID string) []UpcomingItem {
	var upcoming []UpcomingItem

	// Get upcoming tasks
	if i.tasksStore != nil {
		tasks, err := i.tasksStore.GetTasksDueSoon(userID, 24*time.Hour)
		if err == nil {
			for _, task := range tasks {
				if task.DueDate != nil {
					upcoming = append(upcoming, UpcomingItem{
						Type:  "task",
						Title: task.Title,
						Time:  *task.DueDate,
					})
				}
			}
		}

		// Get overdue tasks
		overdue, err := i.tasksStore.GetOverdueTasks(userID)
		if err == nil {
			for _, task := range overdue {
				if task.DueDate != nil {
					upcoming = append(upcoming, UpcomingItem{
						Type:  "task_overdue",
						Title: task.Title + " (overdue)",
						Time:  *task.DueDate,
					})
				}
			}
		}
	}

	// Get upcoming calendar events
	if i.calendarStore != nil {
		events, err := i.calendarStore.GetUpcomingEvents(userID, 10)
		if err == nil {
			for _, event := range events {
				upcoming = append(upcoming, UpcomingItem{
					Type:  "event",
					Title: event.Title,
					Time:  event.StartTime,
				})
			}
		}
	}

	// Get upcoming medication doses
	if i.healthStore != nil {
		medications, err := i.healthStore.ListMedications(userID, true)
		if err == nil {
			now := time.Now()
			for _, med := range medications {
				for _, doseTime := range med.Times {
					// Parse time and check if it's upcoming
					doseHour, doseMin := parseTime(doseTime)
					doseDateTime := time.Date(now.Year(), now.Month(), now.Day(), doseHour, doseMin, 0, 0, now.Location())

					if doseDateTime.After(now) && doseDateTime.Before(now.Add(24*time.Hour)) {
						upcoming = append(upcoming, UpcomingItem{
							Type:  "medication",
							Title: fmt.Sprintf("Take %s (%s)", med.Name, med.Dosage),
							Time:  doseDateTime,
						})
					}
				}
			}
		}
	}

	// Sort by time
	for i := 0; i < len(upcoming)-1; i++ {
		for j := i + 1; j < len(upcoming); j++ {
			if upcoming[j].Time.Before(upcoming[i].Time) {
				upcoming[i], upcoming[j] = upcoming[j], upcoming[i]
			}
		}
	}

	// Limit to 10 items
	if len(upcoming) > 10 {
		upcoming = upcoming[:10]
	}

	return upcoming
}

// parseTime parses time string like "08:00" into hour and minute
func parseTime(t string) (int, int) {
	var hour, min int
	fmt.Sscanf(t, "%d:%d", &hour, &min)
	return hour, min
}

func (i *IntelligenceSkill) handleGetSuggestions(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)

	context := getStringArg(args, "context", "general")
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Generate new suggestions
	_, err := i.suggestionEngine.GenerateSuggestions(userID)
	if err != nil {
		i.logger.Warn("Failed to generate suggestions", zap.Error(err))
	}

	// Get relevant suggestions
	suggestions, err := i.suggestionEngine.GetRelevantSuggestions(userID, context, limit)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, s := range suggestions {
		result = append(result, map[string]interface{}{
			"id":          s.ID,
			"type":        s.Type,
			"category":    s.Category,
			"title":       s.Title,
			"description": s.Description,
			"priority":    s.Priority,
			"actionable":  s.ActionType != "none",
			"action_type": s.ActionType,
		})
	}

	return map[string]interface{}{
		"context":     context,
		"count":       len(result),
		"suggestions": result,
	}, nil
}

func (i *IntelligenceSkill) handleDismissSuggestion(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)

	suggestionID := getStringArg(args, "suggestion_id", "")
	reason := getStringArg(args, "reason", "")

	if suggestionID == "" {
		return nil, fmt.Errorf("suggestion_id is required")
	}

	// Verify ownership
	suggestion, err := i.store.GetSuggestion(suggestionID)
	if err != nil {
		return nil, err
	}
	if suggestion == nil {
		return nil, fmt.Errorf("suggestion not found")
	}
	if suggestion.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := i.store.DismissSuggestion(suggestionID); err != nil {
		return nil, err
	}

	// Save feedback if provided
	if reason != "" {
		suggestion.Feedback = reason
		i.store.UpdateSuggestion(suggestion)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Suggestion dismissed",
	}, nil
}

func (i *IntelligenceSkill) handleAnalyzePatterns(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)
	category := getStringArg(args, "category", "all")

	// Run pattern analysis
	patterns, err := i.analyzer.AnalyzeUser(userID)
	if err != nil {
		return nil, err
	}

	// Filter by category if specified
	var filteredPatterns []*UserPattern
	if category != "all" {
		for _, p := range patterns {
			if p.Category == category {
				filteredPatterns = append(filteredPatterns, p)
			}
		}
	} else {
		filteredPatterns = patterns
	}

	var result []map[string]interface{}
	for _, p := range filteredPatterns {
		result = append(result, map[string]interface{}{
			"id":          p.ID,
			"type":        p.Type,
			"category":    p.Category,
			"name":        p.Name,
			"description": p.Description,
			"confidence":  p.Confidence,
			"occurrences": p.Occurrences,
		})
	}

	return map[string]interface{}{
		"analyzed_category": category,
		"patterns_found":    len(result),
		"patterns":          result,
	}, nil
}

func (i *IntelligenceSkill) handleListPatterns(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)
	category := getStringArg(args, "category", "")

	patterns, err := i.store.ListPatterns(userID, category)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, p := range patterns {
		result = append(result, map[string]interface{}{
			"id":          p.ID,
			"type":        p.Type,
			"category":    p.Category,
			"name":        p.Name,
			"description": p.Description,
			"confidence":  p.Confidence,
			"occurrences": p.Occurrences,
			"first_seen":  p.FirstSeen.Format("Jan 2, 2006"),
			"last_seen":   p.LastSeen.Format("Jan 2, 2006"),
		})
	}

	return map[string]interface{}{
		"count":    len(result),
		"patterns": result,
	}, nil
}

func (i *IntelligenceSkill) handleCreateWorkflow(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)

	name := getStringArg(args, "name", "")
	trigger := getStringArg(args, "trigger", "")

	var actions []string
	if a, ok := args["actions"].([]interface{}); ok {
		for _, action := range a {
			if s, ok := action.(string); ok {
				actions = append(actions, s)
			}
		}
	}

	if name == "" || trigger == "" || len(actions) == 0 {
		return nil, fmt.Errorf("name, trigger, and actions are required")
	}

	triggerData, _ := json.Marshal(map[string]interface{}{
		"description": trigger,
	})

	actionsData, _ := json.Marshal(actions)

	workflow := &AutomatedWorkflow{
		UserID:      userID,
		Name:        name,
		Description: fmt.Sprintf("Auto-created workflow: %s", name),
		TriggerType: "schedule", // Default to schedule
		TriggerData: string(triggerData),
		Actions:     string(actionsData),
		Enabled:     true,
	}

	if err := i.store.CreateWorkflow(workflow); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      workflow.ID,
		"name":    workflow.Name,
		"enabled": workflow.Enabled,
		"message": fmt.Sprintf("Created workflow '%s'", name),
	}, nil
}

func (i *IntelligenceSkill) handleListWorkflows(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)
	activeOnly := getBoolArg(args, "active_only", false)

	workflows, err := i.store.ListWorkflows(userID, activeOnly)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, w := range workflows {
		result = append(result, map[string]interface{}{
			"id":           w.ID,
			"name":         w.Name,
			"description":  w.Description,
			"enabled":      w.Enabled,
			"trigger_type": w.TriggerType,
			"run_count":    w.RunCount,
		})
	}

	return map[string]interface{}{
		"count":     len(result),
		"workflows": result,
	}, nil
}

func (i *IntelligenceSkill) handleToggleWorkflow(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)

	workflowID := getStringArg(args, "workflow_id", "")
	enabled := getBoolArg(args, "enabled", true)

	if workflowID == "" {
		return nil, fmt.Errorf("workflow_id is required")
	}

	workflow, err := i.store.GetWorkflow(workflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("workflow not found")
	}
	if workflow.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	workflow.Enabled = enabled
	if err := i.store.UpdateWorkflow(workflow); err != nil {
		return nil, err
	}

	status := "enabled"
	if !enabled {
		status = "disabled"
	}

	return map[string]interface{}{
		"success": true,
		"status":  status,
		"message": fmt.Sprintf("Workflow '%s' %s", workflow.Name, status),
	}, nil
}

func (i *IntelligenceSkill) handleDeleteWorkflow(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)

	workflowID := getStringArg(args, "workflow_id", "")
	if workflowID == "" {
		return nil, fmt.Errorf("workflow_id is required")
	}

	workflow, err := i.store.GetWorkflow(workflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("workflow not found")
	}
	if workflow.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := i.store.DeleteWorkflow(workflowID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deleted workflow '%s'", workflow.Name),
	}, nil
}

func (i *IntelligenceSkill) handleTrackEvent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := i.getUserID(ctx)

	eventType := getStringArg(args, "event_type", "")
	category := getStringArg(args, "category", "")

	if eventType == "" || category == "" {
		return nil, fmt.Errorf("event_type and category are required")
	}

	data := ""
	if d, ok := args["data"].(map[string]interface{}); ok {
		jsonData, _ := json.Marshal(d)
		data = string(jsonData)
	}

	event := &BehaviorEvent{
		UserID:    userID,
		EventType: eventType,
		Category:  category,
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := i.store.CreateBehaviorEvent(event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":  true,
		"event_id": event.ID,
	}, nil
}

// TrackEvent is a helper method to track events from other skills
func (i *IntelligenceSkill) TrackEvent(userID, eventType, category string, data map[string]interface{}) error {
	jsonData, _ := json.Marshal(data)

	event := &BehaviorEvent{
		UserID:    userID,
		EventType: eventType,
		Category:  category,
		Data:      string(jsonData),
		Timestamp: time.Now(),
	}

	return i.store.CreateBehaviorEvent(event)
}

package intelligence

import (
	"encoding/json"
	"fmt"
	"time"
)

// SuggestionEngine generates proactive suggestions for users
type SuggestionEngine struct {
	store *Store
}

// NewSuggestionEngine creates a new suggestion engine
func NewSuggestionEngine(store *Store) *SuggestionEngine {
	return &SuggestionEngine{store: store}
}

// GenerateSuggestions creates suggestions for a user
func (e *SuggestionEngine) GenerateSuggestions(userID string) ([]*Suggestion, error) {
	var suggestions []*Suggestion
	
	// Health suggestions
	healthSuggestions, err := e.generateHealthSuggestions(userID)
	if err != nil {
		return nil, err
	}
	suggestions = append(suggestions, healthSuggestions...)
	
	// Productivity suggestions
	prodSuggestions, err := e.generateProductivitySuggestions(userID)
	if err != nil {
		return nil, err
	}
	suggestions = append(suggestions, prodSuggestions...)
	
	// Shopping suggestions
	shopSuggestions, err := e.generateShoppingSuggestions(userID)
	if err != nil {
		return nil, err
	}
	suggestions = append(suggestions, shopSuggestions...)
	
	// Finance suggestions
	financeSuggestions, err := e.generateFinanceSuggestions(userID)
	if err != nil {
		return nil, err
	}
	suggestions = append(suggestions, financeSuggestions...)
	
	// Save suggestions
	for _, suggestion := range suggestions {
		e.store.CreateSuggestion(suggestion)
	}
	
	return suggestions, nil
}

// generateHealthSuggestions creates health-related suggestions
func (e *SuggestionEngine) generateHealthSuggestions(userID string) ([]*Suggestion, error) {
	var suggestions []*Suggestion
	
	// Get recent medication logs
	// Note: This would need access to health store, for now we'll simulate
	
	// Suggestion 1: Medication reminder
	suggestion := &Suggestion{
		UserID:      userID,
		Type:        "proactive",
		Category:    "health",
		Title:       "Time for your evening medication",
		Description: "You usually take your medication around this time",
		ActionType:  "send_reminder",
		Priority:    "high",
		SuggestedAt: time.Now(),
		ValidUntil:  &[]time.Time{time.Now().Add(30 * time.Minute)}[0],
		Status:      "pending",
	}
	suggestions = append(suggestions, suggestion)
	
	return suggestions, nil
}

// generateProductivitySuggestions creates productivity suggestions
func (e *SuggestionEngine) generateProductivitySuggestions(userID string) ([]*Suggestion, error) {
	var suggestions []*Suggestion
	
	// Check for patterns
	patterns, _ := e.store.ListPatterns(userID, "tasks")
	
	for _, pattern := range patterns {
		if pattern.Confidence > 70 && pattern.Occurrences >= 5 {
			suggestion := &Suggestion{
				UserID:      userID,
				Type:        "insight",
				Category:    "productivity",
				Title:       fmt.Sprintf("You often %s at this time", pattern.Category),
				Description: pattern.Description,
				ActionType:  "none",
				Priority:    "low",
				SuggestedAt: time.Now(),
				Status:      "pending",
			}
			suggestions = append(suggestions, suggestion)
		}
	}
	
	return suggestions, nil
}

// generateShoppingSuggestions creates shopping-related suggestions
func (e *SuggestionEngine) generateShoppingSuggestions(userID string) ([]*Suggestion, error) {
	var suggestions []*Suggestion
	
	// Check if it's shopping time based on patterns
	hour := time.Now().Hour()
	day := int(time.Now().Weekday())
	
	// Weekend morning shopping suggestion
	if (day == 0 || day == 6) && hour >= 9 && hour <= 11 {
		suggestion := &Suggestion{
			UserID:      userID,
			Type:        "proactive",
			Category:    "shopping",
			Title:       "Weekend grocery shopping time?",
			Description: "You usually go grocery shopping on weekend mornings",
			ActionType:  "show_shopping_list",
			Priority:    "medium",
			SuggestedAt: time.Now(),
			Status:      "pending",
		}
		suggestions = append(suggestions, suggestion)
	}
	
	return suggestions, nil
}

// generateFinanceSuggestions creates finance-related suggestions
func (e *SuggestionEngine) generateFinanceSuggestions(userID string) ([]*Suggestion, error) {
	var suggestions []*Suggestion
	
	// End of month budget check
	now := time.Now()
	if now.Day() >= 25 {
		suggestion := &Suggestion{
			UserID:      userID,
			Type:        "proactive",
			Category:    "finance",
			Title:       "Review your monthly spending",
			Description: fmt.Sprintf("It's almost the end of %s. Would you like to review your spending?", now.Month()),
			ActionType:  "show_expense_summary",
			Priority:    "medium",
			SuggestedAt: time.Now(),
			Status:      "pending",
		}
		suggestions = append(suggestions, suggestion)
	}
	
	return suggestions, nil
}

// GetRelevantSuggestions returns suggestions relevant to current context
func (e *SuggestionEngine) GetRelevantSuggestions(userID string, context string, limit int) ([]Suggestion, error) {
	// Get all pending suggestions
	suggestions, err := e.store.GetPendingSuggestions(userID, limit*2)
	if err != nil {
		return nil, err
	}
	
	// Filter by context relevance
	var relevant []Suggestion
	for _, s := range suggestions {
		if isRelevant(s, context) {
			relevant = append(relevant, s)
			if len(relevant) >= limit {
				break
			}
		}
	}
	
	return relevant, nil
}

// isRelevant checks if a suggestion is relevant to a context
func isRelevant(suggestion Suggestion, context string) bool {
	// Simple keyword matching
	keywords := map[string][]string{
		"morning":   {"breakfast", "medication", "exercise", "daily"},
		"evening":   {"dinner", "medication", "sleep", "relax"},
		"shopping":  {"grocery", "store", "buy", "list"},
		"health":    {"doctor", "medication", "exercise", "appointment"},
		"work":      {"task", "meeting", "deadline", "productivity"},
		"finance":   {"budget", "expense", "spending", "money"},
	}
	
	contextKeywords, ok := keywords[context]
	if !ok {
		return true // If unknown context, include all
	}
	
	for _, kw := range contextKeywords {
		if contains(suggestion.Category, kw) || contains(suggestion.Title, kw) {
			return true
		}
	}
	
	return false
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CreateWorkflowSuggestion suggests creating a workflow
func (e *SuggestionEngine) CreateWorkflowSuggestion(userID string, trigger string, actions []string) *Suggestion {
	actionData, _ := json.Marshal(map[string]interface{}{
		"trigger": trigger,
		"actions": actions,
	})
	
	return &Suggestion{
		UserID:      userID,
		Type:        "automation",
		Category:    "productivity",
		Title:       "Automate this routine?",
		Description: fmt.Sprintf("I noticed you often %s after %s. Would you like me to automate this?", actions[0], trigger),
		ActionType:  "create_workflow",
		ActionData:  string(actionData),
		Priority:    "low",
		SuggestedAt: time.Now(),
		Status:      "pending",
	}
}

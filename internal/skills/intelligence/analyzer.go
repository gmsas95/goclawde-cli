package intelligence

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// PatternAnalyzer analyzes user behavior to detect patterns
type PatternAnalyzer struct {
	store *Store
}

// NewPatternAnalyzer creates a new pattern analyzer
func NewPatternAnalyzer(store *Store) *PatternAnalyzer {
	return &PatternAnalyzer{store: store}
}

// AnalyzeUser analyzes a user's behavior and detects patterns
func (a *PatternAnalyzer) AnalyzeUser(userID string) ([]*UserPattern, error) {
	var detectedPatterns []*UserPattern
	
	// Analyze time-based patterns
	timePatterns, err := a.analyzeTimePatterns(userID)
	if err != nil {
		return nil, err
	}
	detectedPatterns = append(detectedPatterns, timePatterns...)
	
	// Analyze sequence patterns
	seqPatterns, err := a.analyzeSequencePatterns(userID)
	if err != nil {
		return nil, err
	}
	detectedPatterns = append(detectedPatterns, seqPatterns...)
	
	// Analyze frequency patterns
	freqPatterns, err := a.analyzeFrequencyPatterns(userID)
	if err != nil {
		return nil, err
	}
	detectedPatterns = append(detectedPatterns, freqPatterns...)
	
	// Save detected patterns
	for _, pattern := range detectedPatterns {
		existing, err := a.store.FindSimilarPattern(userID, pattern.Type, pattern.Category)
		if err != nil {
			continue
		}
		
		if existing != nil {
			existing.IncrementOccurrence()
			existing.Confidence = math.Min(100, existing.Confidence+5)
			a.store.UpdatePattern(existing)
		} else {
			a.store.CreatePattern(pattern)
		}
	}
	
	return detectedPatterns, nil
}

// analyzeTimePatterns detects time-based patterns
func (a *PatternAnalyzer) analyzeTimePatterns(userID string) ([]*UserPattern, error) {
	var patterns []*UserPattern
	
	// Find peak hours for different categories
	categoryHours := make(map[string]map[int]int)
	events, _ := a.store.GetBehaviorEvents(userID, "", time.Now().AddDate(0, 0, -30), time.Time{}, 1000)
	
	for _, event := range events {
		if _, ok := categoryHours[event.Category]; !ok {
			categoryHours[event.Category] = make(map[int]int)
		}
		categoryHours[event.Category][event.HourOfDay]++
	}
	
	// Detect patterns for each category
	for category, hours := range categoryHours {
		if len(hours) < 5 {
			continue
		}
		
		// Find most common hour
		var maxHour, maxCount int
		for hour, count := range hours {
			if count > maxCount {
				maxHour = hour
				maxCount = count
			}
		}
		
		if maxCount >= 3 {
			data, _ := json.Marshal(map[string]interface{}{
				"preferred_hour": maxHour,
				"occurrences":    maxCount,
			})
			
			pattern := &UserPattern{
				UserID:      userID,
				Type:        "time_based",
				Category:    category,
				Name:        fmt.Sprintf("%s_at_%02d00", category, maxHour),
				Description: fmt.Sprintf("User frequently %s at %02d:00", category, maxHour),
				PatternData: string(data),
				Confidence:  float64(maxCount) * 10,
				Occurrences: maxCount,
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
			}
			
			patterns = append(patterns, pattern)
		}
	}
	
	return patterns, nil
}

// analyzeSequencePatterns detects action sequences
func (a *PatternAnalyzer) analyzeSequencePatterns(userID string) ([]*UserPattern, error) {
	var patterns []*UserPattern
	
	// Get recent events
	events, err := a.store.GetBehaviorEvents(userID, "", time.Now().AddDate(0, 0, -14), time.Time{}, 500)
	if err != nil {
		return nil, err
	}
	
	if len(events) < 10 {
		return patterns, nil
	}
	
	// Sort by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	
	// Find common sequences (simplified: look for pairs within 1 hour)
	sequenceCounts := make(map[string]int)
	
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			diff := events[j].Timestamp.Sub(events[i].Timestamp)
			if diff > time.Hour {
				break
			}
			
			// Create sequence key
			seq := fmt.Sprintf("%s->%s", events[i].Category, events[j].Category)
			sequenceCounts[seq]++
		}
	}
	
	// Report common sequences
	for seq, count := range sequenceCounts {
		if count >= 3 {
			parts := strings.Split(seq, "->")
			data, _ := json.Marshal(map[string]interface{}{
				"first_action":  parts[0],
				"second_action": parts[1],
				"occurrences":   count,
			})
			
			pattern := &UserPattern{
				UserID:      userID,
				Type:        "sequence",
				Category:    parts[0],
				Name:        seq,
				Description: fmt.Sprintf("After %s, user often %s within 1 hour", parts[0], parts[1]),
				PatternData: string(data),
				Confidence:  float64(count) * 15,
				Occurrences: count,
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
			}
			
			patterns = append(patterns, pattern)
		}
	}
	
	return patterns, nil
}

// analyzeFrequencyPatterns detects frequency patterns
func (a *PatternAnalyzer) analyzeFrequencyPatterns(userID string) ([]*UserPattern, error) {
	var patterns []*UserPattern
	
	// Get activity timeline
	timeline, err := a.store.GetActivityTimeline(userID, 30)
	if err != nil {
		return nil, err
	}
	
	if len(timeline) < 7 {
		return patterns, nil
	}
	
	// Calculate average daily activity
	totalCount := 0
	for _, t := range timeline {
		totalCount += t.Count
	}
	avgDaily := float64(totalCount) / float64(len(timeline))
	
	// Check for weekend vs weekday patterns
	events, _ := a.store.GetBehaviorEvents(userID, "", time.Now().AddDate(0, 0, -30), time.Time{}, 1000)
	
	weekdayCount := 0
	weekendCount := 0
	
	for _, event := range events {
		if event.DayOfWeek == 0 || event.DayOfWeek == 6 {
			weekendCount++
		} else {
			weekdayCount++
		}
	}
	
	weekdayAvg := float64(weekdayCount) / 22 // ~22 weekdays
	weekendAvg := float64(weekendCount) / 8  // ~8 weekend days
	
	// Detect weekend pattern
	if weekendAvg > weekdayAvg*1.5 {
		data, _ := json.Marshal(map[string]interface{}{
			"weekday_avg": weekdayAvg,
			"weekend_avg": weekendAvg,
		})
		
		pattern := &UserPattern{
			UserID:      userID,
			Type:        "frequency",
			Category:    "general",
			Name:        "weekend_active",
			Description: "User is more active on weekends",
			PatternData: string(data),
			Confidence:  75,
			Occurrences: len(timeline),
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}
		patterns = append(patterns, pattern)
	}
	
	// Check for consistent daily activity
	variance := 0.0
	for _, t := range timeline {
		diff := float64(t.Count) - avgDaily
		variance += diff * diff
	}
	variance /= float64(len(timeline))
	stdDev := math.Sqrt(variance)
	
	if stdDev < avgDaily*0.3 {
		data, _ := json.Marshal(map[string]interface{}{
			"average_daily": avgDaily,
			"std_dev":       stdDev,
		})
		
		pattern := &UserPattern{
			UserID:      userID,
			Type:        "frequency",
			Category:    "general",
			Name:        "consistent_daily",
			Description: "User maintains consistent daily activity",
			PatternData: string(data),
			Confidence:  80,
			Occurrences: len(timeline),
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}
		patterns = append(patterns, pattern)
	}
	
	return patterns, nil
}

// GenerateInsights generates insights based on patterns
func (a *PatternAnalyzer) GenerateInsights(userID string) ([]DashboardInsight, error) {
	var insights []DashboardInsight
	
	// Get active patterns
	patterns, err := a.store.ListPatterns(userID, "")
	if err != nil {
		return nil, err
	}
	
	// Generate insight from each high-confidence pattern
	for _, pattern := range patterns {
		if pattern.Confidence < 60 {
			continue
		}
		
		insight := DashboardInsight{
			Type:        "insight",
			Title:       pattern.Name,
			Description: pattern.Description,
			Priority:    "low",
			Actionable:  false,
		}
		
		// Make specific insights actionable
		switch pattern.Category {
		case "health":
			insight.Type = "suggestion"
			insight.Priority = "medium"
			insight.Actionable = true
			insight.Action = "Review your health routine"
		case "shopping":
			insight.Type = "trend"
			insight.Priority = "low"
		}
		
		insights = append(insights, insight)
	}
	
	return insights, nil
}

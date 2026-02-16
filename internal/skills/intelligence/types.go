package intelligence

import (
	"fmt"
	"time"
)

// UserPattern represents a detected pattern in user behavior
type UserPattern struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Pattern details
	Type        string    `json:"type"` // time_based, sequence, frequency, anomaly
	Category    string    `json:"category"` // shopping, health, tasks, expenses, etc.
	Name        string    `json:"name"`
	Description string    `json:"description"`
	
	// Pattern data
	PatternData string    `json:"pattern_data" gorm:"type:text"` // JSON serialized pattern
	
	// Confidence and stats
	Confidence  float64   `json:"confidence"` // 0-100
	Occurrences int      `json:"occurrences"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	
	// Status
	Status      string    `json:"status" gorm:"default:active"` // active, inactive, confirmed, dismissed
	
	// Action taken
	AutoAction  string    `json:"auto_action,omitempty"` // what action was automatically taken
	ActionCount int       `json:"action_count"`
	
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Suggestion represents an AI-generated suggestion for the user
type Suggestion struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Suggestion content
	Type        string    `json:"type"` // proactive, reactive, insight, reminder, automation
	Category    string    `json:"category"` // productivity, health, shopping, finance, etc.
	Title       string    `json:"title"`
	Description string    `json:"description"`
	
	// Context
	Context     string    `json:"context,omitempty"` // JSON serialized context
	Trigger     string    `json:"trigger,omitempty"` // what triggered this suggestion
	
	// Action to take
	ActionType  string    `json:"action_type"` // create_task, add_reminder, send_message, etc.
	ActionData  string    `json:"action_data,omitempty" gorm:"type:text"` // JSON data for the action
	
	// Priority and timing
	Priority    string    `json:"priority"` // low, medium, high, urgent
	SuggestedAt time.Time `json:"suggested_at"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	
	// User interaction
	Status      string    `json:"status" gorm:"default:pending"` // pending, shown, accepted, dismissed, acted_upon
	DismissedAt *time.Time `json:"dismissed_at,omitempty"`
	ActedAt     *time.Time `json:"acted_at,omitempty"`
	
	// Feedback
	WasHelpful  *bool     `json:"was_helpful,omitempty"`
	Feedback    string    `json:"feedback,omitempty"`
	
	CreatedAt   time.Time `json:"created_at"`
}

// AutomatedWorkflow represents a user-created automation
type AutomatedWorkflow struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Workflow details
	Name        string    `json:"name"`
	Description string    `json:"description"`
	
	// Trigger
	TriggerType string    `json:"trigger_type"` // schedule, event, location, pattern_detected
	TriggerData string    `json:"trigger_data" gorm:"type:text"` // JSON trigger configuration
	
	// Conditions
	Conditions  string    `json:"conditions,omitempty" gorm:"type:text"` // JSON conditions
	
	// Actions
	Actions     string    `json:"actions" gorm:"type:text"` // JSON array of actions
	
	// Status
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	RunCount    int       `json:"run_count"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	
	// Statistics
	SuccessCount int      `json:"success_count"`
	FailCount    int      `json:"fail_count"`
	
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkflowRun tracks execution of workflows
type WorkflowRun struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	WorkflowID string    `json:"workflow_id" gorm:"index"`
	UserID     string    `json:"user_id"`
	
	Status     string    `json:"status"` // running, completed, failed, cancelled
	StartedAt  time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	
	InputData  string    `json:"input_data,omitempty" gorm:"type:text"`
	OutputData string    `json:"output_data,omitempty" gorm:"type:text"`
	Error      string    `json:"error,omitempty"`
	
	CreatedAt  time.Time `json:"created_at"`
}

// LifeDashboard represents aggregated life data for the dashboard
type LifeDashboard struct {
	UserID    string    `json:"user_id"`
	GeneratedAt time.Time `json:"generated_at"`
	
	// Health summary
	Health    DashboardHealth    `json:"health"`
	
	// Productivity
	Productivity DashboardProductivity `json:"productivity"`
	
	// Finance
	Finance   DashboardFinance   `json:"finance"`
	
	// Shopping
	Shopping  DashboardShopping  `json:"shopping"`
	
	// Social/Communication
	Social    DashboardSocial    `json:"social"`
	
	// Insights
	Insights  []DashboardInsight `json:"insights"`
	
	// Upcoming
	Upcoming  []UpcomingItem     `json:"upcoming"`
}

type DashboardHealth struct {
	MedicationAdherence float64 `json:"medication_adherence"` // percentage
	DosesToday          int     `json:"doses_today"`
	DosesRemaining      int     `json:"doses_remaining"`
	LatestWeight        float64 `json:"latest_weight,omitempty"`
	WeightUnit          string  `json:"weight_unit,omitempty"`
	StepsToday          int     `json:"steps_today,omitempty"`
	SleepLastNight      float64 `json:"sleep_last_night,omitempty"`
	UpcomingAppointments int    `json:"upcoming_appointments"`
	HealthScore         int     `json:"health_score"` // 0-100
}

type DashboardProductivity struct {
	TasksToday     int     `json:"tasks_today"`
	TasksCompleted int     `json:"tasks_completed"`
	CompletionRate float64 `json:"completion_rate"`
	OverdueTasks   int     `json:"overdue_tasks"`
	FocusScore     int     `json:"focus_score"` // 0-100
}

type DashboardFinance struct {
	SpentToday      float64 `json:"spent_today"`
	SpentThisMonth  float64 `json:"spent_this_month"`
	BudgetRemaining float64 `json:"budget_remaining,omitempty"`
	BudgetPercent   float64 `json:"budget_percent"` // percentage used
	TopCategory     string  `json:"top_category"`
}

type DashboardShopping struct {
	ActiveLists     int     `json:"active_lists"`
	ItemsNeeded     int     `json:"items_needed"` // unchecked items
	ItemsChecked    int     `json:"items_checked"`
	CompletionRate  float64 `json:"completion_rate"`
}

type DashboardSocial struct {
	UnreadMessages  int     `json:"unread_messages"`
	UpcomingEvents  int     `json:"upcoming_events"`
}

type DashboardInsight struct {
	Type        string `json:"type"` // trend, alert, suggestion, achievement
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Actionable  bool   `json:"actionable"`
	Action      string `json:"action,omitempty"`
}

type UpcomingItem struct {
	Type        string    `json:"type"` // task, appointment, medication, event
	Title       string    `json:"title"`
	Time        time.Time `json:"time"`
	Description string    `json:"description,omitempty"`
}

// BehaviorEvent represents a tracked user action for pattern analysis
type BehaviorEvent struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	UserID     string    `json:"user_id" gorm:"index"`
	
	EventType  string    `json:"event_type"` // task_created, expense_added, medication_taken, etc.
	Category   string    `json:"category"` // skill category
	
	// Event details
	Data       string    `json:"data" gorm:"type:text"` // JSON event data
	Metadata   string    `json:"metadata,omitempty" gorm:"type:text"` // Additional context
	
	// Time context
	Timestamp  time.Time `json:"timestamp"`
	DayOfWeek  int       `json:"day_of_week"` // 0-6
	HourOfDay  int       `json:"hour_of_day"` // 0-23
	
	CreatedAt  time.Time `json:"created_at"`
}

// PatternMatch represents a detected pattern instance
type PatternMatch struct {
	Pattern   *UserPattern
	Confidence float64
	Data      map[string]interface{}
}

// SuggestionRequest represents a request for suggestions
type SuggestionRequest struct {
	UserID      string
	Context     string
	Limit       int
	Categories  []string
}

// UserProfile stores learned preferences about the user
type UserProfile struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	UserID          string    `json:"user_id" gorm:"uniqueIndex"`
	
	// Preferences
	PreferredTimes  string    `json:"preferred_times,omitempty" gorm:"type:text"` // JSON map of activity -> preferred times
	CommonLocations string    `json:"common_locations,omitempty" gorm:"type:text"` // JSON array
	
	// Habits
	WakeTime        string    `json:"wake_time,omitempty"` // e.g., "07:00"
	SleepTime       string    `json:"sleep_time,omitempty"` // e.g., "23:00"
	WeekendShift    bool      `json:"weekend_shift"` // different schedule on weekends
	
	// Communication
	ResponseTimeAvg int       `json:"response_time_avg,omitempty"` // average minutes to respond
	PreferredChannel string   `json:"preferred_channel,omitempty"` // telegram, discord, etc.
	
	// Productivity
	MostProductiveTime string `json:"most_productive_time,omitempty"` // morning, afternoon, evening
	TaskCompletionRate float64 `json:"task_completion_rate,omitempty"`
	
	// Updated via analysis
	LastAnalyzedAt  *time.Time `json:"last_analyzed_at,omitempty"`
	AnalysisCount   int        `json:"analysis_count"`
	
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Helper methods

// IsUrgent checks if suggestion is urgent
func (s *Suggestion) IsUrgent() bool {
	return s.Priority == "urgent" || s.Priority == "high"
}

// IsExpired checks if suggestion has expired
func (s *Suggestion) IsExpired() bool {
	if s.ValidUntil == nil {
		return false
	}
	return time.Now().After(*s.ValidUntil)
}

// ShouldShow determines if suggestion should be shown to user
func (s *Suggestion) ShouldShow() bool {
	return s.Status == "pending" && !s.IsExpired()
}

// GetPatternKey returns a unique key for the pattern
func (p *UserPattern) GetPatternKey() string {
	return fmt.Sprintf("%s:%s:%s", p.UserID, p.Type, p.Category)
}

// IncrementOccurrence updates pattern stats
func (p *UserPattern) IncrementOccurrence() {
	p.Occurrences++
	p.LastSeen = time.Now()
}

// CalculateHealthScore computes overall health score
func (d *LifeDashboard) CalculateHealthScore() int {
	score := 0
	
	// Medication adherence (40 points max)
	if d.Health.MedicationAdherence >= 90 {
		score += 40
	} else if d.Health.MedicationAdherence >= 75 {
		score += 30
	} else if d.Health.MedicationAdherence >= 50 {
		score += 20
	} else {
		score += 10
	}
	
	// Sleep (30 points max)
	if d.Health.SleepLastNight >= 7 && d.Health.SleepLastNight <= 9 {
		score += 30
	} else if d.Health.SleepLastNight >= 6 {
		score += 20
	} else if d.Health.SleepLastNight > 0 {
		score += 10
	}
	
	// Steps (30 points max)
	if d.Health.StepsToday >= 10000 {
		score += 30
	} else if d.Health.StepsToday >= 7500 {
		score += 20
	} else if d.Health.StepsToday >= 5000 {
		score += 10
	}
	
	return score
}

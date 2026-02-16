package health

import (
	"time"
)

// Medication represents a medication with schedule
type Medication struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Medication details
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Dosage      string  `json:"dosage"` // e.g., "10mg", "1 tablet"
	Form        string  `json:"form,omitempty"` // tablet, capsule, liquid, injection, etc.
	
	// Schedule
	Frequency   string    `json:"frequency"` // daily, weekly, as_needed, specific_days
	TimesPerDay int       `json:"times_per_day,omitempty"`
	Times       []string  `json:"times" gorm:"-"` // ["08:00", "20:00"] - stored in separate table or serialized
	TimesJSON   string    `json:"-" gorm:"type:text"` // Serialized times
	DaysOfWeek  []int     `json:"days_of_week,omitempty" gorm:"-"` // 0=Sunday, 1=Monday, etc.
	DaysJSON    string    `json:"-" gorm:"type:text"` // Serialized days
	
	// Timing
	WithFood    bool   `json:"with_food,omitempty"`
	BeforeBed   bool   `json:"before_bed,omitempty"`
	
	// Supply tracking
	TotalQuantity int       `json:"total_quantity,omitempty"`
	CurrentSupply int       `json:"current_supply,omitempty"`
	RefillDate    *time.Time `json:"refill_date,omitempty"`
	
	// Reminders
	RemindBefore int  `json:"remind_before,omitempty"` // minutes before
	Enabled      bool `json:"enabled" gorm:"default:true"`
	
	// Prescription info
	PrescribedBy   string     `json:"prescribed_by,omitempty"`
	PrescribedDate *time.Time `json:"prescribed_date,omitempty"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	
	// Notes
	Instructions string `json:"instructions,omitempty"`
	SideEffects  string `json:"side_effects,omitempty"`
	Notes        string `json:"notes,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// MedicationLog tracks when medications were taken
type MedicationLog struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"user_id" gorm:"index"`
	MedicationID string    `json:"medication_id" gorm:"index"`
	
	// Log details
	ScheduledTime time.Time  `json:"scheduled_time"`
	TakenTime     *time.Time `json:"taken_time,omitempty"`
	Status        string     `json:"status"` // taken, missed, skipped, late
	
	// Details
	QuantityTaken string `json:"quantity_taken,omitempty"`
	WithFood      bool   `json:"with_food,omitempty"`
	Notes         string `json:"notes,omitempty"`
	
	// Side effects
	HadSideEffects bool   `json:"had_side_effects,omitempty"`
	SideEffects    string `json:"side_effects,omitempty"`
	
	// Reminder tracking
	ReminderSent bool `json:"reminder_sent,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
}

// HealthMetric represents a health measurement
type HealthMetric struct {
	ID     string `json:"id" gorm:"primaryKey"`
	UserID string `json:"user_id" gorm:"index"`
	
	// Metric type
	Type        string  `json:"type" gorm:"index"` // weight, blood_pressure, heart_rate, temperature, blood_sugar, sleep, steps, water, etc.
	SubType     string  `json:"sub_type,omitempty"` // systolic/diastolic for BP, fasting/random for blood sugar
	
	// Value
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	
	// Context
	MeasuredAt  time.Time `json:"measured_at"`
	Context     string    `json:"context,omitempty"` // morning, evening, before_meal, after_meal, resting, after_exercise
	
	// Source
	Source      string `json:"source,omitempty"` // manual, wearable, app, device
	DeviceID    string `json:"device_id,omitempty"`
	
	// Notes
	Notes       string `json:"notes,omitempty"`
	Tags        string `json:"tags,omitempty"` // Comma-separated tags
	
	// Related metrics (e.g., BP has two values)
	RelatedMetricID string `json:"related_metric_id,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HealthAppointment represents a medical appointment
type HealthAppointment struct {
	ID     string `json:"id" gorm:"primaryKey"`
	UserID string `json:"user_id" gorm:"index"`
	
	// Appointment details
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Type        string     `json:"type"` // checkup, specialist, test, procedure, follow_up, vaccination
	
	// Provider
	ProviderName    string `json:"provider_name,omitempty"`
	ProviderType    string `json:"provider_type,omitempty"` // doctor, dentist, specialist, etc.
	Specialty       string `json:"specialty,omitempty"`     // cardiology, dermatology, etc.
	
	// Location
	Location    string     `json:"location,omitempty"`
	Address     string     `json:"address,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	
	// Timing
	DateTime    time.Time  `json:"date_time"`
	Duration    int        `json:"duration,omitempty"` // minutes
	Timezone    string     `json:"timezone,omitempty"`
	
	// Status
	Status      string     `json:"status" gorm:"default:scheduled"` // scheduled, confirmed, completed, cancelled, no_show
	
	// Reminders
	Reminders   []time.Time `json:"reminders,omitempty" gorm:"-"` // Calculated reminder times
	RemindAt    *time.Time  `json:"remind_at,omitempty"` // Primary reminder time
	
	// Preparation
	PreparationNeeded string `json:"preparation_needed,omitempty"` // fasting, bring records, etc.
	DocumentsNeeded   string `json:"documents_needed,omitempty"`
	
	// Follow-up
	FollowUpNeeded bool   `json:"follow_up_needed,omitempty"`
	FollowUpNotes  string `json:"follow_up_notes,omitempty"`
	
	// Insurance
	InsuranceUsed  string `json:"insurance_used,omitempty"`
	Cost           float64 `json:"cost,omitempty"`
	Copay          float64 `json:"copay,omitempty"`
	
	// Notes
	Notes          string `json:"notes,omitempty"`
	Outcome        string `json:"outcome,omitempty"` // What happened at the appointment
	Prescriptions  string `json:"prescriptions,omitempty"` // New prescriptions from appointment
	
	// Calendar sync
	CalendarEventID string `json:"calendar_event_id,omitempty"`
	SyncedToCalendar bool  `json:"synced_to_calendar,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HealthGoal represents a health goal/target
type HealthGoal struct {
	ID     string `json:"id" gorm:"primaryKey"`
	UserID string `json:"user_id" gorm:"index"`
	
	// Goal details
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // weight, exercise, sleep, water, medication_adherence
	
	// Target
	TargetValue    float64 `json:"target_value,omitempty"`
	CurrentValue   float64 `json:"current_value,omitempty"`
	Unit           string  `json:"unit,omitempty"`
	
	// Timeline
	StartDate      time.Time  `json:"start_date"`
	TargetDate     *time.Time `json:"target_date,omitempty"`
	CompletedDate  *time.Time `json:"completed_date,omitempty"`
	
	// Status
	Status         string `json:"status" gorm:"default:active"` // active, completed, paused, abandoned
	Progress       float64 `json:"progress,omitempty"` // 0-100
	
	// Reminders
	RemindDaily    bool   `json:"remind_daily,omitempty"`
	ReminderTime   string `json:"reminder_time,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HealthInsight represents AI-generated health insights
type HealthInsight struct {
	ID     string `json:"id" gorm:"primaryKey"`
	UserID string `json:"user_id" gorm:"index"`
	
	Type        string `json:"type"` // trend, alert, suggestion, milestone
	Category    string `json:"category"` // medication, metric, appointment, general
	Title       string `json:"title"`
	Description string `json:"description"`
	
	// Related data
	RelatedMetricType string `json:"related_metric_type,omitempty"`
	RelatedMedicationID string `json:"related_medication_id,omitempty"`
	
	// Priority
	Priority    string `json:"priority"` // low, medium, high
	Dismissed   bool   `json:"dismissed" gorm:"default:false"`
	DismissedAt *time.Time `json:"dismissed_at,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
}

// MedicationSchedule represents upcoming medication doses
type MedicationSchedule struct {
	MedicationID   string    `json:"medication_id"`
	MedicationName string    `json:"medication_name"`
	Dosage         string    `json:"dosage"`
	ScheduledTime  time.Time `json:"scheduled_time"`
	WithFood       bool      `json:"with_food"`
	IsOverdue      bool      `json:"is_overdue"`
}

// HealthStats represents health tracking statistics
type HealthStats struct {
	// Medications
	TotalMedications   int     `json:"total_medications"`
	ActiveMedications  int     `json:"active_medications"`
	AdherenceRate      float64 `json:"adherence_rate"` // percentage
	DosesTakenToday    int     `json:"doses_taken_today"`
	DosesMissedToday   int     `json:"doses_missed_today"`
	DosesRemainingToday int    `json:"doses_remaining_today"`
	
	// Appointments
	UpcomingAppointments int `json:"upcoming_appointments"`
	AppointmentsThisMonth int `json:"appointments_this_month"`
	
	// Metrics
	MetricsTracked     []string `json:"metrics_tracked"`
	LatestMetrics      map[string]interface{} `json:"latest_metrics"`
	
	// Goals
	ActiveGoals        int     `json:"active_goals"`
	CompletedGoals     int     `json:"completed_goals"`
	GoalsProgress      float64 `json:"goals_progress"` // average progress
}

// MetricSummary provides summary for a specific metric type
type MetricSummary struct {
	Type        string    `json:"type"`
	Unit        string    `json:"unit"`
	LatestValue float64   `json:"latest_value"`
	LatestDate  time.Time `json:"latest_date"`
	Average7Day float64   `json:"average_7day,omitempty"`
	Average30Day float64  `json:"average_30day,omitempty"`
	Min7Day     float64   `json:"min_7day,omitempty"`
	Max7Day     float64   `json:"max_7day,omitempty"`
	Trend       string    `json:"trend,omitempty"` // increasing, decreasing, stable
	Change7Day  float64   `json:"change_7day,omitempty"` // percentage change
}

// Helper methods

// IsOverdue checks if a medication dose is overdue
func (m *MedicationLog) IsOverdue() bool {
	if m.Status == "taken" || m.Status == "skipped" {
		return false
	}
	return time.Now().After(m.ScheduledTime.Add(30 * time.Minute))
}

// IsDueSoon checks if a medication is due within the next hour
func (s *MedicationSchedule) IsDueSoon() bool {
	now := time.Now()
	return s.ScheduledTime.After(now) && s.ScheduledTime.Before(now.Add(time.Hour))
}

// CalculateAdherence calculates adherence percentage for a period
func CalculateAdherence(logs []MedicationLog) float64 {
	if len(logs) == 0 {
		return 0
	}
	
	taken := 0
	for _, log := range logs {
		if log.Status == "taken" {
			taken++
		}
	}
	
	return float64(taken) / float64(len(logs)) * 100
}

// Package persona implements MemoryCore-inspired persona and memory system
package persona

import (
	"time"
)

// PatternType represents the type of detected pattern
type PatternType string

const (
	// FrequencyPattern indicates frequent topic mentions
	FrequencyPattern PatternType = "frequency"
	// TemporalPattern indicates time-based patterns
	TemporalPattern PatternType = "temporal"
	// SkillUsagePattern indicates frequent skill usage
	SkillUsagePattern PatternType = "skill_usage"
	// ContextSwitchPattern indicates context switching patterns
	ContextSwitchPattern PatternType = "context_switch"
)

// ProposalType represents the type of evolution proposal
type ProposalType string

const (
	// ExpertiseProposal suggests adding expertise
	ExpertiseProposal ProposalType = "expertise"
	// PreferenceProposal suggests updating preferences
	PreferenceProposal ProposalType = "preference"
	// ValueProposal suggests adding values
	ValueProposal ProposalType = "value"
	// VoiceProposal suggests adjusting voice/style
	VoiceProposal ProposalType = "voice"
	// GoalProposal suggests tracking goals
	GoalProposal ProposalType = "goal"
)

// ProposalStatus represents the status of an evolution proposal
type ProposalStatus string

const (
	// ProposalPending is awaiting user review
	ProposalPending ProposalStatus = "pending"
	// ProposalApproved has been accepted
	ProposalApproved ProposalStatus = "approved"
	// ProposalRejected has been declined
	ProposalRejected ProposalStatus = "rejected"
	// ProposalApplied has been applied to persona
	ProposalApplied ProposalStatus = "applied"
)

// ChangeType represents the type of change in a version
type ChangeType string

const (
	// ChangeManual indicates user manually edited
	ChangeManual ChangeType = "manual"
	// ChangeProposal indicates from approved proposal
	ChangeProposal ChangeType = "proposal"
	// ChangeRollback indicates a rollback
	ChangeRollback ChangeType = "rollback"
	// ChangeAuto indicates auto-applied (below threshold)
	ChangeAuto ChangeType = "auto"
)

// DetectedPattern represents a pattern detected by the evolution engine
type DetectedPattern struct {
	ID         string                 `json:"id" gorm:"primaryKey"`
	Type       PatternType            `json:"type" gorm:"index"`
	Subject    string                 `json:"subject" gorm:"index"` // e.g., topic, skill, project type
	Frequency  int                    `json:"frequency"`
	Evidence   []string               `json:"evidence" gorm:"serializer:json"`
	Confidence float64                `json:"confidence"` // 0.0 to 1.0
	FirstSeen  time.Time              `json:"first_seen"`
	LastSeen   time.Time              `json:"last_seen"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" gorm:"serializer:json"`
}

// PatternThreshold defines detection thresholds
type PatternThreshold struct {
	MinFrequency        int     `json:"min_frequency"`         // Min mentions (default: 20)
	MinConfidence       float64 `json:"min_confidence"`        // Min confidence (default: 0.7)
	SkillUsageThreshold int     `json:"skill_usage_threshold"` // Min skill uses (default: 50)
	TimeWindowDays      int     `json:"time_window_days"`      // Analysis window (default: 30)
	AutoApplyThreshold  float64 `json:"auto_apply_threshold"`  // Auto-apply confidence (default: 0.95)
}

// DefaultPatternThreshold returns default thresholds
func DefaultPatternThreshold() PatternThreshold {
	return PatternThreshold{
		MinFrequency:        20,
		MinConfidence:       0.7,
		SkillUsageThreshold: 50,
		TimeWindowDays:      30,
		AutoApplyThreshold:  0.95,
	}
}

// EvolutionProposal represents a proposed change to the persona
type EvolutionProposal struct {
	ID              string         `json:"id" gorm:"primaryKey"`
	Type            ProposalType   `json:"type" gorm:"index"`
	Title           string         `json:"title"`
	Description     string         `json:"description" gorm:"type:text"`
	Rationale       string         `json:"rationale" gorm:"type:text"`
	Change          *PersonaChange `json:"change,omitempty" gorm:"serializer:json"`
	Confidence      float64        `json:"confidence"`
	Status          ProposalStatus `json:"status" gorm:"index"`
	PatternIDs      []string       `json:"pattern_ids,omitempty" gorm:"serializer:json"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	AppliedAt       *time.Time     `json:"applied_at,omitempty"`
	RejectedAt      *time.Time     `json:"rejected_at,omitempty"`
	RejectionReason string         `json:"rejection_reason,omitempty"`
}

// PersonaChange represents a specific change to apply
type PersonaChange struct {
	Field     string      `json:"field"`               // e.g., "expertise", "preferences.communication"
	Operation string      `json:"operation"`           // e.g., "add", "update", "remove"
	Value     interface{} `json:"value"`               // The new value
	OldValue  interface{} `json:"old_value,omitempty"` // Previous value if updating
}

// PersonaVersion represents a snapshot of the persona at a point in time
type PersonaVersion struct {
	ID                string       `json:"id" gorm:"primaryKey"`
	Timestamp         time.Time    `json:"timestamp" gorm:"index"`
	Identity          *Identity    `json:"identity,omitempty" gorm:"serializer:json"`
	UserProfile       *UserProfile `json:"user_profile,omitempty" gorm:"serializer:json"`
	ChangeType        ChangeType   `json:"change_type"`
	ChangeDescription string       `json:"change_description"`
	ProposalID        *string      `json:"proposal_id,omitempty"`
	PreviousVersion   *string      `json:"previous_version,omitempty"`
}

// PatternEvidence represents evidence for a pattern
type PatternEvidence struct {
	Source    string    `json:"source"`  // e.g., "conversation", "skill_usage"
	Content   string    `json:"content"` // The actual content
	Timestamp time.Time `json:"timestamp"`
	Context   string    `json:"context,omitempty"` // Additional context
}

// SkillUsageRecord tracks skill usage for pattern detection
type SkillUsageRecord struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	SkillName   string    `json:"skill_name" gorm:"index"`
	ToolName    string    `json:"tool_name,omitempty"`
	UsageCount  int       `json:"usage_count"`
	FirstUsed   time.Time `json:"first_used"`
	LastUsed    time.Time `json:"last_used" gorm:"index"`
	ContextTags []string  `json:"context_tags,omitempty" gorm:"serializer:json"`
}

// AnalysisResult contains results from pattern analysis
type AnalysisResult struct {
	Timestamp     time.Time            `json:"timestamp"`
	PatternsFound int                  `json:"patterns_found"`
	Proposals     []*EvolutionProposal `json:"proposals"`
	Duration      time.Duration        `json:"duration"`
	AnalyzedRange struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"analyzed_range"`
}

// Notification represents a notification for new proposals
type Notification struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	ProposalID *string   `json:"proposal_id,omitempty"`
	Read       bool      `json:"read" gorm:"index"`
	CreatedAt  time.Time `json:"created_at"`
}

// WeeklyAnalysisConfig configures the weekly analysis job
type WeeklyAnalysisConfig struct {
	Enabled      bool `json:"enabled"`
	DayOfWeek    int  `json:"day_of_week"` // 0 = Sunday
	Hour         int  `json:"hour"`
	Minute       int  `json:"minute"`
	MaxProposals int  `json:"max_proposals"` // Max proposals per analysis
}

// DefaultWeeklyAnalysisConfig returns default configuration
func DefaultWeeklyAnalysisConfig() WeeklyAnalysisConfig {
	return WeeklyAnalysisConfig{
		Enabled:      true,
		DayOfWeek:    0, // Sunday
		Hour:         9,
		Minute:       0,
		MaxProposals: 5,
	}
}

// EvolutionConfig contains all evolution-related configuration
type EvolutionConfig struct {
	Enabled                 bool                 `json:"enabled"`
	Thresholds              PatternThreshold     `json:"thresholds"`
	WeeklyAnalysis          WeeklyAnalysisConfig `json:"weekly_analysis"`
	Notifications           bool                 `json:"notifications"`
	AutoApplyHighConfidence bool                 `json:"auto_apply_high_confidence"`
}

// DefaultEvolutionConfig returns default evolution configuration
func DefaultEvolutionConfig() EvolutionConfig {
	return EvolutionConfig{
		Enabled:                 true,
		Thresholds:              DefaultPatternThreshold(),
		WeeklyAnalysis:          DefaultWeeklyAnalysisConfig(),
		Notifications:           true,
		AutoApplyHighConfidence: false, // Disabled by default - user must approve
	}
}

// IsHighConfidence checks if confidence exceeds auto-apply threshold
func (ep *EvolutionProposal) IsHighConfidence(threshold float64) bool {
	return ep.Confidence >= threshold
}

// CanAutoApply checks if this proposal can be auto-applied
func (ep *EvolutionProposal) CanAutoApply(config EvolutionConfig) bool {
	if !config.AutoApplyHighConfidence {
		return false
	}
	return ep.IsHighConfidence(config.Thresholds.AutoApplyThreshold)
}

// Summary returns a brief summary of the proposal
func (ep *EvolutionProposal) Summary() string {
	switch ep.Type {
	case ExpertiseProposal:
		return "Add expertise: " + ep.Title
	case PreferenceProposal:
		return "Update preference: " + ep.Title
	case ValueProposal:
		return "Add value: " + ep.Title
	case VoiceProposal:
		return "Adjust voice: " + ep.Title
	case GoalProposal:
		return "Track goal: " + ep.Title
	default:
		return ep.Title
	}
}

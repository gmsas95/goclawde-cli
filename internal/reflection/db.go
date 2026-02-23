// Package reflection implements the Reflection Engine for self-auditing memory system
package reflection

import (
	"time"
)

// These structs are used for database storage of reflection results
// They mirror the main structs but are defined here for database operations

// DBContradiction is the database model for contradictions
type DBContradiction struct {
	ID                  string     `gorm:"primaryKey" json:"id"`
	MemoryAID           string     `gorm:"not null;index" json:"memory_a_id"`
	MemoryBID           string     `gorm:"not null;index" json:"memory_b_id"`
	Severity            string     `gorm:"not null" json:"severity"` // high, medium, low
	Description         string     `gorm:"not null" json:"description"`
	SuggestedResolution string     `json:"suggested_resolution"`
	DetectedAt          time.Time  `json:"detected_at"`
	Status              string     `gorm:"default:open" json:"status"` // open, resolved, ignored
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	Resolution          string     `json:"resolution,omitempty"`
}

// DBRedundancyGroup is the database model for redundancy groups
type DBRedundancyGroup struct {
	ID              string     `gorm:"primaryKey" json:"id"`
	Theme           string     `json:"theme"`
	MemoryIDsJSON   string     `gorm:"column:memory_ids;type:text" json:"memory_ids"`
	ClusterIDsJSON  string     `gorm:"column:cluster_ids;type:text" json:"cluster_ids"`
	Reason          string     `json:"reason"`
	SuggestedAction string     `json:"suggested_action"` // consolidate, archive, keep
	DetectedAt      time.Time  `json:"detected_at"`
	Status          string     `gorm:"default:open" json:"status"`
	ConsolidatedAt  *time.Time `json:"consolidated_at,omitempty"`
}

// DBGap is the database model for knowledge gaps
type DBGap struct {
	ID              string     `gorm:"primaryKey" json:"id"`
	Topic           string     `gorm:"not null;index" json:"topic"`
	MentionCount    int        `json:"mention_count"`
	MemoryCount     int        `json:"memory_count"`
	GapRatio        float64    `json:"gap_ratio"`
	SuggestedAction string     `json:"suggested_action"`
	DetectedAt      time.Time  `json:"detected_at"`
	Status          string     `gorm:"default:open" json:"status"`
	FilledAt        *time.Time `json:"filled_at,omitempty"`
}

// DBHealthReport is the database model for health reports
type DBHealthReport struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	GeneratedAt    time.Time `json:"generated_at"`
	OverallScore   int       `json:"overall_score"`
	TotalMemories  int       `json:"total_memories"`
	NeuralClusters int       `json:"neural_clusters"`
	Contradictions int       `json:"contradictions"`
	Redundancies   int       `json:"redundancies"`
	Gaps           int       `json:"gaps"`
	DetailsJSON    string    `gorm:"column:details;type:text" json:"details"`
	ActionsJSON    string    `gorm:"column:suggested_actions;type:text" json:"suggested_actions"`
}

// TableName returns the table name for DBContradiction
func (DBContradiction) TableName() string {
	return "contradictions"
}

// TableName returns the table name for DBRedundancyGroup
func (DBRedundancyGroup) TableName() string {
	return "redundancy_groups"
}

// TableName returns the table name for DBGap
func (DBGap) TableName() string {
	return "knowledge_gaps"
}

// TableName returns the table name for DBHealthReport
func (DBHealthReport) TableName() string {
	return "reflection_reports"
}

// ToContradiction converts a Contradiction to DBContradiction
func (c *Contradiction) ToDB() *DBContradiction {
	return &DBContradiction{
		ID:                  c.ID,
		MemoryAID:           c.MemoryAID,
		MemoryBID:           c.MemoryBID,
		Severity:            c.Severity,
		Description:         c.Description,
		SuggestedResolution: c.SuggestedResolution,
		DetectedAt:          c.DetectedAt,
		Status:              c.Status,
		ResolvedAt:          c.ResolvedAt,
		Resolution:          c.Resolution,
	}
}

// ToRedundancyGroup converts a RedundancyGroup to DBRedundancyGroup
func (r *RedundancyGroup) ToDB() *DBRedundancyGroup {
	return &DBRedundancyGroup{
		ID:              r.ID,
		Theme:           r.Theme,
		MemoryIDsJSON:   toJSON(r.MemoryIDs),
		ClusterIDsJSON:  toJSON(r.ClusterIDs),
		Reason:          r.Reason,
		SuggestedAction: r.SuggestedAction,
		DetectedAt:      r.DetectedAt,
		Status:          r.Status,
		ConsolidatedAt:  r.ConsolidatedAt,
	}
}

// ToGap converts a Gap to DBGap
func (g *Gap) ToDB() *DBGap {
	return &DBGap{
		ID:              g.ID,
		Topic:           g.Topic,
		MentionCount:    g.MentionCount,
		MemoryCount:     g.MemoryCount,
		GapRatio:        g.GapRatio,
		SuggestedAction: g.SuggestedAction,
		DetectedAt:      g.DetectedAt,
		Status:          g.Status,
		FilledAt:        g.FilledAt,
	}
}

// ToDB converts a HealthReport to DBHealthReport
func (r *HealthReport) ToDB() *DBHealthReport {
	return &DBHealthReport{
		ID:             r.ID,
		GeneratedAt:    r.GeneratedAt,
		OverallScore:   r.OverallScore,
		TotalMemories:  r.TotalMemories,
		NeuralClusters: r.NeuralClusters,
		Contradictions: r.Contradictions,
		Redundancies:   r.Redundancies,
		Gaps:           r.Gaps,
		DetailsJSON: toJSON(map[string]interface{}{
			"contradictions": r.ContradictionList,
			"redundancies":   r.RedundancyList,
			"gaps":           r.GapList,
		}),
		ActionsJSON: toJSON(r.SuggestedActions),
	}
}

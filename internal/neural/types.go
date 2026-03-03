// Package neural implements neural memory clustering for semantic context retrieval
// Phase 1: Neural Clusters - Groups related memories into semantic clusters
package neural

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// NeuralCluster represents a semantic cluster of related memories
// Clusters are formed based on semantic similarity using vector embeddings
// and represent cohesive themes or topics from memory storage
type NeuralCluster struct {
	ID              string     `gorm:"primaryKey" json:"id"`
	Theme           string     `gorm:"not null" json:"theme"`                // Thematic title/summary
	Essence         string     `gorm:"not null" json:"essence"`              // AI-generated natural language summary
	MemoryIDs       []string   `gorm:"-" json:"memory_ids"`                  // List of memory IDs (serialized as JSON)
	MemoryIDsJSON   string     `gorm:"column:memory_ids;type:text" json:"-"` // JSON storage for memory IDs
	Embedding       []byte     `gorm:"type:bytea" json:"-"`                  // Vector embedding for similarity search
	AccessCount     int        `gorm:"default:0" json:"access_count"`        // Number of times cluster was accessed
	ConfidenceScore float64    `gorm:"default:0" json:"confidence_score"`    // Cluster coherence score (0.0-1.0)
	ClusterSize     int        `gorm:"default:0" json:"cluster_size"`        // Number of memories in cluster
	Metadata        Metadata   `gorm:"-" json:"metadata"`                    // Extensible metadata
	MetadataJSON    string     `gorm:"column:metadata;type:text" json:"-"`   // JSON storage for metadata
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastAccessed    *time.Time `json:"last_accessed"`
}

// Metadata contains extensible information about a cluster
type Metadata struct {
	FormationMethod  string   `json:"formation_method"`  // How cluster was formed ("similarity", "manual", etc.)
	FormationVersion string   `json:"formation_version"` // Version of clustering algorithm
	SourceMemories   int      `json:"source_memories"`   // Original number of memories
	MergeHistory     []string `json:"merge_history"`     // IDs of clusters merged into this one
	Tags             []string `json:"tags"`              // User-defined tags
	Category         string   `json:"category"`          // High-level category
}

// ContextResult holds the result of a context retrieval operation
type ContextResult struct {
	Clusters      []*NeuralCluster `json:"clusters"`       // Relevant clusters found
	Memories      []MemoryRef      `json:"memories"`       // Individual memories from clusters
	TokenEstimate int              `json:"token_estimate"` // Estimated tokens for LLM context
	QueryTime     time.Duration    `json:"query_time"`     // Time taken for retrieval
	TotalClusters int              `json:"total_clusters"` // Total clusters in database
	TotalMemories int              `json:"total_memories"` // Total memories retrieved
}

// MemoryRef represents a memory reference within context results
type MemoryRef struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	Importance int       `json:"importance"`
	ClusterID  string    `json:"cluster_id"`
	Similarity float64   `json:"similarity"` // Similarity to query
	CreatedAt  time.Time `json:"created_at"`
}

// ClusterMemory represents the association between a cluster and a memory
type ClusterMemory struct {
	ClusterID        string    `gorm:"primaryKey" json:"cluster_id"`
	MemoryID         string    `gorm:"primaryKey" json:"memory_id"`
	SimilarityScore  float64   `gorm:"default:0" json:"similarity_score"`
	IsRepresentative bool      `gorm:"default:false" json:"is_representative"`
	AddedAt          time.Time `json:"added_at"`
}

// ClusterFormationLog records cluster formation operations for auditing
type ClusterFormationLog struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	ClusterID     string    `json:"cluster_id"`
	Operation     string    `gorm:"not null" json:"operation"` // create, merge, split, refresh, delete
	Details       string    `json:"details"`                   // JSON details
	MemoryCount   int       `gorm:"default:0" json:"memory_count"`
	PreviousState string    `json:"previous_state"` // JSON of previous state
	NewState      string    `json:"new_state"`      // JSON of new state
	DurationMs    int       `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

// QueryPattern records query patterns for optimization
type QueryPattern struct {
	ID                string    `gorm:"primaryKey" json:"id"`
	QueryText         string    `gorm:"not null" json:"query_text"`
	QueryEmbedding    []byte    `gorm:"type:bytea" json:"-"`
	MatchedClusterIDs []string  `gorm:"-" json:"matched_cluster_ids"`
	MatchedMemoryIDs  []string  `gorm:"-" json:"matched_memory_ids"`
	TokensUsed        int       `gorm:"default:0" json:"tokens_used"`
	LatencyMs         int       `json:"latency_ms"`
	CreatedAt         time.Time `json:"created_at"`
}

// TableName specifies the table name for QueryPattern
func (QueryPattern) TableName() string {
	return "cluster_query_patterns"
}

// ClusterFilter contains filters for cluster queries
type ClusterFilter struct {
	MinConfidence float64
	MaxConfidence float64
	MinSize       int
	MaxSize       int
	ThemeContains string
	Tags          []string
	Category      string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	AccessedAfter *time.Time
	Limit         int
	Offset        int
}

// FormationOptions contains options for cluster formation
type FormationOptions struct {
	SimilarityThreshold float64       // Minimum similarity for cluster membership (default: 0.85)
	MinClusterSize      int           // Minimum memories per cluster (default: 3)
	MaxClusterSize      int           // Maximum memories per cluster (default: 100)
	MaxClusters         int           // Maximum clusters to form (default: 1000)
	ReclusterThreshold  float64       // Re-cluster if confidence below this (default: 0.6)
	BatchSize           int           // Memories to process per batch (default: 100)
	FormationTimeout    time.Duration // Max time for formation (default: 30m)
}

// DefaultFormationOptions returns default cluster formation options
func DefaultFormationOptions() FormationOptions {
	return FormationOptions{
		SimilarityThreshold: 0.85,
		MinClusterSize:      3,
		MaxClusterSize:      100,
		MaxClusters:         1000,
		ReclusterThreshold:  0.6,
		BatchSize:           100,
		FormationTimeout:    30 * time.Minute,
	}
}

// RetrievalOptions contains options for context retrieval
type RetrievalOptions struct {
	MaxClusters           int           // Max clusters to retrieve (default: 5)
	MaxMemoriesPerCluster int           // Max memories per cluster (default: 10)
	MinClusterConfidence  float64       // Min confidence for cluster inclusion (default: 0.7)
	MinMemorySimilarity   float64       // Min similarity for memory inclusion (default: 0.5)
	TokenBudget           int           // Max tokens for context (default: 4000)
	IncludeMetadata       bool          // Include cluster metadata in results
	BoostRecentAccess     bool          // Boost recently accessed clusters
	RecentAccessWindow    time.Duration // Window for recent access boost (default: 7d)
}

// DefaultRetrievalOptions returns default retrieval options
func DefaultRetrievalOptions() RetrievalOptions {
	return RetrievalOptions{
		MaxClusters:           5,
		MaxMemoriesPerCluster: 10,
		MinClusterConfidence:  0.7,
		MinMemorySimilarity:   0.5,
		TokenBudget:           4000,
		IncludeMetadata:       true,
		BoostRecentAccess:     true,
		RecentAccessWindow:    7 * 24 * time.Hour,
	}
}

// FormationResult contains the result of a cluster formation operation
type FormationResult struct {
	ClustersCreated   int           `json:"clusters_created"`
	ClustersUpdated   int           `json:"clusters_updated"`
	ClustersMerged    int           `json:"clusters_merged"`
	ClustersDeleted   int           `json:"clusters_deleted"`
	MemoriesProcessed int           `json:"memories_processed"`
	MemoriesClustered int           `json:"memories_clustered"`
	Duration          time.Duration `json:"duration"`
	Errors            []error       `json:"errors,omitempty"`
}

// BeforeCreate hook for NeuralCluster - serialize JSON fields
func (c *NeuralCluster) BeforeCreate(tx *gorm.DB) error {
	if c.MemoryIDs != nil {
		data, err := json.Marshal(c.MemoryIDs)
		if err != nil {
			return err
		}
		c.MemoryIDsJSON = string(data)
	}
	if c.Metadata.Tags != nil || c.Metadata.MergeHistory != nil || c.Metadata.Category != "" {
		data, err := json.Marshal(c.Metadata)
		if err != nil {
			return err
		}
		c.MetadataJSON = string(data)
	}
	return nil
}

// AfterFind hook for NeuralCluster - deserialize JSON fields
func (c *NeuralCluster) AfterFind(tx *gorm.DB) error {
	if c.MemoryIDsJSON != "" {
		if err := json.Unmarshal([]byte(c.MemoryIDsJSON), &c.MemoryIDs); err != nil {
			return err
		}
	}
	if c.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(c.MetadataJSON), &c.Metadata); err != nil {
			return err
		}
	}
	return nil
}

// GetMemoryIDs returns the list of memory IDs from the cluster
func (c *NeuralCluster) GetMemoryIDs() []string {
	if c.MemoryIDs != nil {
		return c.MemoryIDs
	}
	if c.MemoryIDsJSON != "" {
		var ids []string
		json.Unmarshal([]byte(c.MemoryIDsJSON), &ids)
		return ids
	}
	return []string{}
}

// SetMemoryIDs sets the memory IDs for the cluster
func (c *NeuralCluster) SetMemoryIDs(ids []string) error {
	c.MemoryIDs = ids
	c.ClusterSize = len(ids)
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	c.MemoryIDsJSON = string(data)
	return nil
}

// GetMetadata returns the cluster metadata
func (c *NeuralCluster) GetMetadata() Metadata {
	// Check if metadata is populated in memory
	if c.Metadata.Tags != nil || c.Metadata.MergeHistory != nil ||
		c.Metadata.Category != "" || c.Metadata.FormationMethod != "" {
		return c.Metadata
	}
	// Otherwise try to load from JSON
	if c.MetadataJSON != "" {
		var m Metadata
		json.Unmarshal([]byte(c.MetadataJSON), &m)
		return m
	}
	return Metadata{}
}

// SetMetadata sets the cluster metadata
func (c *NeuralCluster) SetMetadata(m Metadata) error {
	c.Metadata = m
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	c.MetadataJSON = string(data)
	return nil
}

// IncrementAccess updates access statistics
func (c *NeuralCluster) IncrementAccess() {
	c.AccessCount++
	now := time.Now()
	c.LastAccessed = &now
}

// IsStale returns true if the cluster hasn't been accessed recently
func (c *NeuralCluster) IsStale(threshold time.Duration) bool {
	if c.LastAccessed == nil {
		return time.Since(c.CreatedAt) > threshold
	}
	return time.Since(*c.LastAccessed) > threshold
}

// CanMerge returns true if this cluster can be merged with another
func (c *NeuralCluster) CanMerge(other *NeuralCluster, similarity float64) bool {
	if similarity < 0.85 {
		return false
	}
	// Don't merge clusters with very different sizes
	sizeRatio := float64(c.ClusterSize) / float64(other.ClusterSize)
	if sizeRatio < 0.5 || sizeRatio > 2.0 {
		return false
	}
	return true
}

// Package reflection implements the Reflection Engine for self-auditing memory system
package reflection

import (
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/neural"
)

// RedundancyAction represents the suggested action for a redundancy group
type RedundancyAction string

const (
	ActionConsolidate RedundancyAction = "consolidate"
	ActionArchive     RedundancyAction = "archive"
	ActionKeep        RedundancyAction = "keep"
)

// RedundancyStatus represents the status of a redundancy group
type RedundancyStatus string

const (
	RedundancyStatusOpen         RedundancyStatus = "open"
	RedundancyStatusConsolidated RedundancyStatus = "consolidated"
	RedundancyStatusArchived     RedundancyStatus = "archived"
	RedundancyStatusIgnored      RedundancyStatus = "ignored"
)

// RedundancyGroup represents a group of redundant memories or clusters
type RedundancyGroup struct {
	ID              string     `gorm:"primaryKey" json:"id"`
	Theme           string     `json:"theme"`
	MemoryIDs       []string   `gorm:"-" json:"memory_ids"`
	MemoryIDsJSON   string     `gorm:"column:memory_ids;type:text" json:"-"`
	ClusterIDs      []string   `gorm:"-" json:"cluster_ids"`
	ClusterIDsJSON  string     `gorm:"column:cluster_ids;type:text" json:"-"`
	Reason          string     `json:"reason"`
	SuggestedAction string     `json:"suggested_action"` // consolidate, archive, keep
	DetectedAt      time.Time  `json:"detected_at"`
	Status          string     `gorm:"default:open" json:"status"`
	ConsolidatedAt  *time.Time `json:"consolidated_at,omitempty"`

	// Transient fields
	Memories []*neural.MemoryRef     `gorm:"-" json:"memories,omitempty"`
	Clusters []*neural.NeuralCluster `gorm:"-" json:"clusters,omitempty"`
}

// RedundancyFinder finds redundant memories and clusters
type RedundancyFinder struct {
	similarityThreshold float64
	minGroupSize        int
}

// NewRedundancyFinder creates a new redundancy finder
func NewRedundancyFinder() *RedundancyFinder {
	return &RedundancyFinder{
		similarityThreshold: 0.85, // High similarity for redundancy
		minGroupSize:        2,    // At least 2 items to be redundant
	}
}

// SetSimilarityThreshold sets the similarity threshold
func (rf *RedundancyFinder) SetSimilarityThreshold(threshold float64) {
	rf.similarityThreshold = threshold
}

// SetMinGroupSize sets the minimum group size
func (rf *RedundancyFinder) SetMinGroupSize(size int) {
	rf.minGroupSize = size
}

// FindRedundancies finds redundant clusters by theme similarity
func (rf *RedundancyFinder) FindRedundancies(clusters []*neural.NeuralCluster) ([]RedundancyGroup, error) {
	var groups []RedundancyGroup

	// Group clusters by theme (case-insensitive)
	themeMap := make(map[string][]*neural.NeuralCluster)
	for _, cluster := range clusters {
		theme := normalizeTheme(cluster.Theme)
		themeMap[theme] = append(themeMap[theme], cluster)
	}

	// Find themes with multiple clusters
	for theme, themeClusters := range themeMap {
		if len(themeClusters) >= rf.minGroupSize {
			group := rf.createRedundancyGroup(theme, themeClusters)
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// FindMemoryRedundancies finds redundant memories within a set of memories
func (rf *RedundancyFinder) FindMemoryRedundancies(memories []neural.MemoryRef) ([]RedundancyGroup, error) {
	var groups []RedundancyGroup

	// Group memories by content similarity (simple approach)
	contentMap := make(map[string][]neural.MemoryRef)
	for i := range memories {
		key := extractContentKey(memories[i].Content)
		contentMap[key] = append(contentMap[key], memories[i])
	}

	// Find content keys with multiple memories
	for key, mems := range contentMap {
		if len(mems) >= rf.minGroupSize {
			group := rf.createMemoryRedundancyGroup(key, mems)
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// createRedundancyGroup creates a redundancy group from clusters
func (rf *RedundancyFinder) createRedundancyGroup(theme string, clusters []*neural.NeuralCluster) RedundancyGroup {
	var memoryIDs []string
	var clusterIDs []string

	for _, cluster := range clusters {
		clusterIDs = append(clusterIDs, cluster.ID)
		memoryIDs = append(memoryIDs, cluster.GetMemoryIDs()...)
	}

	// Determine suggested action based on cluster characteristics
	suggestedAction := string(ActionConsolidate)
	if len(clusters) > 3 {
		// Many clusters with same theme - consolidate
		suggestedAction = string(ActionConsolidate)
	} else if len(memoryIDs) > 20 {
		// Large number of memories - might be better to archive old ones
		suggestedAction = string(ActionArchive)
	}

	return RedundancyGroup{
		ID:              generateID("redun"),
		Theme:           theme,
		MemoryIDs:       memoryIDs,
		ClusterIDs:      clusterIDs,
		Reason:          fmt.Sprintf("%d clusters with similar theme containing %d total memories", len(clusters), len(memoryIDs)),
		SuggestedAction: suggestedAction,
		DetectedAt:      time.Now(),
		Status:          string(RedundancyStatusOpen),
		Clusters:        clusters,
	}
}

// createMemoryRedundancyGroup creates a redundancy group from memories
func (rf *RedundancyFinder) createMemoryRedundancyGroup(key string, memories []neural.MemoryRef) RedundancyGroup {
	var memoryIDs []string
	memoryPtrs := make([]*neural.MemoryRef, len(memories))
	for i, mem := range memories {
		memoryIDs = append(memoryIDs, mem.ID)
		memoryPtrs[i] = &memories[i]
	}

	return RedundancyGroup{
		ID:              generateID("redun"),
		Theme:           key,
		MemoryIDs:       memoryIDs,
		Reason:          fmt.Sprintf("%d memories with similar content", len(memories)),
		SuggestedAction: string(ActionConsolidate),
		DetectedAt:      time.Now(),
		Status:          string(RedundancyStatusOpen),
		Memories:        memoryPtrs,
	}
}

// normalizeTheme normalizes a theme string for comparison
func normalizeTheme(theme string) string {
	// Convert to lowercase and remove common suffixes/prefixes
	theme = normalizeText(theme)

	// Remove common stop words
	stopWords := []string{"the", "a", "an", "and", "or", "but", "about", "regarding"}
	for _, word := range stopWords {
		theme = removeWord(theme, word)
	}

	return theme
}

// extractContentKey extracts a key for grouping similar content
func extractContentKey(content string) string {
	// Simple extraction: first 50 characters, normalized
	if len(content) > 50 {
		content = content[:50]
	}
	return normalizeText(content)
}

// normalizeText normalizes text for comparison
func normalizeText(text string) string {
	// Simple normalization
	result := ""
	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32) // to lowercase
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r == ' ' && len(result) > 0 && result[len(result)-1] != ' ' {
			result += " "
		}
	}
	return result
}

// removeWord removes a word from text
func removeWord(text, word string) string {
	// Simple word removal
	return text // Simplified for now
}

// CalculateStorageSavings estimates storage savings from consolidating
func (rg *RedundancyGroup) CalculateStorageSavings() int {
	// Rough estimate: each redundant memory saves ~500 bytes after consolidation
	redundantCount := len(rg.MemoryIDs) - 1
	if redundantCount < 0 {
		redundantCount = 0
	}
	return redundantCount * 500
}

// GetPriority returns the priority level for this redundancy group
func (rg *RedundancyGroup) GetPriority() string {
	memoryCount := len(rg.MemoryIDs)

	switch {
	case memoryCount > 20:
		return "high"
	case memoryCount > 10:
		return "medium"
	default:
		return "low"
	}
}

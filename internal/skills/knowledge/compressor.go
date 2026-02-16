package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Compressor handles memory compression and cleanup
type Compressor struct {
	store  *Store
	logger *zap.Logger
}

// NewCompressor creates a new compressor
func NewCompressor(store *Store, logger *zap.Logger) *Compressor {
	return &Compressor{
		store:  store,
		logger: logger,
	}
}

// CompressionConfig contains compression settings
type CompressionConfig struct {
	// Time thresholds
	CompressAfter   time.Duration // Compress memories older than this
	DeleteAfter     time.Duration // Delete compressed memories older than this
	
	// Importance thresholds
	MinImportanceToKeep int // Minimum importance to keep (1-10)
	MaxMemoriesPerBatch int // Maximum memories to compress in one batch
	
	// Compression options
	EnableCompression bool
	EnableDeletion    bool
}

// DefaultCompressionConfig returns default configuration
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		CompressAfter:       90 * 24 * time.Hour,  // 90 days
		DeleteAfter:         365 * 24 * time.Hour, // 1 year
		MinImportanceToKeep: 3,
		MaxMemoriesPerBatch: 100,
		EnableCompression:   true,
		EnableDeletion:      true,
	}
}

// Run performs compression on old memories
func (c *Compressor) Run(ctx context.Context, userID string, config CompressionConfig) (*CompressionResult, error) {
	result := &CompressionResult{
		Compressed: 0,
		Deleted:    0,
		Errors:     []string{},
	}
	
	// Compress old memories
	if config.EnableCompression {
		compressCount, err := c.compressOldMemories(userID, config)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("compression error: %v", err))
		}
		result.Compressed = compressCount
	}
	
	// Delete very old compressed memories
	if config.EnableDeletion {
		deleteCount, err := c.deleteVeryOldMemories(userID, config)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("deletion error: %v", err))
		}
		result.Deleted = deleteCount
	}
	
	c.logger.Info("Memory compression complete",
		zap.String("user_id", userID),
		zap.Int("compressed", result.Compressed),
		zap.Int("deleted", result.Deleted),
	)
	
	return result, nil
}

// compressOldMemories compresses old, low-importance memories
func (c *Compressor) compressOldMemories(userID string, config CompressionConfig) (int, error) {
	// Find old, unaccessed memories
	cutoff := time.Now().Add(-config.CompressAfter)
	
	memories, err := c.store.GetUnaccessedMemories(userID, cutoff, config.MaxMemoriesPerBatch)
	if err != nil {
		return 0, err
	}
	
	compressed := 0
	
	// Group memories by category for batch compression
	byCategory := c.groupByCategory(memories)
	
	for category, mems := range byCategory {
		if len(mems) < 3 {
			// Not enough memories to compress
			continue
		}
		
		// Filter by importance
		eligible := []Memory{}
		for _, mem := range mems {
			if mem.Importance <= config.MinImportanceToKeep && !mem.IsCompressed {
				eligible = append(eligible, mem)
			}
		}
		
		if len(eligible) < 3 {
			continue
		}
		
		// Compress batch
		if err := c.compressBatch(userID, category, eligible); err != nil {
			c.logger.Error("Failed to compress batch",
				zap.String("category", category),
				zap.Error(err),
			)
			continue
		}
		
		compressed += len(eligible)
	}
	
	return compressed, nil
}

// compressBatch compresses a batch of memories into a summary
func (c *Compressor) compressBatch(userID, category string, memories []Memory) error {
	if len(memories) == 0 {
		return nil
	}
	
	// Generate summary
	summary := c.generateBatchSummary(memories, category)
	
	// Collect entity IDs
	entityIDSet := make(map[string]bool)
	memoryIDs := []string{}
	
	for _, mem := range memories {
		memoryIDs = append(memoryIDs, mem.ID)
		for _, id := range mem.GetEntityIDs() {
			entityIDSet[id] = true
		}
		
		// Mark original memory as compressed (content replaced with summary)
		if err := c.store.MarkMemoryCompressed(mem.ID, "", summary); err != nil {
			c.logger.Error("Failed to mark memory compressed", zap.String("id", mem.ID), zap.Error(err))
		}
	}
	
	// Create compressed memory
	entityIDs := []string{}
	for id := range entityIDSet {
		entityIDs = append(entityIDs, id)
	}
	
	compressedMemory := &Memory{
		UserID:         userID,
		Content:        summary,
		Summary:        summary,
		Type:           string(MemoryTypeFact),
		Category:       category,
		EntityIDs:      strings.Join(entityIDs, ","),
		IsCompressed:   true,
		CompressedFrom: strings.Join(memoryIDs, ","),
		Importance:     c.calculateCompressedImportance(memories),
	}
	
	if err := c.store.CreateMemory(compressedMemory); err != nil {
		return fmt.Errorf("failed to create compressed memory: %w", err)
	}
	
	c.logger.Info("Compressed memory batch",
		zap.String("category", category),
		zap.Int("count", len(memories)),
		zap.String("summary", summary[:min(len(summary), 100)]),
	)
	
	return nil
}

// generateBatchSummary generates a summary for a batch of memories
func (c *Compressor) generateBatchSummary(memories []Memory, category string) string {
	// Simple summarization - count by type and key themes
	typeCounts := make(map[string]int)
	keyTerms := make(map[string]int)
	
	for _, mem := range memories {
		typeCounts[mem.Type]++
		
		// Extract key terms (simple approach)
		words := strings.Fields(strings.ToLower(mem.Content))
		for _, word := range words {
			// Filter common words
			if !c.isCommonWord(word) && len(word) > 3 {
				keyTerms[word]++
			}
		}
	}
	
	// Build summary
	parts := []string{}
	
	// Type summary
	if count, ok := typeCounts[string(MemoryTypeEvent)]; ok && count > 0 {
		parts = append(parts, fmt.Sprintf("%d events", count))
	}
	if count, ok := typeCounts[string(MemoryTypePreference)]; ok && count > 0 {
		parts = append(parts, fmt.Sprintf("%d preferences", count))
	}
	if count, ok := typeCounts[string(MemoryTypeFact)]; ok && count > 0 {
		parts = append(parts, fmt.Sprintf("%d facts", count))
	}
	
	// Get top key terms
	topTerms := c.getTopTerms(keyTerms, 5)
	
	summary := fmt.Sprintf("Compressed %d memories from %s category", len(memories), category)
	if len(parts) > 0 {
		summary += " including " + strings.Join(parts, ", ")
	}
	if len(topTerms) > 0 {
		summary += ". Key topics: " + strings.Join(topTerms, ", ")
	}
	
	return summary
}

// deleteVeryOldMemories deletes very old compressed memories
func (c *Compressor) deleteVeryOldMemories(userID string, config CompressionConfig) (int, error) {
	cutoff := time.Now().Add(-config.DeleteAfter)
	
	// This would typically use a bulk delete operation
	// For now, we'll count and return
	filters := MemoryFilters{
		Limit: 1000,
	}
	
	memories, err := c.store.GetMemories(userID, filters)
	if err != nil {
		return 0, err
	}
	
	deleted := 0
	for _, mem := range memories {
		if mem.IsCompressed && mem.CreatedAt.Before(cutoff) && mem.Importance <= 2 {
			if err := c.store.DeleteMemory(mem.ID); err != nil {
				c.logger.Error("Failed to delete memory", zap.String("id", mem.ID), zap.Error(err))
				continue
			}
			deleted++
		}
	}
	
	return deleted, nil
}

// groupByCategory groups memories by category
func (c *Compressor) groupByCategory(memories []Memory) map[string][]Memory {
	groups := make(map[string][]Memory)
	
	for _, mem := range memories {
		category := mem.Category
		if category == "" {
			category = "general"
		}
		groups[category] = append(groups[category], mem)
	}
	
	return groups
}

// calculateCompressedImportance calculates importance for compressed memory
func (c *Compressor) calculateCompressedImportance(memories []Memory) int {
	maxImportance := 0
	for _, mem := range memories {
		if mem.Importance > maxImportance {
			maxImportance = mem.Importance
		}
	}
	
	// Slightly reduce importance for compressed memories
	if maxImportance > 1 {
		maxImportance--
	}
	
	return maxImportance
}

// isCommonWord checks if a word is common
func (c *Compressor) isCommonWord(word string) bool {
	common := map[string]bool{
		"the": true, "be": true, "to": true, "of": true, "and": true,
		"a": true, "in": true, "that": true, "have": true, "i": true,
		"it": true, "for": true, "not": true, "on": true, "with": true,
		"he": true, "as": true, "you": true, "do": true, "at": true,
		"this": true, "but": true, "his": true, "by": true, "from": true,
		"they": true, "we": true, "say": true, "her": true, "she": true,
		"or": true, "an": true, "will": true, "my": true, "one": true,
		"all": true, "would": true, "there": true, "their": true, "what": true,
		"so": true, "up": true, "out": true, "if": true, "about": true,
		"who": true, "get": true, "which": true, "go": true, "me": true,
		"when": true, "make": true, "can": true, "like": true, "time": true,
		"no": true, "just": true, "him": true, "know": true, "take": true,
		"people": true, "into": true, "year": true, "your": true, "good": true,
		"some": true, "could": true, "them": true, "see": true, "other": true,
		"than": true, "then": true, "now": true, "look": true, "only": true,
		"come": true, "its": true, "over": true, "think": true, "also": true,
		"back": true, "after": true, "use": true, "two": true, "how": true,
		"our": true, "work": true, "first": true, "well": true, "way": true,
		"even": true, "new": true, "want": true, "because": true, "any": true,
		"these": true, "give": true, "day": true, "most": true, "us": true,
	}
	
	return common[strings.ToLower(word)]
}

// getTopTerms returns top N terms by frequency
func (c *Compressor) getTopTerms(terms map[string]int, n int) []string {
	type termCount struct {
		term  string
		count int
	}
	
	counts := []termCount{}
	for term, count := range terms {
		counts = append(counts, termCount{term, count})
	}
	
	// Sort by count (simple bubble sort for small lists)
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}
	
	result := []string{}
	for i := 0; i < min(len(counts), n); i++ {
		result = append(result, counts[i].term)
	}
	
	return result
}

// CompressionResult contains compression results
type CompressionResult struct {
	Compressed int      `json:"compressed"`
	Deleted    int      `json:"deleted"`
	Errors     []string `json:"errors,omitempty"`
}

// ScheduleCompression schedules periodic compression
func (c *Compressor) ScheduleCompression(ctx context.Context, interval time.Duration, config CompressionConfig) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// In production, this would iterate over all users
			// For now, we'll just log
			c.logger.Info("Running scheduled memory compression")
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

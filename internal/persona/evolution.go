package persona

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// EvolutionEngine detects patterns and generates evolution proposals
type EvolutionEngine struct {
	store      *store.Store
	llmClient  *llm.Client
	logger     *zap.Logger
	config     EvolutionConfig
	thresholds PatternThreshold
}

// NewEvolutionEngine creates a new evolution engine
func NewEvolutionEngine(store *store.Store, llmClient *llm.Client, logger *zap.Logger) *EvolutionEngine {
	return &EvolutionEngine{
		store:      store,
		llmClient:  llmClient,
		logger:     logger,
		config:     DefaultEvolutionConfig(),
		thresholds: DefaultPatternThreshold(),
	}
}

// SetConfig updates the evolution configuration
func (ee *EvolutionEngine) SetConfig(config EvolutionConfig) {
	ee.config = config
	ee.thresholds = config.Thresholds
}

// GetConfig returns the current evolution configuration
func (ee *EvolutionEngine) GetConfig() EvolutionConfig {
	return ee.config
}

// AnalyzePatterns analyzes data and detects patterns
func (ee *EvolutionEngine) AnalyzePatterns(ctx context.Context) (*AnalysisResult, error) {
	start := time.Now()
	ee.logger.Info("Starting pattern analysis")

	result := &AnalysisResult{
		Timestamp: start,
		AnalyzedRange: struct {
			From time.Time `json:"from"`
			To   time.Time `json:"to"`
		}{
			From: start.AddDate(0, 0, -ee.thresholds.TimeWindowDays),
			To:   start,
		},
	}

	// Detect frequency patterns from conversations
	freqPatterns, err := ee.detectFrequencyPatterns(ctx)
	if err != nil {
		ee.logger.Warn("Failed to detect frequency patterns", zap.Error(err))
	}

	// Detect skill usage patterns
	skillPatterns, err := ee.detectSkillUsagePatterns(ctx)
	if err != nil {
		ee.logger.Warn("Failed to detect skill usage patterns", zap.Error(err))
	}

	// Detect temporal patterns
	temporalPatterns, err := ee.detectTemporalPatterns(ctx)
	if err != nil {
		ee.logger.Warn("Failed to detect temporal patterns", zap.Error(err))
	}

	// Combine all patterns
	allPatterns := append(append(freqPatterns, skillPatterns...), temporalPatterns...)
	result.PatternsFound = len(allPatterns)

	// Save patterns to store
	for _, pattern := range allPatterns {
		if err := ee.savePattern(ctx, pattern); err != nil {
			ee.logger.Warn("Failed to save pattern", zap.Error(err))
		}
	}

	// Generate proposals from patterns
	proposals, err := ee.GenerateProposals(ctx, allPatterns)
	if err != nil {
		return result, fmt.Errorf("failed to generate proposals: %w", err)
	}

	result.Proposals = proposals
	result.Duration = time.Since(start)

	ee.logger.Info("Pattern analysis complete",
		zap.Int("patterns_found", result.PatternsFound),
		zap.Int("proposals_generated", len(proposals)),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// detectFrequencyPatterns detects frequently mentioned topics
func (ee *EvolutionEngine) detectFrequencyPatterns(ctx context.Context) ([]*DetectedPattern, error) {
	var patterns []*DetectedPattern

	// Get messages from the time window
	fromDate := time.Now().AddDate(0, 0, -ee.thresholds.TimeWindowDays)

	var messages []store.Message
	err := ee.store.DB().Where("created_at >= ? AND role = ?", fromDate, "user").
		Order("created_at DESC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	// Count topic mentions
	topicCounts := make(map[string]*DetectedPattern)

	for _, msg := range messages {
		// Extract topics (simple keyword extraction)
		topics := ee.extractTopics(msg.Content)

		for _, topic := range topics {
			if _, exists := topicCounts[topic]; !exists {
				topicCounts[topic] = &DetectedPattern{
					ID:         ee.generatePatternID(),
					Type:       FrequencyPattern,
					Subject:    topic,
					Frequency:  0,
					Evidence:   []string{},
					FirstSeen:  msg.CreatedAt,
					Confidence: 0.0,
				}
			}

			p := topicCounts[topic]
			p.Frequency++
			p.LastSeen = msg.CreatedAt

			// Keep only recent evidence (max 5)
			if len(p.Evidence) < 5 {
				p.Evidence = append(p.Evidence, msg.Content[:min(len(msg.Content), 100)])
			}
		}
	}

	// Filter by threshold and calculate confidence
	for topic, pattern := range topicCounts {
		if pattern.Frequency >= ee.thresholds.MinFrequency {
			// Calculate confidence based on frequency and consistency
			pattern.Confidence = ee.calculateFrequencyConfidence(pattern)

			if pattern.Confidence >= ee.thresholds.MinConfidence {
				patterns = append(patterns, pattern)
				ee.logger.Debug("Detected frequency pattern",
					zap.String("topic", topic),
					zap.Int("frequency", pattern.Frequency),
					zap.Float64("confidence", pattern.Confidence),
				)
			}
		}
	}

	return patterns, nil
}

// detectSkillUsagePatterns detects frequently used skills
func (ee *EvolutionEngine) detectSkillUsagePatterns(ctx context.Context) ([]*DetectedPattern, error) {
	var patterns []*DetectedPattern

	// Get skill usage from the time window
	fromDate := time.Now().AddDate(0, 0, -ee.thresholds.TimeWindowDays)

	var records []SkillUsageRecord
	err := ee.store.DB().Where("last_used >= ?", fromDate).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	// Group by skill
	skillCounts := make(map[string]*DetectedPattern)

	for _, record := range records {
		if _, exists := skillCounts[record.SkillName]; !exists {
			skillCounts[record.SkillName] = &DetectedPattern{
				ID:         ee.generatePatternID(),
				Type:       SkillUsagePattern,
				Subject:    record.SkillName,
				Frequency:  0,
				Evidence:   []string{},
				FirstSeen:  record.FirstUsed,
				Confidence: 0.0,
				Metadata: map[string]interface{}{
					"context_tags": record.ContextTags,
				},
			}
		}

		p := skillCounts[record.SkillName]
		p.Frequency += record.UsageCount
		p.LastSeen = record.LastUsed
	}

	// Filter by skill usage threshold
	for skill, pattern := range skillCounts {
		if pattern.Frequency >= ee.thresholds.SkillUsageThreshold {
			pattern.Confidence = ee.calculateSkillConfidence(pattern)

			if pattern.Confidence >= ee.thresholds.MinConfidence {
				patterns = append(patterns, pattern)
				ee.logger.Debug("Detected skill usage pattern",
					zap.String("skill", skill),
					zap.Int("frequency", pattern.Frequency),
					zap.Float64("confidence", pattern.Confidence),
				)
			}
		}
	}

	return patterns, nil
}

// detectTemporalPatterns detects time-based patterns
func (ee *EvolutionEngine) detectTemporalPatterns(ctx context.Context) ([]*DetectedPattern, error) {
	var patterns []*DetectedPattern

	// Analyze conversation times
	fromDate := time.Now().AddDate(0, 0, -ee.thresholds.TimeWindowDays)

	var messages []store.Message
	err := ee.store.DB().Where("created_at >= ?", fromDate).
		Select("created_at").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	// Count by hour of day
	hourCounts := make(map[int]int)
	for _, msg := range messages {
		hour := msg.CreatedAt.Hour()
		hourCounts[hour]++
	}

	// Find peak hours (more than 20% of messages)
	totalMessages := len(messages)
	if totalMessages > 0 {
		for hour, count := range hourCounts {
			percentage := float64(count) / float64(totalMessages)
			if percentage >= 0.20 {
				timeOfDay := ee.getTimeOfDayLabel(hour)

				pattern := &DetectedPattern{
					ID:         ee.generatePatternID(),
					Type:       TemporalPattern,
					Subject:    timeOfDay,
					Frequency:  count,
					Evidence:   []string{fmt.Sprintf("Peak activity hour: %d:00 (%d messages, %.1f%%)", hour, count, percentage*100)},
					FirstSeen:  fromDate,
					LastSeen:   time.Now(),
					Confidence: percentage,
					Metadata: map[string]interface{}{
						"peak_hour":     hour,
						"percentage":    percentage,
						"message_count": count,
					},
				}

				if pattern.Confidence >= ee.thresholds.MinConfidence {
					patterns = append(patterns, pattern)
				}
			}
		}
	}

	return patterns, nil
}

// GenerateProposals creates evolution proposals based on detected patterns
func (ee *EvolutionEngine) GenerateProposals(ctx context.Context, patterns []*DetectedPattern) ([]*EvolutionProposal, error) {
	var proposals []*EvolutionProposal

	for _, pattern := range patterns {
		proposal := ee.createProposalFromPattern(pattern)
		if proposal != nil {
			proposals = append(proposals, proposal)
		}
	}

	// Sort by confidence (highest first)
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].Confidence > proposals[j].Confidence
	})

	// Limit to max proposals
	if len(proposals) > ee.config.WeeklyAnalysis.MaxProposals {
		proposals = proposals[:ee.config.WeeklyAnalysis.MaxProposals]
	}

	return proposals, nil
}

// createProposalFromPattern creates a proposal from a detected pattern
func (ee *EvolutionEngine) createProposalFromPattern(pattern *DetectedPattern) *EvolutionProposal {
	switch pattern.Type {
	case FrequencyPattern:
		return ee.createExpertiseProposal(pattern)
	case SkillUsagePattern:
		return ee.createSkillExpertiseProposal(pattern)
	case TemporalPattern:
		return ee.createTemporalPreferenceProposal(pattern)
	default:
		return nil
	}
}

// createExpertiseProposal creates a proposal for adding expertise
func (ee *EvolutionEngine) createExpertiseProposal(pattern *DetectedPattern) *EvolutionProposal {
	expertise := ee.normalizeExpertise(pattern.Subject)

	return &EvolutionProposal{
		ID:          ee.generateProposalID(),
		Type:        ExpertiseProposal,
		Title:       fmt.Sprintf("Add %s expertise", expertise),
		Description: fmt.Sprintf("You've mentioned '%s' %d times in the last %d days.", pattern.Subject, pattern.Frequency, ee.thresholds.TimeWindowDays),
		Rationale:   fmt.Sprintf("Based on frequency analysis, '%s' appears to be an area of significant interest or work. Adding this to my expertise will help me provide more relevant and nuanced responses.", pattern.Subject),
		Change: &PersonaChange{
			Field:     "identity.expertise",
			Operation: "add",
			Value:     expertise,
		},
		Confidence: pattern.Confidence,
		Status:     ProposalPending,
		PatternIDs: []string{pattern.ID},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// createSkillExpertiseProposal creates a proposal based on skill usage
func (ee *EvolutionEngine) createSkillExpertiseProposal(pattern *DetectedPattern) *EvolutionProposal {
	return &EvolutionProposal{
		ID:          ee.generateProposalID(),
		Type:        ExpertiseProposal,
		Title:       fmt.Sprintf("Add %s to expertise", pattern.Subject),
		Description: fmt.Sprintf("You've used the '%s' skill %d times in the last %d days.", pattern.Subject, pattern.Frequency, ee.thresholds.TimeWindowDays),
		Rationale:   fmt.Sprintf("Heavy usage of '%s' suggests it's a key part of your workflow. Adding this to my expertise will help me better understand your context and provide more helpful suggestions.", pattern.Subject),
		Change: &PersonaChange{
			Field:     "identity.expertise",
			Operation: "add",
			Value:     pattern.Subject,
		},
		Confidence: pattern.Confidence,
		Status:     ProposalPending,
		PatternIDs: []string{pattern.ID},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// createTemporalPreferenceProposal creates a proposal for temporal preferences
func (ee *EvolutionEngine) createTemporalPreferenceProposal(pattern *DetectedPattern) *EvolutionProposal {
	hour, _ := pattern.Metadata["peak_hour"].(int)
	timeLabel := ee.getTimeOfDayLabel(hour)

	return &EvolutionProposal{
		ID:          ee.generateProposalID(),
		Type:        PreferenceProposal,
		Title:       fmt.Sprintf("Note %s activity pattern", timeLabel),
		Description: fmt.Sprintf("You're most active during %s hours.", timeLabel),
		Rationale:   fmt.Sprintf("Analysis shows %.0f%% of your messages occur during %s hours. I can use this to time notifications and suggestions appropriately.", pattern.Confidence*100, timeLabel),
		Change: &PersonaChange{
			Field:     "user.preferences.active_hours",
			Operation: "add",
			Value:     timeLabel,
		},
		Confidence: pattern.Confidence,
		Status:     ProposalPending,
		PatternIDs: []string{pattern.ID},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// extractTopics extracts potential topics from text
func (ee *EvolutionEngine) extractTopics(text string) []string {
	var topics []string

	// Simple keyword extraction - in production, use NLP
	words := strings.Fields(strings.ToLower(text))
	wordCounts := make(map[string]int)

	// Common words to filter out
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "can": true,
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
		"my": true, "your": true, "his": true, "her": true, "its": true, "our": true, "their": true,
		"this": true, "that": true, "these": true, "those": true,
		"and": true, "or": true, "but": true, "if": true, "then": true, "else": true,
		"of": true, "in": true, "to": true, "for": true, "with": true, "on": true, "at": true,
		"by": true, "from": true, "as": true, "into": true, "through": true, "during": true,
		"before": true, "after": true, "above": true, "below": true, "up": true, "down": true,
		"out": true, "off": true, "over": true, "under": true, "again": true, "further": true,
		"once": true, "here": true, "there": true, "when": true, "where": true,
		"why": true, "how": true, "all": true, "any": true, "both": true, "each": true,
		"few": true, "more": true, "most": true, "other": true, "some": true, "such": true,
		"no": true, "nor": true, "not": true, "only": true, "own": true, "same": true,
		"so": true, "than": true, "too": true, "very": true, "just": true, "now": true,
	}

	for _, word := range words {
		// Clean word
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) < 3 {
			continue
		}
		if !stopWords[word] {
			wordCounts[word]++
		}
	}

	// Return words that appear to be topics (mentioned in this message)
	for word, count := range wordCounts {
		if count >= 1 && len(word) >= 4 {
			topics = append(topics, word)
		}
	}

	return topics
}

// normalizeExpertise normalizes an expertise string
func (ee *EvolutionEngine) normalizeExpertise(topic string) string {
	// Capitalize first letter
	topic = strings.ToLower(topic)
	if len(topic) > 0 {
		topic = strings.ToUpper(topic[:1]) + topic[1:]
	}
	return topic
}

// getTimeOfDayLabel returns a label for the hour
func (ee *EvolutionEngine) getTimeOfDayLabel(hour int) string {
	switch {
	case hour >= 5 && hour < 8:
		return "early morning"
	case hour >= 8 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 14:
		return "midday"
	case hour >= 14 && hour < 17:
		return "afternoon"
	case hour >= 17 && hour < 20:
		return "evening"
	case hour >= 20 && hour < 22:
		return "night"
	default:
		return "late night"
	}
}

// calculateFrequencyConfidence calculates confidence for frequency patterns
func (ee *EvolutionEngine) calculateFrequencyConfidence(pattern *DetectedPattern) float64 {
	// Base confidence on frequency
	baseConfidence := float64(pattern.Frequency) / float64(ee.thresholds.MinFrequency*2)
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	// Adjust for time span (longer = more confident)
	timeSpan := pattern.LastSeen.Sub(pattern.FirstSeen).Hours()
	timeFactor := timeSpan / (24 * 7) // Week factor
	if timeFactor > 1.0 {
		timeFactor = 1.0
	}

	return (baseConfidence*0.7 + timeFactor*0.3)
}

// calculateSkillConfidence calculates confidence for skill patterns
func (ee *EvolutionEngine) calculateSkillConfidence(pattern *DetectedPattern) float64 {
	// Base confidence on usage count
	baseConfidence := float64(pattern.Frequency) / float64(ee.thresholds.SkillUsageThreshold*2)
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// savePattern saves a detected pattern to the store
func (ee *EvolutionEngine) savePattern(ctx context.Context, pattern *DetectedPattern) error {
	// Save to store if not exists
	var existing DetectedPattern
	err := ee.store.DB().Where("id = ?", pattern.ID).First(&existing).Error
	if err != nil {
		// Create new
		return ee.store.DB().Create(pattern).Error
	}
	// Update existing
	return ee.store.DB().Model(&existing).Updates(map[string]interface{}{
		"frequency":  pattern.Frequency,
		"last_seen":  pattern.LastSeen,
		"confidence": pattern.Confidence,
	}).Error
}

// generatePatternID generates a unique pattern ID
func (ee *EvolutionEngine) generatePatternID() string {
	return fmt.Sprintf("pattern_%d", time.Now().UnixNano())
}

// generateProposalID generates a unique proposal ID
func (ee *EvolutionEngine) generateProposalID() string {
	return fmt.Sprintf("proposal_%d", time.Now().UnixNano())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

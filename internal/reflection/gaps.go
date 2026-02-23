// Package reflection implements the Reflection Engine for self-auditing memory system
package reflection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// GapAction represents the suggested action for a knowledge gap
type GapAction string

const (
	GapActionAskUser  GapAction = "ask_user"
	GapActionAutoFill GapAction = "auto_fill"
	GapActionDismiss  GapAction = "dismiss"
)

// GapStatus represents the status of a knowledge gap
type GapStatus string

const (
	GapStatusOpen    GapStatus = "open"
	GapStatusFilled  GapStatus = "filled"
	GapStatusIgnored GapStatus = "ignored"
)

// Gap represents a knowledge gap - topics mentioned frequently but not well documented
type Gap struct {
	ID              string     `gorm:"primaryKey" json:"id"`
	Topic           string     `gorm:"not null;index" json:"topic"`
	MentionCount    int        `json:"mention_count"`
	MemoryCount     int        `json:"memory_count"`
	GapRatio        float64    `json:"gap_ratio"` // memory_count / mention_count
	SuggestedAction string     `json:"suggested_action"`
	DetectedAt      time.Time  `json:"detected_at"`
	Status          string     `gorm:"default:open" json:"status"`
	FilledAt        *time.Time `json:"filled_at,omitempty"`

	// Transient fields
	SampleMentions []string `gorm:"-" json:"sample_mentions,omitempty"`
}

// GapAnalyzer identifies knowledge gaps in the memory system
type GapAnalyzer struct {
	llmClient         *llm.Client
	logger            *zap.Logger
	mentionThreshold  int     // Minimum mentions to consider a topic
	gapRatioThreshold float64 // Below this ratio is considered a gap
}

// NewGapAnalyzer creates a new gap analyzer
func NewGapAnalyzer(llmClient *llm.Client, logger *zap.Logger) *GapAnalyzer {
	return &GapAnalyzer{
		llmClient:         llmClient,
		logger:            logger,
		mentionThreshold:  10,  // Topic must be mentioned at least 10 times
		gapRatioThreshold: 0.3, // Less than 30% memory coverage is a gap
	}
}

// SetMentionThreshold sets the minimum mention threshold
func (ga *GapAnalyzer) SetMentionThreshold(threshold int) {
	ga.mentionThreshold = threshold
}

// SetGapRatioThreshold sets the gap ratio threshold
func (ga *GapAnalyzer) SetGapRatioThreshold(threshold float64) {
	ga.gapRatioThreshold = threshold
}

// IdentifyGaps finds under-documented topics in conversations
func (ga *GapAnalyzer) IdentifyGaps(ctx context.Context, conversations []store.Conversation, memories []store.Memory) ([]Gap, error) {
	ga.logger.Info("Starting gap analysis",
		zap.Int("conversation_count", len(conversations)),
		zap.Int("memory_count", len(memories)))

	// Extract topics from conversations
	topicMentions := ga.extractTopicsFromConversations(conversations)

	// Count memories per topic
	topicMemoryCounts := ga.countMemoriesPerTopic(memories, topicMentions)

	// Identify gaps
	var gaps []Gap
	for topic, mentionCount := range topicMentions {
		if mentionCount < ga.mentionThreshold {
			continue
		}

		memoryCount := topicMemoryCounts[topic]
		gapRatio := float64(memoryCount) / float64(mentionCount)

		if gapRatio < ga.gapRatioThreshold {
			gap := Gap{
				ID:              generateID("gap"),
				Topic:           topic,
				MentionCount:    mentionCount,
				MemoryCount:     memoryCount,
				GapRatio:        gapRatio,
				SuggestedAction: string(GapActionAskUser),
				DetectedAt:      time.Now(),
				Status:          string(GapStatusOpen),
			}
			gaps = append(gaps, gap)
		}
	}

	ga.logger.Info("Gap analysis complete", zap.Int("gaps_found", len(gaps)))
	return gaps, nil
}

// IdentifyGapsWithLLM uses LLM to extract topics and identify gaps more intelligently
func (ga *GapAnalyzer) IdentifyGapsWithLLM(ctx context.Context, conversations []store.Conversation, memories []store.Memory) ([]Gap, error) {
	ga.logger.Info("Starting LLM-based gap analysis")

	// Prepare conversation summaries for LLM
	summaries := ga.prepareConversationSummaries(conversations)

	// Use LLM to extract topics
	topics, err := ga.extractTopicsWithLLM(ctx, summaries)
	if err != nil {
		ga.logger.Warn("LLM topic extraction failed, falling back to simple method", zap.Error(err))
		return ga.IdentifyGaps(ctx, conversations, memories)
	}

	// Count memories per topic
	var gaps []Gap
	for topic, mentionCount := range topics {
		if mentionCount < ga.mentionThreshold {
			continue
		}

		memoryCount := ga.countMemoriesForTopic(memories, topic)
		gapRatio := float64(memoryCount) / float64(mentionCount)

		if gapRatio < ga.gapRatioThreshold {
			gap := Gap{
				ID:              generateID("gap"),
				Topic:           topic,
				MentionCount:    mentionCount,
				MemoryCount:     memoryCount,
				GapRatio:        gapRatio,
				SuggestedAction: string(GapActionAskUser),
				DetectedAt:      time.Now(),
				Status:          string(GapStatusOpen),
			}
			gaps = append(gaps, gap)
		}
	}

	return gaps, nil
}

// extractTopicsFromConversations extracts topics from conversation content
func (ga *GapAnalyzer) extractTopicsFromConversations(conversations []store.Conversation) map[string]int {
	topicMentions := make(map[string]int)

	for _, conv := range conversations {
		// Extract topics from conversation title
		titleTopics := ga.extractTopicsFromText(conv.Title)
		for _, topic := range titleTopics {
			topicMentions[topic]++
		}

		// Count the conversation itself as a mention for its main topic
		if conv.Title != "" {
			topicMentions[normalizeTopic(conv.Title)]++
		}
	}

	return topicMentions
}

// extractTopicsFromText extracts potential topics from text
func (ga *GapAnalyzer) extractTopicsFromText(text string) []string {
	var topics []string

	// Simple keyword extraction
	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		word = cleanWord(word)
		if ga.isSignificantWord(word) {
			topics = append(topics, word)
		}
	}

	return topics
}

// countMemoriesPerTopic counts how many memories exist for each topic
func (ga *GapAnalyzer) countMemoriesPerTopic(memories []store.Memory, topics map[string]int) map[string]int {
	counts := make(map[string]int)

	for topic := range topics {
		counts[topic] = ga.countMemoriesForTopic(memories, topic)
	}

	return counts
}

// countMemoriesForTopic counts memories related to a specific topic
func (ga *GapAnalyzer) countMemoriesForTopic(memories []store.Memory, topic string) int {
	count := 0
	topicLower := strings.ToLower(topic)

	for _, mem := range memories {
		contentLower := strings.ToLower(mem.Content)
		if strings.Contains(contentLower, topicLower) {
			count++
		}
	}

	return count
}

// prepareConversationSummaries prepares summaries for LLM analysis
func (ga *GapAnalyzer) prepareConversationSummaries(conversations []store.Conversation) string {
	var summaries []string

	for _, conv := range conversations {
		summary := fmt.Sprintf("- %s (messages: %d)", conv.Title, conv.MessageCount)
		summaries = append(summaries, summary)
	}

	if len(summaries) > 50 {
		summaries = summaries[:50] // Limit to avoid token limits
	}

	return strings.Join(summaries, "\n")
}

// extractTopicsWithLLM uses LLM to intelligently extract topics
func (ga *GapAnalyzer) extractTopicsWithLLM(ctx context.Context, summaries string) (map[string]int, error) {
	systemPrompt := `You are a topic extraction specialist. Analyze conversation summaries and extract main topics/themes.

Respond with a JSON object where keys are topic names and values are mention counts (how many conversations relate to this topic).

Example response:
{
  "docker": 15,
  "kubernetes": 12,
  "python": 8,
  "microservices": 5
}

Only include topics that appear multiple times. Be specific but not too granular."
`

	userPrompt := fmt.Sprintf("Analyze these conversation summaries and extract the main topics:\n\n%s", summaries)

	response, err := ga.llmClient.SimpleChat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var topics map[string]int
	if err := parseJSONResponse(response, &topics); err != nil {
		return nil, err
	}

	return topics, nil
}

// isSignificantWord checks if a word is significant enough to be a topic
func (ga *GapAnalyzer) isSignificantWord(word string) bool {
	if len(word) < 3 {
		return false
	}

	// Common stop words to exclude
	stopWords := []string{"the", "and", "for", "are", "but", "not", "you", "all", "can", "had", "her", "was", "one", "our", "out", "day", "get", "has", "him", "his", "how", "its", "may", "new", "now", "old", "see", "two", "who", "boy", "did", "she", "use", "her", "way", "many", "oil", "sit", "set", "run", "eat", "far", "sea", "eye", "ago", "off", "too", "any", "say", "man", "try", "ask", "end", "why", "let", "put", "say", "she", "try", "way", "own", "say", "too", "old", "tell", "very", "when", "much", "would", "there", "their", "what", "said", "each", "which", "will", "about", "could", "other", "after", "first", "never", "these", "think", "where", "being", "every", "great", "might", "shall", "still", "those", "while", "this", "that", "with", "have", "from", "they", "know", "want", "been", "good", "much", "some", "time", "very", "when", "come", "here", "just", "like", "long", "make", "many", "over", "such", "take", "than", "them", "well", "were"}

	for _, stop := range stopWords {
		if word == stop {
			return false
		}
	}

	return true
}

// cleanWord removes punctuation from a word
func cleanWord(word string) string {
	var result strings.Builder
	for _, r := range word {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	return strings.ToLower(result.String())
}

// normalizeTopic normalizes a topic string
func normalizeTopic(topic string) string {
	return strings.ToLower(strings.TrimSpace(topic))
}

// parseJSONResponse extracts and parses JSON from LLM response
func parseJSONResponse(response string, v interface{}) error {
	// Find JSON block
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start >= 0 && end > start {
		jsonStr := response[start : end+1]
		return parseJSON(jsonStr, v)
	}

	return fmt.Errorf("no JSON found in response")
}

// GetSeverity returns the severity level of the gap
func (g *Gap) GetSeverity() string {
	switch {
	case g.GapRatio < 0.1 && g.MentionCount > 20:
		return "high"
	case g.GapRatio < 0.2 && g.MentionCount > 15:
		return "medium"
	default:
		return "low"
	}
}

// GetImpactEstimate returns an estimate of the impact of filling this gap
func (g *Gap) GetImpactEstimate() string {
	switch g.GetSeverity() {
	case "high":
		return fmt.Sprintf("Filling this gap could improve responses on '%s' by ~30%%", g.Topic)
	case "medium":
		return fmt.Sprintf("Better context for %s-related queries", g.Topic)
	default:
		return fmt.Sprintf("Minor improvement in %s understanding", g.Topic)
	}
}

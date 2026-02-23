// Package reflection implements the Reflection Engine for self-auditing memory system
// Phase 5: Detects contradictions, redundancies, and gaps in the memory system
package reflection

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// Severity levels for contradictions
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// Status for contradictions
type ContradictionStatus string

const (
	StatusOpen     ContradictionStatus = "open"
	StatusResolved ContradictionStatus = "resolved"
	StatusIgnored  ContradictionStatus = "ignored"
)

// Contradiction represents a detected contradiction between two memories
type Contradiction struct {
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

	// Transient fields (not stored in DB)
	MemoryA *store.Memory `gorm:"-" json:"memory_a,omitempty"`
	MemoryB *store.Memory `gorm:"-" json:"memory_b,omitempty"`
}

// ContradictionDetector detects contradictions in memories using embeddings and LLM
type ContradictionDetector struct {
	llmClient           *llm.Client
	logger              *zap.Logger
	similarityThreshold float64
}

// NewContradictionDetector creates a new contradiction detector
func NewContradictionDetector(llmClient *llm.Client, logger *zap.Logger) *ContradictionDetector {
	return &ContradictionDetector{
		llmClient:           llmClient,
		logger:              logger,
		similarityThreshold: 0.8, // Default threshold for same topic detection
	}
}

// SetSimilarityThreshold sets the similarity threshold for topic matching
func (cd *ContradictionDetector) SetSimilarityThreshold(threshold float64) {
	cd.similarityThreshold = threshold
}

// Detect finds contradictions between memories
// Uses a two-phase approach:
// 1. Find semantically similar memories using embeddings (>0.8 similarity)
// 2. Use LLM to verify contradictions
func (cd *ContradictionDetector) Detect(ctx context.Context, memories []store.Memory) ([]Contradiction, error) {
	cd.logger.Info("Starting contradiction detection", zap.Int("memory_count", len(memories)))

	var contradictions []Contradiction
	checked := make(map[string]bool) // Track checked pairs to avoid duplicates

	// Compare each memory with others
	for i, memA := range memories {
		for j := i + 1; j < len(memories); j++ {
			memB := memories[j]

			// Create unique key for this pair
			pairKey := fmt.Sprintf("%s:%s", memA.ID, memB.ID)
			if checked[pairKey] {
				continue
			}
			checked[pairKey] = true

			// Check if memories have embeddings
			if len(memA.Embedding) == 0 || len(memB.Embedding) == 0 {
				continue
			}

			// Phase 1: Check similarity
			similarity, err := cd.cosineSimilarity(memA.Embedding, memB.Embedding)
			if err != nil {
				cd.logger.Warn("Failed to calculate similarity",
					zap.String("mem_a", memA.ID),
					zap.String("mem_b", memB.ID),
					zap.Error(err))
				continue
			}

			// Skip if not similar enough (not about the same topic)
			if similarity < cd.similarityThreshold {
				continue
			}

			cd.logger.Debug("Found similar memories",
				zap.String("mem_a", memA.ID),
				zap.String("mem_b", memB.ID),
				zap.Float64("similarity", similarity))

			// Phase 2: Use LLM to check for contradiction
			contradiction, err := cd.checkContradictionWithLLM(ctx, &memA, &memB, similarity)
			if err != nil {
				cd.logger.Warn("Failed to check contradiction with LLM",
					zap.String("mem_a", memA.ID),
					zap.String("mem_b", memB.ID),
					zap.Error(err))
				continue
			}

			if contradiction != nil {
				contradictions = append(contradictions, *contradiction)
			}
		}
	}

	cd.logger.Info("Contradiction detection complete",
		zap.Int("contradictions_found", len(contradictions)))

	return contradictions, nil
}

// DetectForNewMemories checks only new memories against existing ones
// This is used for the daily lightweight job
func (cd *ContradictionDetector) DetectForNewMemories(ctx context.Context, newMemories []store.Memory, existingMemories []store.Memory) ([]Contradiction, error) {
	cd.logger.Info("Checking new memories for contradictions",
		zap.Int("new_count", len(newMemories)),
		zap.Int("existing_count", len(existingMemories)))

	var contradictions []Contradiction

	// Compare each new memory with existing ones
	for _, newMem := range newMemories {
		for _, existingMem := range existingMemories {
			// Skip if same memory
			if newMem.ID == existingMem.ID {
				continue
			}

			// Check if memories have embeddings
			if len(newMem.Embedding) == 0 || len(existingMem.Embedding) == 0 {
				continue
			}

			// Check similarity
			similarity, err := cd.cosineSimilarity(newMem.Embedding, existingMem.Embedding)
			if err != nil {
				continue
			}

			if similarity < cd.similarityThreshold {
				continue
			}

			// Check for contradiction
			contradiction, err := cd.checkContradictionWithLLM(ctx, &newMem, &existingMem, similarity)
			if err != nil {
				continue
			}

			if contradiction != nil {
				contradictions = append(contradictions, *contradiction)
			}
		}
	}

	return contradictions, nil
}

// cosineSimilarity calculates cosine similarity between two embedding vectors
func (cd *ContradictionDetector) cosineSimilarity(embeddingA, embeddingB []byte) (float64, error) {
	// Parse embeddings from bytes (assuming float32 arrays)
	vecA, err := parseEmbedding(embeddingA)
	if err != nil {
		return 0, fmt.Errorf("failed to parse embedding A: %w", err)
	}

	vecB, err := parseEmbedding(embeddingB)
	if err != nil {
		return 0, fmt.Errorf("failed to parse embedding B: %w", err)
	}

	if len(vecA) != len(vecB) {
		return 0, fmt.Errorf("embeddings have different dimensions: %d vs %d", len(vecA), len(vecB))
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(vecA); i++ {
		dotProduct += vecA[i] * vecB[i]
		normA += vecA[i] * vecA[i]
		normB += vecB[i] * vecB[i]
	}

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("zero vector detected")
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	return similarity, nil
}

// parseEmbedding parses byte slice to float64 slice
// Assumes the embedding is stored as JSON array of floats
func parseEmbedding(data []byte) ([]float64, error) {
	// Try JSON format first
	var floats []float64
	if err := json.Unmarshal(data, &floats); err == nil {
		return floats, nil
	}

	// Try binary format (float32 array)
	if len(data)%4 == 0 {
		floats = make([]float64, len(data)/4)
		for i := 0; i < len(floats); i++ {
			bits := uint32(data[i*4]) | uint32(data[i*4+1])<<8 |
				uint32(data[i*4+2])<<16 | uint32(data[i*4+3])<<24
			floats[i] = float64(math.Float32frombits(bits))
		}
		return floats, nil
	}

	return nil, fmt.Errorf("unknown embedding format")
}

// checkContradictionWithLLM uses LLM to verify if two memories contradict
func (cd *ContradictionDetector) checkContradictionWithLLM(ctx context.Context, memA, memB *store.Memory, similarity float64) (*Contradiction, error) {
	systemPrompt := `You are a contradiction detection specialist. Analyze two memories and determine if they contradict each other.

A contradiction occurs when:
1. The memories make opposite claims about the same thing
2. One memory negates or reverses what the other says
3. They express opposing preferences, facts, or beliefs

IMPORTANT: Slight variations or additional details do NOT constitute contradictions. Only flag TRUE contradictions.

Respond in this exact JSON format:
{
  "is_contradiction": true/false,
  "severity": "high/medium/low",
  "description": "Brief explanation of the contradiction",
  "suggested_resolution": "How to resolve this contradiction"
}

Severity levels:
- high: Direct logical conflict, cannot both be true
- medium: Significant inconsistency in preferences or facts
- low: Minor inconsistency or contextual difference`

	userPrompt := fmt.Sprintf(`Analyze these two memories for contradictions:

Memory 1: "%s"
Memory 2: "%s"

Semantic Similarity: %.2f (1.0 = identical topic)

Are these memories contradictory?`, memA.Content, memB.Content, similarity)

	response, err := cd.llmClient.SimpleChat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response
	var result struct {
		IsContradiction     bool   `json:"is_contradiction"`
		Severity            string `json:"severity"`
		Description         string `json:"description"`
		SuggestedResolution string `json:"suggested_resolution"`
	}

	// Try to extract JSON from response (in case there's extra text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		response = response[jsonStart : jsonEnd+1]
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If JSON parsing fails, try to interpret text response
		return cd.parseTextResponse(memA.ID, memB.ID, response), nil
	}

	if !result.IsContradiction {
		return nil, nil
	}

	return &Contradiction{
		ID:                  generateID("contr"),
		MemoryAID:           memA.ID,
		MemoryBID:           memB.ID,
		Severity:            result.Severity,
		Description:         result.Description,
		SuggestedResolution: result.SuggestedResolution,
		DetectedAt:          time.Now(),
		Status:              string(StatusOpen),
		MemoryA:             memA,
		MemoryB:             memB,
	}, nil
}

// parseTextResponse attempts to parse a non-JSON LLM response
func (cd *ContradictionDetector) parseTextResponse(memAID, memBID string, response string) *Contradiction {
	responseLower := strings.ToLower(response)

	// Check if contradiction is mentioned
	if !strings.Contains(responseLower, "contradiction") &&
		!strings.Contains(responseLower, "conflict") &&
		!strings.Contains(responseLower, "opposite") {
		return nil
	}

	// Determine severity
	severity := SeverityLow
	if strings.Contains(responseLower, "high") || strings.Contains(responseLower, "direct") {
		severity = SeverityHigh
	} else if strings.Contains(responseLower, "medium") || strings.Contains(responseLower, "moderate") {
		severity = SeverityMedium
	}

	return &Contradiction{
		ID:          generateID("contr"),
		MemoryAID:   memAID,
		MemoryBID:   memBID,
		Severity:    string(severity),
		Description: strings.TrimSpace(response),
		DetectedAt:  time.Now(),
		Status:      string(StatusOpen),
	}
}

// generateID creates a unique ID with prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().Unix(), randomString(6))
}

// randomString generates a random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

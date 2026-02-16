package knowledge

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Extractor extracts entities and relationships from text
type Extractor struct {
	logger *zap.Logger
	store  *Store
}

// NewExtractor creates a new extractor
func NewExtractor(store *Store, logger *zap.Logger) *Extractor {
	return &Extractor{
		logger: logger,
		store:  store,
	}
}

// ExtractionOptions contains options for extraction
type ExtractionOptions struct {
	ExtractEntities      bool
	ExtractRelationships bool
	ExtractMemories      bool
	MinConfidence        float64
	ConversationID       string
}

// DefaultExtractionOptions returns default options
func DefaultExtractionOptions() ExtractionOptions {
	return ExtractionOptions{
		ExtractEntities:      true,
		ExtractRelationships: true,
		ExtractMemories:      true,
		MinConfidence:        0.6,
	}
}

// ExtractFromText extracts knowledge from plain text using rule-based approach
func (e *Extractor) ExtractFromText(ctx context.Context, userID, text string, opts ExtractionOptions) (*ExtractionResult, error) {
	result := &ExtractionResult{
		Entities:      []ExtractedEntity{},
		Relationships: []ExtractedRelationship{},
		Memories:      []ExtractedMemory{},
	}
	
	// Extract entities
	if opts.ExtractEntities {
		entities := e.extractEntities(text)
		result.Entities = e.filterByConfidence(entities, opts.MinConfidence)
	}
	
	// Extract relationships
	if opts.ExtractRelationships && len(result.Entities) > 1 {
		relationships := e.extractRelationships(text, result.Entities)
		result.Relationships = e.filterRelByConfidence(relationships, opts.MinConfidence)
	}
	
	// Extract memories/facts
	if opts.ExtractMemories {
		memories := e.extractMemories(text, result.Entities)
		result.Memories = e.filterMemByConfidence(memories, opts.MinConfidence)
	}
	
	// Calculate overall confidence
	if len(result.Entities) > 0 {
		totalConf := 0.0
		for _, ent := range result.Entities {
			totalConf += ent.Confidence
		}
		result.Confidence = totalConf / float64(len(result.Entities))
	}
	
	return result, nil
}

// ProcessAndStore processes text and stores extracted knowledge
func (e *Extractor) ProcessAndStore(ctx context.Context, userID, text, conversationID string) error {
	opts := DefaultExtractionOptions()
	opts.ConversationID = conversationID
	
	result, err := e.ExtractFromText(ctx, userID, text, opts)
	if err != nil {
		return err
	}
	
	// Store entities and build ID mapping
	entityIDMap := make(map[string]string) // extracted name -> stored ID
	
	for _, ent := range result.Entities {
		stored, err := e.store.FindOrCreateEntity(userID, ent.Name, ent.Type)
		if err != nil {
			e.logger.Error("Failed to store entity", zap.Error(err), zap.String("name", ent.Name))
			continue
		}
		
		// Update entity with additional info
		if ent.Description != "" && stored.Description == "" {
			stored.Description = ent.Description
		}
		if len(ent.Aliases) > 0 {
			for _, alias := range ent.Aliases {
				stored.AddAlias(alias)
			}
		}
		stored.Confidence = ent.Confidence
		stored.SourceConversation = conversationID
		
		if err := e.store.UpdateEntity(stored); err != nil {
			e.logger.Error("Failed to update entity", zap.Error(err))
		}
		
		entityIDMap[ent.Name] = stored.ID
	}
	
	// Store relationships
	for _, rel := range result.Relationships {
		sourceID, ok1 := entityIDMap[rel.Source]
		targetID, ok2 := entityIDMap[rel.Target]
		
		if !ok1 || !ok2 {
			continue // Skip if entities weren't stored
		}
		
		_, err := e.store.FindOrCreateRelationship(userID, sourceID, targetID, rel.Type)
		if err != nil {
			e.logger.Error("Failed to store relationship", zap.Error(err))
		}
	}
	
	// Store memories
	for _, mem := range result.Memories {
		memory := &Memory{
			UserID:         userID,
			Content:        mem.Content,
			Type:           mem.Type,
			Category:       mem.Category,
			ConversationID: conversationID,
			Confidence:     mem.Confidence,
			Importance:     e.calculateImportance(mem),
		}
		
		// Link entities
		for _, entityName := range mem.Entities {
			if entityID, ok := entityIDMap[entityName]; ok {
				memory.LinkEntity(entityID)
			}
		}
		
		// Try to parse timestamp
		if ts := e.extractTimestamp(mem.Content); ts != nil {
			memory.Timestamp = ts
		}
		
		if err := e.store.CreateMemory(memory); err != nil {
			e.logger.Error("Failed to store memory", zap.Error(err))
		}
	}
	
	e.logger.Info("Knowledge extraction complete",
		zap.Int("entities", len(result.Entities)),
		zap.Int("relationships", len(result.Relationships)),
		zap.Int("memories", len(result.Memories)),
	)
	
	return nil
}

// extractEntities extracts entities using pattern matching
func (e *Extractor) extractEntities(text string) []ExtractedEntity {
	entities := []ExtractedEntity{}
	
	// Person patterns
	personPatterns := []struct {
		pattern *regexp.Regexp
		source  string
	}{
		{regexp.MustCompile(`(?i)(?:I met|met with|talked to|called|spoke with)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`), "met"},
		{regexp.MustCompile(`(?i)(?:my friend|my colleague|my boss|my manager)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`), "relationship"},
		{regexp.MustCompile(`(?i)(?:\b[A-Z][a-z]+\b)(?:\s+said|\s+told me|\s+mentioned)`), "speaker"},
	}
	
	for _, pp := range personPatterns {
		matches := pp.pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				name := text[match[2]:match[3]]
				entities = append(entities, ExtractedEntity{
					Name:       name,
					Type:       string(EntityTypePerson),
					Confidence: 0.8,
					StartPos:   match[2],
					EndPos:     match[3],
				})
			}
		}
	}
	
	// Place patterns
	placePatterns := []struct {
		pattern *regexp.Regexp
		type_   string
	}{
		{regexp.MustCompile(`(?i)(?:at|in|from)\s+(?:the\s+)?([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?\s+(?:Cafe|Restaurant|Shop|Store|Mall|Park|Hotel|Hospital|School|University|Office|Building|Street|Avenue|Road|Plaza|Center|Centre))`), string(EntityTypePlace)},
		{regexp.MustCompile(`(?i)(?:at|in)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2})`), string(EntityTypePlace)},
	}
	
	for _, pp := range placePatterns {
		matches := pp.pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				name := text[match[2]:match[3]]
				// Filter out common non-places
				if !e.isCommonWord(name) {
					entities = append(entities, ExtractedEntity{
						Name:       name,
						Type:       pp.type_,
						Confidence: 0.7,
						StartPos:   match[2],
						EndPos:     match[3],
					})
				}
			}
		}
	}
	
	// Organization patterns
	orgPattern := regexp.MustCompile(`(?i)(?:at|for|with)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?\s+(?:Inc|LLC|Corp|Corporation|Ltd|Company|Co|Group|Team|Department|Dept))`)
	matches := orgPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			name := text[match[2]:match[3]]
			entities = append(entities, ExtractedEntity{
				Name:       name,
				Type:       string(EntityTypeOrganization),
				Confidence: 0.75,
				StartPos:   match[2],
				EndPos:     match[3],
			})
		}
	}
	
	// Time references
	timePatterns := []string{
		`\b(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\b`,
		`\b(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2}(?:st|nd|rd|th)?\b`,
		`\b(?:last|next|this)\s+(?:week|month|year|Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\b`,
		`\b\d{1,2}:\d{2}\s*(?:AM|PM|am|pm)?\b`,
	}
	
	for _, pattern := range timePatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindAllStringIndex(text, -1)
		for _, match := range matches {
			timeRef := text[match[0]:match[1]]
			entities = append(entities, ExtractedEntity{
				Name:       timeRef,
				Type:       string(EntityTypeTime),
				Confidence: 0.9,
				StartPos:   match[0],
				EndPos:     match[1],
			})
		}
	}
	
	return e.deduplicateEntities(entities)
}

// extractRelationships extracts relationships between entities
func (e *Extractor) extractRelationships(text string, entities []ExtractedEntity) []ExtractedRelationship {
	relationships := []ExtractedRelationship{}
	
	// Relationship patterns
	patterns := []struct {
		pattern *regexp.Regexp
		relType string
	}{
		{regexp.MustCompile(`(?i)(\w+)\s+(?:works?\s+(?:at|for)|employed\s+(?:at|by))\s+(\w+)`), string(RelTypeWorksAt)},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:lives?\s+in|located\s+in)\s+(\w+)`), string(RelTypeLivesIn)},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:met|saw|visited)\s+(\w+)`), string(RelTypeMetAt)},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:is\s+(?:married\s+to|dating))\s+(\w+)`), string(RelTypeMarriedTo)},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:is\s+(?:friends?\s+with|knows))\s+(\w+)`), string(RelTypeFriendOf)},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:is\s+(?:a\s+)?(?:colleague|coworker)\s+(?:of|with))\s+(\w+)`), string(RelTypeColleagueOf)},
	}
	
	// Build name to entity map for lookup
	nameMap := make(map[string]ExtractedEntity)
	for _, ent := range entities {
		nameMap[strings.ToLower(ent.Name)] = ent
	}
	
	for _, p := range patterns {
		matches := p.pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				sourceName := strings.ToLower(match[1])
				targetName := strings.ToLower(match[2])
				
				_, sourceExists := nameMap[sourceName]
				_, targetExists := nameMap[targetName]
				
				if sourceExists && targetExists {
					relationships = append(relationships, ExtractedRelationship{
						Source:     match[1],
						Target:     match[2],
						Type:       p.relType,
						Confidence: 0.75,
					})
				}
			}
		}
	}
	
	return relationships
}

// extractMemories extracts memories/facts from text
func (e *Extractor) extractMemories(text string, entities []ExtractedEntity) []ExtractedMemory {
	memories := []ExtractedMemory{}
	
	// Fact patterns
	factPatterns := []struct {
		pattern     *regexp.Regexp
		memType     string
		category    string
		confidence  float64
	}{
		{regexp.MustCompile(`(?i)(?:I\s+(?:like|love|enjoy|prefer))\s+(.+?)(?:\.|,|;|$)`), string(MemoryTypePreference), "personal", 0.85},
		{regexp.MustCompile(`(?i)(?:I\s+(?:hate|dislike|can't\s+stand))\s+(.+?)(?:\.|,|;|$)`), string(MemoryTypePreference), "personal", 0.85},
		{regexp.MustCompile(`(?i)(?:I\s+(?:want|plan|intend)\s+to)\s+(.+?)(?:\.|,|;|$)`), string(MemoryTypeGoal), "personal", 0.8},
		{regexp.MustCompile(`(?i)(?:I\s+(?:need|must|have\s+to))\s+(.+?)(?:\.|,|;|$)`), string(MemoryTypeGoal), "personal", 0.8},
		{regexp.MustCompile(`(?i)(?:I\s+(?:am|work\s+as)\s+(?:a\s+)?)([^.]+(?:engineer|developer|manager|designer|teacher|doctor|lawyer|consultant|analyst))`), string(MemoryTypeFact), "professional", 0.85},
		{regexp.MustCompile(`(?i)(?:my\s+(?:birthday|anniversary)\s+is\s+(?:on\s+)?)([^.]+)`), string(MemoryTypeFact), "personal", 0.9},
	}
	
	for _, fp := range factPatterns {
		matches := fp.pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				content := strings.TrimSpace(match[0])
				memories = append(memories, ExtractedMemory{
					Content:    content,
					Type:       fp.memType,
					Category:   fp.category,
					Entities:   e.extractEntityNames(content, entities),
					Confidence: fp.confidence,
				})
			}
		}
	}
	
	// Event patterns
	eventPattern := regexp.MustCompile(`(?i)(?:I\s+(?:went|visited|attended|had))\s+(.+?)(?:\.|,|;|$)`)
	matches := eventPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			content := strings.TrimSpace(match[0])
			memories = append(memories, ExtractedMemory{
				Content:    content,
				Type:       string(MemoryTypeEvent),
				Category:   "life",
				Entities:   e.extractEntityNames(content, entities),
				Confidence: 0.75,
			})
		}
	}
	
	return memories
}

// Helper methods

func (e *Extractor) filterByConfidence(entities []ExtractedEntity, minConf float64) []ExtractedEntity {
	filtered := []ExtractedEntity{}
	for _, ent := range entities {
		if ent.Confidence >= minConf {
			filtered = append(filtered, ent)
		}
	}
	return filtered
}

func (e *Extractor) filterRelByConfidence(rels []ExtractedRelationship, minConf float64) []ExtractedRelationship {
	filtered := []ExtractedRelationship{}
	for _, rel := range rels {
		if rel.Confidence >= minConf {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}

func (e *Extractor) filterMemByConfidence(memories []ExtractedMemory, minConf float64) []ExtractedMemory {
	filtered := []ExtractedMemory{}
	for _, mem := range memories {
		if mem.Confidence >= minConf {
			filtered = append(filtered, mem)
		}
	}
	return filtered
}

func (e *Extractor) deduplicateEntities(entities []ExtractedEntity) []ExtractedEntity {
	seen := make(map[string]bool)
	unique := []ExtractedEntity{}
	
	for _, ent := range entities {
		key := strings.ToLower(ent.Name) + "|" + ent.Type
		if !seen[key] {
			seen[key] = true
			unique = append(unique, ent)
		}
	}
	
	return unique
}

func (e *Extractor) extractEntityNames(text string, entities []ExtractedEntity) []string {
	names := []string{}
	textLower := strings.ToLower(text)
	
	for _, ent := range entities {
		if strings.Contains(textLower, strings.ToLower(ent.Name)) {
			names = append(names, ent.Name)
		}
	}
	
	return names
}

func (e *Extractor) isCommonWord(word string) bool {
	common := map[string]bool{
		"the": true, "a": true, "an": true, "this": true, "that": true,
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
		"my": true, "your": true, "his": true, "her": true, "its": true, "our": true, "their": true,
		"is": true, "am": true, "are": true, "was": true, "were": true,
		"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true, "might": true,
		"and": true, "or": true, "but": true, "so": true, "because": true, "if": true, "then": true,
		"what": true, "when": true, "where": true, "why": true, "how": true, "who": true,
		"today": true, "tomorrow": true, "yesterday": true, "now": true, "later": true,
	}
	
	return common[strings.ToLower(word)]
}

func (e *Extractor) extractTimestamp(text string) *time.Time {
	// Try to parse dates like "last Tuesday", "January 15"
	// For now, return nil - would use the tasks date parser in production
	return nil
}

func (e *Extractor) calculateImportance(mem ExtractedMemory) int {
	// Calculate importance based on type and content
	baseImportance := 5
	
	switch MemoryType(mem.Type) {
	case MemoryTypePreference:
		baseImportance = 7
	case MemoryTypeGoal:
		baseImportance = 8
	case MemoryTypeRelationship:
		baseImportance = 9
	}
	
	// Adjust based on confidence
	if mem.Confidence > 0.9 {
		baseImportance += 1
	}
	
	if baseImportance > 10 {
		baseImportance = 10
	}
	
	return baseImportance
}

// ExtractFromConversation extracts knowledge from a full conversation
func (e *Extractor) ExtractFromConversation(ctx context.Context, userID string, messages []struct {
	Role    string
	Content string
}, conversationID string) error {
	// Combine all user messages
	var combinedText strings.Builder
	for _, msg := range messages {
		if msg.Role == "user" {
			combinedText.WriteString(msg.Content)
			combinedText.WriteString(" ")
		}
	}
	
	if combinedText.Len() == 0 {
		return nil
	}
	
	return e.ProcessAndStore(ctx, userID, combinedText.String(), conversationID)
}

// SerializeResult serializes extraction result to JSON
func (e *Extractor) SerializeResult(result *ExtractionResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

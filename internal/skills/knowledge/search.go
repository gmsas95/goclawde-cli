package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Search provides semantic and structured search over the knowledge graph
type Search struct {
	store  *Store
	logger *zap.Logger
}

// NewSearch creates a new search instance
func NewSearch(store *Store, logger *zap.Logger) *Search {
	return &Search{
		store:  store,
		logger: logger,
	}
}

// SearchQuery represents a search query
type SearchQuery struct {
	Text         string
	EntityTypes  []string
	MemoryTypes  []string
	Categories   []string
	TimeRange    *TimeRange
	Entities     []string // Specific entity names to search
	Limit        int
}

// TimeRange represents a time range for search
type TimeRange struct {
	Start *time.Time
	End   *time.Time
}

// SearchResult represents search results
type SearchResult struct {
	Entities      []EntityResult  `json:"entities"`
	Memories      []MemoryResult  `json:"memories"`
	Relationships []RelResult     `json:"relationships"`
	Answer        string          `json:"answer,omitempty"`
	Confidence    float64         `json:"confidence"`
}

// EntityResult represents an entity in search results
type EntityResult struct {
	Entity     Entity   `json:"entity"`
	Relevance  float64  `json:"relevance"`
	Matches    []string `json:"matches"`
}

// MemoryResult represents a memory in search results
type MemoryResult struct {
	Memory    Memory   `json:"memory"`
	Relevance float64  `json:"relevance"`
	Matches   []string `json:"matches"`
}

// RelResult represents a relationship in search results
type RelResult struct {
	Relationship Relationship `json:"relationship"`
	Source       Entity       `json:"source"`
	Target       Entity       `json:"target"`
	Relevance    float64      `json:"relevance"`
}

// Execute performs a search query
func (s *Search) Execute(ctx context.Context, userID string, query SearchQuery) (*SearchResult, error) {
	result := &SearchResult{
		Entities:      []EntityResult{},
		Memories:      []MemoryResult{},
		Relationships: []RelResult{},
	}
	
	// Search entities
	entities, err := s.searchEntities(userID, query)
	if err != nil {
		s.logger.Error("Entity search failed", zap.Error(err))
	}
	result.Entities = entities
	
	// Search memories
	memories, err := s.searchMemories(userID, query)
	if err != nil {
		s.logger.Error("Memory search failed", zap.Error(err))
	}
	result.Memories = memories
	
	// If we found specific entities, get their relationships
	if len(entities) > 0 {
		for _, er := range entities {
			rels, err := s.getEntityRelationships(er.Entity.ID)
			if err != nil {
				continue
			}
			for _, rel := range rels {
				result.Relationships = append(result.Relationships, RelResult{
					Relationship: rel.Relationship,
					Source:       rel.Source,
					Target:       rel.Target,
					Relevance:    er.Relevance * 0.9,
				})
			}
		}
	}
	
	// Calculate overall confidence
	if len(result.Entities) > 0 || len(result.Memories) > 0 {
		result.Confidence = 0.8
	}
	
	return result, nil
}

// QueryAnswer answers a natural language question using the knowledge graph
func (s *Search) QueryAnswer(ctx context.Context, userID, question string) (*SearchResult, error) {
	// Parse the question to understand what's being asked
	queryType, entities, timeRef := s.parseQuestion(question)
	
	s.logger.Info("Processing question",
		zap.String("type", queryType),
		zap.Strings("entities", entities),
		zap.String("time", timeRef),
	)
	
	// Build search query based on question type
	query := SearchQuery{
		Text:  question,
		Limit: 10,
	}
	
	// Add time range if detected
	if timeRef != "" {
		query.TimeRange = s.parseTimeReference(timeRef)
	}
	
	// Add entity names
	query.Entities = entities
	
	// Execute search
	result, err := s.Execute(ctx, userID, query)
	if err != nil {
		return nil, err
	}
	
	// Generate answer based on results
	result.Answer = s.generateAnswer(question, queryType, result)
	
	return result, nil
}

// searchEntities searches for matching entities
func (s *Search) searchEntities(userID string, query SearchQuery) ([]EntityResult, error) {
	results := []EntityResult{}
	
	// If specific entity names provided, search for those
	if len(query.Entities) > 0 {
		for _, name := range query.Entities {
			entity, err := s.store.GetEntityByName(userID, name)
			if err != nil {
				continue
			}
			if entity != nil {
				results = append(results, EntityResult{
					Entity:    *entity,
					Relevance: 1.0,
					Matches:   []string{"name"},
				})
			}
		}
	}
	
	// Text-based search
	if query.Text != "" {
		// Search by name
		entities, err := s.store.SearchEntities(userID, query.Text, query.Limit)
		if err != nil {
			return results, err
		}
		
		for _, ent := range entities {
			// Calculate relevance
			relevance := s.calculateEntityRelevance(ent, query)
			
			// Check if already in results
			found := false
			for _, r := range results {
				if r.Entity.ID == ent.ID {
					found = true
					break
				}
			}
			
			if !found {
				results = append(results, EntityResult{
					Entity:    ent,
					Relevance: relevance,
					Matches:   s.getEntityMatches(ent, query.Text),
				})
			}
		}
	}
	
	// Filter by entity types if specified
	if len(query.EntityTypes) > 0 {
		filtered := []EntityResult{}
		for _, r := range results {
			for _, t := range query.EntityTypes {
				if r.Entity.Type == t {
					filtered = append(filtered, r)
					break
				}
			}
		}
		results = filtered
	}
	
	return results, nil
}

// searchMemories searches for matching memories
func (s *Search) searchMemories(userID string, query SearchQuery) ([]MemoryResult, error) {
	filters := MemoryFilters{
		Search: query.Text,
		Limit:  query.Limit,
	}
	
	if query.TimeRange != nil {
		filters.Since = query.TimeRange.Start
		filters.Until = query.TimeRange.End
	}
	
	// If specific entity IDs are known, filter by them
	if len(query.Entities) > 0 {
		// First get entity IDs
		for _, name := range query.Entities {
			entity, err := s.store.GetEntityByName(userID, name)
			if err != nil || entity == nil {
				continue
			}
			filters.EntityID = entity.ID
			break // Use first found entity
		}
	}
	
	memories, err := s.store.GetMemories(userID, filters)
	if err != nil {
		return nil, err
	}
	
	results := make([]MemoryResult, len(memories))
	for i, mem := range memories {
		results[i] = MemoryResult{
			Memory:    mem,
			Relevance: s.calculateMemoryRelevance(mem, query),
			Matches:   s.getMemoryMatches(mem, query.Text),
		}
	}
	
	return results, nil
}

// getEntityRelationships gets relationships for an entity
func (s *Search) getEntityRelationships(entityID string) ([]RelationshipWithContext, error) {
	return s.store.GetRelationships(entityID)
}

// parseQuestion parses a natural language question
func (s *Search) parseQuestion(question string) (queryType string, entities []string, timeRef string) {
	questionLower := strings.ToLower(question)
	
	// Determine query type
	switch {
	case strings.Contains(questionLower, "who"):
		queryType = "who"
	case strings.Contains(questionLower, "where"):
		queryType = "where"
	case strings.Contains(questionLower, "when"):
		queryType = "when"
	case strings.Contains(questionLower, "what"):
		queryType = "what"
	case strings.Contains(questionLower, "how"):
		queryType = "how"
	default:
		queryType = "general"
	}
	
	// Extract time references
	timePatterns := []string{
		"last week", "last month", "last year",
		"this week", "this month", "this year",
		"yesterday", "today", "tomorrow",
		"recently", "lately",
	}
	
	for _, pattern := range timePatterns {
		if strings.Contains(questionLower, pattern) {
			timeRef = pattern
			break
		}
	}
	
	// Extract potential entity names (capitalized words)
	words := strings.Fields(question)
	for i, word := range words {
		// Clean word
		cleanWord := strings.TrimRight(word, "?.,;:!")
		
		// Check if capitalized and not sentence start
		if len(cleanWord) > 1 && cleanWord[0] >= 'A' && cleanWord[0] <= 'Z' {
			if i > 0 || (i == 0 && !strings.Contains("Who What Where When Why How", cleanWord)) {
				entities = append(entities, cleanWord)
			}
		}
	}
	
	return queryType, entities, timeRef
}

// parseTimeReference converts a time reference to a time range
func (s *Search) parseTimeReference(ref string) *TimeRange {
	now := time.Now()
	tr := &TimeRange{}
	
	switch strings.ToLower(ref) {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, 1)
		tr.Start = &start
		tr.End = &end
		
	case "yesterday":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		start := end.AddDate(0, 0, -1)
		tr.Start = &start
		tr.End = &end
		
	case "last week":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		start := end.AddDate(0, 0, -7)
		tr.Start = &start
		tr.End = &end
		
	case "last month":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		start := end.AddDate(0, -1, 0)
		tr.Start = &start
		tr.End = &end
		
	case "recently", "lately":
		end := now
		start := end.AddDate(0, 0, -14) // Last 2 weeks
		tr.Start = &start
		tr.End = &end
	}
	
	return tr
}

// generateAnswer generates a natural language answer
func (s *Search) generateAnswer(question, queryType string, result *SearchResult) string {
	if len(result.Entities) == 0 && len(result.Memories) == 0 {
		return "I don't have any information about that in my memory."
	}
	
	// Build answer based on query type
	switch queryType {
	case "who":
		return s.generateWhoAnswer(result)
	case "where":
		return s.generateWhereAnswer(result)
	case "when":
		return s.generateWhenAnswer(result)
	case "what":
		return s.generateWhatAnswer(result)
	default:
		return s.generateGeneralAnswer(result)
	}
}

func (s *Search) generateWhoAnswer(result *SearchResult) string {
	persons := []string{}
	for _, er := range result.Entities {
		if er.Entity.Type == string(EntityTypePerson) {
			persons = append(persons, er.Entity.Name)
		}
	}
	
	if len(persons) == 1 {
		return fmt.Sprintf("That would be %s.", persons[0])
	} else if len(persons) > 1 {
		return fmt.Sprintf("I found these people: %s.", strings.Join(persons, ", "))
	}
	
	// Check memories for person mentions
	for _, mr := range result.Memories {
		if strings.Contains(mr.Memory.Content, "met") || strings.Contains(mr.Memory.Content, "saw") {
			return mr.Memory.Content
		}
	}
	
	return "I'm not sure who you're referring to."
}

func (s *Search) generateWhereAnswer(result *SearchResult) string {
	places := []string{}
	for _, er := range result.Entities {
		if er.Entity.Type == string(EntityTypePlace) {
			places = append(places, er.Entity.Name)
		}
	}
	
	if len(places) == 1 {
		return fmt.Sprintf("That would be at %s.", places[0])
	} else if len(places) > 1 {
		return fmt.Sprintf("I found these places: %s.", strings.Join(places, ", "))
	}
	
	// Check relationships for location info
	for _, rel := range result.Relationships {
		if rel.Relationship.Type == string(RelTypeLocatedIn) || rel.Relationship.Type == string(RelTypeLivesIn) {
			return fmt.Sprintf("%s is at %s.", rel.Source.Name, rel.Target.Name)
		}
	}
	
	return "I'm not sure about the location."
}

func (s *Search) generateWhenAnswer(result *SearchResult) string {
	// Look for time entities or memory timestamps
	for _, er := range result.Entities {
		if er.Entity.Type == string(EntityTypeTime) {
			return fmt.Sprintf("That was %s.", er.Entity.Name)
		}
	}
	
	// Check memories for timestamps
	for _, mr := range result.Memories {
		if mr.Memory.Timestamp != nil {
			return fmt.Sprintf("That was on %s.", mr.Memory.Timestamp.Format("January 2, 2006"))
		}
		if mr.Memory.DateText != "" {
			return fmt.Sprintf("That was %s.", mr.Memory.DateText)
		}
	}
	
	return "I'm not sure about the exact time."
}

func (s *Search) generateWhatAnswer(result *SearchResult) string {
	if len(result.Memories) > 0 {
		return result.Memories[0].Memory.Content
	}
	
	if len(result.Entities) > 0 {
		ent := result.Entities[0].Entity
		if ent.Description != "" {
			return fmt.Sprintf("%s is %s.", ent.Name, ent.Description)
		}
		return fmt.Sprintf("I know about %s (%s).", ent.Name, ent.Type)
	}
	
	return "I'm not sure about that."
}

func (s *Search) generateGeneralAnswer(result *SearchResult) string {
	parts := []string{}
	
	if len(result.Entities) > 0 {
		entNames := []string{}
		for _, er := range result.Entities {
			entNames = append(entNames, er.Entity.Name)
		}
		parts = append(parts, fmt.Sprintf("I found information about: %s.", strings.Join(entNames, ", ")))
	}
	
	if len(result.Memories) > 0 {
		parts = append(parts, result.Memories[0].Memory.Content)
	}
	
	if len(parts) == 0 {
		return "I found some related information but I'm not sure how to answer that specifically."
	}
	
	return strings.Join(parts, " ")
}

// Helper methods

func (s *Search) calculateEntityRelevance(ent Entity, query SearchQuery) float64 {
	relevance := 0.5
	
	// Boost by mention count (more mentioned = more relevant)
	if ent.MentionCount > 5 {
		relevance += 0.1
	}
	if ent.MentionCount > 10 {
		relevance += 0.1
	}
	
	// Boost by recency
	if ent.LastMentioned != nil {
		daysSince := time.Since(*ent.LastMentioned).Hours() / 24
		if daysSince < 7 {
			relevance += 0.1
		}
	}
	
	// Boost by importance
	relevance += float64(ent.Importance) / 100.0
	
	if relevance > 1.0 {
		relevance = 1.0
	}
	
	return relevance
}

func (s *Search) calculateMemoryRelevance(mem Memory, query SearchQuery) float64 {
	relevance := 0.5
	
	// Boost by importance
	relevance += float64(mem.Importance) / 20.0
	
	// Boost by access count
	if mem.AccessCount > 5 {
		relevance += 0.1
	}
	
	// Boost by recency
	if mem.LastAccessed != nil {
		daysSince := time.Since(*mem.LastAccessed).Hours() / 24
		if daysSince < 7 {
			relevance += 0.1
		}
	}
	
	if relevance > 1.0 {
		relevance = 1.0
	}
	
	return relevance
}

func (s *Search) getEntityMatches(ent Entity, query string) []string {
	matches := []string{}
	queryLower := strings.ToLower(query)
	
	if strings.Contains(strings.ToLower(ent.Name), queryLower) {
		matches = append(matches, "name")
	}
	if strings.Contains(strings.ToLower(ent.Description), queryLower) {
		matches = append(matches, "description")
	}
	
	return matches
}

func (s *Search) getMemoryMatches(mem Memory, query string) []string {
	matches := []string{}
	queryLower := strings.ToLower(query)
	
	if strings.Contains(strings.ToLower(mem.Content), queryLower) {
		matches = append(matches, "content")
	}
	if strings.Contains(strings.ToLower(mem.Summary), queryLower) {
		matches = append(matches, "summary")
	}
	
	return matches
}

// FindPath finds a path between two entities
func (s *Search) FindPath(ctx context.Context, sourceID, targetID string, maxDepth int) ([]Relationship, error) {
	// Simple BFS to find path
	if maxDepth <= 0 {
		maxDepth = 3
	}
	
	visited := make(map[string]bool)
	queue := [][]string{{sourceID}}
	
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		
		currentID := path[len(path)-1]
		
		if currentID == targetID {
			// Found path, get relationships
			return s.getRelationshipsForPath(path)
		}
		
		if len(path) >= maxDepth {
			continue
		}
		
		if visited[currentID] {
			continue
		}
		visited[currentID] = true
		
		// Get neighbors
		rels, err := s.store.GetRelationships(currentID)
		if err != nil {
			continue
		}
		
		for _, rel := range rels {
			nextID := rel.Target.ID
			if rel.Source.ID != currentID {
				nextID = rel.Source.ID
			}
			
			if !visited[nextID] {
				newPath := append([]string{}, path...)
				newPath = append(newPath, nextID)
				queue = append(queue, newPath)
			}
		}
	}
	
	return nil, fmt.Errorf("no path found between entities")
}

func (s *Search) getRelationshipsForPath(entityIDs []string) ([]Relationship, error) {
	relationships := []Relationship{}
	
	for i := 0; i < len(entityIDs)-1; i++ {
		rels, err := s.store.GetRelationships(entityIDs[i])
		if err != nil {
			return nil, err
		}
		
		for _, rel := range rels {
			if rel.Target.ID == entityIDs[i+1] || rel.Source.ID == entityIDs[i+1] {
				relationships = append(relationships, rel.Relationship)
				break
			}
		}
	}
	
	return relationships, nil
}

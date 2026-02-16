package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// KnowledgeSkill provides personal knowledge graph capabilities
type KnowledgeSkill struct {
	*skills.BaseSkill
	store      *Store
	extractor  *Extractor
	search     *Search
	compressor *Compressor
	logger     *zap.Logger
}

// KnowledgeConfig contains knowledge skill configuration
type KnowledgeConfig struct {
	Enabled            bool
	AutoExtract        bool   // Extract on every conversation
	ExtractOnRequest   bool   // Extract only when requested
	EnableCompression  bool
	MinConfidence      float64
	MaxEntitiesPerMsg  int
}

// DefaultKnowledgeConfig returns default configuration
func DefaultKnowledgeConfig() KnowledgeConfig {
	return KnowledgeConfig{
		Enabled:           true,
		AutoExtract:       true,
		ExtractOnRequest:  false,
		EnableCompression: true,
		MinConfidence:     0.6,
		MaxEntitiesPerMsg: 10,
	}
}

// NewKnowledgeSkill creates a new knowledge skill
func NewKnowledgeSkill(db *gorm.DB, config KnowledgeConfig, logger *zap.Logger) (*KnowledgeSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge store: %w", err)
	}
	
	skill := &KnowledgeSkill{
		BaseSkill:  skills.NewBaseSkill("knowledge", "Personal Knowledge Graph", "1.0.0"),
		store:      store,
		extractor:  NewExtractor(store, logger),
		search:     NewSearch(store, logger),
		compressor: NewCompressor(store, logger),
		logger:     logger,
	}
	
	skill.registerTools()
	
	return skill, nil
}

// registerTools registers all knowledge management tools
func (k *KnowledgeSkill) registerTools() {
	tools := []skills.Tool{
		{
			Name:        "remember",
			Description: "Extract and store knowledge from text. Automatically identifies people, places, events, preferences, and facts.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to extract knowledge from",
					},
					"conversation_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional conversation ID for context",
					},
				},
				"required": []string{"text"},
			},
		},
		{
			Name:        "recall",
			Description: "Search and retrieve information from your knowledge graph. Ask natural language questions like 'Who did I meet last week?' or 'What are my preferences?'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query or question",
					},
					"entity_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"person", "place", "organization", "event", "concept", "preference", "goal", "all"},
						"description": "Filter by entity type",
					},
					"time_range": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"today", "yesterday", "week", "month", "year", "all"},
						"description": "Time range filter",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"default":     10,
						"description": "Maximum results",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_entity",
			Description: "Get detailed information about a specific person, place, or thing you know",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the entity",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "add_memory",
			Description: "Manually add a specific memory or fact",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The memory content",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"fact", "preference", "event", "goal", "observation"},
						"description": "Type of memory",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category (personal, work, health, etc.)",
					},
					"importance": map[string]interface{}{
						"type":        "integer",
						"minimum":     1,
						"maximum":     10,
						"default":     5,
						"description": "Importance level",
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "list_entities",
			Description: "List all entities of a certain type that you know about",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entity_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"person", "place", "organization", "event", "concept", "preference", "goal", "all"},
						"default":     "all",
						"description": "Type of entities to list",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"default":     20,
						"description": "Maximum entities to return",
					},
				},
			},
		},
		{
			Name:        "get_stats",
			Description: "Get statistics about your knowledge graph",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "forget",
			Description: "Delete a specific memory or entity from your knowledge graph",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"memory", "entity"},
						"description": "What to delete",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the item to delete",
					},
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"default":     false,
						"description": "Confirm deletion",
					},
				},
				"required": []string{"type", "id"},
			},
		},
	}
	
	for _, tool := range tools {
		tool.Handler = k.handleTool(tool.Name)
		k.AddTool(tool)
	}
}

// handleTool handles tool calls
func (k *KnowledgeSkill) handleTool(name string) skills.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "remember":
			return k.handleRemember(ctx, args)
		case "recall":
			return k.handleRecall(ctx, args)
		case "get_entity":
			return k.handleGetEntity(ctx, args)
		case "add_memory":
			return k.handleAddMemory(ctx, args)
		case "list_entities":
			return k.handleListEntities(ctx, args)
		case "get_stats":
			return k.handleGetStats(ctx, args)
		case "forget":
			return k.handleForget(ctx, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	}
}

// handleRemember extracts and stores knowledge
func (k *KnowledgeSkill) handleRemember(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	text, _ := args["text"].(string)
	conversationID, _ := args["conversation_id"].(string)
	
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	
	userID := k.getUserID(ctx)
	
	opts := DefaultExtractionOptions()
	result, err := k.extractor.ExtractFromText(ctx, userID, text, opts)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}
	
	// Store the extracted knowledge
	if err := k.extractor.ProcessAndStore(ctx, userID, text, conversationID); err != nil {
		return nil, fmt.Errorf("failed to store knowledge: %w", err)
	}
	
	return map[string]interface{}{
		"extracted": map[string]interface{}{
			"entities":      len(result.Entities),
			"relationships": len(result.Relationships),
			"memories":      len(result.Memories),
		},
		"stored":    true,
		"confidence": result.Confidence,
		"summary":   k.summarizeExtraction(result),
	}, nil
}

// handleRecall searches the knowledge graph
func (k *KnowledgeSkill) handleRecall(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	entityType, _ := args["entity_type"].(string)
	timeRange, _ := args["time_range"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	
	userID := k.getUserID(ctx)
	
	// Build search query
	searchQuery := SearchQuery{
		Text:  query,
		Limit: limit,
	}
	
	if entityType != "" && entityType != "all" {
		searchQuery.EntityTypes = []string{entityType}
	}
	
	if timeRange != "" && timeRange != "all" {
		searchQuery.TimeRange = k.parseTimeRange(timeRange)
	}
	
	// Execute search
	result, err := k.search.Execute(ctx, userID, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Also try to answer as a question
	if k.isQuestion(query) {
		answerResult, err := k.search.QueryAnswer(ctx, userID, query)
		if err == nil && answerResult.Answer != "" {
			result.Answer = answerResult.Answer
		}
	}
	
	return map[string]interface{}{
		"query":         query,
		"entities":      k.formatEntityResults(result.Entities),
		"memories":      k.formatMemoryResults(result.Memories),
		"answer":        result.Answer,
		"confidence":    result.Confidence,
	}, nil
}

// handleGetEntity gets detailed entity information
func (k *KnowledgeSkill) handleGetEntity(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	
	userID := k.getUserID(ctx)
	
	// Find entity
	entity, err := k.store.GetEntityByName(userID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	if entity == nil {
		return map[string]interface{}{
			"found": false,
			"message": fmt.Sprintf("I don't know anything about '%s'", name),
		}, nil
	}
	
	// Get relationships
	relationships, err := k.store.GetRelationships(entity.ID)
	if err != nil {
		k.logger.Error("Failed to get relationships", zap.Error(err))
	}
	
	// Get memories
	memories, err := k.store.GetMemoriesForEntity(entity.ID, 10)
	if err != nil {
		k.logger.Error("Failed to get memories", zap.Error(err))
	}
	
	return map[string]interface{}{
		"found":         true,
		"entity":        entity,
		"relationships": relationships,
		"memories":      memories,
	}, nil
}

// handleAddMemory manually adds a memory
func (k *KnowledgeSkill) handleAddMemory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	content, _ := args["content"].(string)
	memType, _ := args["type"].(string)
	category, _ := args["category"].(string)
	importance := 5
	if i, ok := args["importance"].(float64); ok {
		importance = int(i)
	}
	
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	
	if memType == "" {
		memType = string(MemoryTypeFact)
	}
	
	userID := k.getUserID(ctx)
	
	memory := &Memory{
		UserID:     userID,
		Content:    content,
		Type:       memType,
		Category:   category,
		Importance: importance,
		Confidence: 1.0, // User-provided is high confidence
	}
	
	if err := k.store.CreateMemory(memory); err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}
	
	// Also extract any entities from the content
	k.extractor.ProcessAndStore(ctx, userID, content, "")
	
	return map[string]interface{}{
		"memory_id": memory.ID,
		"stored":    true,
	}, nil
}

// handleListEntities lists entities
func (k *KnowledgeSkill) handleListEntities(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	entityType, _ := args["entity_type"].(string)
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	
	userID := k.getUserID(ctx)
	
	var entities []Entity
	var err error
	
	if entityType == "" || entityType == "all" {
		// Get all recent entities (last 30 days)
		since := time.Now().AddDate(0, 0, -30)
		entities, err = k.store.GetRecentEntities(userID, since, limit)
	} else {
		entities, err = k.store.GetEntitiesByType(userID, entityType, limit)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	
	return map[string]interface{}{
		"entities": k.formatEntities(entities),
		"count":    len(entities),
	}, nil
}

// handleGetStats gets knowledge graph statistics
func (k *KnowledgeSkill) handleGetStats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := k.getUserID(ctx)
	
	stats, err := k.store.GetStats(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	
	return map[string]interface{}{
		"total_entities":       stats.TotalEntities,
		"total_relationships":  stats.TotalRelationships,
		"total_memories":       stats.TotalMemories,
		"entity_types":         stats.EntityTypes,
		"relationship_types":   stats.RelationshipTypes,
		"recent_mentions":      stats.RecentMentions,
	}, nil
}

// handleForget deletes a memory or entity
func (k *KnowledgeSkill) handleForget(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	itemType, _ := args["type"].(string)
	id, _ := args["id"].(string)
	confirm, _ := args["confirm"].(bool)
	
	if itemType == "" || id == "" {
		return nil, fmt.Errorf("type and id are required")
	}
	
	if !confirm {
		return map[string]interface{}{
			"confirm_required": true,
			"message":          fmt.Sprintf("Set confirm=true to delete this %s", itemType),
		}, nil
	}
	
	var err error
	switch itemType {
	case "memory":
		err = k.store.DeleteMemory(id)
	default:
		return nil, fmt.Errorf("unsupported type: %s", itemType)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}
	
	return map[string]interface{}{
		"deleted": true,
		"type":    itemType,
		"id":      id,
	}, nil
}

// Helper methods

func (k *KnowledgeSkill) getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "default_user"
}

func (k *KnowledgeSkill) isQuestion(text string) bool {
	questions := []string{"who", "what", "where", "when", "why", "how", "which", "did", "do", "is", "are", "was", "were"}
	lower := strings.ToLower(strings.TrimSpace(text))
	
	for _, q := range questions {
		if strings.HasPrefix(lower, q+" ") || strings.HasPrefix(lower, q+"?") {
			return true
		}
	}
	
	return strings.Contains(lower, "?")
}

func (k *KnowledgeSkill) parseTimeRange(range_ string) *TimeRange {
	return k.search.parseTimeReference(range_)
}



func (k *KnowledgeSkill) summarizeExtraction(result *ExtractionResult) string {
	parts := []string{}
	
	if len(result.Entities) > 0 {
		types := make(map[string]int)
		for _, e := range result.Entities {
			types[e.Type]++
		}
		
		typeNames := []string{}
		for t, count := range types {
			if count == 1 {
				typeNames = append(typeNames, fmt.Sprintf("1 %s", t))
			} else {
				typeNames = append(typeNames, fmt.Sprintf("%d %ss", count, t))
			}
		}
		
		parts = append(parts, fmt.Sprintf("Found %s", strings.Join(typeNames, ", ")))
	}
	
	if len(result.Memories) > 0 {
		parts = append(parts, fmt.Sprintf("extracted %d memories", len(result.Memories)))
	}
	
	if len(parts) == 0 {
		return "No knowledge extracted"
	}
	
	return strings.Join(parts, " and ") + "."
}

func (k *KnowledgeSkill) formatEntityResults(results []EntityResult) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(results))
	for i, r := range results {
		formatted[i] = map[string]interface{}{
			"id":        r.Entity.ID,
			"name":      r.Entity.Name,
			"type":      r.Entity.Type,
			"relevance": r.Relevance,
		}
	}
	return formatted
}

func (k *KnowledgeSkill) formatMemoryResults(results []MemoryResult) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(results))
	for i, r := range results {
		formatted[i] = map[string]interface{}{
			"id":        r.Memory.ID,
			"content":   r.Memory.Content,
			"type":      r.Memory.Type,
			"relevance": r.Relevance,
		}
	}
	return formatted
}

func (k *KnowledgeSkill) formatEntities(entities []Entity) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(entities))
	for i, e := range entities {
		formatted[i] = map[string]interface{}{
			"id":          e.ID,
			"name":        e.Name,
			"type":        e.Type,
			"mention_count": e.MentionCount,
		}
	}
	return formatted
}

// ExtractFromConversation extracts knowledge from a conversation
func (k *KnowledgeSkill) ExtractFromConversation(ctx context.Context, userID, conversationID string, messages []struct {
	Role    string
	Content string
}) error {
	return k.extractor.ExtractFromConversation(ctx, userID, messages, conversationID)
}

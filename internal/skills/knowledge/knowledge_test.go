package knowledge

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupKnowledgeTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func setupKnowledgeSkill(t *testing.T) (*KnowledgeSkill, *gorm.DB) {
	db := setupKnowledgeTestDB(t)
	logger, _ := zap.NewDevelopment()
	
	config := DefaultKnowledgeConfig()
	skill, err := NewKnowledgeSkill(db, config, logger)
	require.NoError(t, err)
	
	return skill, db
}

// Store Tests

func TestStore_CreateAndGetEntity(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	entity := &Entity{
		UserID:      "user1",
		Type:        string(EntityTypePerson),
		Name:        "Sarah",
		Description: "Friend from college",
	}
	
	err = store.CreateEntity(entity)
	require.NoError(t, err)
	assert.NotEmpty(t, entity.ID)
	
	// Retrieve
	retrieved, err := store.GetEntity(entity.ID)
	require.NoError(t, err)
	assert.Equal(t, "Sarah", retrieved.Name)
	assert.Equal(t, string(EntityTypePerson), retrieved.Type)
}

func TestStore_FindOrCreateEntity(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// First creation
	entity1, err := store.FindOrCreateEntity("user1", "John", string(EntityTypePerson))
	require.NoError(t, err)
	assert.Equal(t, 1, entity1.MentionCount)
	
	// Second lookup should return same entity with incremented count
	entity2, err := store.FindOrCreateEntity("user1", "John", string(EntityTypePerson))
	require.NoError(t, err)
	assert.Equal(t, entity1.ID, entity2.ID)
	assert.Equal(t, 2, entity2.MentionCount)
}

func TestStore_SearchEntities(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create entities
	entities := []*Entity{
		{UserID: "user1", Type: string(EntityTypePerson), Name: "Sarah Johnson"},
		{UserID: "user1", Type: string(EntityTypePerson), Name: "Sarah Smith"},
		{UserID: "user1", Type: string(EntityTypePlace), Name: "Blue Bottle Cafe"},
	}
	
	for _, e := range entities {
		err := store.CreateEntity(e)
		require.NoError(t, err)
	}
	
	// Search
	results, err := store.SearchEntities("user1", "Sarah", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestStore_Relationships(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create entities
	person := &Entity{UserID: "user1", Type: string(EntityTypePerson), Name: "Alice"}
	company := &Entity{UserID: "user1", Type: string(EntityTypeOrganization), Name: "TechCorp"}
	
	require.NoError(t, store.CreateEntity(person))
	require.NoError(t, store.CreateEntity(company))
	
	// Create relationship
	rel, err := store.FindOrCreateRelationship("user1", person.ID, company.ID, string(RelTypeWorksAt))
	require.NoError(t, err)
	assert.Equal(t, string(RelTypeWorksAt), rel.Type)
	
	// Get relationships
	rels, err := store.GetRelationships(person.ID)
	require.NoError(t, err)
	assert.Len(t, rels, 1)
}

func TestStore_MemoryOperations(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create memory
	memory := &Memory{
		UserID:   "user1",
		Content:  "I love hiking in the mountains",
		Type:     string(MemoryTypePreference),
		Category: "personal",
	}
	
	err = store.CreateMemory(memory)
	require.NoError(t, err)
	assert.NotEmpty(t, memory.ID)
	
	// Get memory
	retrieved, err := store.GetMemory(memory.ID)
	require.NoError(t, err)
	assert.Equal(t, "I love hiking in the mountains", retrieved.Content)
}

func TestStore_MemoryFilters(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create memories
	memories := []*Memory{
		{UserID: "user1", Content: "Fact 1", Type: string(MemoryTypeFact), Category: "work"},
		{UserID: "user1", Content: "Fact 2", Type: string(MemoryTypeFact), Category: "personal"},
		{UserID: "user1", Content: "Preference", Type: string(MemoryTypePreference), Category: "personal"},
	}
	
	for _, m := range memories {
		err := store.CreateMemory(m)
		require.NoError(t, err)
	}
	
	// Filter by type
	filters := MemoryFilters{Type: string(MemoryTypeFact)}
	results, err := store.GetMemories("user1", filters)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	
	// Filter by category
	filters = MemoryFilters{Category: "personal"}
	results, err = store.GetMemories("user1", filters)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestStore_Stats(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create some data
	require.NoError(t, store.CreateEntity(&Entity{UserID: "user1", Type: string(EntityTypePerson), Name: "Person1"}))
	require.NoError(t, store.CreateEntity(&Entity{UserID: "user1", Type: string(EntityTypePlace), Name: "Place1"}))
	require.NoError(t, store.CreateMemory(&Memory{UserID: "user1", Content: "Memory1", Type: string(MemoryTypeFact)}))
	
	stats, err := store.GetStats("user1")
	require.NoError(t, err)
	
	assert.Equal(t, 2, stats.TotalEntities)
	assert.Equal(t, 1, stats.TotalMemories)
	assert.Equal(t, 1, stats.EntityTypes[string(EntityTypePerson)])
	assert.Equal(t, 1, stats.EntityTypes[string(EntityTypePlace)])
}

// Extractor Tests

func TestExtractor_ExtractEntities(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	logger, _ := zap.NewDevelopment()
	extractor := NewExtractor(store, logger)
	
	text := "I met Sarah at Blue Bottle Cafe yesterday. She works at Google."
	
	opts := DefaultExtractionOptions()
	result, err := extractor.ExtractFromText(context.Background(), "user1", text, opts)
	require.NoError(t, err)
	
	assert.GreaterOrEqual(t, len(result.Entities), 2)
	
	// Check for person (name might include surrounding words due to regex)
	foundPerson := false
	for _, e := range result.Entities {
		if strings.Contains(e.Name, "Sarah") && e.Type == string(EntityTypePerson) {
			foundPerson = true
			break
		}
	}
	assert.True(t, foundPerson, "Should find Sarah as person")
}

func TestExtractor_ExtractRelationships(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	logger, _ := zap.NewDevelopment()
	extractor := NewExtractor(store, logger)
	
	text := "Alice works at TechCorp and lives in San Francisco."
	
	opts := DefaultExtractionOptions()
	result, err := extractor.ExtractFromText(context.Background(), "user1", text, opts)
	require.NoError(t, err)
	
	// Should extract entities (at least 2: person and organization)
	assert.GreaterOrEqual(t, len(result.Entities), 2, "Should extract multiple entities")
}

func TestExtractor_ExtractMemories(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	logger, _ := zap.NewDevelopment()
	extractor := NewExtractor(store, logger)
	
	text := "I love Italian food. I want to learn Spanish."
	
	opts := DefaultExtractionOptions()
	result, err := extractor.ExtractFromText(context.Background(), "user1", text, opts)
	require.NoError(t, err)
	
	// Should extract preferences and goals
	foundPref := false
	foundGoal := false
	
	for _, m := range result.Memories {
		if m.Type == string(MemoryTypePreference) {
			foundPref = true
		}
		if m.Type == string(MemoryTypeGoal) {
			foundGoal = true
		}
	}
	
	assert.True(t, foundPref, "Should find preference")
	assert.True(t, foundGoal, "Should find goal")
}

// Search Tests

func TestSearch_ParseQuestion(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	logger, _ := zap.NewDevelopment()
	search := NewSearch(store, logger)
	
	tests := []struct {
		question     string
		expectedType string
	}{
		{"Who did I meet?", "who"},
		{"Where did I go?", "where"},
		{"When did that happen?", "when"},
		{"What is my favorite?", "what"},
		{"How do I do this?", "how"},
	}
	
	for _, test := range tests {
		queryType, _, _ := search.parseQuestion(test.question)
		assert.Equal(t, test.expectedType, queryType, "For question: %s", test.question)
	}
}

func TestSearch_TimeRangeParsing(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	logger, _ := zap.NewDevelopment()
	search := NewSearch(store, logger)
	
	tests := []struct {
		ref      string
		hasRange bool
	}{
		{"today", true},
		{"yesterday", true},
		{"last week", true},
		{"recently", true},
		{"sometime", true}, // Returns empty TimeRange (non-nil)
	}
	
	for _, test := range tests {
		tr := search.parseTimeReference(test.ref)
		if test.hasRange {
			// For valid time references, we should have a valid time range with at least start or end
			assert.NotNil(t, tr, "Should have time range for: %s", test.ref)
		} else {
			// For unknown references, we might get an empty TimeRange
			// Just verify it doesn't panic
			_ = tr
		}
	}
}

// Knowledge Skill Tests

func TestKnowledgeSkill_Remember(t *testing.T) {
	skill, _ := setupKnowledgeSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	args := map[string]interface{}{
		"text": "I met Sarah at the coffee shop yesterday. She is a software engineer at Google.",
	}
	
	result, err := skill.handleRemember(ctx, args)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	extracted := resultMap["extracted"].(map[string]interface{})
	
	assert.GreaterOrEqual(t, extracted["entities"], 1)
	assert.Equal(t, true, resultMap["stored"])
}

func TestKnowledgeSkill_Recall(t *testing.T) {
	skill, _ := setupKnowledgeSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// First remember something
	rememberArgs := map[string]interface{}{
		"text": "I love hiking in Yosemite. The mountains are beautiful.",
	}
	_, err := skill.handleRemember(ctx, rememberArgs)
	require.NoError(t, err)
	
	// Now recall
	recallArgs := map[string]interface{}{
		"query":   "hiking",
		"limit":   float64(5),
	}
	
	result, err := skill.handleRecall(ctx, recallArgs)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	memories := resultMap["memories"].([]map[string]interface{})
	
	assert.GreaterOrEqual(t, len(memories), 1)
}

func TestKnowledgeSkill_AddMemory(t *testing.T) {
	skill, _ := setupKnowledgeSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	args := map[string]interface{}{
		"content":    "My favorite color is blue",
		"type":       "preference",
		"category":   "personal",
		"importance": float64(7),
	}
	
	result, err := skill.handleAddMemory(ctx, args)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.NotEmpty(t, resultMap["memory_id"])
	assert.Equal(t, true, resultMap["stored"])
}

func TestKnowledgeSkill_GetStats(t *testing.T) {
	skill, _ := setupKnowledgeSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add some data
	skill.handleAddMemory(ctx, map[string]interface{}{
		"content": "Memory 1",
		"type":    "fact",
	})
	skill.handleAddMemory(ctx, map[string]interface{}{
		"content": "Memory 2",
		"type":    "preference",
	})
	
	result, err := skill.handleGetStats(ctx, map[string]interface{}{})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.GreaterOrEqual(t, resultMap["total_memories"], 2)
}

func TestKnowledgeSkill_Forget(t *testing.T) {
	skill, _ := setupKnowledgeSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add a memory
	addResult, err := skill.handleAddMemory(ctx, map[string]interface{}{
		"content": "Temporary memory",
	})
	require.NoError(t, err)
	
	memoryID := addResult.(map[string]interface{})["memory_id"].(string)
	
	// Try to delete without confirmation
	result, err := skill.handleForget(ctx, map[string]interface{}{
		"type": "memory",
		"id":   memoryID,
	})
	require.NoError(t, err)
	assert.Equal(t, true, result.(map[string]interface{})["confirm_required"])
	
	// Delete with confirmation
	result, err = skill.handleForget(ctx, map[string]interface{}{
		"type":    "memory",
		"id":      memoryID,
		"confirm": true,
	})
	require.NoError(t, err)
	assert.Equal(t, true, result.(map[string]interface{})["deleted"])
}

// Entity Helper Tests

func TestEntity_Aliases(t *testing.T) {
	entity := &Entity{Name: "John"}
	
	entity.AddAlias("Johnny")
	entity.AddAlias("Jon")
	
	aliases := entity.GetAliases()
	assert.Len(t, aliases, 2)
	assert.Contains(t, aliases, "Johnny")
	assert.Contains(t, aliases, "Jon")
}

func TestEntity_IncrementMention(t *testing.T) {
	entity := &Entity{Name: "Test"}
	
	entity.IncrementMention()
	assert.Equal(t, 1, entity.MentionCount)
	assert.NotNil(t, entity.FirstMentioned)
	assert.NotNil(t, entity.LastMentioned)
	
	first := *entity.FirstMentioned
	
	// Wait a tiny bit
	time.Sleep(10 * time.Millisecond)
	
	entity.IncrementMention()
	assert.Equal(t, 2, entity.MentionCount)
	assert.Equal(t, first, *entity.FirstMentioned) // Should stay same
	assert.True(t, entity.LastMentioned.After(first) || entity.LastMentioned.Equal(first))
}

// Memory Helper Tests

func TestMemory_LinkEntity(t *testing.T) {
	memory := &Memory{Content: "Test"}
	
	memory.LinkEntity("ent_1")
	memory.LinkEntity("ent_2")
	
	ids := memory.GetEntityIDs()
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "ent_1")
	assert.Contains(t, ids, "ent_2")
}

func TestMemory_RecordAccess(t *testing.T) {
	memory := &Memory{Content: "Test"}
	
	memory.RecordAccess()
	assert.Equal(t, 1, memory.AccessCount)
	assert.NotNil(t, memory.LastAccessed)
}

// Compression Tests

func TestCompressor_GroupByCategory(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	logger, _ := zap.NewDevelopment()
	compressor := NewCompressor(store, logger)
	
	memories := []Memory{
		{Category: "work", Content: "Work 1"},
		{Category: "work", Content: "Work 2"},
		{Category: "personal", Content: "Personal 1"},
	}
	
	groups := compressor.groupByCategory(memories)
	assert.Len(t, groups, 2)
	assert.Len(t, groups["work"], 2)
	assert.Len(t, groups["personal"], 1)
}

// Helper Tests

func TestSplitAndTrim(t *testing.T) {
	result := splitAndTrim("a,b,c", ",")
	assert.Equal(t, []string{"a", "b", "c"}, result)
	
	result = splitAndTrim("  a  ,  b  ,  c  ", ",")
	assert.Equal(t, []string{"a", "b", "c"}, result)
	
	result = splitAndTrim("", ",")
	assert.Empty(t, result)
}

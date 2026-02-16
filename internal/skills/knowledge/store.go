package knowledge

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles knowledge graph persistence
type Store struct {
	db *gorm.DB
}

// NewStore creates a new knowledge store
func NewStore(db *gorm.DB) (*Store, error) {
	store := &Store{db: db}
	
	// Auto-migrate schemas
	if err := db.AutoMigrate(&Entity{}, &Relationship{}, &Memory{}); err != nil {
		return nil, fmt.Errorf("failed to migrate knowledge schemas: %w", err)
	}
	
	// Create indexes for performance
	store.createIndexes()
	
	return store, nil
}

// createIndexes creates database indexes
func (s *Store) createIndexes() {
	// Entity indexes
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_entities_user_type ON entities(user_id, type)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_entities_user_name ON entities(user_id, name)")
	
	// Relationship indexes
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_id)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_id)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_relationships_type ON relationships(type)")
	
	// Memory indexes
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_memories_user_type ON memories(user_id, type)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_memories_timestamp ON memories(timestamp)")
}

// generateID generates a unique ID
func generateID(prefix string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return prefix + "_" + hex.EncodeToString(bytes)
}

// Entity operations

// CreateEntity creates a new entity
func (s *Store) CreateEntity(entity *Entity) error {
	if entity.ID == "" {
		entity.ID = generateID("ent")
	}
	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()
	
	return s.db.Create(entity).Error
}

// GetEntity retrieves an entity by ID
func (s *Store) GetEntity(entityID string) (*Entity, error) {
	var entity Entity
	err := s.db.Where("id = ?", entityID).First(&entity).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &entity, err
}

// GetEntityByName retrieves an entity by name (case-insensitive)
func (s *Store) GetEntityByName(userID, name string) (*Entity, error) {
	var entity Entity
	err := s.db.Where("user_id = ? AND LOWER(name) = LOWER(?)", userID, name).First(&entity).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &entity, err
}

// UpdateEntity updates an existing entity
func (s *Store) UpdateEntity(entity *Entity) error {
	entity.UpdatedAt = time.Now()
	return s.db.Save(entity).Error
}

// FindOrCreateEntity finds an entity by name or creates a new one
func (s *Store) FindOrCreateEntity(userID string, name, entityType string) (*Entity, error) {
	// Try to find existing entity
	entity, err := s.GetEntityByName(userID, name)
	if err != nil {
		return nil, err
	}
	if entity != nil {
		// Update mention count and last mentioned
		entity.IncrementMention()
		if err := s.UpdateEntity(entity); err != nil {
			return nil, err
		}
		return entity, nil
	}
	
	// Create new entity
	entity = &Entity{
		UserID: userID,
		Name:   name,
		Type:   entityType,
	}
	entity.IncrementMention()
	
	if err := s.CreateEntity(entity); err != nil {
		return nil, err
	}
	
	return entity, nil
}

// SearchEntities searches for entities by name (partial match)
func (s *Store) SearchEntities(userID, query string, limit int) ([]Entity, error) {
	var entities []Entity
	searchPattern := "%" + query + "%"
	
	err := s.db.Where(
		"user_id = ? AND (name LIKE ? OR aliases LIKE ?)",
		userID, searchPattern, searchPattern,
	).
	Order("mention_count DESC, updated_at DESC").
	Limit(limit).
	Find(&entities).Error
	
	return entities, err
}

// GetEntitiesByType retrieves entities by type
func (s *Store) GetEntitiesByType(userID, entityType string, limit int) ([]Entity, error) {
	var entities []Entity
	err := s.db.Where("user_id = ? AND type = ?", userID, entityType).
		Order("mention_count DESC").
		Limit(limit).
		Find(&entities).Error
	return entities, err
}

// GetRecentEntities gets recently mentioned entities
func (s *Store) GetRecentEntities(userID string, since time.Time, limit int) ([]Entity, error) {
	var entities []Entity
	err := s.db.Where("user_id = ? AND last_mentioned >= ?", userID, since).
		Order("last_mentioned DESC").
		Limit(limit).
		Find(&entities).Error
	return entities, err
}

// Relationship operations

// CreateRelationship creates a new relationship
func (s *Store) CreateRelationship(rel *Relationship) error {
	if rel.ID == "" {
		rel.ID = generateID("rel")
	}
	rel.CreatedAt = time.Now()
	rel.UpdatedAt = time.Now()
	
	return s.db.Create(rel).Error
}

// FindOrCreateRelationship finds or creates a relationship
func (s *Store) FindOrCreateRelationship(userID, sourceID, targetID, relType string) (*Relationship, error) {
	// Check if relationship exists
	var rel Relationship
	err := s.db.Where(
		"user_id = ? AND source_id = ? AND target_id = ? AND type = ?",
		userID, sourceID, targetID, relType,
	).First(&rel).Error
	
	if err == nil {
		// Update existing relationship
		rel.MentionCount++
		now := time.Now()
		rel.LastMentioned = &now
		rel.UpdatedAt = now
		if err := s.db.Save(&rel).Error; err != nil {
			return nil, err
		}
		return &rel, nil
	}
	
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	
	// Create new relationship
	rel = Relationship{
		UserID:      userID,
		SourceID:    sourceID,
		TargetID:    targetID,
		Type:        relType,
		MentionCount: 1,
	}
	now := time.Now()
	rel.FirstMentioned = &now
	rel.LastMentioned = &now
	
	if err := s.CreateRelationship(&rel); err != nil {
		return nil, err
	}
	
	return &rel, nil
}

// GetRelationships gets all relationships for an entity
func (s *Store) GetRelationships(entityID string) ([]RelationshipWithContext, error) {
	var rels []Relationship
	err := s.db.Where("source_id = ? OR target_id = ?", entityID, entityID).
		Order("mention_count DESC").
		Find(&rels).Error
	if err != nil {
		return nil, err
	}
	
	result := make([]RelationshipWithContext, len(rels))
	for i, rel := range rels {
		source, _ := s.GetEntity(rel.SourceID)
		target, _ := s.GetEntity(rel.TargetID)
		
		if source != nil && target != nil {
			result[i] = RelationshipWithContext{
				Relationship: rel,
				Source:       *source,
				Target:       *target,
			}
		}
	}
	
	return result, nil
}

// GetRelatedEntities gets entities related to a given entity
func (s *Store) GetRelatedEntities(entityID string) ([]Entity, error) {
	// Get relationship IDs
	var rels []Relationship
	err := s.db.Where("source_id = ? OR target_id = ?", entityID, entityID).Find(&rels).Error
	if err != nil {
		return nil, err
	}
	
	// Collect related entity IDs
	entityIDs := make([]string, 0, len(rels))
	for _, rel := range rels {
		if rel.SourceID == entityID {
			entityIDs = append(entityIDs, rel.TargetID)
		} else {
			entityIDs = append(entityIDs, rel.SourceID)
		}
	}
	
	if len(entityIDs) == 0 {
		return []Entity{}, nil
	}
	
	// Fetch entities
	var entities []Entity
	err = s.db.Where("id IN ?", entityIDs).Find(&entities).Error
	return entities, err
}

// Memory operations

// CreateMemory creates a new memory
func (s *Store) CreateMemory(memory *Memory) error {
	if memory.ID == "" {
		memory.ID = generateID("mem")
	}
	memory.CreatedAt = time.Now()
	memory.UpdatedAt = time.Now()
	
	return s.db.Create(memory).Error
}

// GetMemory retrieves a memory by ID
func (s *Store) GetMemory(memoryID string) (*Memory, error) {
	var memory Memory
	err := s.db.Where("id = ?", memoryID).First(&memory).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &memory, err
}

// UpdateMemory updates a memory
func (s *Store) UpdateMemory(memory *Memory) error {
	memory.UpdatedAt = time.Now()
	return s.db.Save(memory).Error
}

// GetMemories retrieves memories with optional filters
func (s *Store) GetMemories(userID string, filters MemoryFilters) ([]Memory, error) {
	query := s.db.Where("user_id = ?", userID)
	
	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.EntityID != "" {
		query = query.Where("entity_ids LIKE ?", "%"+filters.EntityID+"%")
	}
	if filters.Since != nil {
		query = query.Where("timestamp >= ?", filters.Since)
	}
	if filters.Until != nil {
		query = query.Where("timestamp <= ?", filters.Until)
	}
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where("content LIKE ? OR summary LIKE ?", searchPattern, searchPattern)
	}
	
	// Order by importance and recency
	query = query.Order("importance DESC, timestamp DESC")
	
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	
	var memories []Memory
	err := query.Find(&memories).Error
	return memories, err
}

// MemoryFilters contains filters for memory queries
type MemoryFilters struct {
	Type     string
	Category string
	EntityID string
	Since    *time.Time
	Until    *time.Time
	Search   string
	Limit    int
}

// GetMemoriesForEntity gets all memories linked to an entity
func (s *Store) GetMemoriesForEntity(entityID string, limit int) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("entity_ids LIKE ?", "%"+entityID+"%").
		Order("timestamp DESC").
		Limit(limit).
		Find(&memories).Error
	return memories, err
}

// GetRecentMemories gets memories from a time range
func (s *Store) GetRecentMemories(userID string, since time.Time, limit int) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("user_id = ? AND created_at >= ?", userID, since).
		Order("created_at DESC").
		Limit(limit).
		Find(&memories).Error
	return memories, err
}

// GetUnaccessedMemories gets memories that haven't been accessed recently
func (s *Store) GetUnaccessedMemories(userID string, since time.Time, limit int) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where(
		"user_id = ? AND (last_accessed IS NULL OR last_accessed < ?)",
		userID, since,
	).
	Order("importance ASC, created_at ASC").
	Limit(limit).
	Find(&memories).Error
	return memories, err
}

// DeleteMemory deletes a memory
func (s *Store) DeleteMemory(memoryID string) error {
	return s.db.Where("id = ?", memoryID).Delete(&Memory{}).Error
}

// Statistics

// GetStats gets knowledge graph statistics
func (s *Store) GetStats(userID string) (*GraphStats, error) {
	stats := &GraphStats{
		EntityTypes:       make(map[string]int),
		RelationshipTypes: make(map[string]int),
		MemoryTypes:       make(map[string]int),
	}
	
	// Count entities
	var entityCount int64
	s.db.Model(&Entity{}).Where("user_id = ?", userID).Count(&entityCount)
	stats.TotalEntities = int(entityCount)
	
	// Count relationships
	var relCount int64
	s.db.Model(&Relationship{}).Where("user_id = ?", userID).Count(&relCount)
	stats.TotalRelationships = int(relCount)
	
	// Count memories
	var memCount int64
	s.db.Model(&Memory{}).Where("user_id = ?", userID).Count(&memCount)
	stats.TotalMemories = int(memCount)
	
	// Entity types
	var entityTypes []struct {
		Type  string
		Count int64
	}
	s.db.Model(&Entity{}).Select("type, COUNT(*) as count").Where("user_id = ?", userID).Group("type").Scan(&entityTypes)
	for _, et := range entityTypes {
		stats.EntityTypes[et.Type] = int(et.Count)
	}
	
	// Relationship types
	var relTypes []struct {
		Type  string
		Count int64
	}
	s.db.Model(&Relationship{}).Select("type, COUNT(*) as count").Where("user_id = ?", userID).Group("type").Scan(&relTypes)
	for _, rt := range relTypes {
		stats.RelationshipTypes[rt.Type] = int(rt.Count)
	}
	
	// Memory types
	var memTypes []struct {
		Type  string
		Count int64
	}
	s.db.Model(&Memory{}).Select("type, COUNT(*) as count").Where("user_id = ?", userID).Group("type").Scan(&memTypes)
	for _, mt := range memTypes {
		stats.MemoryTypes[mt.Type] = int(mt.Count)
	}
	
	// Recent mentions (last 30 days)
	recent := time.Now().AddDate(0, 0, -30)
	var recentCount int64
	s.db.Model(&Entity{}).Where("user_id = ? AND last_mentioned >= ?", userID, recent).Count(&recentCount)
	stats.RecentMentions = int(recentCount)
	
	return stats, nil
}

// Compression operations

// MarkMemoryCompressed marks a memory as compressed
func (s *Store) MarkMemoryCompressed(memoryID, compressedFrom string, summary string) error {
	return s.db.Model(&Memory{}).Where("id = ?", memoryID).Updates(map[string]interface{}{
		"is_compressed":    true,
		"compressed_from":  compressedFrom,
		"summary":          summary,
		"content":          summary,
		"updated_at":       time.Now(),
	}).Error
}

// DeleteOldMemories deletes memories older than specified date with low importance
func (s *Store) DeleteOldMemories(userID string, olderThan time.Time, maxImportance int) error {
	return s.db.Where(
		"user_id = ? AND created_at < ? AND importance <= ? AND is_compressed = ?",
		userID, olderThan, maxImportance, false,
	).Delete(&Memory{}).Error
}

// UpsertEntity creates or updates an entity
func (s *Store) UpsertEntity(entity *Entity) error {
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(entity).Error
}

// UpsertMemory creates or updates a memory
func (s *Store) UpsertMemory(memory *Memory) error {
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(memory).Error
}

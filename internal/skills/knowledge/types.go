package knowledge

import (
	"time"
)

// Entity represents a node in the knowledge graph
type Entity struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	Type        string    `json:"type" gorm:"index"` // person, place, event, concept, preference, etc.
	Name        string    `json:"name" gorm:"index"`
	Aliases     string    `json:"aliases"` // Comma-separated alternative names
	Description string    `json:"description"`
	
	// Context
	FirstMentioned   *time.Time `json:"first_mentioned"`
	LastMentioned    *time.Time `json:"last_mentioned"`
	MentionCount     int        `json:"mention_count"`
	
	// Source tracking
	SourceConversation string `json:"source_conversation,omitempty"`
	SourceMessage      string `json:"source_message,omitempty"`
	
	// Metadata
	Confidence  float64        `json:"confidence"` // 0-1 extraction confidence
	Importance  int            `json:"importance"` // 1-10 user-defined importance
	IsVerified  bool           `json:"is_verified"`
	Metadata    string         `json:"metadata"` // JSON-encoded additional data
	
	// Timestamps
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// EntityType represents different types of entities
type EntityType string

const (
	EntityTypePerson      EntityType = "person"
	EntityTypePlace       EntityType = "place"
	EntityTypeOrganization EntityType = "organization"
	EntityTypeEvent       EntityType = "event"
	EntityTypeConcept     EntityType = "concept"
	EntityTypePreference  EntityType = "preference"
	EntityTypeGoal        EntityType = "goal"
	EntityTypeHabit       EntityType = "habit"
	EntityTypeRelationship EntityType = "relationship"
	EntityTypeItem        EntityType = "item"
	EntityTypeTime        EntityType = "time"
)

// Relationship represents an edge between two entities
type Relationship struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Entities
	SourceID    string    `json:"source_id" gorm:"index"`
	TargetID    string    `json:"target_id" gorm:"index"`
	
	// Relationship type
	Type        string    `json:"type" gorm:"index"` // knows, works_at, located_in, etc.
	Directional bool      `json:"directional"` // true if A->B is different from B->A
	
	// Context
	FirstMentioned   *time.Time `json:"first_mentioned"`
	LastMentioned    *time.Time `json:"last_mentioned"`
	MentionCount     int        `json:"mention_count"`
	
	// Properties
	Confidence  float64   `json:"confidence"`
	Properties  string    `json:"properties"` // JSON-encoded relationship properties
	
	// Source
	SourceConversation string `json:"source_conversation,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RelationshipType represents common relationship types
type RelationshipType string

const (
	RelTypeKnows         RelationshipType = "knows"
	RelTypeWorksAt       RelationshipType = "works_at"
	RelTypeLocatedIn     RelationshipType = "located_in"
	RelTypeLivesIn       RelationshipType = "lives_in"
	RelTypeBornIn        RelationshipType = "born_in"
	RelTypeMarriedTo     RelationshipType = "married_to"
	RelTypeRelatedTo     RelationshipType = "related_to"
	RelTypeFriendOf      RelationshipType = "friend_of"
	RelTypeColleagueOf   RelationshipType = "colleague_of"
	RelTypeMemberOf      RelationshipType = "member_of"
	RelTypeCreated       RelationshipType = "created"
	RelTypePartOf        RelationshipType = "part_of"
	RelTypeHas           RelationshipType = "has"
	RelTypePrefers       RelationshipType = "prefers"
	RelTypeDislikes      RelationshipType = "dislikes"
	RelTypeInterestedIn  RelationshipType = "interested_in"
	RelTypeMetAt         RelationshipType = "met_at"
	RelTypeAttended      RelationshipType = "attended"
	RelTypeScheduledFor  RelationshipType = "scheduled_for"
)

// Memory represents a specific memory/fact extracted from conversations
type Memory struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Content
	Content     string    `json:"content" gorm:"type:text"`
	Summary     string    `json:"summary"`
	
	// Classification
	Type        string    `json:"type"` // fact, preference, event, plan, observation
	Category    string    `json:"category"` // personal, work, health, finance, etc.
	
	// Entity links
	EntityIDs   string    `json:"entity_ids"` // Comma-separated entity IDs
	
	// Temporal
	Timestamp   *time.Time `json:"timestamp,omitempty"` // When the memory occurred
	DateText    string     `json:"date_text,omitempty"` // Original date text ("last Tuesday")
	
	// Context
	ConversationID string `json:"conversation_id,omitempty"`
	Context        string `json:"context,omitempty" gorm:"type:text"` // Surrounding context
	
	// Memory management
	Confidence     float64    `json:"confidence"`
	Importance     int        `json:"importance"` // 1-10
	AccessCount    int        `json:"access_count"`
	LastAccessed   *time.Time `json:"last_accessed"`
	
	// Compression
	IsCompressed   bool       `json:"is_compressed"`
	CompressedFrom string     `json:"compressed_from,omitempty"` // IDs of memories this was compressed from
	
	// Vector embedding (stored separately or as reference)
	EmbeddingID    string     `json:"embedding_id,omitempty"`
	
	// Timestamps
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// MemoryType represents types of memories
type MemoryType string

const (
	MemoryTypeFact         MemoryType = "fact"
	MemoryTypePreference   MemoryType = "preference"
	MemoryTypeEvent        MemoryType = "event"
	MemoryTypePlan         MemoryType = "plan"
	MemoryTypeObservation  MemoryType = "observation"
	MemoryTypeGoal         MemoryType = "goal"
	MemoryTypeRelationship MemoryType = "relationship"
)

// ExtractionResult contains entities and relationships extracted from text
type ExtractionResult struct {
	Entities      []ExtractedEntity      `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
	Memories      []ExtractedMemory      `json:"memories"`
	Confidence    float64                `json:"confidence"`
}

// ExtractedEntity represents an entity extracted from text
type ExtractedEntity struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Aliases     []string  `json:"aliases,omitempty"`
	Description string    `json:"description,omitempty"`
	Confidence  float64   `json:"confidence"`
	StartPos    int       `json:"start_pos"`
	EndPos      int       `json:"end_pos"`
}

// ExtractedRelationship represents a relationship extracted from text
type ExtractedRelationship struct {
	Source     string    `json:"source"`
	Target     string    `json:"target"`
	Type       string    `json:"type"`
	Confidence float64   `json:"confidence"`
}

// ExtractedMemory represents a memory extracted from text
type ExtractedMemory struct {
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	Category   string    `json:"category"`
	Entities   []string  `json:"entities"`
	Confidence float64   `json:"confidence"`
}

// QueryResult represents the result of querying the knowledge graph
type QueryResult struct {
	Entities      []EntityWithContext      `json:"entities"`
	Relationships []RelationshipWithContext `json:"relationships"`
	Memories      []Memory                 `json:"memories"`
	Answer        string                   `json:"answer,omitempty"`
	Confidence    float64                  `json:"confidence"`
}

// EntityWithContext includes entity with related information
type EntityWithContext struct {
	Entity       Entity         `json:"entity"`
	Related      []Entity       `json:"related"`
	Relationships []Relationship `json:"relationships"`
	Memories     []Memory       `json:"memories"`
}

// RelationshipWithContext includes relationship with entity details
type RelationshipWithContext struct {
	Relationship Relationship `json:"relationship"`
	Source       Entity       `json:"source"`
	Target       Entity       `json:"target"`
}

// GraphStats contains statistics about the knowledge graph
type GraphStats struct {
	TotalEntities      int            `json:"total_entities"`
	TotalRelationships int            `json:"total_relationships"`
	TotalMemories      int            `json:"total_memories"`
	EntityTypes        map[string]int `json:"entity_types"`
	RelationshipTypes  map[string]int `json:"relationship_types"`
	MemoryTypes        map[string]int `json:"memory_types"`
	RecentMentions     int            `json:"recent_mentions"` // Last 30 days
}

// CompressionBatch represents a batch of memories to compress
type CompressionBatch struct {
	Memories   []Memory  `json:"memories"`
	Summary    string    `json:"summary"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
}

// Helper methods

// AddAlias adds an alias to the entity
func (e *Entity) AddAlias(alias string) {
	if e.Aliases == "" {
		e.Aliases = alias
	} else {
		e.Aliases = e.Aliases + "," + alias
	}
}

// GetAliases returns the list of aliases
func (e *Entity) GetAliases() []string {
	if e.Aliases == "" {
		return []string{}
	}
	return splitAndTrim(e.Aliases, ",")
}

// IncrementMention updates mention tracking
func (e *Entity) IncrementMention() {
	e.MentionCount++
	now := time.Now()
	e.LastMentioned = &now
	if e.FirstMentioned == nil {
		e.FirstMentioned = &now
	}
}

// LinkEntity adds an entity ID to the memory
func (m *Memory) LinkEntity(entityID string) {
	if m.EntityIDs == "" {
		m.EntityIDs = entityID
	} else {
		m.EntityIDs = m.EntityIDs + "," + entityID
	}
}

// GetEntityIDs returns the list of linked entity IDs
func (m *Memory) GetEntityIDs() []string {
	if m.EntityIDs == "" {
		return []string{}
	}
	return splitAndTrim(m.EntityIDs, ",")
}

// RecordAccess updates access tracking
func (m *Memory) RecordAccess() {
	m.AccessCount++
	now := time.Now()
	m.LastAccessed = &now
}

// splitAndTrim splits a string and trims whitespace
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	parts := []string{}
	for _, p := range splitString(s, sep) {
		trimmed := trimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// Helper functions (simplified)
func splitString(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i < len(s)-len(sep)+1 && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

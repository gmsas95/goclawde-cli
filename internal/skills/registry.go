package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gmsas95/myrai-cli/internal/store"
)

// Skill represents a plugin/skill that can be registered
type Skill interface {
	Name() string
	Description() string
	Version() string
	Tools() []Tool
	IsEnabled() bool
	Enable() error
	Disable() error
}

// Tool represents a tool provided by a skill
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler is the function that executes a tool
type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// Registry manages all skills
type Registry struct {
	skills map[string]Skill
	tools  map[string]Tool
	mu     sync.RWMutex
	store  *store.Store
}

// NewRegistry creates a new skill registry
func NewRegistry(store *store.Store) *Registry {
	r := &Registry{
		skills: make(map[string]Skill),
		tools:  make(map[string]Tool),
		store:  store,
	}
	return r
}

// Register adds a skill to the registry
func (r *Registry) Register(skill Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := skill.Name()
	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill %s already registered", name)
	}

	r.skills[name] = skill

	// Register tools
	for _, tool := range skill.Tools() {
		r.tools[tool.Name] = tool
	}

	return nil
}

// GetSkill retrieves a skill by name
func (r *Registry) GetSkill(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	skill, ok := r.skills[name]
	return skill, ok
}

// GetTool retrieves a tool by name
func (r *Registry) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// ExecuteTool executes a tool by name
func (r *Registry) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	tool, ok := r.GetTool(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Parse arguments
	var argsMap map[string]interface{}
	if err := json.Unmarshal(args, &argsMap); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return tool.Handler(ctx, argsMap)
}

// ListSkills returns all registered skills
func (r *Registry) ListSkills() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	return skills
}

// ListTools returns all available tools
func (r *Registry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolDefinitions returns tool definitions for LLM
func (r *Registry) GetToolDefinitions() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		})
	}
	return defs
}

// BaseSkill provides a base implementation for skills
type BaseSkill struct {
	name        string
	description string
	version     string
	enabled     bool
	tools       []Tool
}

// Name returns the skill name
func (s *BaseSkill) Name() string { return s.name }

// Description returns the skill description
func (s *BaseSkill) Description() string { return s.description }

// Version returns the skill version
func (s *BaseSkill) Version() string { return s.version }

// Tools returns the skill's tools
func (s *BaseSkill) Tools() []Tool { return s.tools }

// IsEnabled returns if the skill is enabled
func (s *BaseSkill) IsEnabled() bool { return s.enabled }

// Enable enables the skill
func (s *BaseSkill) Enable() error {
	s.enabled = true
	return nil
}

// Disable disables the skill
func (s *BaseSkill) Disable() error {
	s.enabled = false
	return nil
}

// NewBaseSkill creates a new base skill
func NewBaseSkill(name, description, version string) *BaseSkill {
	return &BaseSkill{
		name:        name,
		description: description,
		version:     version,
		enabled:     true,
		tools:       []Tool{},
	}
}

// AddTool adds a tool to the skill
func (s *BaseSkill) AddTool(tool Tool) {
	s.tools = append(s.tools, tool)
}

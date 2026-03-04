// Package tools provides dynamic tool registration and execution
// allowing tools to be loaded from skills/plugins without recompilation.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gmsas95/myrai-cli/internal/types"
)

// ToolHandler is the function signature for tool implementations
type ToolHandler func(ctx context.Context, args json.RawMessage) (*ToolResult, error)

// ToolResult represents the result of executing a tool
type ToolResult struct {
	Content []types.ContentBlock
	IsError bool
	Error   error
}

// ToolSource indicates where a tool came from
type ToolSource string

const (
	ToolSourceBuiltin ToolSource = "builtin"
	ToolSourceSkill   ToolSource = "skill"
	ToolSourcePlugin  ToolSource = "plugin"
)

// ToolDefinition defines a tool that can be invoked
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
	Handler     ToolHandler     `json:"-"`          // Function to execute (not serialized)
	Source      ToolSource      `json:"source"`
	SourcePath  string          `json:"source_path,omitempty"` // For skills/plugins
	Version     string          `json:"version,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ToFunctionDefinition converts to a format suitable for LLM APIs
func (t *ToolDefinition) ToFunctionDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        t.Name,
		"description": t.Description,
		"parameters":  t.Parameters,
	}
}

// Registry manages available tools
type Registry struct {
	tools map[string]*ToolDefinition
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*ToolDefinition),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *ToolDefinition) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool %s has no handler", tool.Name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}

	if tool.CreatedAt.IsZero() {
		tool.CreatedAt = time.Now()
	}

	r.tools[tool.Name] = tool
	return nil
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(r.tools, name)
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools
func (r *Registry) List() []*ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// ListBySource returns tools from a specific source
func (r *Registry) ListBySource(source ToolSource) []*ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ToolDefinition
	for _, tool := range r.tools {
		if tool.Source == source {
			result = append(result, tool)
		}
	}
	return result
}

// Execute runs a tool with the given arguments
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) (*ToolResult, error) {
	tool, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool.Handler(ctx, args)
}

// ToProviderTools converts all tools to provider format
func (r *Registry) ToProviderTools() []*ToolDefinition {
	return r.List()
}

// SkillLoader loads tools from skill directories
type SkillLoader struct {
	registry  *Registry
	skillsDir string
}

// NewSkillLoader creates a new skill loader
func NewSkillLoader(registry *Registry, skillsDir string) *SkillLoader {
	return &SkillLoader{
		registry:  registry,
		skillsDir: skillsDir,
	}
}

// SkillManifest defines the structure of a skill.json file
type SkillManifest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Tools       []SkillToolDef `json:"tools"`
	EntryPoint  string         `json:"entry_point,omitempty"` // Python/JS file to execute
}

// SkillToolDef defines a tool within a skill
type SkillToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Handler     string          `json:"handler"` // Function name in entry point
}

// LoadSkill loads all tools from a skill directory
func (l *SkillLoader) LoadSkill(skillName string) error {
	skillPath := filepath.Join(l.skillsDir, skillName)

	// Read skill manifest
	manifestData, err := readFile(filepath.Join(skillPath, "skill.json"))
	if err != nil {
		return fmt.Errorf("failed to read skill manifest: %w", err)
	}

	var manifest SkillManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse skill manifest: %w", err)
	}

	// Register each tool
	for _, toolDef := range manifest.Tools {
		tool := &ToolDefinition{
			Name:        toolDef.Name,
			Description: toolDef.Description,
			Parameters:  toolDef.Parameters,
			Source:      ToolSourceSkill,
			SourcePath:  skillPath,
			Version:     manifest.Version,
			Handler:     l.createSkillHandler(skillPath, manifest.EntryPoint, toolDef.Handler),
		}

		if err := l.registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", toolDef.Name, err)
		}
	}

	return nil
}

// createSkillHandler creates a handler that executes a skill tool
func (l *SkillLoader) createSkillHandler(skillPath, entryPoint, handler string) ToolHandler {
	return func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
		// This is a placeholder - actual implementation would:
		// 1. Execute the entry point (Python/JS) with the handler name and args
		// 2. Parse the output
		// 3. Return as ToolResult

		// For now, return a placeholder result
		return &ToolResult{
			Content: []types.ContentBlock{
				types.TextBlock{Text: fmt.Sprintf("Skill handler %s executed", handler)},
			},
		}, nil
	}
}

// LoadAllSkills loads all skills from the skills directory
func (l *SkillLoader) LoadAllSkills() error {
	entries, err := listDir(l.skillsDir)
	if err != nil {
		return fmt.Errorf("failed to list skills directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir {
			if err := l.LoadSkill(entry.Name); err != nil {
				// Log error but continue loading other skills
				fmt.Printf("Warning: failed to load skill %s: %v\n", entry.Name, err)
			}
		}
	}

	return nil
}

// BuiltinTools provides the default set of built-in tools
type BuiltinTools struct {
	registry *Registry
}

// NewBuiltinTools creates a builtin tools registrar
func NewBuiltinTools(registry *Registry) *BuiltinTools {
	return &BuiltinTools{registry: registry}
}

// RegisterAll registers all built-in tools
func (b *BuiltinTools) RegisterAll() error {
	// Register built-in tools here
	// Example:
	// if err := b.registerReadFile(); err != nil { return err }
	// if err := b.registerWriteFile(); err != nil { return err }
	// etc.

	return nil
}

// File system helpers (these would be implemented with actual file operations)

type dirEntry struct {
	Name  string
	IsDir bool
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func listDir(path string) ([]dirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	result := make([]dirEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dirEntry{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
		})
	}
	return result, nil
}

// DefaultRegistry is the global tool registry instance
var DefaultRegistry = NewRegistry()

// Register is a convenience function to register a tool on the default registry
func Register(tool *ToolDefinition) error {
	return DefaultRegistry.Register(tool)
}

// Get is a convenience function to get a tool from the default registry
func Get(name string) (*ToolDefinition, bool) {
	return DefaultRegistry.Get(name)
}

// Execute is a convenience function to execute a tool on the default registry
func Execute(ctx context.Context, name string, args json.RawMessage) (*ToolResult, error) {
	return DefaultRegistry.Execute(ctx, name, args)
}

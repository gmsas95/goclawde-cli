package skills

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParameterType represents valid parameter types
type ParameterType string

const (
	TypeString  ParameterType = "string"
	TypeInteger ParameterType = "integer"
	TypeNumber  ParameterType = "number"
	TypeBoolean ParameterType = "boolean"
	TypeArray   ParameterType = "array"
	TypeObject  ParameterType = "object"
)

// ToolParameter defines a tool parameter
type ToolParameter struct {
	Name        string                 `yaml:"name" json:"name"`
	Type        ParameterType          `yaml:"type" json:"type"`
	Required    bool                   `yaml:"required,omitempty" json:"required,omitempty"`
	Default     interface{}            `yaml:"default,omitempty" json:"default,omitempty"`
	Description string                 `yaml:"description" json:"description"`
	Enum        []string               `yaml:"enum,omitempty" json:"enum,omitempty"`
	Items       *ToolParameter         `yaml:"items,omitempty" json:"items,omitempty"`           // For array types
	Properties  map[string]interface{} `yaml:"properties,omitempty" json:"properties,omitempty"` // For object types
}

// ManifestTool defines a tool in the manifest
type ManifestTool struct {
	Name        string          `yaml:"name" json:"name"`
	Description string          `yaml:"description" json:"description"`
	Parameters  []ToolParameter `yaml:"parameters" json:"parameters"`
}

// MCPServerConfig defines MCP server configuration in a skill
type MCPServerConfig struct {
	Name     string            `yaml:"name" json:"name"`
	Required bool              `yaml:"required,omitempty" json:"required,omitempty"`
	Command  string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args     []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env      map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// SkillManifest represents the YAML frontmatter in SKILL.md
type SkillManifest struct {
	Name            string           `yaml:"name" json:"name"`
	Version         string           `yaml:"version" json:"version"`
	Description     string           `yaml:"description" json:"description"`
	Author          string           `yaml:"author,omitempty" json:"author,omitempty"`
	Tags            []string         `yaml:"tags,omitempty" json:"tags,omitempty"`
	MinMyraiVersion string           `yaml:"min_myrai_version,omitempty" json:"min_myrai_version,omitempty"`
	MCP             *MCPServerConfig `yaml:"mcp,omitempty" json:"mcp,omitempty"`
	Tools           []ManifestTool   `yaml:"tools" json:"tools"`
}

// Validate checks if the manifest is valid
func (m *SkillManifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("skill version is required")
	}
	if m.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	if !isValidVersion(m.Version) {
		return fmt.Errorf("invalid version format: %s (expected semver like 1.2.0)", m.Version)
	}
	if m.MinMyraiVersion != "" && !isValidVersion(m.MinMyraiVersion) {
		return fmt.Errorf("invalid min_myrai_version format: %s", m.MinMyraiVersion)
	}

	// Validate tool names are unique
	toolNames := make(map[string]bool)
	for _, tool := range m.Tools {
		if tool.Name == "" {
			return fmt.Errorf("tool name is required")
		}
		if toolNames[tool.Name] {
			return fmt.Errorf("duplicate tool name: %s", tool.Name)
		}
		toolNames[tool.Name] = true

		// Validate parameters
		paramNames := make(map[string]bool)
		for _, param := range tool.Parameters {
			if param.Name == "" {
				return fmt.Errorf("parameter name is required in tool %s", tool.Name)
			}
			if paramNames[param.Name] {
				return fmt.Errorf("duplicate parameter name %s in tool %s", param.Name, tool.Name)
			}
			paramNames[param.Name] = true

			if !isValidParameterType(param.Type) {
				return fmt.Errorf("invalid parameter type %s for parameter %s in tool %s", param.Type, param.Name, tool.Name)
			}
		}
	}

	return nil
}

// GetTool returns a tool by name from the manifest
func (m *SkillManifest) GetTool(name string) (*ManifestTool, bool) {
	for _, tool := range m.Tools {
		if tool.Name == name {
			return &tool, true
		}
	}
	return nil, false
}

// HasTool checks if a tool exists in the manifest
func (m *SkillManifest) HasTool(name string) bool {
	_, ok := m.GetTool(name)
	return ok
}

// GetParameter returns a parameter by name for a given tool
func (t *ManifestTool) GetParameter(name string) (*ToolParameter, bool) {
	for _, param := range t.Parameters {
		if param.Name == name {
			return &param, true
		}
	}
	return nil, false
}

// ToJSONSchema converts parameters to JSON schema format
func (t *ManifestTool) ToJSONSchema() map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	for _, param := range t.Parameters {
		prop := map[string]interface{}{
			"type":        string(param.Type),
			"description": param.Description,
		}
		if len(param.Enum) > 0 {
			prop["enum"] = param.Enum
		}
		if param.Default != nil {
			prop["default"] = param.Default
		}
		if param.Type == TypeArray && param.Items != nil {
			prop["items"] = map[string]interface{}{
				"type": string(param.Items.Type),
			}
		}
		if param.Type == TypeObject && param.Properties != nil {
			prop["properties"] = param.Properties
		}
		properties[param.Name] = prop

		if param.Required {
			required = append(required, param.Name)
		}
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// isValidVersion checks if a version string follows semver format
func isValidVersion(version string) bool {
	// Simple semver validation: major.minor.patch or major.minor.patch-prerelease
	pattern := `^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`
	matched, _ := regexp.MatchString(pattern, version)
	return matched
}

// isValidParameterType checks if a parameter type is valid
func isValidParameterType(t ParameterType) bool {
	validTypes := []ParameterType{TypeString, TypeInteger, TypeNumber, TypeBoolean, TypeArray, TypeObject}
	for _, valid := range validTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// ParseSkillMarkdown parses SKILL.md content and extracts the manifest and documentation
func ParseSkillMarkdown(content string) (*SkillManifest, string, error) {
	// Split frontmatter from content
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, "", fmt.Errorf("invalid SKILL.md format: missing frontmatter delimiter")
	}

	frontmatter := strings.TrimSpace(parts[1])
	docs := strings.TrimSpace(parts[2])

	var manifest SkillManifest
	if err := parseFrontmatter(frontmatter, &manifest); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, "", fmt.Errorf("invalid manifest: %w", err)
	}

	return &manifest, docs, nil
}

// parseFrontmatter parses YAML frontmatter using proper YAML parser
func parseFrontmatter(content string, manifest *SkillManifest) error {
	if err := yaml.Unmarshal([]byte(content), manifest); err != nil {
		return fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}
	return nil
}

// parseStringArray parses a comma-separated string into a slice
func parseStringArray(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseValue attempts to parse a value into the appropriate type
func parseValue(s string) interface{} {
	s = strings.TrimSpace(s)

	// Try boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Try number
	if i, err := parseInt(s); err == nil {
		return i
	}
	if f, err := parseFloat(s); err == nil {
		return f
	}

	// Return as string
	return strings.Trim(s, `"'`)
}

// parseInt attempts to parse a string as int
func parseInt(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// parseFloat attempts to parse a string as float
func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

package skills

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Validator provides validation for skills and MCP servers
type Validator struct {
	mu      sync.RWMutex
	results map[string]*ValidationResult
	timeout time.Duration
}

// ValidationResult contains validation results
type ValidationResult struct {
	SkillName   string
	Valid       bool
	Errors      []ValidationError
	Warnings    []string
	ValidatedAt time.Time
	Duration    time.Duration
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		results: make(map[string]*ValidationResult),
		timeout: 30 * time.Second,
	}
}

// SetTimeout sets the validation timeout
func (v *Validator) SetTimeout(timeout time.Duration) {
	v.timeout = timeout
}

// ValidateManifest validates a skill manifest
func (v *Validator) ValidateManifest(manifest *SkillManifest) *ValidationResult {
	start := time.Now()
	result := &ValidationResult{
		SkillName: manifest.Name,
		Valid:     true,
		Errors:    []ValidationError{},
		Warnings:  []string{},
	}

	// Check required fields
	if manifest.Name == "" {
		result.addError("name", "Skill name is required", "REQUIRED_FIELD_MISSING")
	} else if !isValidSkillName(manifest.Name) {
		result.addError("name", "Invalid skill name format (use lowercase letters, numbers, and hyphens)", "INVALID_FORMAT")
	}

	if manifest.Version == "" {
		result.addError("version", "Version is required", "REQUIRED_FIELD_MISSING")
	} else if !isValidVersion(manifest.Version) {
		result.addError("version", "Invalid version format (expected semver like 1.2.0)", "INVALID_VERSION")
	}

	if manifest.Description == "" {
		result.addError("description", "Description is required", "REQUIRED_FIELD_MISSING")
	} else if len(manifest.Description) < 10 {
		result.Warnings = append(result.Warnings, "Description should be at least 10 characters")
	}

	// Check min_myrai_version if provided
	if manifest.MinMyraiVersion != "" && !isValidVersion(manifest.MinMyraiVersion) {
		result.addError("min_myrai_version", "Invalid version format", "INVALID_VERSION")
	}

	// Validate tools
	if len(manifest.Tools) == 0 {
		result.addError("tools", "At least one tool is required", "NO_TOOLS_DEFINED")
	} else {
		v.validateTools(manifest, result)
	}

	// Validate MCP configuration if present
	if manifest.MCP != nil {
		v.validateMCPConfig(manifest.MCP, result)
	}

	result.Valid = len(result.Errors) == 0
	result.ValidatedAt = time.Now()
	result.Duration = time.Since(start)

	// Store result
	v.mu.Lock()
	v.results[manifest.Name] = result
	v.mu.Unlock()

	return result
}

// validateTools validates all tools in a manifest
func (v *Validator) validateTools(manifest *SkillManifest, result *ValidationResult) {
	toolNames := make(map[string]bool)

	for i, tool := range manifest.Tools {
		prefix := fmt.Sprintf("tools[%d]", i)

		// Check tool name
		if tool.Name == "" {
			result.addError(prefix+".name", "Tool name is required", "REQUIRED_FIELD_MISSING")
			continue
		}

		if toolNames[tool.Name] {
			result.addError(prefix+".name", fmt.Sprintf("Duplicate tool name: %s", tool.Name), "DUPLICATE_TOOL")
		} else {
			toolNames[tool.Name] = true
		}

		if !isValidToolName(tool.Name) {
			result.addError(prefix+".name", "Invalid tool name format (use lowercase letters, numbers, and underscores)", "INVALID_FORMAT")
		}

		// Check tool description
		if tool.Description == "" {
			result.addError(prefix+".description", "Tool description is required", "REQUIRED_FIELD_MISSING")
		}

		// Validate parameters
		v.validateParameters(tool, prefix, result)
	}
}

// validateParameters validates tool parameters
func (v *Validator) validateParameters(tool ManifestTool, prefix string, result *ValidationResult) {
	paramNames := make(map[string]bool)

	for i, param := range tool.Parameters {
		paramPrefix := fmt.Sprintf("%s.parameters[%d]", prefix, i)

		// Check parameter name
		if param.Name == "" {
			result.addError(paramPrefix+".name", "Parameter name is required", "REQUIRED_FIELD_MISSING")
			continue
		}

		if paramNames[param.Name] {
			result.addError(paramPrefix+".name", fmt.Sprintf("Duplicate parameter name: %s", param.Name), "DUPLICATE_PARAMETER")
		} else {
			paramNames[param.Name] = true
		}

		// Validate parameter type
		if !isValidParameterType(param.Type) {
			result.addError(paramPrefix+".type", fmt.Sprintf("Invalid parameter type: %s", param.Type), "INVALID_TYPE")
		}

		// Validate description
		if param.Description == "" {
			result.addError(paramPrefix+".description", "Parameter description is required", "REQUIRED_FIELD_MISSING")
		}

		// Validate enum values if provided
		if len(param.Enum) > 0 {
			if param.Type != TypeString {
				result.addError(paramPrefix+".enum", "Enum can only be used with string type", "INVALID_ENUM")
			}
		}

		// Validate default value matches type
		if param.Default != nil {
			if !v.validateDefaultValue(param.Default, param.Type) {
				result.addError(paramPrefix+".default", "Default value doesn't match parameter type", "INVALID_DEFAULT")
			}
		}
	}
}

// validateMCPConfig validates MCP server configuration
func (v *Validator) validateMCPConfig(mcp *MCPServerConfig, result *ValidationResult) {
	if mcp.Name == "" {
		result.addError("mcp.name", "MCP server name is required", "REQUIRED_FIELD_MISSING")
	}

	// If command is provided, validate it
	if mcp.Command != "" {
		// Check for potentially dangerous commands
		dangerousCommands := []string{"rm", "del", "format", "mkfs", "dd"}
		for _, dangerous := range dangerousCommands {
			if mcp.Command == dangerous || contains(mcp.Args, dangerous) {
				result.addError("mcp.command", fmt.Sprintf("Potentially dangerous command detected: %s", dangerous), "DANGEROUS_COMMAND")
			}
		}
	}
}

// ValidateMCPConnectivity checks if an MCP server is reachable
func (v *Validator) ValidateMCPConnectivity(ctx context.Context, serverName, command string, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	// This is a placeholder - actual implementation would:
	// 1. Start the MCP server process
	// 2. Send initialization request
	// 3. Check for proper response
	// 4. Clean up

	// For now, just return nil (assume success)
	return nil
}

// ValidateSkillFile validates a SKILL.md file
func (v *Validator) ValidateSkillFile(content string) *ValidationResult {
	manifest, _, err := ParseSkillMarkdown(content)
	if err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []ValidationError{{Field: "file", Message: err.Error(), Code: "PARSE_ERROR"}},
		}
	}

	return v.ValidateManifest(manifest)
}

// GetValidationResult retrieves a stored validation result
func (v *Validator) GetValidationResult(skillName string) (*ValidationResult, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	result, ok := v.results[skillName]
	return result, ok
}

// GetAllValidationResults returns all stored validation results
func (v *Validator) GetAllValidationResults() map[string]*ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	results := make(map[string]*ValidationResult)
	for k, v := range v.results {
		results[k] = v
	}
	return results
}

// ClearValidationResults clears all stored validation results
func (v *Validator) ClearValidationResults() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.results = make(map[string]*ValidationResult)
}

// Helper methods

func (r *ValidationResult) addError(field, message, code string) {
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

func isValidSkillName(name string) bool {
	if name == "" {
		return false
	}
	// Allow lowercase letters, numbers, and hyphens
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

func isValidToolName(name string) bool {
	if name == "" {
		return false
	}
	// Allow lowercase letters, numbers, and underscores
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func (v *Validator) validateDefaultValue(value interface{}, paramType ParameterType) bool {
	switch paramType {
	case TypeString:
		_, ok := value.(string)
		return ok
	case TypeInteger:
		switch v := value.(type) {
		case int, int8, int16, int32, int64:
			return true
		case float64:
			return v == float64(int64(v))
		default:
			return false
		}
	case TypeNumber:
		switch value.(type) {
		case int, int8, int16, int32, int64, float32, float64:
			return true
		default:
			return false
		}
	case TypeBoolean:
		_, ok := value.(bool)
		return ok
	case TypeArray:
		_, ok := value.([]interface{})
		return ok
	case TypeObject:
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return false
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// FormatErrors returns a formatted string of all errors
func (r *ValidationResult) FormatErrors() string {
	if len(r.Errors) == 0 {
		return "No errors"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Validation failed for '%s':\n", r.SkillName))
	for _, err := range r.Errors {
		sb.WriteString(fmt.Sprintf("  - [%s] %s: %s\n", err.Code, err.Field, err.Message))
	}
	return sb.String()
}

// FormatWarnings returns a formatted string of all warnings
func (r *ValidationResult) FormatWarnings() string {
	if len(r.Warnings) == 0 {
		return "No warnings"
	}

	var sb strings.Builder
	sb.WriteString("Warnings:\n")
	for _, warning := range r.Warnings {
		sb.WriteString(fmt.Sprintf("  - %s\n", warning))
	}
	return sb.String()
}

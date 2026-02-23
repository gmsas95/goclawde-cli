package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/stretchr/testify/assert"
)

func TestNewSkillLoader(t *testing.T) {
	// Create a mock registry
	registry := skills.NewRegistry(nil)
	loader := NewSkillLoader(registry)

	assert.NotNil(t, loader)
	assert.NotNil(t, loader.registry)
	assert.NotNil(t, loader.watchers)
	assert.NotNil(t, loader.loaded)
	assert.NotNil(t, loader.stopCh)
}

func TestSkillSourceType(t *testing.T) {
	assert.Equal(t, SkillSourceType("github"), SourceGitHub)
	assert.Equal(t, SkillSourceType("local"), SourceLocal)
	assert.Equal(t, SkillSourceType("builtin"), SourceBuiltIn)
}

func TestLoadedSkill(t *testing.T) {
	skill := &LoadedSkill{
		Name:        "test-skill",
		Version:     "1.0.0",
		Description: "Test skill",
		Source:      "github.com/user/repo",
		SourceType:  SourceGitHub,
		Tools:       []skills.Tool{},
		LoadedAt:    "2026-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-skill", skill.Name)
	assert.Equal(t, "1.0.0", skill.Version)
	assert.Equal(t, SourceGitHub, skill.SourceType)
}

func TestSkillManifest(t *testing.T) {
	manifest := &SkillManifest{
		Name:            "docker-helper",
		Version:         "1.2.0",
		Description:     "Docker helper skill",
		Author:          "test-author",
		Tags:            []string{"docker", "devops"},
		MinMyraiVersion: "2.0.0",
		MCP: &MCPConfig{
			Server:   "docker",
			Required: false,
		},
		Tools: []ManifestTool{
			{
				Name:        "docker_ps",
				Description: "List containers",
				Parameters: []ManifestParameter{
					{
						Name:        "all",
						Type:        "boolean",
						Required:    false,
						Default:     false,
						Description: "Show all containers",
					},
				},
			},
		},
	}

	assert.Equal(t, "docker-helper", manifest.Name)
	assert.Equal(t, "1.2.0", manifest.Version)
	assert.NotNil(t, manifest.MCP)
	assert.Equal(t, "docker", manifest.MCP.Server)
	assert.Len(t, manifest.Tools, 1)
}

func TestManifestParameter(t *testing.T) {
	param := ManifestParameter{
		Name:        "format",
		Type:        "string",
		Required:    false,
		Default:     "table",
		Description: "Output format",
		Enum:        []string{"table", "json"},
	}

	assert.Equal(t, "format", param.Name)
	assert.Equal(t, "string", param.Type)
	assert.Equal(t, "table", param.Default)
	assert.Len(t, param.Enum, 2)
}

func TestListLoaded(t *testing.T) {
	registry := skills.NewRegistry(nil)
	loader := NewSkillLoader(registry)

	// Initially empty
	skills := loader.ListLoaded()
	assert.Empty(t, skills)
}

func TestGetSkillNotFound(t *testing.T) {
	registry := skills.NewRegistry(nil)
	loader := NewSkillLoader(registry)

	skill, ok := loader.GetSkill("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, skill)
}

func TestLoadFromLocalNotFound(t *testing.T) {
	registry := skills.NewRegistry(nil)
	loader := NewSkillLoader(registry)

	_, err := loader.LoadFromLocal("/nonexistent/path")
	assert.Error(t, err)
	// Error message should indicate failure to read or path not found
	assert.True(t, err != nil)
}

func TestParseManifestBasic(t *testing.T) {
	loader := NewSkillLoader(nil)

	content := `---
name: test-skill
version: 1.0.0
description: Test skill
author: test-author
---

# Test Skill

This is a test skill.
`

	manifest, err := loader.parseManifest([]byte(content))
	assert.NoError(t, err)
	assert.Equal(t, "test-skill", manifest.Name)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, "Test skill", manifest.Description)
	assert.Equal(t, "test-author", manifest.Author)
}

func TestParseManifestNoFrontMatter(t *testing.T) {
	loader := NewSkillLoader(nil)

	content := `# Just Markdown

No YAML front matter here.
`

	_, err := loader.parseManifest([]byte(content))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no YAML front matter")
}

func TestStopWatcher(t *testing.T) {
	registry := skills.NewRegistry(nil)
	loader := NewSkillLoader(registry)

	// Create a temp directory to watch
	tempDir := t.TempDir()

	// Start watching
	err := loader.Watch(tempDir)
	assert.NoError(t, err)

	// Stop should not panic
	loader.Stop()
}

func TestCreateTempSkillFile(t *testing.T) {
	// Create a temporary skill file
	tempDir := t.TempDir()
	skillPath := filepath.Join(tempDir, "SKILL.md")

	content := `---
name: temp-skill
version: 0.1.0
description: Temporary test skill
author: test
---

# Temp Skill
`

	err := os.WriteFile(skillPath, []byte(content), 0644)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(skillPath)
	assert.NoError(t, err)
}

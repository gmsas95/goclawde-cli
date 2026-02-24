package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewLoader(t *testing.T) {
	logger := zap.NewNop()
	loader := NewLoader(logger)

	assert.NotNil(t, loader)
	assert.NotNil(t, loader.skills)
	assert.NotNil(t, loader.paths)
	assert.NotNil(t, loader.logger)
}

func TestSkillStatus(t *testing.T) {
	tests := []struct {
		status   SkillStatus
		expected string
	}{
		{SkillStatusUnknown, "unknown"},
		{SkillStatusLoading, "loading"},
		{SkillStatusLoaded, "loaded"},
		{SkillStatusValidating, "validating"},
		{SkillStatusValidated, "validated"},
		{SkillStatusError, "error"},
		{SkillStatusDisabled, "disabled"},
		{SkillStatus(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestSkillManifest(t *testing.T) {
	manifest := SkillManifest{
		Name:        "docker-helper",
		Version:     "1.2.0",
		Description: "Docker helper skill",
		Author:      "test-author",
		Tags:        []string{"docker", "devops"},
		Tools: []ToolDefinition{
			{
				Name:        "docker_ps",
				Description: "List containers",
				Parameters: map[string]interface{}{
					"all": map[string]interface{}{
						"type":        "boolean",
						"description": "Show all containers",
					},
				},
			},
		},
	}

	assert.Equal(t, "docker-helper", manifest.Name)
	assert.Equal(t, "1.2.0", manifest.Version)
	assert.NotNil(t, manifest.Tools)
	assert.Len(t, manifest.Tools, 1)
	assert.Equal(t, "docker_ps", manifest.Tools[0].Name)
}

func TestSandboxConfig(t *testing.T) {
	config := SandboxConfig{
		Enabled:     true,
		AllowFS:     true,
		AllowNet:    false,
		AllowExec:   false,
		AllowedDirs: []string{"/tmp", "/data"},
	}

	assert.True(t, config.Enabled)
	assert.True(t, config.AllowFS)
	assert.False(t, config.AllowNet)
	assert.Len(t, config.AllowedDirs, 2)
}

func TestLoaderAddPath(t *testing.T) {
	logger := zap.NewNop()
	loader := NewLoader(logger)

	loader.AddPath("/test/path/1")
	loader.AddPath("/test/path/2")

	assert.Len(t, loader.paths, 2)
}

func TestLoadSkillNotFound(t *testing.T) {
	logger := zap.NewNop()
	loader := NewLoader(logger)

	_, err := loader.LoadSkill("/nonexistent/path/SKILL.md")
	assert.Error(t, err)
}

func TestGetSkillNotFound(t *testing.T) {
	logger := zap.NewNop()
	loader := NewLoader(logger)

	skill := loader.GetSkill("nonexistent")
	assert.Nil(t, skill)
}

func TestListSkillsEmpty(t *testing.T) {
	logger := zap.NewNop()
	loader := NewLoader(logger)

	skills := loader.ListSkills()
	assert.Empty(t, skills)
}

func TestParseFrontmatterBasic(t *testing.T) {
	content := `---
name: test-skill
version: 1.0.0
description: Test skill
author: test-author
---

# Test Skill

This is a test skill.
`

	manifest, rawManifest, err := parseFrontmatter(content)
	assert.NoError(t, err)
	assert.Equal(t, "test-skill", manifest.Name)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, "Test skill", manifest.Description)
	assert.Equal(t, "test-author", manifest.Author)
	assert.NotEmpty(t, rawManifest)
}

func TestParseFrontmatterNoFrontMatter(t *testing.T) {
	content := `# Just Markdown

No YAML front matter here.
`

	_, _, err := parseFrontmatter(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no frontmatter")
}

func TestParseFrontmatterInvalidYAML(t *testing.T) {
	content := `---
name: test-skill
version: [
---

# Test Skill
`

	_, _, err := parseFrontmatter(content)
	assert.Error(t, err)
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name      string
		manifest  SkillManifest
		wantError bool
	}{
		{
			name: "valid manifest",
			manifest: SkillManifest{
				Name:        "test-skill",
				Version:     "1.0.0",
				Description: "Test skill",
			},
			wantError: false,
		},
		{
			name: "missing name",
			manifest: SkillManifest{
				Version:     "1.0.0",
				Description: "Test skill",
			},
			wantError: true,
		},
		{
			name: "missing version",
			manifest: SkillManifest{
				Name:        "test-skill",
				Description: "Test skill",
			},
			wantError: true,
		},
		{
			name: "missing description",
			manifest: SkillManifest{
				Name:    "test-skill",
				Version: "1.0.0",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &Skill{
				Manifest: tt.manifest,
			}
			err := ValidateManifest(skill)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadSkillFromFile(t *testing.T) {
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

This is a temporary test skill.
`

	err := os.WriteFile(skillPath, []byte(content), 0644)
	assert.NoError(t, err)

	// Load the skill
	logger := zap.NewNop()
	loader := NewLoader(logger)
	skill, err := loader.LoadSkill(skillPath)

	assert.NoError(t, err)
	assert.NotNil(t, skill)
	assert.Equal(t, "temp-skill", skill.Manifest.Name)
	assert.Equal(t, "0.1.0", skill.Manifest.Version)
	assert.Equal(t, skillPath, skill.Path)
}

func TestSkillStruct(t *testing.T) {
	skill := &Skill{
		Manifest: SkillManifest{
			Name:        "test",
			Version:     "1.0.0",
			Description: "Test skill",
		},
		Path:        "/test/path",
		Content:     "Test content",
		RawManifest: "raw manifest",
		Status:      SkillStatusLoading,
	}

	assert.Equal(t, "test", skill.Manifest.Name)
	assert.Equal(t, SkillStatusLoading, skill.Status)
	assert.Equal(t, "/test/path", skill.Path)
}

func TestToolDefinition(t *testing.T) {
	tool := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]interface{}{
			"param1": map[string]interface{}{
				"type": "string",
			},
		},
	}

	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "A test tool", tool.Description)
	assert.NotNil(t, tool.Parameters)
}

func TestGetSkillAfterLoad(t *testing.T) {
	// Create a temporary skill file
	tempDir := t.TempDir()
	skillPath := filepath.Join(tempDir, "SKILL.md")

	content := `---
name: get-test-skill
version: 1.0.0
description: Test get skill
---

# Get Test Skill
`

	err := os.WriteFile(skillPath, []byte(content), 0644)
	assert.NoError(t, err)

	// Load and then get
	logger := zap.NewNop()
	loader := NewLoader(logger)
	loader.LoadSkill(skillPath)

	// Get by path
	skill := loader.GetSkill(skillPath)
	assert.NotNil(t, skill)
	assert.Equal(t, "get-test-skill", skill.Manifest.Name)
}

func TestListSkillsAfterLoad(t *testing.T) {
	// Create temporary skill files
	tempDir := t.TempDir()

	for i := 0; i < 3; i++ {
		skillPath := filepath.Join(tempDir, "SKILL"+string(rune('0'+i))+".md")
		content := `---
name: skill-` + string(rune('0'+i)) + `
version: 1.0.0
description: Test skill
---

# Test Skill
`
		os.WriteFile(skillPath, []byte(content), 0644)

		logger := zap.NewNop()
		loader := NewLoader(logger)
		loader.LoadSkill(skillPath)

		skills := loader.ListSkills()
		assert.GreaterOrEqual(t, len(skills), 1)
	}
}

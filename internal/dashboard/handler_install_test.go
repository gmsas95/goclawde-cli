package dashboard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInstallSkill(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	skillsRegistry := skills.NewRegistry(nil)
	handler := NewHandler(cfg, skillsRegistry, logger, nil)

	app := fiber.New()
	app.Post("/api/skills/install", handler.installSkill)

	t.Run("empty repo returns error", func(t *testing.T) {
		body := map[string]string{
			"repo": "",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/skills/install", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("missing repo field returns error", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/skills/install", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://github.com/user/repo",
			expected: "github.com_user_repo",
		},
		{
			input:    "github.com/user/repo",
			expected: "github.com_user_repo",
		},
		{
			input:    "git@github.com:user/repo.git",
			expected: "user_repo",
		},
		{
			input:    "https://github.com/user/repo.git",
			expected: "github.com_user_repo",
		},
		{
			input:    "user/repo",
			expected: "user_repo",
		},
		{
			input:    "https://gitlab.com/group/project",
			expected: "gitlab.com_group_project",
		},
		{
			input:    "repo with spaces",
			expected: "repo_with_spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeRepoName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVersionCompatible(t *testing.T) {
	tests := []struct {
		current  string
		minimum  string
		expected bool
	}{
		{"2.0.0", "1.0.0", true},  // Higher major
		{"1.5.0", "1.0.0", true},  // Higher minor
		{"1.0.5", "1.0.0", true},  // Higher patch
		{"1.0.0", "1.0.0", true},  // Exact match
		{"1.0.0", "2.0.0", false}, // Lower major
		{"1.0.0", "1.5.0", false}, // Lower minor
		{"1.0.0", "1.0.5", false}, // Lower patch
		{"2.0", "1.0", true},      // Short version
		{"1", "1", true},          // Single digit
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.minimum, func(t *testing.T) {
			result := isVersionCompatible(tt.current, tt.minimum)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCloneRepository(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	handler := NewHandler(cfg, nil, logger, nil)

	t.Run("non-existent repo fails gracefully", func(t *testing.T) {
		destPath := filepath.Join(tempDir, "test_clone")
		err := handler.cloneRepository("https://github.com/nonexistent-user-12345/nonexistent-repo-67890", destPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "git clone failed")
	})

	t.Run("git not installed would fail", func(t *testing.T) {
		// This test documents that git must be installed
		// We can't easily test this without modifying PATH
		t.Skip("Requires git to not be in PATH - skipping")
	})
}

func TestInstallSkillDependencies(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	handler := NewHandler(cfg, nil, logger, nil)

	t.Run("no dependencies returns nil", func(t *testing.T) {
		skillPath := t.TempDir()
		err := handler.installSkillDependencies(skillPath)
		assert.NoError(t, err)
	})

	t.Run("package.json detected", func(t *testing.T) {
		skillPath := t.TempDir()
		packageJSON := []byte(`{"name": "test"}`)
		err := os.WriteFile(filepath.Join(skillPath, "package.json"), packageJSON, 0644)
		require.NoError(t, err)

		err = handler.installSkillDependencies(skillPath)
		// May succeed or fail depending on npm availability
		// Just verify it doesn't panic
		t.Logf("package.json install result: %v", err)
	})

	t.Run("requirements.txt detected", func(t *testing.T) {
		skillPath := t.TempDir()
		requirements := []byte("# No packages\n")
		err := os.WriteFile(filepath.Join(skillPath, "requirements.txt"), requirements, 0644)
		require.NoError(t, err)

		err = handler.installSkillDependencies(skillPath)
		// May succeed or fail depending on pip availability
		// Just verify it doesn't panic
		t.Logf("requirements.txt install result: %v", err)
	})
}

func TestInstallSkill_InvalidRepoFormat(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	skillsRegistry := skills.NewRegistry(nil)
	handler := NewHandler(cfg, skillsRegistry, logger, nil)

	app := fiber.New()
	app.Post("/api/skills/install", handler.installSkill)

	// Test with invalid URL format
	body := map[string]string{
		"repo": "not-a-valid-url-or-shorthand",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/skills/install", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should try to clone and fail
	assert.Equal(t, 500, resp.StatusCode)

	bodyBytes, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(bodyBytes), "Failed to clone")
}

func TestInstallSkill_MissingManifest(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	handler := NewHandler(cfg, nil, logger, nil)

	app := fiber.New()
	app.Post("/api/skills/install", handler.installSkill)

	// Create a temporary repo without SKILL.md
	tempRepo := t.TempDir()
	// Just create some random file
	os.WriteFile(filepath.Join(tempRepo, "README.md"), []byte("test"), 0644)

	// This test would need actual git repo to work properly
	// Skipping as it requires more setup
	t.Skip("Requires actual git repository setup")
}

// Integration test - would require actual git repo
func TestInstallSkill_Success(t *testing.T) {
	t.Skip("Integration test - requires actual git repository with SKILL.md")
}

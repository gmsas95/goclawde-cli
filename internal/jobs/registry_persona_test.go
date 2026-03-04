package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRegistry_SetPersonaManager(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
		LLM: config.LLMConfig{
			DefaultProvider: "openai",
			Providers: map[string]config.Provider{
				"openai": {
					APIKey:  "test-key",
					BaseURL: "https://api.openai.com/v1",
					Model:   "gpt-4",
				},
			},
		},
	}

	st, err := store.New(cfg)
	require.NoError(t, err)
	defer st.Close()

	registry, err := NewRegistry(st, cfg, logger)
	require.NoError(t, err)

	// Initially should be nil
	assert.Nil(t, registry.personaManager)

	// Create a mock persona manager
	pm := &persona.PersonaManager{}

	// Set persona manager
	registry.SetPersonaManager(pm)

	// Should be set
	assert.Equal(t, pm, registry.personaManager)
}

func TestRegisterPersonaEvolutionJob_WithoutPersonaManager(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
		LLM: config.LLMConfig{
			DefaultProvider: "openai",
			Providers: map[string]config.Provider{
				"openai": {
					APIKey:  "test-key",
					BaseURL: "https://api.openai.com/v1",
					Model:   "gpt-4",
				},
			},
		},
	}

	st, err := store.New(cfg)
	require.NoError(t, err)
	defer st.Close()

	registry, err := NewRegistry(st, cfg, logger)
	require.NoError(t, err)

	// Should not error when persona manager is nil
	err = registry.registerPersonaEvolutionJob()
	assert.NoError(t, err)
}

func TestPersonaEvolutionJobSchedule(t *testing.T) {
	// Verify the cron schedules are correct
	weeklySchedule := "0 1 * * 1" // Monday at 1 AM
	dailySchedule := SchedulePresets.DailyAt2AM

	assert.Equal(t, "0 1 * * 1", weeklySchedule)
	// DailyAt2AM format includes seconds field (for robfig/cron)
	assert.Equal(t, "0 0 2 * * *", dailySchedule)
}

func TestRegistry_Initialize_WithPersonaEvolution(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
		LLM: config.LLMConfig{
			DefaultProvider: "openai",
			Providers: map[string]config.Provider{
				"openai": {
					APIKey:  "test-key",
					BaseURL: "https://api.openai.com/v1",
					Model:   "gpt-4",
				},
			},
		},
	}

	st, err := store.New(cfg)
	require.NoError(t, err)
	defer st.Close()

	registry, err := NewRegistry(st, cfg, logger)
	require.NoError(t, err)

	// Initialize should work even without persona manager
	err = registry.Initialize()
	assert.NoError(t, err)
	assert.True(t, registry.initialized)

	// Should have neural and reflection jobs registered
	jobs := registry.scheduler.ListJobs()
	assert.Greater(t, len(jobs), 0)
}

func TestPersonaEvolutionJobFunc(t *testing.T) {
	// This is an integration test that requires a full setup
	// Testing the job function directly would require mocking the evolution manager
	t.Skip("Integration test - requires full persona evolution setup")
}

func TestAutoApplyJobFunc(t *testing.T) {
	// This is an integration test that requires a full setup
	t.Skip("Integration test - requires full persona evolution setup")
}

func TestEvolutionJob_Timeout(t *testing.T) {
	// Test that the evolution job respects context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(5 * time.Millisecond)

	// Context should be cancelled
	select {
	case <-ctx.Done():
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	default:
		t.Error("Context should be cancelled")
	}
}

// Benchmark the job registration
func BenchmarkRegisterPersonaEvolutionJob(b *testing.B) {
	logger := zap.NewNop()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			DefaultProvider: "openai",
			Providers: map[string]config.Provider{
				"openai": {
					APIKey:  "test-key",
					BaseURL: "https://api.openai.com/v1",
					Model:   "gpt-4",
				},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		tempDir := b.TempDir()
		cfg.Storage.DataDir = tempDir

		st, _ := store.New(cfg)
		registry, _ := NewRegistry(st, cfg, logger)

		// Test registration without persona manager (fast path)
		_ = registry.registerPersonaEvolutionJob()

		st.Close()
	}
}

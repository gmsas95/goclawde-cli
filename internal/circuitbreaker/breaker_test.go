package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewManager(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.breakers)
	assert.NotNil(t, manager.configs)
	assert.Equal(t, logger, manager.logger)
}

func TestManager_Register(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	cfg := Config{
		MaxRequests:                  10,
		Interval:                     30 * time.Second,
		Timeout:                      60 * time.Second,
		ConsecutiveFailuresThreshold: 3,
	}

	manager.Register("test-service", cfg)

	// Verify breaker was registered
	state, err := manager.GetState("test-service")
	require.NoError(t, err)
	assert.Equal(t, gobreaker.StateClosed, state)
}

func TestManager_Execute_Success(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	cfg := DefaultConfig()
	manager.Register("success-service", cfg)

	result, err := manager.Execute("success-service", func() (interface{}, error) {
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestManager_Execute_Failure(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	cfg := DefaultConfig()
	manager.Register("fail-service", cfg)

	_, err := manager.Execute("fail-service", func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test error")
}

func TestManager_CircuitOpens(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	// Configure to trip after 2 failures
	cfg := Config{
		MaxRequests:                  1,
		Interval:                     30 * time.Second,
		Timeout:                      60 * time.Second,
		ConsecutiveFailuresThreshold: 2,
	}
	manager.Register("circuit-test", cfg)

	// First failure
	_, _ = manager.Execute("circuit-test", func() (interface{}, error) {
		return nil, errors.New("error 1")
	})

	// Second failure - should trip circuit
	_, _ = manager.Execute("circuit-test", func() (interface{}, error) {
		return nil, errors.New("error 2")
	})

	// Third call - circuit should be open
	_, err := manager.Execute("circuit-test", func() (interface{}, error) {
		return "should not execute", nil
	})

	// Should get circuit breaker error
	assert.Error(t, err)
	// The error should be from the circuit breaker, not our function
}

func TestManager_GetState_NotFound(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	_, err := manager.GetState("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_Counts(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	cfg := DefaultConfig()
	manager.Register("count-service", cfg)

	// Execute some requests
	for i := 0; i < 3; i++ {
		manager.Execute("count-service", func() (interface{}, error) {
			return "ok", nil
		})
	}

	counts, err := manager.Counts("count-service")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, counts.Requests, uint32(3))
}

func TestNewHTTPClient(t *testing.T) {
	logger := zap.NewNop()
	client := NewHTTPClient("test-api", 30*time.Second, logger)

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.cb)
	assert.Equal(t, "test-api", client.serviceName)
}

func TestHTTPClient_Do_Success(t *testing.T) {
	logger := zap.NewNop()
	client := NewHTTPClient("success-api", 30*time.Second, logger)

	// This would require a real HTTP server or mock
	// For now, just test that the client was created properly
	assert.NotNil(t, client)
}

func TestHTTPClient_State(t *testing.T) {
	logger := zap.NewNop()
	client := NewHTTPClient("state-api", 30*time.Second, logger)

	state := client.State()
	assert.Equal(t, gobreaker.StateClosed, state)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, uint32(100), cfg.MaxRequests)
	assert.Equal(t, 30*time.Second, cfg.Interval)
	assert.Equal(t, 60*time.Second, cfg.Timeout)
	assert.Equal(t, uint32(5), cfg.ConsecutiveFailuresThreshold)
}

func TestNewDefaultManager(t *testing.T) {
	logger := zap.NewNop()
	manager := NewDefaultManager(logger)

	// Verify all default breakers are registered
	services := []string{GitHubAPI, MCPStdio, MCPSSE, LLM}

	for _, service := range services {
		state, err := manager.GetState(service)
		require.NoError(t, err, "Service %s should be registered", service)
		assert.Equal(t, gobreaker.StateClosed, state)
	}
}

func TestCircuitBreaker_Constants(t *testing.T) {
	assert.Equal(t, "github-api", GitHubAPI)
	assert.Equal(t, "mcp-stdio", MCPStdio)
	assert.Equal(t, "mcp-sse", MCPSSE)
	assert.Equal(t, "llm", LLM)
}

// Package circuitbreaker provides circuit breaker implementations for external services
package circuitbreaker

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"
)

// Config holds circuit breaker configuration
type Config struct {
	// MaxRequests is the maximum number of requests allowed to pass through
	// when the CircuitBreaker is half-open. If MaxRequests is 0,
	// CircuitBreaker allows only 1 request.
	MaxRequests uint32

	// Interval is the cyclic period of the closed state for CircuitBreaker to
	// clear the internal Counts. If Interval is 0, CircuitBreaker doesn't clear
	// internal Counts during the closed state.
	Interval time.Duration

	// Timeout is the period of the open state, after which the state of
	// CircuitBreaker becomes half-open. If Timeout is 0, the default value (60s) is used.
	Timeout time.Duration

	// ConsecutiveFailuresThreshold is the number of consecutive failures that
	// will trip the circuit breaker. Default is 5.
	ConsecutiveFailuresThreshold uint32
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		MaxRequests:                  100,
		Interval:                     30 * time.Second,
		Timeout:                      60 * time.Second,
		ConsecutiveFailuresThreshold: 5,
	}
}

// Manager manages circuit breakers for different services
type Manager struct {
	breakers map[string]*gobreaker.CircuitBreaker[interface{}]
	configs  map[string]Config
	logger   *zap.Logger
}

// NewManager creates a new circuit breaker manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		breakers: make(map[string]*gobreaker.CircuitBreaker[interface{}]),
		configs:  make(map[string]Config),
		logger:   logger,
	}
}

// Register registers a circuit breaker for a service
func (m *Manager) Register(name string, cfg Config) {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.ConsecutiveFailuresThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			m.logger.Warn("Circuit breaker state changed",
				zap.String("service", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}

	cb := gobreaker.NewCircuitBreaker[interface{}](settings)
	m.breakers[name] = cb
	m.configs[name] = cfg

	m.logger.Info("Circuit breaker registered",
		zap.String("service", name),
		zap.Uint32("max_requests", cfg.MaxRequests),
		zap.Duration("interval", cfg.Interval),
		zap.Duration("timeout", cfg.Timeout),
	)
}

// Execute executes a function with circuit breaker protection
func (m *Manager) Execute(name string, fn func() (interface{}, error)) (interface{}, error) {
	cb, exists := m.breakers[name]
	if !exists {
		return nil, fmt.Errorf("circuit breaker '%s' not found", name)
	}

	return cb.Execute(fn)
}

// GetState returns the current state of a circuit breaker
func (m *Manager) GetState(name string) (gobreaker.State, error) {
	cb, exists := m.breakers[name]
	if !exists {
		return gobreaker.StateClosed, fmt.Errorf("circuit breaker '%s' not found", name)
	}

	return cb.State(), nil
}

// Counts returns the current counts of a circuit breaker
func (m *Manager) Counts(name string) (gobreaker.Counts, error) {
	cb, exists := m.breakers[name]
	if !exists {
		return gobreaker.Counts{}, fmt.Errorf("circuit breaker '%s' not found", name)
	}

	return cb.Counts(), nil
}

// Predefined circuit breaker names
const (
	GitHubAPI = "github-api"
	MCPStdio  = "mcp-stdio"
	MCPSSE    = "mcp-sse"
	LLM       = "llm"
)

// NewDefaultManager creates a manager with default circuit breakers
func NewDefaultManager(logger *zap.Logger) *Manager {
	manager := NewManager(logger)
	cfg := DefaultConfig()

	// Register default circuit breakers
	manager.Register(GitHubAPI, cfg)
	manager.Register(MCPStdio, cfg)
	manager.Register(MCPSSE, cfg)
	manager.Register(LLM, cfg)

	return manager
}

// HTTPClient wraps http.Client with circuit breaker
type HTTPClient struct {
	client      *http.Client
	cb          *gobreaker.CircuitBreaker[*http.Response]
	logger      *zap.Logger
	serviceName string
}

// NewHTTPClient creates a new HTTP client with circuit breaker
func NewHTTPClient(serviceName string, timeout time.Duration, logger *zap.Logger) *HTTPClient {
	settings := gobreaker.Settings{
		Name:        serviceName,
		MaxRequests: 100,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("HTTP client circuit breaker state changed",
				zap.String("service", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
		IsSuccessful: func(err error) bool {
			// Consider 4xx errors as successful (client errors, not server failures)
			if err == nil {
				return true
			}
			return false
		},
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		cb:          gobreaker.NewCircuitBreaker[*http.Response](settings),
		logger:      logger,
		serviceName: serviceName,
	}
}

// Do executes an HTTP request with circuit breaker protection
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.cb.Execute(func() (*http.Response, error) {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}

		// Treat 5xx errors as failures for circuit breaker
		if resp.StatusCode >= 500 {
			return resp, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return resp, nil
	})
}

// Get performs an HTTP GET request with circuit breaker protection
func (c *HTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// State returns the current state of the circuit breaker
func (c *HTTPClient) State() gobreaker.State {
	return c.cb.State()
}

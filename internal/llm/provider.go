// Package llm provides LLM client with multi-provider failover support
package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ProviderManager manages multiple LLM providers with failover
type ProviderManager struct {
	providers []ProviderConfig
	current   int
	mu        sync.RWMutex
	logger    *zap.Logger
}

// ProviderConfig holds provider configuration with priority
type ProviderConfig struct {
	Name     string
	Client   *Client
	Priority int    // Lower = higher priority
	Enabled  bool
	LastErr  error
	LastUsed time.Time
}

// NewProviderManager creates a new provider manager
func NewProviderManager(logger *zap.Logger) *ProviderManager {
	return &ProviderManager{
		providers: make([]ProviderConfig, 0),
		current:   0,
		logger:    logger,
	}
}

// AddProvider adds a provider to the manager
func (pm *ProviderManager) AddProvider(name string, client *Client, priority int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.providers = append(pm.providers, ProviderConfig{
		Name:     name,
		Client:   client,
		Priority: priority,
		Enabled:  true,
	})
	
	// Sort by priority
	pm.sortProviders()
}

// GetActiveProvider returns the current active provider
func (pm *ProviderManager) GetActiveProvider() (*Client, string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	for i, p := range pm.providers {
		if p.Enabled && p.LastErr == nil {
			return p.Client, p.Name, nil
		}
		// Skip failed provider but try next
		if i == pm.current {
			continue
		}
	}
	
	return nil, "", fmt.Errorf("no available providers")
}

// ChatCompletion sends a request with automatic failover
func (pm *ProviderManager) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	pm.mu.Lock()
	startIdx := pm.current
	pm.mu.Unlock()
	
	var lastErr error
	
	for i := 0; i < len(pm.providers); i++ {
		idx := (startIdx + i) % len(pm.providers)
		
		pm.mu.RLock()
		provider := pm.providers[idx]
		pm.mu.RUnlock()
		
		if !provider.Enabled {
			continue
		}
		
		resp, err := provider.Client.ChatCompletion(ctx, req)
		if err == nil {
			// Success - update current index
			pm.mu.Lock()
			pm.current = idx
			pm.providers[idx].LastUsed = time.Now()
			pm.providers[idx].LastErr = nil
			pm.mu.Unlock()
			
			if i > 0 {
				pm.logger.Info("Failover successful",
					zap.String("provider", provider.Name),
					zap.Int("attempt", i+1),
				)
			}
			
			return resp, nil
		}
		
		// Mark provider as failed
		pm.mu.Lock()
		pm.providers[idx].LastErr = err
		pm.mu.Unlock()
		
		lastErr = err
		pm.logger.Warn("Provider failed, trying next",
			zap.String("provider", provider.Name),
			zap.Error(err),
		)
	}
	
	// All providers failed
	pm.resetProviders()
	return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
}

// SimpleChat sends a simple chat with failover
func (pm *ProviderManager) SimpleChat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	client, _, err := pm.GetActiveProvider()
	if err != nil {
		return "", err
	}
	
	result, err := client.SimpleChat(ctx, systemPrompt, userMessage)
	if err != nil {
		// Mark current provider as failed and retry
		pm.mu.Lock()
		pm.providers[pm.current].LastErr = err
		pm.mu.Unlock()
		
		// Try next provider
		return pm.SimpleChat(ctx, systemPrompt, userMessage)
	}
	
	return result, nil
}

// resetProviders clears error states after all failed
func (pm *ProviderManager) resetProviders() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for i := range pm.providers {
		pm.providers[i].LastErr = nil
	}
}

// sortProviders sorts providers by priority
func (pm *ProviderManager) sortProviders() {
	// Simple bubble sort for small list
	for i := 0; i < len(pm.providers); i++ {
		for j := i + 1; j < len(pm.providers); j++ {
			if pm.providers[j].Priority < pm.providers[i].Priority {
				pm.providers[i], pm.providers[j] = pm.providers[j], pm.providers[i]
			}
		}
	}
}

// GetProviderStatus returns status of all providers
func (pm *ProviderManager) GetProviderStatus() []map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	status := make([]map[string]interface{}, 0, len(pm.providers))
	for _, p := range pm.providers {
		status = append(status, map[string]interface{}{
			"name":     p.Name,
			"enabled":  p.Enabled,
			"priority": p.Priority,
			"healthy":  p.LastErr == nil,
			"lastUsed": p.LastUsed,
		})
	}
	return status
}

// DisableProvider disables a provider by name
func (pm *ProviderManager) DisableProvider(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for i := range pm.providers {
		if pm.providers[i].Name == name {
			pm.providers[i].Enabled = false
			break
		}
	}
}

// EnableProvider enables a provider by name
func (pm *ProviderManager) EnableProvider(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for i := range pm.providers {
		if pm.providers[i].Name == name {
			pm.providers[i].Enabled = true
			pm.providers[i].LastErr = nil
			break
		}
	}
}

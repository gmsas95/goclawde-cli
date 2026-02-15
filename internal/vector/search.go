// Package vector implements semantic search using vector embeddings
package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/config"
	"github.com/gmsas95/goclawde-cli/internal/store"
	"go.uber.org/zap"
)

// Searcher provides vector search capabilities
type Searcher struct {
	config    *config.VectorConfig
	store     *store.Store
	logger    *zap.Logger
	mu        sync.RWMutex
	cache     map[string][]float32 // In-memory cache for embeddings
	providers map[string]Provider
}

// Provider interface for embedding generation
type Provider interface {
	Name() string
	GenerateEmbedding(text string) ([]float32, error)
	Dimension() int
}

// Result represents a search result
type Result struct {
	MemoryID   string
	Content    string
	Similarity float64
	Type       string
	Importance int
}

// NewSearcher creates a new vector searcher
func NewSearcher(cfg *config.VectorConfig, st *store.Store, logger *zap.Logger) (*Searcher, error) {
	if cfg == nil {
		cfg = &config.VectorConfig{
			Enabled:   false,
			Provider:  "local",
			Dimension: 384,
		}
	}

	s := &Searcher{
		config:    cfg,
		store:     st,
		logger:    logger,
		cache:     make(map[string][]float32),
		providers: make(map[string]Provider),
	}

	// Register providers
	s.registerProviders()

	return s, nil
}

// IsEnabled returns whether vector search is enabled
func (s *Searcher) IsEnabled() bool {
	return s.config.Enabled
}

// registerProviders initializes embedding providers
func (s *Searcher) registerProviders() {
	// Local provider (simplified word-based embeddings)
	s.providers["local"] = NewLocalProvider(s.config.Dimension)

	// OpenAI provider
	if s.config.OpenAIAPIKey != "" {
		s.providers["openai"] = NewOpenAIProvider(s.config.OpenAIAPIKey, s.config.EmbeddingModel)
	}

	// Ollama provider
	s.providers["ollama"] = NewOllamaProvider(s.config.OllamaHost, s.config.EmbeddingModel)
}

// getProvider returns the active provider
func (s *Searcher) getProvider() Provider {
	if provider, ok := s.providers[s.config.Provider]; ok {
		return provider
	}
	// Fallback to local
	return s.providers["local"]
}

// GenerateEmbedding creates an embedding for the given text
func (s *Searcher) GenerateEmbedding(text string) ([]float32, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("vector search is disabled")
	}

	provider := s.getProvider()
	return provider.GenerateEmbedding(text)
}

// IndexMemory indexes a memory for vector search
func (s *Searcher) IndexMemory(memoryID string, content string) error {
	if !s.config.Enabled {
		return nil
	}

	embedding, err := s.GenerateEmbedding(content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Convert to bytes for storage
	embeddingBytes := float32SliceToBytes(embedding)

	// Update memory in database
	if err := s.store.DB().Model(&store.Memory{}).
		Where("id = ?", memoryID).
		Update("embedding", embeddingBytes).Error; err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	// Cache in memory
	s.mu.Lock()
	s.cache[memoryID] = embedding
	s.mu.Unlock()

	s.logger.Debug("Indexed memory for vector search",
		zap.String("memory_id", memoryID),
		zap.Int("dimension", len(embedding)),
	)

	return nil
}

// Search performs semantic search on memories
func (s *Searcher) Search(query string, limit int) ([]Result, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("vector search is disabled")
	}

	// Generate query embedding
	queryEmbedding, err := s.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get all memories with embeddings
	memories, err := s.getMemoriesWithEmbeddings()
	if err != nil {
		return nil, fmt.Errorf("failed to get memories: %w", err)
	}

	if len(memories) == 0 {
		return []Result{}, nil
	}

	// Calculate similarities
	results := make([]Result, 0, len(memories))
	for _, mem := range memories {
		if len(mem.Embedding) == 0 {
			continue
		}

		embedding := bytesToFloat32Slice(mem.Embedding)
		if len(embedding) != len(queryEmbedding) {
			continue // Skip if dimensions don't match
		}

		similarity := cosineSimilarity(queryEmbedding, embedding)
		if similarity > 0.5 { // Threshold for relevance
			results = append(results, Result{
				MemoryID:   mem.ID,
				Content:    mem.Content,
				Similarity: similarity,
				Type:       mem.Type,
				Importance: mem.Importance,
			})
		}
	}

	// Sort by similarity (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Return top results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// SearchWithThreshold performs search with custom similarity threshold
func (s *Searcher) SearchWithThreshold(query string, limit int, threshold float64) ([]Result, error) {
	results, err := s.Search(query, limit*2) // Get more to filter
	if err != nil {
		return nil, err
	}

	// Filter by threshold
	filtered := make([]Result, 0, len(results))
	for _, r := range results {
		if r.Similarity >= threshold {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

// getMemoriesWithEmbeddings retrieves all memories with embeddings
func (s *Searcher) getMemoriesWithEmbeddings() ([]store.Memory, error) {
	var memories []store.Memory
	err := s.store.DB().
		Where("embedding IS NOT NULL AND length(embedding) > 0").
		Order("created_at DESC").
		Limit(1000). // Reasonable limit for in-memory search
		Find(&memories).Error
	return memories, err
}

// ReindexAll reindexes all memories (useful when changing embedding models)
func (s *Searcher) ReindexAll() error {
	if !s.config.Enabled {
		return fmt.Errorf("vector search is disabled")
	}

	s.logger.Info("Starting full memory reindex")

	var memories []store.Memory
	if err := s.store.DB().Find(&memories).Error; err != nil {
		return err
	}

	for _, mem := range memories {
		if err := s.IndexMemory(mem.ID, mem.Content); err != nil {
			s.logger.Error("Failed to index memory",
				zap.String("memory_id", mem.ID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("Memory reindex complete", zap.Int("count", len(memories)))
	return nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// float32SliceToBytes converts float32 slice to bytes
func float32SliceToBytes(f []float32) []byte {
	buf := make([]byte, len(f)*4)
	for i, v := range f {
		bits := math.Float32bits(v)
		buf[i*4] = byte(bits)
		buf[i*4+1] = byte(bits >> 8)
		buf[i*4+2] = byte(bits >> 16)
		buf[i*4+3] = byte(bits >> 24)
	}
	return buf
}

// bytesToFloat32Slice converts bytes to float32 slice
func bytesToFloat32Slice(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	
	f := make([]float32, len(b)/4)
	for i := 0; i < len(f); i++ {
		bits := uint32(b[i*4]) |
			uint32(b[i*4+1])<<8 |
			uint32(b[i*4+2])<<16 |
			uint32(b[i*4+3])<<24
		f[i] = math.Float32frombits(bits)
	}
	return f
}

// ==================== Providers ====================

// LocalProvider provides simple local embeddings
type LocalProvider struct {
	dimension int
	vocab     map[string][]float32
	mu        sync.RWMutex
}

// NewLocalProvider creates a local embedding provider
func NewLocalProvider(dimension int) *LocalProvider {
	return &LocalProvider{
		dimension: dimension,
		vocab:     make(map[string][]float32),
	}
}

func (p *LocalProvider) Name() string { return "local" }

func (p *LocalProvider) Dimension() int { return p.dimension }

func (p *LocalProvider) GenerateEmbedding(text string) ([]float32, error) {
	// Simple word-based embedding using random projections
	// In production, use a proper embedding model like sentence-transformers
	
	words := tokenize(text)
	embedding := make([]float32, p.dimension)
	
	for _, word := range words {
		vec := p.getWordVector(word)
		for i := range embedding {
			embedding[i] += vec[i]
		}
	}
	
	// Normalize
	normalize(embedding)
	
	return embedding, nil
}

func (p *LocalProvider) getWordVector(word string) []float32 {
	p.mu.RLock()
	if vec, ok := p.vocab[word]; ok {
		p.mu.RUnlock()
		return vec
	}
	p.mu.RUnlock()
	
	// Generate deterministic vector for new word
	vec := make([]float32, p.dimension)
	seed := hashString(word)
	for i := range vec {
		vec[i] = float32((seed+uint64(i)*6364136223846793005) % 1000) / 1000.0
	}
	normalize(vec)
	
	p.mu.Lock()
	p.vocab[word] = vec
	p.mu.Unlock()
	
	return vec
}

// OpenAIProvider uses OpenAI's embedding API
type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIProvider creates an OpenAI embedding provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Dimension() int {
	// text-embedding-3-small = 1536, text-embedding-3-large = 3072
	if p.model == "text-embedding-3-large" {
		return 3072
	}
	return 1536
}

func (p *OpenAIProvider) GenerateEmbedding(text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"input": text,
		"model": p.model,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %s", resp.Status)
	}
	
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	
	return result.Data[0].Embedding, nil
}

// OllamaProvider uses local Ollama embeddings
type OllamaProvider struct {
	host  string
	model string
	client *http.Client
}

// NewOllamaProvider creates an Ollama embedding provider
func NewOllamaProvider(host, model string) *OllamaProvider {
	if host == "" {
		host = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaProvider{
		host:   host,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) Dimension() int {
	// nomic-embed-text = 768
	if p.model == "nomic-embed-text" {
		return 768
	}
	return 4096 // Default for most models
}

func (p *OllamaProvider) GenerateEmbedding(text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"model":  p.model,
		"prompt": text,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("POST", p.host+"/api/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API error: %s", resp.Status)
	}
	
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result.Embedding, nil
}

// Helper functions

func tokenize(text string) []string {
	// Simple tokenization - split on non-alphanumeric
	var tokens []string
	var current []rune
	
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			current = append(current, r)
		} else if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = nil
		}
	}
	
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	
	return tokens
}

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x * x)
	}
	
	if sum == 0 {
		return
	}
	
	norm := math.Sqrt(sum)
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
}

func hashString(s string) uint64 {
	var h uint64 = 14695981039346656037 // FNV offset basis
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211 // FNV prime
	}
	return h
}

// Package neural implements neural memory clustering for semantic context retrieval
package neural

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/errors"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/vector"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Retriever handles context retrieval operations using neural clusters
type Retriever struct {
	db        *gorm.DB
	llmClient *llm.Client
	searcher  *vector.Searcher
	logger    *zap.Logger
}

// NewRetriever creates a new context retriever
func NewRetriever(db *gorm.DB, llmClient *llm.Client, searcher *vector.Searcher, logger *zap.Logger) *Retriever {
	return &Retriever{
		db:        db,
		llmClient: llmClient,
		searcher:  searcher,
		logger:    logger,
	}
}

// RetrieveContext retrieves relevant context for a query using neural clusters
// It performs vector similarity search on cluster centroids and returns
// the most relevant clusters and their memories
func (r *Retriever) RetrieveContext(ctx context.Context, query string, opts RetrievalOptions) (*ContextResult, error) {
	start := time.Now()
	result := &ContextResult{
		QueryTime: time.Since(start),
	}

	r.logger.Debug("Retrieving context",
		zap.String("query", query),
		zap.Int("max_clusters", opts.MaxClusters),
	)

	// Generate query embedding
	queryEmbedding, err := r.generateQueryEmbedding(query)
	if err != nil {
		r.logger.Warn("Failed to generate query embedding, falling back to text search",
			zap.Error(err),
		)
		// Fall back to text-based retrieval
		return r.retrieveContextTextSearch(ctx, query, opts)
	}

	// Find relevant clusters via vector search
	clusters, err := r.findRelevantClusters(queryEmbedding, opts)
	if err != nil {
		r.logger.Error("Failed to find relevant clusters",
			zap.Error(err),
		)
		return result, errors.Wrap(err, "NEURAL_014", "cluster search failed")
	}

	result.Clusters = clusters
	result.TotalClusters = len(clusters)

	// Get memories from clusters
	memories, err := r.getMemoriesFromClusters(clusters, queryEmbedding, opts)
	if err != nil {
		r.logger.Error("Failed to get memories from clusters",
			zap.Error(err),
		)
		return result, errors.Wrap(err, "NEURAL_015", "memory retrieval failed")
	}

	result.Memories = memories
	result.TotalMemories = len(memories)

	// Estimate tokens
	result.TokenEstimate = r.estimateTokens(memories, clusters, opts)

	// Update cluster access statistics
	r.updateClusterAccess(clusters)

	// Log query pattern for optimization
	r.logQueryPattern(query, queryEmbedding, clusters, memories, start)

	result.QueryTime = time.Since(start)

	r.logger.Debug("Context retrieval complete",
		zap.Int("clusters", len(clusters)),
		zap.Int("memories", len(memories)),
		zap.Int("tokens", result.TokenEstimate),
		zap.Duration("duration", result.QueryTime),
	)

	return result, nil
}

// generateQueryEmbedding generates an embedding for the query
func (r *Retriever) generateQueryEmbedding(query string) ([]float32, error) {
	if r.searcher != nil && r.searcher.IsEnabled() {
		return r.searcher.GenerateEmbedding(query)
	}

	// Fallback to local embedding generation
	return r.generateLocalEmbedding(query), nil
}

// generateLocalEmbedding creates a simple local embedding for queries
func (r *Retriever) generateLocalEmbedding(text string) []float32 {
	// Simple word-based embedding - same as vector package
	dimension := 384
	words := r.tokenize(text)
	embedding := make([]float32, dimension)

	for _, word := range words {
		vec := r.getWordVector(word, dimension)
		for i := range embedding {
			embedding[i] += vec[i]
		}
	}

	// Normalize
	r.normalize(embedding)

	return embedding
}

func (r *Retriever) tokenize(text string) []string {
	var tokens []string
	var current []rune

	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			current = append(current, c)
		} else if len(current) > 0 {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = nil
		}
	}

	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}

	return tokens
}

func (r *Retriever) getWordVector(word string, dimension int) []float32 {
	vec := make([]float32, dimension)
	seed := r.hashString(word)
	for i := range vec {
		vec[i] = float32((seed+uint64(i)*6364136223846793005)%1000) / 1000.0
	}
	r.normalize(vec)
	return vec
}

func (r *Retriever) normalize(v []float32) {
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

func (r *Retriever) hashString(s string) uint64 {
	var h uint64 = 14695981039346656037
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h
}

// findRelevantClusters finds clusters similar to the query embedding
func (r *Retriever) findRelevantClusters(queryEmbedding []float32, opts RetrievalOptions) ([]*NeuralCluster, error) {
	// Get all clusters with embeddings
	var clusters []NeuralCluster
	err := r.db.Where("embedding IS NOT NULL AND length(embedding) > 0").
		Order("confidence_score DESC").
		Limit(1000). // Reasonable limit for in-memory search
		Find(&clusters).Error

	if err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return []*NeuralCluster{}, nil
	}

	// Calculate similarity scores
	type scoredCluster struct {
		cluster    *NeuralCluster
		similarity float64
	}

	scored := make([]scoredCluster, 0, len(clusters))

	for i := range clusters {
		cluster := &clusters[i]
		clusterEmbedding := r.bytesToFloat32Slice(cluster.Embedding)

		if len(clusterEmbedding) != len(queryEmbedding) {
			continue
		}

		similarity := r.cosineSimilarity(queryEmbedding, clusterEmbedding)

		// Apply confidence threshold
		if cluster.ConfidenceScore < opts.MinClusterConfidence {
			similarity *= 0.5 // Penalize low confidence clusters
		}

		// Boost recently accessed clusters
		if opts.BoostRecentAccess && cluster.LastAccessed != nil {
			if time.Since(*cluster.LastAccessed) < opts.RecentAccessWindow {
				similarity *= 1.1 // 10% boost
			}
		}

		scored = append(scored, scoredCluster{
			cluster:    cluster,
			similarity: similarity,
		})
	}

	// Sort by similarity
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].similarity > scored[j].similarity
	})

	// Take top N
	limit := opts.MaxClusters
	if limit > len(scored) {
		limit = len(scored)
	}

	result := make([]*NeuralCluster, limit)
	for i := 0; i < limit; i++ {
		result[i] = scored[i].cluster
	}

	return result, nil
}

// getMemoriesFromClusters retrieves the most relevant memories from clusters
func (r *Retriever) getMemoriesFromClusters(clusters []*NeuralCluster, queryEmbedding []float32, opts RetrievalOptions) ([]MemoryRef, error) {
	if len(clusters) == 0 {
		return []MemoryRef{}, nil
	}

	// Collect cluster IDs
	clusterIDs := make([]string, len(clusters))
	for i, c := range clusters {
		clusterIDs[i] = c.ID
	}

	// Get cluster-memory associations
	var clusterMems []ClusterMemory
	err := r.db.Where("cluster_id IN ?", clusterIDs).
		Order("similarity_score DESC").
		Find(&clusterMems).Error

	if err != nil {
		return nil, err
	}

	// Group by cluster
	memoriesByCluster := make(map[string][]ClusterMemory)
	for _, cm := range clusterMems {
		memoriesByCluster[cm.ClusterID] = append(memoriesByCluster[cm.ClusterID], cm)
	}

	// Get memory details
	memoryIDs := make([]string, len(clusterMems))
	for i, cm := range clusterMems {
		memoryIDs[i] = cm.MemoryID
	}

	var memories []store.Memory
	if len(memoryIDs) > 0 {
		r.db.Where("id IN ?", memoryIDs).Find(&memories)
	}

	memoryMap := make(map[string]store.Memory)
	for _, mem := range memories {
		memoryMap[mem.ID] = mem
	}

	// Create memory refs with scores
	result := make([]MemoryRef, 0)

	for _, cluster := range clusters {
		clusterMemories := memoriesByCluster[cluster.ID]

		// Sort by similarity score within cluster
		sort.Slice(clusterMemories, func(i, j int) bool {
			return clusterMemories[i].SimilarityScore > clusterMemories[j].SimilarityScore
		})

		// Limit memories per cluster
		limit := opts.MaxMemoriesPerCluster
		if limit > len(clusterMemories) {
			limit = len(clusterMemories)
		}

		for i := 0; i < limit; i++ {
			cm := clusterMemories[i]
			mem, ok := memoryMap[cm.MemoryID]
			if !ok {
				continue
			}

			// Calculate final similarity
			similarity := cm.SimilarityScore
			if similarity < opts.MinMemorySimilarity {
				continue
			}

			result = append(result, MemoryRef{
				ID:         mem.ID,
				Content:    mem.Content,
				Type:       mem.Type,
				Importance: mem.Importance,
				ClusterID:  cluster.ID,
				Similarity: similarity,
				CreatedAt:  mem.CreatedAt,
			})
		}
	}

	// Sort all memories by similarity
	sort.Slice(result, func(i, j int) bool {
		return result[i].Similarity > result[j].Similarity
	})

	return result, nil
}

// retrieveContextTextSearch performs text-based retrieval as fallback
func (r *Retriever) retrieveContextTextSearch(ctx context.Context, query string, opts RetrievalOptions) (*ContextResult, error) {
	result := &ContextResult{}

	// Search for memories containing query terms
	searchPattern := "%" + query + "%"
	var memories []store.Memory
	err := r.db.Where("content LIKE ?", searchPattern).
		Order("importance DESC, created_at DESC").
		Limit(opts.MaxClusters * opts.MaxMemoriesPerCluster).
		Find(&memories).Error

	if err != nil {
		return result, err
	}

	// Convert to memory refs
	refs := make([]MemoryRef, len(memories))
	for i, mem := range memories {
		refs[i] = MemoryRef{
			ID:         mem.ID,
			Content:    mem.Content,
			Type:       mem.Type,
			Importance: mem.Importance,
			Similarity: 0.5, // Default similarity for text search
			CreatedAt:  mem.CreatedAt,
		}
	}

	result.Memories = refs
	result.TotalMemories = len(refs)
	result.TokenEstimate = r.estimateTokens(refs, nil, opts)

	return result, nil
}

// updateClusterAccess updates access statistics for clusters
func (r *Retriever) updateClusterAccess(clusters []*NeuralCluster) {
	for _, cluster := range clusters {
		cluster.IncrementAccess()
		r.db.Save(cluster)
	}
}

// logQueryPattern logs query patterns for future optimization
func (r *Retriever) logQueryPattern(query string, embedding []float32, clusters []*NeuralCluster, memories []MemoryRef, start time.Time) {
	clusterIDs := make([]string, len(clusters))
	for i, c := range clusters {
		clusterIDs[i] = c.ID
	}

	memoryIDs := make([]string, len(memories))
	for i, m := range memories {
		memoryIDs[i] = m.ID
	}

	clusterIDsJSON, _ := json.Marshal(clusterIDs)
	memoryIDsJSON, _ := json.Marshal(memoryIDs)

	pattern := &QueryPattern{
		QueryText:         query,
		QueryEmbedding:    r.float32SliceToBytes(embedding),
		MatchedClusterIDs: clusterIDs,
		MatchedMemoryIDs:  memoryIDs,
		LatencyMs:         int(time.Since(start).Milliseconds()),
		CreatedAt:         time.Now(),
	}

	// Store JSON versions for database
	patternData := map[string]interface{}{
		"matched_cluster_ids": string(clusterIDsJSON),
		"matched_memory_ids":  string(memoryIDsJSON),
	}
	detailsJSON, _ := json.Marshal(patternData)

	// We don't have a field for these, but we can extend the schema if needed
	_ = detailsJSON

	r.db.Create(pattern)
}

// estimateTokens estimates the total token count for the context
func (r *Retriever) estimateTokens(memories []MemoryRef, clusters []*NeuralCluster, opts RetrievalOptions) int {
	// Rough estimate: ~4 characters per token
	totalChars := 0

	// Cluster metadata
	if opts.IncludeMetadata && clusters != nil {
		for _, cluster := range clusters {
			totalChars += len(cluster.Theme)
			totalChars += len(cluster.Essence)
		}
	}

	// Memory contents
	for _, mem := range memories {
		totalChars += len(mem.Content)
	}

	tokens := totalChars / 4

	// Add overhead for formatting
	tokens += len(memories) * 10

	return tokens
}

// cosineSimilarity calculates cosine similarity
func (r *Retriever) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
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

// bytesToFloat32Slice converts bytes to float32 slice
func (r *Retriever) bytesToFloat32Slice(b []byte) []float32 {
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

// float32SliceToBytes converts float32 slice to bytes
func (r *Retriever) float32SliceToBytes(f []float32) []byte {
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

// SearchClusters searches for clusters by theme/content (text search)
func (r *Retriever) SearchClusters(query string, limit int) ([]*NeuralCluster, error) {
	searchPattern := "%" + query + "%"

	var clusters []NeuralCluster
	err := r.db.Where("theme LIKE ? OR essence LIKE ?", searchPattern, searchPattern).
		Order("confidence_score DESC, access_count DESC").
		Limit(limit).
		Find(&clusters).Error

	if err != nil {
		return nil, errors.Wrap(err, "NEURAL_016", "cluster search failed")
	}

	result := make([]*NeuralCluster, len(clusters))
	for i := range clusters {
		result[i] = &clusters[i]
	}

	return result, nil
}

// GetCluster retrieves a specific cluster by ID with all its memories
func (r *Retriever) GetCluster(clusterID string) (*NeuralCluster, []store.Memory, error) {
	var cluster NeuralCluster
	if err := r.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return nil, nil, errors.Wrap(err, "NEURAL_017", "cluster not found")
	}

	// Get memory IDs
	var clusterMems []ClusterMemory
	r.db.Where("cluster_id = ?", clusterID).Find(&clusterMems)

	memoryIDs := make([]string, len(clusterMems))
	for i, cm := range clusterMems {
		memoryIDs[i] = cm.MemoryID
	}

	// Get memories
	var memories []store.Memory
	if len(memoryIDs) > 0 {
		r.db.Where("id IN ?", memoryIDs).Find(&memories)
	}

	// Update access
	cluster.IncrementAccess()
	r.db.Save(&cluster)

	return &cluster, memories, nil
}

// ListClusters returns a paginated list of clusters
func (r *Retriever) ListClusters(filter ClusterFilter) ([]*NeuralCluster, int64, error) {
	query := r.db.Model(&NeuralCluster{})

	// Apply filters
	if filter.MinConfidence > 0 {
		query = query.Where("confidence_score >= ?", filter.MinConfidence)
	}
	if filter.MaxConfidence > 0 {
		query = query.Where("confidence_score <= ?", filter.MaxConfidence)
	}
	if filter.MinSize > 0 {
		query = query.Where("cluster_size >= ?", filter.MinSize)
	}
	if filter.MaxSize > 0 {
		query = query.Where("cluster_size <= ?", filter.MaxSize)
	}
	if filter.ThemeContains != "" {
		query = query.Where("theme LIKE ?", "%"+filter.ThemeContains+"%")
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", filter.CreatedBefore)
	}
	if filter.AccessedAfter != nil {
		query = query.Where("last_accessed >= ?", filter.AccessedAfter)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply ordering
	query = query.Order("confidence_score DESC, cluster_size DESC")

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Get results
	var clusters []NeuralCluster
	if err := query.Find(&clusters).Error; err != nil {
		return nil, 0, errors.Wrap(err, "NEURAL_018", "failed to list clusters")
	}

	result := make([]*NeuralCluster, len(clusters))
	for i := range clusters {
		result[i] = &clusters[i]
	}

	return result, total, nil
}

// GetStatistics returns cluster statistics
func (r *Retriever) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalClusters int64
	r.db.Model(&NeuralCluster{}).Count(&totalClusters)
	stats["total_clusters"] = totalClusters

	var totalMemories int64
	r.db.Model(&ClusterMemory{}).Count(&totalMemories)
	stats["total_memories_clustered"] = totalMemories

	var avgConfidence float64
	r.db.Model(&NeuralCluster{}).Select("AVG(confidence_score)").Scan(&avgConfidence)
	stats["average_confidence"] = avgConfidence

	var avgSize float64
	r.db.Model(&NeuralCluster{}).Select("AVG(cluster_size)").Scan(&avgSize)
	stats["average_cluster_size"] = avgSize

	var maxSize int
	r.db.Model(&NeuralCluster{}).Select("MAX(cluster_size)").Scan(&maxSize)
	stats["max_cluster_size"] = maxSize

	var totalAccesses int64
	r.db.Model(&NeuralCluster{}).Select("SUM(access_count)").Scan(&totalAccesses)
	stats["total_accesses"] = totalAccesses

	// Recently created clusters
	weekAgo := time.Now().AddDate(0, 0, -7)
	var recentClusters int64
	r.db.Model(&NeuralCluster{}).Where("created_at >= ?", weekAgo).Count(&recentClusters)
	stats["clusters_created_last_7_days"] = recentClusters

	return stats, nil
}

// Package neural implements neural memory clustering for semantic context retrieval
package neural

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/errors"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/vector"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ClusterManager handles neural cluster operations
type ClusterManager struct {
	db        *gorm.DB
	llmClient *llm.Client
	searcher  *vector.Searcher
	logger    *zap.Logger
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(db *gorm.DB, llmClient *llm.Client, searcher *vector.Searcher, logger *zap.Logger) (*ClusterManager, error) {
	// Auto-migrate schemas
	if err := db.AutoMigrate(
		&NeuralCluster{},
		&ClusterMemory{},
		&ClusterFormationLog{},
		&QueryPattern{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate neural schemas: %w", err)
	}

	// Create indexes for performance
	createIndexes(db)

	return &ClusterManager{
		db:        db,
		llmClient: llmClient,
		searcher:  searcher,
		logger:    logger,
	}, nil
}

// createIndexes creates database indexes for optimal query performance
func createIndexes(db *gorm.DB) {
	// Neural cluster indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_neural_clusters_theme ON neural_clusters(theme)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_neural_clusters_confidence ON neural_clusters(confidence_score DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_neural_clusters_size ON neural_clusters(cluster_size DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_neural_clusters_access ON neural_clusters(access_count DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_neural_clusters_created ON neural_clusters(created_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_neural_clusters_updated ON neural_clusters(updated_at DESC)")

	// Cluster memories indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_cluster_memories_memory ON cluster_memories(memory_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_cluster_memories_similarity ON cluster_memories(similarity_score DESC)")

	// Formation logs indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_formation_logs_cluster ON cluster_formation_logs(cluster_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_formation_logs_operation ON cluster_formation_logs(operation)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_formation_logs_created ON cluster_formation_logs(created_at DESC)")

	// Query patterns indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_query_patterns_created ON cluster_query_patterns(created_at DESC)")
}

// FormClusters performs cluster formation on unclustered memories
// It groups memories by semantic similarity and generates cluster metadata
func (cm *ClusterManager) FormClusters(ctx context.Context, opts FormationOptions) (*FormationResult, error) {
	start := time.Now()
	result := &FormationResult{}

	cm.logger.Info("Starting cluster formation",
		zap.Float64("similarity_threshold", opts.SimilarityThreshold),
		zap.Int("min_cluster_size", opts.MinClusterSize),
		zap.Int("max_clusters", opts.MaxClusters),
	)

	// Get unclustered memories
	memories, err := cm.getUnclusteredMemories(opts.BatchSize)
	if err != nil {
		return result, errors.Wrap(err, "NEURAL_001", "failed to get unclustered memories")
	}

	if len(memories) == 0 {
		cm.logger.Info("No unclustered memories found")
		return result, nil
	}

	result.MemoriesProcessed = len(memories)
	cm.logger.Info("Processing memories", zap.Int("count", len(memories)))

	// Generate embeddings for memories if needed
	memoriesWithEmbeddings, err := cm.ensureEmbeddings(ctx, memories)
	if err != nil {
		return result, errors.Wrap(err, "NEURAL_002", "failed to generate embeddings")
	}

	// Perform clustering
	clusters, err := cm.clusterMemories(memoriesWithEmbeddings, opts)
	if err != nil {
		return result, errors.Wrap(err, "NEURAL_003", "failed to cluster memories")
	}

	// Process each cluster
	for _, clusterMems := range clusters {
		if err := ctx.Err(); err != nil {
			return result, errors.Wrap(err, "NEURAL_004", "formation cancelled")
		}

		if len(clusterMems) < opts.MinClusterSize {
			continue // Skip small clusters
		}

		// Create or update cluster
		cluster, err := cm.createClusterFromMemories(ctx, clusterMems, opts)
		if err != nil {
			cm.logger.Error("Failed to create cluster",
				zap.Int("memory_count", len(clusterMems)),
				zap.Error(err),
			)
			result.Errors = append(result.Errors, err)
			continue
		}

		if cluster != nil {
			result.ClustersCreated++
			result.MemoriesClustered += len(clusterMems)
			cm.logFormation(cluster, "create", len(clusterMems), start)
		}
	}

	result.Duration = time.Since(start)
	cm.logger.Info("Cluster formation complete",
		zap.Int("clusters_created", result.ClustersCreated),
		zap.Int("memories_clustered", result.MemoriesClustered),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// getUnclusteredMemories retrieves memories that aren't yet in any cluster
func (cm *ClusterManager) getUnclusteredMemories(limit int) ([]store.Memory, error) {
	var memories []store.Memory

	// Get memories not in any cluster
	err := cm.db.Raw(`
		SELECT m.* FROM memories m
		WHERE m.embedding IS NOT NULL 
		AND m.embedding != ''
		AND m.id NOT IN (
			SELECT DISTINCT memory_id FROM cluster_memories
		)
		ORDER BY m.created_at DESC
		LIMIT ?
	`, limit).Scan(&memories).Error

	return memories, err
}

// ensureEmbeddings generates embeddings for memories that don't have them
func (cm *ClusterManager) ensureEmbeddings(ctx context.Context, memories []store.Memory) ([]*MemoryWithEmbedding, error) {
	result := make([]*MemoryWithEmbedding, 0, len(memories))

	for _, mem := range memories {
		embedding := cm.bytesToFloat32Slice(mem.Embedding)

		// If no embedding and vector search is enabled, generate one
		if len(embedding) == 0 && cm.searcher != nil && cm.searcher.IsEnabled() {
			newEmbedding, err := cm.searcher.GenerateEmbedding(mem.Content)
			if err != nil {
				cm.logger.Warn("Failed to generate embedding",
					zap.String("memory_id", mem.ID),
					zap.Error(err),
				)
				continue
			}
			embedding = newEmbedding

			// Store the new embedding
			embeddingBytes := cm.float32SliceToBytes(embedding)
			cm.db.Model(&store.Memory{}).Where("id = ?", mem.ID).Update("embedding", embeddingBytes)
		}

		result = append(result, &MemoryWithEmbedding{
			Memory:    mem,
			Embedding: embedding,
		})
	}

	return result, nil
}

// MemoryWithEmbedding combines a memory with its vector embedding
type MemoryWithEmbedding struct {
	store.Memory
	Embedding []float32
}

// clusterMemories groups memories by semantic similarity using hierarchical clustering
func (cm *ClusterManager) clusterMemories(memories []*MemoryWithEmbedding, opts FormationOptions) ([][]*MemoryWithEmbedding, error) {
	if len(memories) < opts.MinClusterSize {
		return nil, nil
	}

	// Simple greedy clustering algorithm
	// For production, consider using DBSCAN or HDBSCAN for better results
	clusters := make([][]*MemoryWithEmbedding, 0)
	used := make(map[string]bool)

	for i, mem1 := range memories {
		if used[mem1.ID] {
			continue
		}

		cluster := []*MemoryWithEmbedding{mem1}
		used[mem1.ID] = true

		for j, mem2 := range memories {
			if i == j || used[mem2.ID] {
				continue
			}

			similarity := cosineSimilarity(mem1.Embedding, mem2.Embedding)
			if float64(similarity) >= opts.SimilarityThreshold {
				cluster = append(cluster, mem2)
				used[mem2.ID] = true
			}
		}

		if len(cluster) >= opts.MinClusterSize {
			clusters = append(clusters, cluster)
		}

		// Check max clusters limit
		if len(clusters) >= opts.MaxClusters {
			break
		}
	}

	return clusters, nil
}

// createClusterFromMemories creates a new cluster from a group of memories
func (cm *ClusterManager) createClusterFromMemories(ctx context.Context, memories []*MemoryWithEmbedding, opts FormationOptions) (*NeuralCluster, error) {
	if len(memories) == 0 {
		return nil, nil
	}

	// Extract memory IDs
	memoryIDs := make([]string, len(memories))
	memoryContents := make([]string, len(memories))
	for i, mem := range memories {
		memoryIDs[i] = mem.ID
		memoryContents[i] = mem.Content
	}

	// Calculate centroid embedding
	centroid := cm.calculateCentroid(memories)
	embeddingBytes := cm.float32SliceToBytes(centroid)

	// Generate theme
	theme, err := cm.extractTheme(ctx, memoryContents)
	if err != nil {
		cm.logger.Warn("Failed to extract theme", zap.Error(err))
		theme = cm.generateSimpleTheme(memoryContents)
	}

	// Generate essence using LLM
	essence, err := cm.generateEssence(ctx, memoryContents, theme)
	if err != nil {
		cm.logger.Warn("Failed to generate essence", zap.Error(err))
		essence = cm.generateSimpleEssence(memoryContents)
	}

	// Calculate confidence score
	confidence := cm.calculateConfidence(memories, centroid)

	// Create cluster
	cluster := &NeuralCluster{
		Theme:           theme,
		Essence:         essence,
		Embedding:       embeddingBytes,
		ConfidenceScore: confidence,
		ClusterSize:     len(memories),
		AccessCount:     0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Set memory IDs
	if err := cluster.SetMemoryIDs(memoryIDs); err != nil {
		return nil, err
	}

	// Set metadata
	metadata := Metadata{
		FormationMethod:  "similarity",
		FormationVersion: "1.0",
		SourceMemories:   len(memories),
		Tags:             []string{},
	}
	if err := cluster.SetMetadata(metadata); err != nil {
		return nil, err
	}

	// Save to database
	if err := cm.db.Create(cluster).Error; err != nil {
		return nil, errors.Wrap(err, "NEURAL_005", "failed to save cluster")
	}

	// Create cluster-memory associations
	for _, mem := range memories {
		similarity := cosineSimilarity(centroid, mem.Embedding)
		clusterMem := &ClusterMemory{
			ClusterID:        cluster.ID,
			MemoryID:         mem.ID,
			SimilarityScore:  float64(similarity),
			IsRepresentative: false,
			AddedAt:          time.Now(),
		}
		cm.db.Create(clusterMem)
	}

	// Mark the most representative memory
	if err := cm.markRepresentativeMemory(cluster.ID, centroid); err != nil {
		cm.logger.Warn("Failed to mark representative memory", zap.Error(err))
	}

	return cluster, nil
}

// markRepresentativeMemory marks the memory most similar to centroid as representative
func (cm *ClusterManager) markRepresentativeMemory(clusterID string, centroid []float32) error {
	var bestMemID string
	var bestSimilarity float32 = -1

	var clusterMems []ClusterMemory
	cm.db.Where("cluster_id = ?", clusterID).Find(&clusterMems)

	for _, cm := range clusterMems {
		if cm.SimilarityScore > float64(bestSimilarity) {
			bestSimilarity = float32(cm.SimilarityScore)
			bestMemID = cm.MemoryID
		}
	}

	if bestMemID != "" {
		return cm.db.Model(&ClusterMemory{}).
			Where("cluster_id = ? AND memory_id = ?", clusterID, bestMemID).
			Update("is_representative", true).Error
	}

	return nil
}

// generateEssence generates a natural language summary of a cluster using LLM
func (cm *ClusterManager) generateEssence(ctx context.Context, memoryContents []string, theme string) (string, error) {
	if cm.llmClient == nil {
		return cm.generateSimpleEssence(memoryContents), nil
	}

	// Limit content length for LLM
	combinedContent := strings.Join(memoryContents, "\n---\n")
	if len(combinedContent) > 8000 {
		combinedContent = combinedContent[:8000] + "..."
	}

	prompt := fmt.Sprintf(`Generate a concise, natural language summary (2-3 sentences) that captures the essence of these related memories.

Theme: %s

Memories:
%s

Essence:`, theme, combinedContent)

	systemPrompt := "You are a helpful assistant that synthesizes information into clear, concise summaries. Focus on the key insights and relationships between the memories."

	resp, err := cm.llmClient.SimpleChat(ctx, systemPrompt, prompt)
	if err != nil {
		return "", err
	}

	// Clean up response
	resp = strings.TrimSpace(resp)
	resp = strings.Trim(resp, `"'`)

	return resp, nil
}

// extractTheme extracts a short theme/title for a cluster using LLM
func (cm *ClusterManager) extractTheme(ctx context.Context, memoryContents []string) (string, error) {
	if cm.llmClient == nil {
		return cm.generateSimpleTheme(memoryContents), nil
	}

	// Limit to first few memories for theme extraction
	sampleSize := 5
	if len(memoryContents) < sampleSize {
		sampleSize = len(memoryContents)
	}

	combinedContent := strings.Join(memoryContents[:sampleSize], "\n")
	if len(combinedContent) > 2000 {
		combinedContent = combinedContent[:2000] + "..."
	}

	prompt := fmt.Sprintf(`Generate a short, descriptive theme (3-5 words) that captures the common topic of these memories:

%s

Theme:`, combinedContent)

	systemPrompt := "You are a helpful assistant that identifies themes. Respond with just the theme, no explanation."

	resp, err := cm.llmClient.SimpleChat(ctx, systemPrompt, prompt)
	if err != nil {
		return "", err
	}

	// Clean up response
	resp = strings.TrimSpace(resp)
	resp = strings.Trim(resp, `"'`)

	// Limit length
	if len(resp) > 255 {
		resp = resp[:255]
	}

	return resp, nil
}

// calculateConfidence calculates a confidence score for a cluster
// based on internal similarity and coherence
func (cm *ClusterManager) calculateConfidence(memories []*MemoryWithEmbedding, centroid []float32) float64 {
	if len(memories) < 2 {
		return 0.5
	}

	// Calculate average similarity to centroid
	var totalSimilarity float64
	for _, mem := range memories {
		sim := cosineSimilarity(centroid, mem.Embedding)
		totalSimilarity += float64(sim)
	}
	avgSimilarity := totalSimilarity / float64(len(memories))

	// Penalize very large clusters slightly (diminishing returns)
	sizePenalty := 1.0
	if len(memories) > 50 {
		sizePenalty = 0.95
	}

	// Calculate final confidence
	confidence := avgSimilarity * sizePenalty

	// Ensure within bounds
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

// generateSimpleEssence generates a simple essence without LLM
func (cm *ClusterManager) generateSimpleEssence(memoryContents []string) string {
	if len(memoryContents) == 0 {
		return "Empty cluster"
	}

	// Extract key words from first memory
	words := strings.Fields(memoryContents[0])
	if len(words) > 20 {
		words = words[:20]
	}

	essence := fmt.Sprintf("Collection of %d memories about %s", len(memoryContents), strings.Join(words, " "))
	if len(essence) > 500 {
		essence = essence[:500]
	}

	return essence
}

// generateSimpleTheme generates a simple theme without LLM
func (cm *ClusterManager) generateSimpleTheme(memoryContents []string) string {
	if len(memoryContents) == 0 {
		return "Untitled Cluster"
	}

	// Extract first few words from first memory
	words := strings.Fields(memoryContents[0])
	if len(words) > 5 {
		words = words[:5]
	}

	theme := strings.Join(words, " ")
	if len(theme) > 255 {
		theme = theme[:255]
	}

	return theme
}

// calculateCentroid calculates the centroid (average) of a set of embeddings
func (cm *ClusterManager) calculateCentroid(memories []*MemoryWithEmbedding) []float32 {
	if len(memories) == 0 {
		return nil
	}

	dim := len(memories[0].Embedding)
	centroid := make([]float32, dim)

	for _, mem := range memories {
		for i, val := range mem.Embedding {
			centroid[i] += val
		}
	}

	for i := range centroid {
		centroid[i] /= float32(len(memories))
	}

	return centroid
}

// logFormation logs a cluster formation event
func (cm *ClusterManager) logFormation(cluster *NeuralCluster, operation string, memoryCount int, start time.Time) {
	log := &ClusterFormationLog{
		ClusterID:   cluster.ID,
		Operation:   operation,
		MemoryCount: memoryCount,
		NewState:    cluster.Essence,
		DurationMs:  int(time.Since(start).Milliseconds()),
		CreatedAt:   time.Now(),
	}

	details := map[string]interface{}{
		"theme":      cluster.Theme,
		"confidence": cluster.ConfidenceScore,
		"size":       cluster.ClusterSize,
	}
	detailsJSON, _ := json.Marshal(details)
	log.Details = string(detailsJSON)

	cm.db.Create(log)
}

// float32SliceToBytes converts a float32 slice to bytes
func (cm *ClusterManager) float32SliceToBytes(f []float32) []byte {
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
func (cm *ClusterManager) bytesToFloat32Slice(b []byte) []float32 {
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

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
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

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// RefreshCluster refreshes a cluster by re-analyzing its memories
func (cm *ClusterManager) RefreshCluster(ctx context.Context, clusterID string) error {
	var cluster NeuralCluster
	if err := cm.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return errors.Wrap(err, "NEURAL_006", "cluster not found")
	}

	// Get current memories in cluster
	var clusterMems []ClusterMemory
	if err := cm.db.Where("cluster_id = ?", clusterID).Find(&clusterMems).Error; err != nil {
		return errors.Wrap(err, "NEURAL_007", "failed to get cluster memories")
	}

	memoryIDs := make([]string, len(clusterMems))
	for i, cm := range clusterMems {
		memoryIDs[i] = cm.MemoryID
	}

	// Get memory contents
	var memories []store.Memory
	if err := cm.db.Where("id IN ?", memoryIDs).Find(&memories).Error; err != nil {
		return errors.Wrap(err, "NEURAL_008", "failed to fetch memories")
	}

	if len(memories) == 0 {
		// Delete empty cluster
		cm.db.Delete(&cluster)
		cm.logFormation(&cluster, "delete", 0, time.Now())
		return nil
	}

	// Refresh embeddings
	memoriesWithEmbeddings, err := cm.ensureEmbeddings(ctx, memories)
	if err != nil {
		return errors.Wrap(err, "NEURAL_009", "failed to ensure embeddings")
	}

	// Recalculate centroid
	centroid := cm.calculateCentroid(memoriesWithEmbeddings)
	cluster.Embedding = cm.float32SliceToBytes(centroid)

	// Recalculate confidence
	cluster.ConfidenceScore = cm.calculateConfidence(memoriesWithEmbeddings, centroid)

	// Regenerate theme and essence
	memoryContents := make([]string, len(memories))
	for i, mem := range memories {
		memoryContents[i] = mem.Content
	}

	theme, err := cm.extractTheme(ctx, memoryContents)
	if err == nil {
		cluster.Theme = theme
	}

	essence, err := cm.generateEssence(ctx, memoryContents, cluster.Theme)
	if err == nil {
		cluster.Essence = essence
	}

	cluster.UpdatedAt = time.Now()

	// Save updated cluster
	if err := cm.db.Save(&cluster).Error; err != nil {
		return errors.Wrap(err, "NEURAL_010", "failed to save refreshed cluster")
	}

	cm.logFormation(&cluster, "refresh", len(memories), time.Now())

	return nil
}

// MergeClusters merges two clusters into one
func (cm *ClusterManager) MergeClusters(ctx context.Context, clusterID1, clusterID2 string) (*NeuralCluster, error) {
	var cluster1, cluster2 NeuralCluster

	if err := cm.db.First(&cluster1, "id = ?", clusterID1).Error; err != nil {
		return nil, errors.Wrap(err, "NEURAL_011", "first cluster not found")
	}

	if err := cm.db.First(&cluster2, "id = ?", clusterID2).Error; err != nil {
		return nil, errors.Wrap(err, "NEURAL_012", "second cluster not found")
	}

	// Get all memories from both clusters
	var clusterMems1, clusterMems2 []ClusterMemory
	cm.db.Where("cluster_id = ?", clusterID1).Find(&clusterMems1)
	cm.db.Where("cluster_id = ?", clusterID2).Find(&clusterMems2)

	allMemIDs := make(map[string]bool)
	for _, cm := range clusterMems1 {
		allMemIDs[cm.MemoryID] = true
	}
	for _, cm := range clusterMems2 {
		allMemIDs[cm.MemoryID] = true
	}

	memoryIDs := make([]string, 0, len(allMemIDs))
	for id := range allMemIDs {
		memoryIDs = append(memoryIDs, id)
	}

	// Delete old clusters
	cm.db.Where("id IN ?", []string{clusterID1, clusterID2}).Delete(&NeuralCluster{})
	cm.db.Where("cluster_id IN ?", []string{clusterID1, clusterID2}).Delete(&ClusterMemory{})

	// Create new merged cluster
	var memories []store.Memory
	cm.db.Where("id IN ?", memoryIDs).Find(&memories)

	memoriesWithEmbeddings, _ := cm.ensureEmbeddings(ctx, memories)
	newCluster, err := cm.createClusterFromMemories(ctx, memoriesWithEmbeddings, DefaultFormationOptions())
	if err != nil {
		return nil, err
	}

	// Update metadata with merge history
	metadata := newCluster.GetMetadata()
	metadata.MergeHistory = []string{clusterID1, clusterID2}
	newCluster.SetMetadata(metadata)
	cm.db.Save(newCluster)

	// Log merge
	cm.logFormation(newCluster, "merge", len(memoryIDs), time.Now())

	return newCluster, nil
}

// DeleteCluster removes a cluster and its associations
func (cm *ClusterManager) DeleteCluster(clusterID string) error {
	var cluster NeuralCluster
	if err := cm.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return errors.Wrap(err, "NEURAL_013", "cluster not found")
	}

	// Delete cluster-memory associations
	cm.db.Where("cluster_id = ?", clusterID).Delete(&ClusterMemory{})

	// Delete cluster
	cm.db.Delete(&cluster)

	// Log deletion
	cm.logFormation(&cluster, "delete", cluster.ClusterSize, time.Now())

	return nil
}

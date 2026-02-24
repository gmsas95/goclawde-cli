package neural

import (
	"testing"

	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		vec1     []float32
		vec2     []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			vec1:     []float32{1.0, 0.0, 0.0},
			vec2:     []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "opposite vectors",
			vec1:     []float32{1.0, 0.0, 0.0},
			vec2:     []float32{-1.0, 0.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "orthogonal vectors",
			vec1:     []float32{1.0, 0.0, 0.0},
			vec2:     []float32{0.0, 1.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "zero vectors",
			vec1:     []float32{0.0, 0.0, 0.0},
			vec2:     []float32{0.0, 0.0, 0.0},
			expected: 0.0, // Handle zero vectors
		},
		{
			name:     "different lengths",
			vec1:     []float32{1.0, 0.0},
			vec2:     []float32{1.0, 0.0, 0.0},
			expected: 0.0, // Should return 0 for different lengths
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec1, tt.vec2)
			assert.InDelta(t, tt.expected, similarity, 0.01)
		})
	}
}

func TestClusterManager_calculateCentroid(t *testing.T) {
	logger := zap.NewNop()
	cm := &ClusterManager{logger: logger}

	// Test with multiple embeddings
	memories := []*MemoryWithEmbedding{
		{Embedding: []float32{1.0, 0.0, 0.0}},
		{Embedding: []float32{0.0, 1.0, 0.0}},
		{Embedding: []float32{0.0, 0.0, 1.0}},
	}

	centroid := cm.calculateCentroid(memories)

	// Centroid should be the average
	require.NotNil(t, centroid)
	assert.Len(t, centroid, 3)

	// Each component should be 1/3 (average of the three unit vectors)
	expected := float32(1.0 / 3.0)
	assert.InDelta(t, expected, centroid[0], 0.001)
	assert.InDelta(t, expected, centroid[1], 0.001)
	assert.InDelta(t, expected, centroid[2], 0.001)
}

func TestClusterManager_calculateConfidence(t *testing.T) {
	logger := zap.NewNop()
	cm := &ClusterManager{logger: logger}

	memories := []*MemoryWithEmbedding{
		{Embedding: []float32{1.0, 0.0, 0.0}},
		{Embedding: []float32{0.9, 0.1, 0.0}},
		{Embedding: []float32{0.8, 0.2, 0.0}},
	}

	centroid := cm.calculateCentroid(memories)
	confidence := cm.calculateConfidence(memories, centroid)

	// Confidence should be between 0 and 1
	assert.GreaterOrEqual(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 1.0)

	// With similar memories, confidence should be high
	assert.Greater(t, confidence, 0.5)
}

func TestClusterManager_float32SliceToBytes(t *testing.T) {
	logger := zap.NewNop()
	cm := &ClusterManager{logger: logger}

	original := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	bytes := cm.float32SliceToBytes(original)

	// Should be 4 bytes per float32
	assert.Equal(t, len(original)*4, len(bytes))

	// Convert back and verify
	converted := cm.bytesToFloat32Slice(bytes)
	assert.Equal(t, original, converted)
}

func TestClusterManager_bytesToFloat32Slice(t *testing.T) {
	logger := zap.NewNop()
	cm := &ClusterManager{logger: logger}

	// Test empty bytes
	empty := []byte{}
	result := cm.bytesToFloat32Slice(empty)
	assert.Empty(t, result)

	// Test with actual data
	original := []float32{1.5, 2.5, 3.5}
	bytes := cm.float32SliceToBytes(original)
	result = cm.bytesToFloat32Slice(bytes)
	assert.Equal(t, original, result)
}

func TestClusterManager_generateSimpleEssence(t *testing.T) {
	logger := zap.NewNop()
	cm := &ClusterManager{logger: logger}

	contents := []string{
		"User likes Python programming",
		"User prefers Python over JavaScript",
		"User uses Python for data science",
	}

	essence := cm.generateSimpleEssence(contents)

	// Should not be empty
	assert.NotEmpty(t, essence)
	// Should mention Python (common theme)
	assert.Contains(t, essence, "Python")
}

func TestClusterManager_generateSimpleTheme(t *testing.T) {
	logger := zap.NewNop()
	cm := &ClusterManager{logger: logger}

	contents := []string{
		"User likes coffee in the morning",
		"User drinks coffee at work",
		"User prefers coffee over tea",
	}

	theme := cm.generateSimpleTheme(contents)

	// Should not be empty
	assert.NotEmpty(t, theme)
	// Should be a short phrase
	assert.Less(t, len(theme), 50)
}

// Database-dependent tests (skipped by default)

func TestNewClusterManager(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestClusterManager_FormClusters(t *testing.T) {
	t.Skip("Requires database and LLM client")
}

func TestClusterManager_RefreshCluster(t *testing.T) {
	t.Skip("Requires database and LLM client")
}

func TestClusterManager_MergeClusters(t *testing.T) {
	t.Skip("Requires database")
}

func TestClusterManager_DeleteCluster(t *testing.T) {
	t.Skip("Requires database")
}

// MemoryWithEmbedding struct tests

func TestMemoryWithEmbedding_Struct(t *testing.T) {
	mem := &MemoryWithEmbedding{
		Memory: store.Memory{
			ID:      "mem1",
			Content: "test content",
			Type:    "fact",
		},
		Embedding: []float32{0.1, 0.2, 0.3},
	}

	assert.Equal(t, "mem1", mem.ID)
	assert.Equal(t, "test content", mem.Content)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, mem.Embedding)
}

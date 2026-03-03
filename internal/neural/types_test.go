package neural

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeuralCluster_BeforeCreate(t *testing.T) {
	tests := []struct {
		name    string
		cluster *NeuralCluster
		wantErr bool
	}{
		{
			name: "valid cluster with memory IDs",
			cluster: &NeuralCluster{
				Theme:     "test-theme",
				Essence:   "test essence",
				MemoryIDs: []string{"mem1", "mem2", "mem3"},
				Metadata:  Metadata{Category: "test"},
			},
			wantErr: false,
		},
		{
			name: "cluster without memory IDs",
			cluster: &NeuralCluster{
				Theme:   "test-theme",
				Essence: "test essence",
			},
			wantErr: false,
		},
		{
			name: "cluster with metadata",
			cluster: &NeuralCluster{
				Theme:   "test-theme",
				Essence: "test essence",
				Metadata: Metadata{
					Tags: []string{"tag1", "tag2"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cluster.BeforeCreate(nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if len(tt.cluster.MemoryIDs) > 0 {
					assert.NotEmpty(t, tt.cluster.MemoryIDsJSON)
					// Verify JSON is valid
					var ids []string
					err := json.Unmarshal([]byte(tt.cluster.MemoryIDsJSON), &ids)
					assert.NoError(t, err)
					assert.Equal(t, tt.cluster.MemoryIDs, ids)
				}
			}
		})
	}
}

func TestNeuralCluster_AfterFind(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *NeuralCluster
		expected *NeuralCluster
		wantErr  bool
	}{
		{
			name: "with memory IDs JSON",
			cluster: &NeuralCluster{
				MemoryIDsJSON: `["mem1", "mem2"]`,
			},
			expected: &NeuralCluster{
				MemoryIDs: []string{"mem1", "mem2"},
			},
			wantErr: false,
		},
		{
			name: "with metadata JSON",
			cluster: &NeuralCluster{
				MetadataJSON: `{"category":"test", "tags":["tag1"]}`,
			},
			expected: &NeuralCluster{
				Metadata: Metadata{
					Category: "test",
					Tags:     []string{"tag1"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty JSON fields",
			cluster: &NeuralCluster{
				MemoryIDsJSON: "",
				MetadataJSON:  "",
			},
			expected: &NeuralCluster{},
			wantErr:  false,
		},
		{
			name: "invalid JSON",
			cluster: &NeuralCluster{
				MemoryIDsJSON: "invalid",
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cluster.AfterFind(nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.expected != nil {
				assert.Equal(t, tt.expected.MemoryIDs, tt.cluster.MemoryIDs)
				assert.Equal(t, tt.expected.Metadata.Category, tt.cluster.Metadata.Category)
				assert.Equal(t, tt.expected.Metadata.Tags, tt.cluster.Metadata.Tags)
			}
		})
	}
}

func TestNeuralCluster_GetMemoryIDs(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *NeuralCluster
		expected []string
	}{
		{
			name: "from memory field",
			cluster: &NeuralCluster{
				MemoryIDs: []string{"mem1", "mem2"},
			},
			expected: []string{"mem1", "mem2"},
		},
		{
			name: "from JSON field",
			cluster: &NeuralCluster{
				MemoryIDsJSON: `["mem3", "mem4"]`,
			},
			expected: []string{"mem3", "mem4"},
		},
		{
			name:     "empty",
			cluster:  &NeuralCluster{},
			expected: []string{},
		},
		{
			name: "memory field takes precedence",
			cluster: &NeuralCluster{
				MemoryIDs:     []string{"a", "b"},
				MemoryIDsJSON: `["c", "d"]`,
			},
			expected: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cluster.GetMemoryIDs()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNeuralCluster_SetMemoryIDs(t *testing.T) {
	cluster := &NeuralCluster{}
	ids := []string{"mem1", "mem2", "mem3"}

	err := cluster.SetMemoryIDs(ids)
	require.NoError(t, err)

	assert.Equal(t, ids, cluster.MemoryIDs)
	assert.Equal(t, 3, cluster.ClusterSize)
	assert.NotEmpty(t, cluster.MemoryIDsJSON)

	// Verify JSON is valid
	var parsedIDs []string
	err = json.Unmarshal([]byte(cluster.MemoryIDsJSON), &parsedIDs)
	require.NoError(t, err)
	assert.Equal(t, ids, parsedIDs)
}

func TestNeuralCluster_GetMetadata(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *NeuralCluster
		expected Metadata
	}{
		{
			name: "from memory field",
			cluster: &NeuralCluster{
				Metadata: Metadata{
					Category: "test",
					Tags:     []string{"tag1"},
				},
			},
			expected: Metadata{
				Category: "test",
				Tags:     []string{"tag1"},
			},
		},
		{
			name: "from JSON field",
			cluster: &NeuralCluster{
				MetadataJSON: `{"category":"json", "tags":["tag2"], "formation_method":"auto"}`,
			},
			expected: Metadata{
				Category:        "json",
				Tags:            []string{"tag2"},
				FormationMethod: "auto",
			},
		},
		{
			name:     "empty",
			cluster:  &NeuralCluster{},
			expected: Metadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cluster.GetMetadata()
			assert.Equal(t, tt.expected.Category, result.Category)
			assert.Equal(t, tt.expected.Tags, result.Tags)
			assert.Equal(t, tt.expected.FormationMethod, result.FormationMethod)
		})
	}
}

func TestNeuralCluster_SetMetadata(t *testing.T) {
	cluster := &NeuralCluster{}
	metadata := Metadata{
		Category:         "test",
		Tags:             []string{"tag1", "tag2"},
		FormationMethod:  "manual",
		FormationVersion: "1.0",
	}

	err := cluster.SetMetadata(metadata)
	require.NoError(t, err)

	assert.Equal(t, metadata, cluster.Metadata)
	assert.NotEmpty(t, cluster.MetadataJSON)

	// Verify JSON is valid
	var parsedMetadata Metadata
	err = json.Unmarshal([]byte(cluster.MetadataJSON), &parsedMetadata)
	require.NoError(t, err)
	assert.Equal(t, metadata.Category, parsedMetadata.Category)
	assert.Equal(t, metadata.Tags, parsedMetadata.Tags)
}

func TestNeuralCluster_IncrementAccess(t *testing.T) {
	cluster := &NeuralCluster{
		ID:          "test-cluster",
		AccessCount: 5,
	}

	cluster.IncrementAccess()

	assert.Equal(t, 6, cluster.AccessCount)
	assert.NotNil(t, cluster.LastAccessed)
	assert.True(t, cluster.LastAccessed.After(cluster.UpdatedAt) || cluster.LastAccessed.Equal(cluster.UpdatedAt))
}

func TestNeuralCluster_IsStale(t *testing.T) {
	now := time.Now()
	threshold := 90 * 24 * time.Hour // 90 days

	tests := []struct {
		name     string
		cluster  *NeuralCluster
		expected bool
	}{
		{
			name: "fresh with last accessed",
			cluster: &NeuralCluster{
				LastAccessed: func() *time.Time { t := now.Add(-time.Hour); return &t }(),
				CreatedAt:    now.Add(-24 * time.Hour),
			},
			expected: false,
		},
		{
			name: "stale by last accessed",
			cluster: &NeuralCluster{
				LastAccessed: func() *time.Time { t := now.Add(-100 * 24 * time.Hour); return &t }(),
				CreatedAt:    now.Add(-200 * 24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "stale by creation (no last accessed)",
			cluster: &NeuralCluster{
				LastAccessed: nil,
				CreatedAt:    now.Add(-100 * 24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "fresh by creation (no last accessed)",
			cluster: &NeuralCluster{
				LastAccessed: nil,
				CreatedAt:    now.Add(-time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cluster.IsStale(threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNeuralCluster_CanMerge(t *testing.T) {
	tests := []struct {
		name       string
		cluster1   *NeuralCluster
		cluster2   *NeuralCluster
		similarity float64
		expected   bool
	}{
		{
			name:       "similarity too low",
			cluster1:   &NeuralCluster{ClusterSize: 10},
			cluster2:   &NeuralCluster{ClusterSize: 10},
			similarity: 0.8,
			expected:   false,
		},
		{
			name:       "similarity high enough",
			cluster1:   &NeuralCluster{ClusterSize: 10},
			cluster2:   &NeuralCluster{ClusterSize: 10},
			similarity: 0.9,
			expected:   true,
		},
		{
			name:       "size ratio too different",
			cluster1:   &NeuralCluster{ClusterSize: 10},
			cluster2:   &NeuralCluster{ClusterSize: 30},
			similarity: 0.9,
			expected:   false,
		},
		{
			name:       "size ratio acceptable",
			cluster1:   &NeuralCluster{ClusterSize: 10},
			cluster2:   &NeuralCluster{ClusterSize: 15},
			similarity: 0.9,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cluster1.CanMerge(tt.cluster2, tt.similarity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultFormationOptions(t *testing.T) {
	opts := DefaultFormationOptions()

	assert.InDelta(t, 0.85, opts.SimilarityThreshold, 0.001)
	assert.Equal(t, 3, opts.MinClusterSize)
	assert.Equal(t, 100, opts.MaxClusterSize)
	assert.Equal(t, 1000, opts.MaxClusters)
	assert.InDelta(t, 0.6, opts.ReclusterThreshold, 0.001)
	assert.Equal(t, 100, opts.BatchSize)
	assert.Equal(t, 30*time.Minute, opts.FormationTimeout)
}

func TestDefaultRetrievalOptions(t *testing.T) {
	opts := DefaultRetrievalOptions()

	assert.Equal(t, 5, opts.MaxClusters)
	assert.Equal(t, 10, opts.MaxMemoriesPerCluster)
	assert.InDelta(t, 0.7, opts.MinClusterConfidence, 0.001)
	assert.InDelta(t, 0.5, opts.MinMemorySimilarity, 0.001)
	assert.Equal(t, 4000, opts.TokenBudget)
	assert.True(t, opts.IncludeMetadata)
	assert.True(t, opts.BoostRecentAccess)
	assert.Equal(t, 7*24*time.Hour, opts.RecentAccessWindow)
}

func TestContextResult_TotalCounts(t *testing.T) {
	result := &ContextResult{
		Clusters:      []*NeuralCluster{{ID: "c1"}, {ID: "c2"}},
		Memories:      []MemoryRef{{ID: "m1"}, {ID: "m2"}, {ID: "m3"}},
		TotalClusters: 10,
		TotalMemories: 100,
	}

	assert.Equal(t, 2, len(result.Clusters))
	assert.Equal(t, 3, len(result.Memories))
	assert.Equal(t, 10, result.TotalClusters)
	assert.Equal(t, 100, result.TotalMemories)
}

func TestMemoryRef_Struct(t *testing.T) {
	mem := MemoryRef{
		ID:         "mem1",
		Content:    "test content",
		Type:       "fact",
		Importance: 5,
		ClusterID:  "cluster1",
		Similarity: 0.85,
		CreatedAt:  time.Now(),
	}

	assert.Equal(t, "mem1", mem.ID)
	assert.Equal(t, "test content", mem.Content)
	assert.Equal(t, "fact", mem.Type)
	assert.Equal(t, 5, mem.Importance)
	assert.Equal(t, "cluster1", mem.ClusterID)
	assert.InDelta(t, 0.85, mem.Similarity, 0.001)
}

func TestClusterMemory_Struct(t *testing.T) {
	cm := ClusterMemory{
		ClusterID:        "cluster1",
		MemoryID:         "mem1",
		SimilarityScore:  0.9,
		IsRepresentative: true,
		AddedAt:          time.Now(),
	}

	assert.Equal(t, "cluster1", cm.ClusterID)
	assert.Equal(t, "mem1", cm.MemoryID)
	assert.InDelta(t, 0.9, cm.SimilarityScore, 0.001)
	assert.True(t, cm.IsRepresentative)
}

func TestQueryPattern_Struct(t *testing.T) {
	now := time.Now()
	pattern := QueryPattern{
		ID:                "pattern1",
		QueryText:         "test query",
		MatchedClusterIDs: []string{"c1", "c2"},
		MatchedMemoryIDs:  []string{"m1", "m2"},
		TokensUsed:        100,
		LatencyMs:         50,
		CreatedAt:         now,
	}

	assert.Equal(t, "pattern1", pattern.ID)
	assert.Equal(t, "test query", pattern.QueryText)
	assert.Equal(t, []string{"c1", "c2"}, pattern.MatchedClusterIDs)
	assert.Equal(t, []string{"m1", "m2"}, pattern.MatchedMemoryIDs)
	assert.Equal(t, 100, pattern.TokensUsed)
	assert.Equal(t, 50, pattern.LatencyMs)
}

func TestClusterFilter_Struct(t *testing.T) {
	now := time.Now()
	filter := ClusterFilter{
		MinConfidence: 0.7,
		MaxConfidence: 0.95,
		MinSize:       5,
		MaxSize:       50,
		ThemeContains: "test",
		Tags:          []string{"important"},
		Category:      "knowledge",
		CreatedAfter:  &now,
		Limit:         10,
		Offset:        0,
	}

	assert.InDelta(t, 0.7, filter.MinConfidence, 0.001)
	assert.Equal(t, 5, filter.MinSize)
	assert.Equal(t, "test", filter.ThemeContains)
	assert.Equal(t, []string{"important"}, filter.Tags)
}

func TestClusterFormationLog_Struct(t *testing.T) {
	log := ClusterFormationLog{
		ID:            "log1",
		ClusterID:     "cluster1",
		Operation:     "create",
		Details:       `{"source": "auto"}`,
		MemoryCount:   10,
		PreviousState: "",
		NewState:      `{"size": 10}`,
		DurationMs:    100,
	}

	assert.Equal(t, "log1", log.ID)
	assert.Equal(t, "cluster1", log.ClusterID)
	assert.Equal(t, "create", log.Operation)
	assert.Equal(t, 10, log.MemoryCount)
	assert.Equal(t, 100, log.DurationMs)
}

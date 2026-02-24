package reflection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test helper functions first (no DB needed)

func TestNormalizeTopic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"UPPER CASE", "upper case"},
		{"MiXeD CaSe", "mixed case"},
		{"  spaces  ", "spaces"},
		{"special!@#chars", "special!@#chars"}, // Only lowercases, doesn't remove special chars
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTopic(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"Multiple   Spaces", "multiple spaces"},
		{"Special!@#Chars", "specialchars"},
		{"123Numbers", "123numbers"}, // Keeps numbers
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello!", "hello"},
		{"world.", "world"},
		{"test?", "test"},
		{"clean", "clean"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanWord(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractContentKey(t *testing.T) {
	longContent := "This is a very long content that should be truncated to 50 characters for the key"

	tests := []struct {
		input    string
		expected int // expected max length
	}{
		{"short", 5},
		{longContent, 50},
	}

	for _, tt := range tests {
		t.Run("length check", func(t *testing.T) {
			result := extractContentKey(tt.input)
			assert.LessOrEqual(t, len(result), tt.expected)
		})
	}
}

func TestNormalizeTheme(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Note: removeWord is currently a stub, so stop words aren't actually removed
		{"The Docker Configuration", "the docker configuration"},
		{"API and Webhook Setup", "api and webhook setup"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTheme(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID("test")
	id2 := generateID("test")

	// Should start with prefix
	assert.Contains(t, id1, "test")

	// Should be unique
	assert.NotEqual(t, id1, id2)

	// Should not be empty
	assert.NotEmpty(t, id1)
}

func TestRandomString(t *testing.T) {
	s1 := randomString(10)
	s2 := randomString(10)

	// Should have correct length
	assert.Len(t, s1, 10)
	assert.Len(t, s2, 10)

	// Should be different (likely)
	// Note: There's a tiny chance they could be the same
	if s1 == s2 {
		t.Log("Warning: random strings were equal (very unlikely)")
	}
}

func TestMin(t *testing.T) {
	assert.Equal(t, 1, min(1, 2))
	assert.Equal(t, 1, min(2, 1))
	assert.Equal(t, 5, min(5, 5))
	assert.Equal(t, -1, min(-1, 1))
}

func TestMax(t *testing.T) {
	assert.Equal(t, 2, max(1, 2))
	assert.Equal(t, 2, max(2, 1))
	assert.Equal(t, 5, max(5, 5))
	assert.Equal(t, 1, max(-1, 1))
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s        string
		maxLen   int
		expected string
	}{
		{"hello world", 5, "hello..."},
		{"hi", 10, "hi"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := truncateString(tt.s, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	assert.True(t, contains("hello world", "world"))
	assert.True(t, contains("hello world", "hello"))
	assert.False(t, contains("hello world", "foo"))
	assert.True(t, contains("", ""))
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{100, "100"},
		{1000, "1,000"},
		{1000000, "1,000,000"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatNumber(tt.n)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseJSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			json:    `{"name":"test","value":42}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{"name":"test",}`,
			wantErr: true,
		},
		{
			name:    "empty string",
			json:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := parseJSON(tt.json, &result)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "test", result.Name)
				assert.Equal(t, 42, result.Value)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	obj := TestStruct{Name: "test", Value: 42}
	jsonStr := toJSON(obj)

	assert.Contains(t, jsonStr, "test")
	assert.Contains(t, jsonStr, "42")
}

// Tests for types and structs

func TestRedundancyGroup_CalculateStorageSavings(t *testing.T) {
	tests := []struct {
		name     string
		memories []string
		expected int
	}{
		{
			name:     "no memories",
			memories: []string{},
			expected: 0,
		},
		{
			name:     "one memory",
			memories: []string{"mem1"},
			expected: 0,
		},
		{
			name:     "three memories",
			memories: []string{"mem1", "mem2", "mem3"},
			expected: 1000, // (3-1) * 500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := RedundancyGroup{
				MemoryIDs: tt.memories,
			}
			result := group.CalculateStorageSavings()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedundancyGroup_GetPriority(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected string
	}{
		{"low", 5, "low"},
		{"medium", 15, "medium"},
		{"high", 25, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := RedundancyGroup{
				MemoryIDs: make([]string, tt.count),
			}
			result := group.GetPriority()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGap_GetSeverity(t *testing.T) {
	tests := []struct {
		name       string
		mentionCnt int
		memoryCnt  int
		expected   string
	}{
		{"critical", 50, 2, "critical"},
		{"high", 20, 1, "high"},
		{"medium", 15, 5, "medium"},
		{"low", 5, 3, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gap := Gap{
				MentionCount: tt.mentionCnt,
				MemoryCount:  tt.memoryCnt,
			}
			result := gap.GetSeverity()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGap_GetImpactEstimate(t *testing.T) {
	tests := []struct {
		name       string
		mentionCnt int
		memoryCnt  int
		expected   string
	}{
		{"major", 100, 5, "Major"},
		{"significant", 50, 10, "Significant"},
		{"moderate", 20, 15, "Moderate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gap := Gap{
				MentionCount: tt.mentionCnt,
				MemoryCount:  tt.memoryCnt,
			}
			result := gap.GetImpactEstimate()
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestHealthReport_GetScoreCategory(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{95, "excellent"},
		{85, "good"},
		{70, "fair"},
		{50, "poor"},
		{30, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			report := HealthReport{OverallScore: tt.score}
			result := report.GetScoreCategory()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthReport_GetScoreEmoji(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{95, "🌟"},
		{85, "✅"},
		{70, "⚠️"},
		{50, "❌"},
		{30, "🚨"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			report := HealthReport{OverallScore: tt.score}
			result := report.GetScoreEmoji()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthReport_calculateScore(t *testing.T) {
	report := HealthReport{
		TotalMemories:  100,
		NeuralClusters: 10,
		Contradictions: 1,
		Redundancies:   2,
		Gaps:           1,
		Metrics: Metrics{
			MemoriesPerCluster:   10,
			AvgClusterConfidence: 0.8,
		},
	}

	report.calculateScore()

	// Score should be between 0 and 100
	assert.GreaterOrEqual(t, report.OverallScore, 0)
	assert.LessOrEqual(t, report.OverallScore, 100)

	// With minimal issues and good metrics, should be in good range
	assert.Greater(t, report.OverallScore, 70)
}

// Tests that require DB/LLM (skipped)

func TestNewHealthReporter(t *testing.T) {
	t.Skip("Requires database")
}

func TestHealthReporter_GenerateReport(t *testing.T) {
	t.Skip("Requires database and LLM")
}

func TestNewContradictionDetector(t *testing.T) {
	t.Skip("Requires LLM client")
}

func TestContradictionDetector_Detect(t *testing.T) {
	t.Skip("Requires LLM client and database")
}

func TestNewGapAnalyzer(t *testing.T) {
	t.Skip("Requires LLM client")
}

func TestGapAnalyzer_IdentifyGaps(t *testing.T) {
	t.Skip("Requires LLM client and database")
}

func TestNewRedundancyFinder(t *testing.T) {
	t.Skip("Requires database")
}

func TestRedundancyFinder_FindRedundancies(t *testing.T) {
	t.Skip("Requires database")
}

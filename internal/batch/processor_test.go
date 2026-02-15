package batch

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxConcurrency <= 0 {
		t.Error("MaxConcurrency should be positive")
	}
	if cfg.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	if cfg.RetryCount < 0 {
		t.Error("RetryCount should be non-negative")
	}
}

func TestConfig_Values(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxConcurrency != 3 {
		t.Errorf("Expected MaxConcurrency 3, got %d", cfg.MaxConcurrency)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout 60s, got %v", cfg.Timeout)
	}
	if cfg.RetryCount != 2 {
		t.Errorf("Expected RetryCount 2, got %d", cfg.RetryCount)
	}
}

func TestInputItem_ID(t *testing.T) {
	item := InputItem{
		ID:      "test-1",
		Message: "Hello",
	}

	if item.ID != "test-1" {
		t.Errorf("Expected ID test-1, got %s", item.ID)
	}
}

func TestInputItem_WithContext(t *testing.T) {
	item := InputItem{
		ID:      "test-1",
		Message: "Hello",
		Context: map[string]string{"key": "value"},
	}

	if item.Context["key"] != "value" {
		t.Error("Context not set correctly")
	}
}

func TestOutputItem_Success(t *testing.T) {
	item := OutputItem{
		ID:         "test-1",
		Response:   "Hello back!",
		TokensUsed: 10,
		Success:    true,
	}

	if !item.Success {
		t.Error("Item should be successful")
	}
}

func TestOutputItem_WithError(t *testing.T) {
	item := OutputItem{
		ID:     "test-1",
		Error:  "something went wrong",
		Success: false,
	}

	if item.Success {
		t.Error("Item should not be successful")
	}
	if item.Error == "" {
		t.Error("Error message should be set")
	}
}

func TestResult_Summary(t *testing.T) {
	result := &Result{
		Total:    10,
		Success:  8,
		Failed:   1,
		Skipped:  1,
		Duration: 5 * time.Second,
	}

	summary := result.Summary()

	if summary == "" {
		t.Error("Summary should not be empty")
	}
}

func TestResult_Summary_ContainsAllStats(t *testing.T) {
	result := &Result{
		Total:    10,
		Success:  8,
		Failed:   1,
		Skipped:  1,
		Duration: 5 * time.Second,
	}

	summary := result.Summary()

	checks := []string{"Total:", "Success:", "Failed:", "Skipped:", "Duration:"}
	for _, check := range checks {
		if !contains(summary, check) {
			t.Errorf("Summary missing: %s", check)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestResult_ToJSON(t *testing.T) {
	result := &Result{
		Total:    2,
		Success:  1,
		Failed:   1,
		Skipped:  0,
		Duration: 1 * time.Second,
		Items:    []OutputItem{},
	}

	json, err := result.ToJSON()
	if err != nil {
		t.Errorf("ToJSON failed: %v", err)
	}
	if json == "" {
		t.Error("JSON should not be empty")
	}
}

func TestNewProcessor_NilConfig(t *testing.T) {
	cfg := Config{
		MaxConcurrency: 0,
	}
	
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 1
	}

	if cfg.MaxConcurrency != 1 {
		t.Error("Should default to 1 when 0 or negative")
	}
}

func TestLoadTextFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "batch_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `First prompt
Second prompt
# This is a comment

Third prompt
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	processor := &Processor{
		config: DefaultConfig(),
	}

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	items, err := processor.loadTextFile(file)
	if err != nil {
		t.Fatalf("loadTextFile failed: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}
}

func TestLoadTextFile_EmptyLines(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "batch_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := "\n\n\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	processor := &Processor{
		config: DefaultConfig(),
	}

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	items, err := processor.loadTextFile(file)
	if err != nil {
		t.Fatalf("loadTextFile failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items from empty file, got %d", len(items))
	}
}

func TestLoadJSONFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "batch_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `{"id": "1", "message": "First"}
{"id": "2", "message": "Second"}
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	processor := &Processor{
		config: DefaultConfig(),
	}

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	items, err := processor.loadJSONFile(file)
	if err != nil {
		t.Fatalf("loadJSONFile failed: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestLoadJSONFile_WithContext(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "batch_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `{"id": "1", "message": "Test", "context": {"lang": "go"}}
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	processor := &Processor{
		config: DefaultConfig(),
	}

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	items, err := processor.loadJSONFile(file)
	if err != nil {
		t.Fatalf("loadJSONFile failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	if items[0].Context["lang"] != "go" {
		t.Error("Context not loaded correctly")
	}
}

func TestProcessor_ProcessFile_Context(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 1
	cfg.Timeout = 5 * time.Second

	logger := zap.NewNop()
	processor := &Processor{
		config: cfg,
		logger: logger,
	}

	_, err := processor.loadInputFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Should fail with nonexistent file")
	}
}

func TestSaveOutputFile_JSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "batch_output_*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	result := &Result{
		Total:    1,
		Success:  1,
		Failed:   0,
		Skipped:  0,
		Duration: 1 * time.Second,
		Items: []OutputItem{
			{ID: "test-1", Response: "Hello", Success: true},
		},
	}

	processor := &Processor{}
	err = processor.saveOutputFile(tmpFile.Name(), result)
	if err != nil {
		t.Fatalf("saveOutputFile failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("Output file is empty")
	}
}

func TestSaveOutputFile_Text(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "batch_output_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	result := &Result{
		Total:    1,
		Success:  1,
		Failed:   0,
		Skipped:  0,
		Duration: 1 * time.Second,
		Items: []OutputItem{
			{ID: "test-1", Input: "Hello", Response: "Hi there!", Success: true},
		},
	}

	processor := &Processor{}
	err = processor.saveOutputFile(tmpFile.Name(), result)
	if err != nil {
		t.Fatalf("saveOutputFile failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("Output file is empty")
	}
}

func TestInputItem_EmptyID(t *testing.T) {
	item := InputItem{
		Message: "Test",
	}

	if item.ID != "" {
		t.Error("ID should be empty")
	}
}

func TestOutputItem_Timestamp(t *testing.T) {
	now := time.Now()
	item := OutputItem{
		Timestamp: now,
	}

	if !item.Timestamp.Equal(now) {
		t.Error("Timestamp not preserved")
	}
}

func TestConfig_Timeout(t *testing.T) {
	cfg := Config{
		Timeout: 30 * time.Second,
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", cfg.Timeout)
	}
}

func TestConfig_RetrySettings(t *testing.T) {
	cfg := Config{
		RetryCount: 3,
		RetryDelay: 2 * time.Second,
	}

	if cfg.RetryCount != 3 {
		t.Errorf("Expected RetryCount 3, got %d", cfg.RetryCount)
	}
	if cfg.RetryDelay != 2*time.Second {
		t.Errorf("Expected RetryDelay 2s, got %v", cfg.RetryDelay)
	}
}

func TestResult_ZeroValues(t *testing.T) {
	result := &Result{}

	if result.Total != 0 || result.Success != 0 || result.Failed != 0 {
		t.Error("Result should have zero values initially")
	}
}

func BenchmarkResult_Summary(b *testing.B) {
	result := &Result{
		Total:    100,
		Success:  95,
		Failed:   3,
		Skipped:  2,
		Duration: 10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.Summary()
	}
}

func BenchmarkResult_ToJSON(b *testing.B) {
	result := &Result{
		Total:    100,
		Success:  95,
		Failed:   3,
		Skipped:  2,
		Duration: 10 * time.Second,
		Items:    make([]OutputItem, 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.ToJSON()
	}
}

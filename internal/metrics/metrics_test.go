package metrics

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Error("New() returned nil")
	}
}

func TestDefault(t *testing.T) {
	m1 := Default()
	m2 := Default()

	if m1 != m2 {
		t.Error("Default() should return same instance")
	}
}

func TestRecordRequest_Success(t *testing.T) {
	m := New()
	m.RecordRequest(true)

	if m.requestsTotal.Load() != 1 {
		t.Error("Total requests not incremented")
	}
	if m.requestsSuccess.Load() != 1 {
		t.Error("Success requests not incremented")
	}
}

func TestRecordRequest_Failure(t *testing.T) {
	m := New()
	m.RecordRequest(false)

	if m.requestsTotal.Load() != 1 {
		t.Error("Total requests not incremented")
	}
	if m.requestsFailed.Load() != 1 {
		t.Error("Failed requests not incremented")
	}
}

func TestRecordRequestBlocked(t *testing.T) {
	m := New()
	m.RecordRequestBlocked()

	if m.requestsBlocked.Load() != 1 {
		t.Error("Blocked requests not incremented")
	}
}

func TestRecordTokens(t *testing.T) {
	m := New()
	m.RecordTokens(100, 50)

	if m.tokensUsed.Load() != 150 {
		t.Errorf("Expected 150 tokens, got %d", m.tokensUsed.Load())
	}
	if m.tokensPrompt.Load() != 100 {
		t.Errorf("Expected 100 prompt tokens, got %d", m.tokensPrompt.Load())
	}
	if m.tokensCompletion.Load() != 50 {
		t.Errorf("Expected 50 completion tokens, got %d", m.tokensCompletion.Load())
	}
}

func TestRecordMessage(t *testing.T) {
	m := New()
	m.RecordMessage(false)

	if m.messagesProcessed.Load() != 1 {
		t.Error("Messages processed not incremented")
	}

	m.RecordMessage(true)
	if m.messagesBlocked.Load() != 1 {
		t.Error("Messages blocked not incremented")
	}
}

func TestRecordToolCall_Success(t *testing.T) {
	m := New()
	m.RecordToolCall(true)

	if m.toolCallsTotal.Load() != 1 {
		t.Error("Tool calls total not incremented")
	}
	if m.toolCallsSuccess.Load() != 1 {
		t.Error("Tool calls success not incremented")
	}
}

func TestRecordToolCall_Failure(t *testing.T) {
	m := New()
	m.RecordToolCall(false)

	if m.toolCallsFailed.Load() != 1 {
		t.Error("Tool calls failed not incremented")
	}
}

func TestRecordSkillCall(t *testing.T) {
	m := New()
	m.RecordSkillCall("github")
	m.RecordSkillCall("github")
	m.RecordSkillCall("weather")

	m.skillLock.Lock()
	defer m.skillLock.Unlock()

	if m.skillCalls["github"].Load() != 2 {
		t.Error("GitHub skill calls not counted correctly")
	}
	if m.skillCalls["weather"].Load() != 1 {
		t.Error("Weather skill calls not counted correctly")
	}
}

func TestRecordProviderRequest(t *testing.T) {
	m := New()
	m.RecordProviderRequest("openai")
	m.RecordProviderRequest("openai")
	m.RecordProviderRequest("anthropic")

	m.providerLock.Lock()
	defer m.providerLock.Unlock()

	if m.providerRequests["openai"].Load() != 2 {
		t.Error("OpenAI requests not counted correctly")
	}
	if m.providerRequests["anthropic"].Load() != 1 {
		t.Error("Anthropic requests not counted correctly")
	}
}

func TestRecordResponseTime(t *testing.T) {
	m := New()
	m.RecordResponseTime(100 * time.Millisecond)
	m.RecordResponseTime(200 * time.Millisecond)

	m.responseTimesLock.Lock()
	defer m.responseTimesLock.Unlock()

	if len(m.responseTimes) != 2 {
		t.Errorf("Expected 2 response times, got %d", len(m.responseTimes))
	}
}

func TestSetActiveConnections(t *testing.T) {
	m := New()
	m.SetActiveConnections(5)

	if m.activeConnections.Load() != 5 {
		t.Error("Active connections not set correctly")
	}
}

func TestIncrementActiveConnections(t *testing.T) {
	m := New()
	m.SetActiveConnections(5)
	m.IncrementActiveConnections()

	if m.activeConnections.Load() != 6 {
		t.Error("Active connections not incremented")
	}
}

func TestDecrementActiveConnections(t *testing.T) {
	m := New()
	m.SetActiveConnections(5)
	m.DecrementActiveConnections()

	if m.activeConnections.Load() != 4 {
		t.Error("Active connections not decremented")
	}
}

func TestSecurityMetrics(t *testing.T) {
	m := New()
	m.RecordSecurityBlock()
	m.RecordInjectionBlocked()
	m.RecordSecretsDetected()
	m.RecordPathTraversal()
	m.RecordDangerousCommand()

	if m.securityBlocks.Load() != 1 {
		t.Error("Security blocks not recorded")
	}
	if m.injectionBlocked.Load() != 1 {
		t.Error("Injection blocked not recorded")
	}
	if m.secretsDetected.Load() != 1 {
		t.Error("Secrets detected not recorded")
	}
	if m.pathTraversals.Load() != 1 {
		t.Error("Path traversals not recorded")
	}
	if m.dangerousCommands.Load() != 1 {
		t.Error("Dangerous commands not recorded")
	}
}

func TestSnapshot(t *testing.T) {
	m := New()
	m.RecordRequest(true)
	m.RecordRequest(false)
	m.RecordTokens(100, 50)
	m.RecordToolCall(true)

	s := m.Snapshot()

	if s.RequestsTotal != 2 {
		t.Errorf("Expected 2 total requests, got %d", s.RequestsTotal)
	}
	if s.RequestsSuccess != 1 {
		t.Errorf("Expected 1 success, got %d", s.RequestsSuccess)
	}
	if s.RequestsFailed != 1 {
		t.Errorf("Expected 1 failed, got %d", s.RequestsFailed)
	}
	if s.TokensUsed != 150 {
		t.Errorf("Expected 150 tokens, got %d", s.TokensUsed)
	}
	if s.Uptime <= 0 {
		t.Error("Uptime should be positive")
	}
}

func TestSnapshot_SuccessRate(t *testing.T) {
	m := New()
	m.RecordRequest(true)
	m.RecordRequest(true)
	m.RecordRequest(false)

	s := m.Snapshot()

	if s.SuccessRate != 66.66666666666666 {
		t.Errorf("Expected ~66.67%% success rate, got %f", s.SuccessRate)
	}
}

func TestSnapshot_ZeroRequests(t *testing.T) {
	m := New()
	s := m.Snapshot()

	if s.SuccessRate != 0 {
		t.Errorf("Expected 0%% success rate with no requests, got %f", s.SuccessRate)
	}
}

func TestPrometheus(t *testing.T) {
	m := New()
	m.RecordRequest(true)
	m.RecordTokens(100, 50)

	output := m.Prometheus()

	if output == "" {
		t.Error("Prometheus output should not be empty")
	}

	expectedStrings := []string{
		"goclawde_requests_total",
		"goclawde_tokens_used",
		"goclawde_uptime_seconds",
	}

	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Prometheus output missing: %s", expected)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPrometheus_WithProviders(t *testing.T) {
	m := New()
	m.RecordProviderRequest("openai")
	m.RecordProviderRequest("anthropic")

	output := m.Prometheus()

	if !contains(output, `provider="openai"`) {
		t.Error("OpenAI provider not in output")
	}
	if !contains(output, `provider="anthropic"`) {
		t.Error("Anthropic provider not in output")
	}
}

func TestPrometheus_WithSkills(t *testing.T) {
	m := New()
	m.RecordSkillCall("github")
	m.RecordSkillCall("weather")

	output := m.Prometheus()

	if !contains(output, `skill="github"`) {
		t.Error("GitHub skill not in output")
	}
	if !contains(output, `skill="weather"`) {
		t.Error("Weather skill not in output")
	}
}

func TestHelperFunctions(t *testing.T) {
	m := Default()

	initialRequests := m.requestsTotal.Load()
	RecordRequest(true)
	if m.requestsTotal.Load() != initialRequests+1 {
		t.Error("RecordRequest helper failed")
	}

	RecordTokens(100, 50)
	if m.tokensUsed.Load() < 150 {
		t.Error("RecordTokens helper failed")
	}

	RecordToolCall(true)
	if m.toolCallsTotal.Load() < 1 {
		t.Error("RecordToolCall helper failed")
	}

	RecordSkillCall("test")
	RecordProviderRequest("test")
	RecordSecurityBlock()
	RecordInjectionBlocked()
	RecordSecretsDetected()
	RecordDangerousCommand()

	s := GetSnapshot()
	if s == nil {
		t.Error("GetSnapshot helper returned nil")
	}

	p := GetPrometheus()
	if p == "" {
		t.Error("GetPrometheus helper returned empty string")
	}
}

func TestResponseTimePercentile(t *testing.T) {
	m := New()

	for i := 0; i < 100; i++ {
		m.RecordResponseTime(time.Duration(i+1) * time.Millisecond)
	}

	s := m.Snapshot()

	if s.AvgResponseTime <= 0 {
		t.Error("Average response time should be positive")
	}
	if s.P99ResponseTime <= 0 {
		t.Error("P99 response time should be positive")
	}
}

func TestResponseTimeRolling(t *testing.T) {
	m := New()

	for i := 0; i < 1100; i++ {
		m.RecordResponseTime(time.Duration(i+1) * time.Millisecond)
	}

	m.responseTimesLock.Lock()
	count := len(m.responseTimes)
	m.responseTimesLock.Unlock()

	if count > 1000 {
		t.Errorf("Response times should be capped at 1000, got %d", count)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := New()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.RecordRequest(true)
				m.RecordTokens(10, 5)
				m.RecordToolCall(j%2 == 0)
				m.RecordSkillCall("test")
				m.RecordProviderRequest("test")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	s := m.Snapshot()
	if s.RequestsTotal != 1000 {
		t.Errorf("Expected 1000 requests, got %d", s.RequestsTotal)
	}
}

func BenchmarkRecordRequest(b *testing.B) {
	m := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordRequest(true)
	}
}

func BenchmarkRecordTokens(b *testing.B) {
	m := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordTokens(100, 50)
	}
}

func BenchmarkSnapshot(b *testing.B) {
	m := New()
	for i := 0; i < 100; i++ {
		m.RecordRequest(true)
		m.RecordTokens(100, 50)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Snapshot()
	}
}

func BenchmarkPrometheus(b *testing.B) {
	m := New()
	for i := 0; i < 100; i++ {
		m.RecordRequest(true)
		m.RecordTokens(100, 50)
		m.RecordSkillCall("test")
		m.RecordProviderRequest("test")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Prometheus()
	}
}

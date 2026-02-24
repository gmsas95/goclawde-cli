// Package metrics provides Prometheus metrics for Myrai
package metrics

import (
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// JobMetrics tracks background job execution
	JobExecutions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myrai_job_executions_total",
		Help: "Total number of job executions",
	}, []string{"job_name", "status"})

	JobDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "myrai_job_duration_seconds",
		Help:    "Job execution duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"job_name"})

	// NeuralClusterMetrics tracks neural cluster statistics
	ClusterCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "myrai_neural_clusters_total",
		Help: "Total number of neural clusters",
	})

	ClusterRetrievals = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myrai_cluster_retrievals_total",
		Help: "Total number of cluster retrievals",
	}, []string{"status"})

	// LLMMetrics tracks LLM API usage
	LLMRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myrai_llm_requests_total",
		Help: "Total LLM requests",
	}, []string{"provider", "status"})

	LLMLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "myrai_llm_latency_seconds",
		Help:    "LLM request latency",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
	}, []string{"provider"})

	// CircuitBreakerMetrics tracks circuit breaker state
	CircuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "myrai_circuit_breaker_state",
		Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
	}, []string{"name"})

	// MemoryMetrics tracks memory operations
	MemoryOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myrai_memory_operations_total",
		Help: "Total memory operations",
	}, []string{"operation", "status"})

	// ConversationMetrics tracks conversation statistics
	MessagesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myrai_messages_processed_total",
		Help: "Total messages processed",
	})

	// ReflectionMetrics tracks reflection engine
	ReflectionIssues = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "myrai_reflection_issues",
		Help: "Number of reflection issues by type",
	}, []string{"type"})

	// SkillMetrics tracks skill usage
	SkillExecutions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myrai_skill_executions_total",
		Help: "Total skill executions",
	}, []string{"skill_name", "status"})
)

// Metrics holds all application metrics
type Metrics struct {
	requestsTotal     atomic.Int64
	requestsSuccess   atomic.Int64
	requestsFailed    atomic.Int64
	requestsBlocked   atomic.Int64
	tokensUsed        atomic.Int64
	tokensPrompt      atomic.Int64
	tokensCompletion  atomic.Int64
	messagesProcessed atomic.Int64
	messagesBlocked   atomic.Int64
	toolCallsTotal    atomic.Int64
	toolCallsSuccess  atomic.Int64
	toolCallsFailed   atomic.Int64
	skillCalls        map[string]*atomic.Int64
	providerRequests  map[string]*atomic.Int64
	responseTimes     []time.Duration
	activeConnections atomic.Int64
	securityBlocks    atomic.Int64
	injectionBlocked  atomic.Int64
	secretsDetected   atomic.Int64
	pathTraversals    atomic.Int64
	dangerousCommands atomic.Int64
	skillLock         sync.Mutex
	providerLock      sync.Mutex
	responseTimesLock sync.Mutex
	startTime         time.Time
}

// Snapshot represents a snapshot of metrics at a point in time
type Snapshot struct {
	RequestsTotal     int64
	RequestsSuccess   int64
	RequestsFailed    int64
	RequestsBlocked   int64
	TokensUsed        int64
	TokensPrompt      int64
	TokensCompletion  int64
	MessagesProcessed int64
	MessagesBlocked   int64
	ToolCallsTotal    int64
	ToolCallsSuccess  int64
	ToolCallsFailed   int64
	ActiveConnections int64
	SecurityBlocks    int64
	InjectionBlocked  int64
	SecretsDetected   int64
	PathTraversals    int64
	DangerousCommands int64
	AvgResponseTime   time.Duration
	P99ResponseTime   time.Duration
	Uptime            time.Duration
	SuccessRate       float64
	SkillCalls        map[string]int64
	ProviderRequests  map[string]int64
}

var (
	defaultInstance *Metrics
	defaultOnce     sync.Once
)

// New creates a new Metrics instance
func New() *Metrics {
	return &Metrics{
		skillCalls:       make(map[string]*atomic.Int64),
		providerRequests: make(map[string]*atomic.Int64),
		responseTimes:    make([]time.Duration, 0),
		startTime:        time.Now(),
	}
}

// Default returns the singleton default Metrics instance
func Default() *Metrics {
	defaultOnce.Do(func() {
		defaultInstance = New()
	})
	return defaultInstance
}

// RecordRequest records a request with its success status
func (m *Metrics) RecordRequest(success bool) {
	m.requestsTotal.Add(1)
	if success {
		m.requestsSuccess.Add(1)
	} else {
		m.requestsFailed.Add(1)
	}
}

// RecordRequestBlocked records a blocked request
func (m *Metrics) RecordRequestBlocked() {
	m.requestsBlocked.Add(1)
}

// RecordTokens records token usage
func (m *Metrics) RecordTokens(prompt, completion int64) {
	m.tokensPrompt.Add(prompt)
	m.tokensCompletion.Add(completion)
	m.tokensUsed.Add(prompt + completion)
}

// RecordMessage records a message (blocked or processed)
func (m *Metrics) RecordMessage(blocked bool) {
	if blocked {
		m.messagesBlocked.Add(1)
	} else {
		m.messagesProcessed.Add(1)
	}
}

// RecordToolCall records a tool call with its success status
func (m *Metrics) RecordToolCall(success bool) {
	m.toolCallsTotal.Add(1)
	if success {
		m.toolCallsSuccess.Add(1)
	} else {
		m.toolCallsFailed.Add(1)
	}
}

// RecordSkillCall records a skill call by name
func (m *Metrics) RecordSkillCall(name string) {
	m.skillLock.Lock()
	defer m.skillLock.Unlock()

	if _, exists := m.skillCalls[name]; !exists {
		m.skillCalls[name] = &atomic.Int64{}
	}
	m.skillCalls[name].Add(1)
}

// RecordProviderRequest records a provider request
func (m *Metrics) RecordProviderRequest(provider string) {
	m.providerLock.Lock()
	defer m.providerLock.Unlock()

	if _, exists := m.providerRequests[provider]; !exists {
		m.providerRequests[provider] = &atomic.Int64{}
	}
	m.providerRequests[provider].Add(1)
}

// RecordResponseTime records a response time
func (m *Metrics) RecordResponseTime(d time.Duration) {
	m.responseTimesLock.Lock()
	defer m.responseTimesLock.Unlock()

	m.responseTimes = append(m.responseTimes, d)

	// Cap at 1000 entries
	if len(m.responseTimes) > 1000 {
		m.responseTimes = m.responseTimes[len(m.responseTimes)-1000:]
	}
}

// SetActiveConnections sets the number of active connections
func (m *Metrics) SetActiveConnections(n int64) {
	m.activeConnections.Store(n)
}

// IncrementActiveConnections increments the active connections counter
func (m *Metrics) IncrementActiveConnections() {
	m.activeConnections.Add(1)
}

// DecrementActiveConnections decrements the active connections counter
func (m *Metrics) DecrementActiveConnections() {
	m.activeConnections.Add(-1)
}

// RecordSecurityBlock records a security block
func (m *Metrics) RecordSecurityBlock() {
	m.securityBlocks.Add(1)
}

// RecordInjectionBlocked records an injection block
func (m *Metrics) RecordInjectionBlocked() {
	m.injectionBlocked.Add(1)
}

// RecordSecretsDetected records detected secrets
func (m *Metrics) RecordSecretsDetected() {
	m.secretsDetected.Add(1)
}

// RecordPathTraversal records a path traversal attempt
func (m *Metrics) RecordPathTraversal() {
	m.pathTraversals.Add(1)
}

// RecordDangerousCommand records a dangerous command attempt
func (m *Metrics) RecordDangerousCommand() {
	m.dangerousCommands.Add(1)
}

// Snapshot returns a snapshot of current metrics
func (m *Metrics) Snapshot() *Snapshot {
	s := &Snapshot{
		RequestsTotal:     m.requestsTotal.Load(),
		RequestsSuccess:   m.requestsSuccess.Load(),
		RequestsFailed:    m.requestsFailed.Load(),
		RequestsBlocked:   m.requestsBlocked.Load(),
		TokensUsed:        m.tokensUsed.Load(),
		TokensPrompt:      m.tokensPrompt.Load(),
		TokensCompletion:  m.tokensCompletion.Load(),
		MessagesProcessed: m.messagesProcessed.Load(),
		MessagesBlocked:   m.messagesBlocked.Load(),
		ToolCallsTotal:    m.toolCallsTotal.Load(),
		ToolCallsSuccess:  m.toolCallsSuccess.Load(),
		ToolCallsFailed:   m.toolCallsFailed.Load(),
		ActiveConnections: m.activeConnections.Load(),
		SecurityBlocks:    m.securityBlocks.Load(),
		InjectionBlocked:  m.injectionBlocked.Load(),
		SecretsDetected:   m.secretsDetected.Load(),
		PathTraversals:    m.pathTraversals.Load(),
		DangerousCommands: m.dangerousCommands.Load(),
		Uptime:            time.Since(m.startTime),
		SkillCalls:        make(map[string]int64),
		ProviderRequests:  make(map[string]int64),
	}

	// Calculate success rate
	if s.RequestsTotal > 0 {
		s.SuccessRate = float64(s.RequestsSuccess) / float64(s.RequestsTotal) * 100
	}

	// Copy skill calls
	m.skillLock.Lock()
	for name, counter := range m.skillCalls {
		s.SkillCalls[name] = counter.Load()
	}
	m.skillLock.Unlock()

	// Copy provider requests
	m.providerLock.Lock()
	for name, counter := range m.providerRequests {
		s.ProviderRequests[name] = counter.Load()
	}
	m.providerLock.Unlock()

	// Calculate response time statistics
	m.responseTimesLock.Lock()
	if len(m.responseTimes) > 0 {
		times := make([]time.Duration, len(m.responseTimes))
		copy(times, m.responseTimes)
		m.responseTimesLock.Unlock()

		// Calculate average
		var total time.Duration
		for _, d := range times {
			total += d
		}
		s.AvgResponseTime = total / time.Duration(len(times))

		// Calculate P99
		sort.Slice(times, func(i, j int) bool {
			return times[i] < times[j]
		})
		p99Index := int(float64(len(times)) * 0.99)
		if p99Index >= len(times) {
			p99Index = len(times) - 1
		}
		s.P99ResponseTime = times[p99Index]
	} else {
		m.responseTimesLock.Unlock()
	}

	return s
}

// Prometheus returns Prometheus-formatted metrics
func (m *Metrics) Prometheus() string {
	s := m.Snapshot()

	var buf bytes.Buffer

	// Write metrics in Prometheus format
	fmt.Fprintf(&buf, "# HELP myrai_requests_total Total requests\n")
	fmt.Fprintf(&buf, "# TYPE myrai_requests_total counter\n")
	fmt.Fprintf(&buf, "myrai_requests_total %d\n", s.RequestsTotal)

	fmt.Fprintf(&buf, "# HELP myrai_requests_success Total successful requests\n")
	fmt.Fprintf(&buf, "# TYPE myrai_requests_success counter\n")
	fmt.Fprintf(&buf, "myrai_requests_success %d\n", s.RequestsSuccess)

	fmt.Fprintf(&buf, "# HELP myrai_requests_failed Total failed requests\n")
	fmt.Fprintf(&buf, "# TYPE myrai_requests_failed counter\n")
	fmt.Fprintf(&buf, "myrai_requests_failed %d\n", s.RequestsFailed)

	fmt.Fprintf(&buf, "# HELP myrai_requests_blocked Total blocked requests\n")
	fmt.Fprintf(&buf, "# TYPE myrai_requests_blocked counter\n")
	fmt.Fprintf(&buf, "myrai_requests_blocked %d\n", s.RequestsBlocked)

	fmt.Fprintf(&buf, "# HELP myrai_tokens_used Total tokens used\n")
	fmt.Fprintf(&buf, "# TYPE myrai_tokens_used counter\n")
	fmt.Fprintf(&buf, "myrai_tokens_used %d\n", s.TokensUsed)

	fmt.Fprintf(&buf, "# HELP myrai_tokens_prompt Total prompt tokens\n")
	fmt.Fprintf(&buf, "# TYPE myrai_tokens_prompt counter\n")
	fmt.Fprintf(&buf, "myrai_tokens_prompt %d\n", s.TokensPrompt)

	fmt.Fprintf(&buf, "# HELP myrai_tokens_completion Total completion tokens\n")
	fmt.Fprintf(&buf, "# TYPE myrai_tokens_completion counter\n")
	fmt.Fprintf(&buf, "myrai_tokens_completion %d\n", s.TokensCompletion)

	fmt.Fprintf(&buf, "# HELP myrai_messages_processed Total messages processed\n")
	fmt.Fprintf(&buf, "# TYPE myrai_messages_processed counter\n")
	fmt.Fprintf(&buf, "myrai_messages_processed %d\n", s.MessagesProcessed)

	fmt.Fprintf(&buf, "# HELP myrai_messages_blocked Total messages blocked\n")
	fmt.Fprintf(&buf, "# TYPE myrai_messages_blocked counter\n")
	fmt.Fprintf(&buf, "myrai_messages_blocked %d\n", s.MessagesBlocked)

	fmt.Fprintf(&buf, "# HELP myrai_tool_calls_total Total tool calls\n")
	fmt.Fprintf(&buf, "# TYPE myrai_tool_calls_total counter\n")
	fmt.Fprintf(&buf, "myrai_tool_calls_total %d\n", s.ToolCallsTotal)

	fmt.Fprintf(&buf, "# HELP myrai_tool_calls_success Total successful tool calls\n")
	fmt.Fprintf(&buf, "# TYPE myrai_tool_calls_success counter\n")
	fmt.Fprintf(&buf, "myrai_tool_calls_success %d\n", s.ToolCallsSuccess)

	fmt.Fprintf(&buf, "# HELP myrai_tool_calls_failed Total failed tool calls\n")
	fmt.Fprintf(&buf, "# TYPE myrai_tool_calls_failed counter\n")
	fmt.Fprintf(&buf, "myrai_tool_calls_failed %d\n", s.ToolCallsFailed)

	fmt.Fprintf(&buf, "# HELP myrai_active_connections Current active connections\n")
	fmt.Fprintf(&buf, "# TYPE myrai_active_connections gauge\n")
	fmt.Fprintf(&buf, "myrai_active_connections %d\n", s.ActiveConnections)

	fmt.Fprintf(&buf, "# HELP myrai_security_blocks_total Total security blocks\n")
	fmt.Fprintf(&buf, "# TYPE myrai_security_blocks_total counter\n")
	fmt.Fprintf(&buf, "myrai_security_blocks_total %d\n", s.SecurityBlocks)

	fmt.Fprintf(&buf, "# HELP myrai_injection_blocked_total Total injection blocks\n")
	fmt.Fprintf(&buf, "# TYPE myrai_injection_blocked_total counter\n")
	fmt.Fprintf(&buf, "myrai_injection_blocked_total %d\n", s.InjectionBlocked)

	fmt.Fprintf(&buf, "# HELP myrai_secrets_detected_total Total secrets detected\n")
	fmt.Fprintf(&buf, "# TYPE myrai_secrets_detected_total counter\n")
	fmt.Fprintf(&buf, "myrai_secrets_detected_total %d\n", s.SecretsDetected)

	fmt.Fprintf(&buf, "# HELP myrai_path_traversals_total Total path traversal attempts\n")
	fmt.Fprintf(&buf, "# TYPE myrai_path_traversals_total counter\n")
	fmt.Fprintf(&buf, "myrai_path_traversals_total %d\n", s.PathTraversals)

	fmt.Fprintf(&buf, "# HELP myrai_dangerous_commands_total Total dangerous command attempts\n")
	fmt.Fprintf(&buf, "# TYPE myrai_dangerous_commands_total counter\n")
	fmt.Fprintf(&buf, "myrai_dangerous_commands_total %d\n", s.DangerousCommands)

	fmt.Fprintf(&buf, "# HELP myrai_avg_response_time_seconds Average response time\n")
	fmt.Fprintf(&buf, "# TYPE myrai_avg_response_time_seconds gauge\n")
	fmt.Fprintf(&buf, "myrai_avg_response_time_seconds %.6f\n", s.AvgResponseTime.Seconds())

	fmt.Fprintf(&buf, "# HELP myrai_p99_response_time_seconds P99 response time\n")
	fmt.Fprintf(&buf, "# TYPE myrai_p99_response_time_seconds gauge\n")
	fmt.Fprintf(&buf, "myrai_p99_response_time_seconds %.6f\n", s.P99ResponseTime.Seconds())

	fmt.Fprintf(&buf, "# HELP myrai_uptime_seconds Uptime in seconds\n")
	fmt.Fprintf(&buf, "# TYPE myrai_uptime_seconds gauge\n")
	fmt.Fprintf(&buf, "myrai_uptime_seconds %.6f\n", s.Uptime.Seconds())

	fmt.Fprintf(&buf, "# HELP myrai_success_rate Success rate percentage\n")
	fmt.Fprintf(&buf, "# TYPE myrai_success_rate gauge\n")
	fmt.Fprintf(&buf, "myrai_success_rate %.6f\n", s.SuccessRate)

	// Skill calls with labels
	fmt.Fprintf(&buf, "# HELP myrai_skill_calls_total Total skill calls\n")
	fmt.Fprintf(&buf, "# TYPE myrai_skill_calls_total counter\n")
	for skill, count := range s.SkillCalls {
		fmt.Fprintf(&buf, `myrai_skill_calls_total{skill="%s"} %d`+"\n", skill, count)
	}

	// Provider requests with labels
	fmt.Fprintf(&buf, "# HELP myrai_provider_requests_total Total provider requests\n")
	fmt.Fprintf(&buf, "# TYPE myrai_provider_requests_total counter\n")
	for provider, count := range s.ProviderRequests {
		fmt.Fprintf(&buf, `myrai_provider_requests_total{provider="%s"} %d`+"\n", provider, count)
	}

	return buf.String()
}

// Helper functions that use the default instance

// RecordRequest records a request with the default metrics
func RecordRequest(success bool) {
	Default().RecordRequest(success)
}

// RecordTokens records token usage with the default metrics
func RecordTokens(prompt, completion int64) {
	Default().RecordTokens(prompt, completion)
}

// RecordToolCall records a tool call with the default metrics
func RecordToolCall(success bool) {
	Default().RecordToolCall(success)
}

// RecordSkillCall records a skill call with the default metrics
func RecordSkillCall(name string) {
	Default().RecordSkillCall(name)
}

// RecordProviderRequest records a provider request with the default metrics
func RecordProviderRequest(provider string) {
	Default().RecordProviderRequest(provider)
}

// RecordSecurityBlock records a security block with the default metrics
func RecordSecurityBlock() {
	Default().RecordSecurityBlock()
}

// RecordInjectionBlocked records an injection block with the default metrics
func RecordInjectionBlocked() {
	Default().RecordInjectionBlocked()
}

// RecordSecretsDetected records detected secrets with the default metrics
func RecordSecretsDetected() {
	Default().RecordSecretsDetected()
}

// RecordDangerousCommand records a dangerous command with the default metrics
func RecordDangerousCommand() {
	Default().RecordDangerousCommand()
}

// GetSnapshot returns a snapshot from the default metrics
func GetSnapshot() *Snapshot {
	return Default().Snapshot()
}

// GetPrometheus returns Prometheus metrics in text format
func GetPrometheus() string {
	buf := &bytes.Buffer{}
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
	req, _ := http.NewRequest("GET", "/metrics", nil)
	handler.ServeHTTP(&mockResponseWriter{buf: buf}, req)
	return buf.String()
}

// mockResponseWriter implements http.ResponseWriter for metrics
type mockResponseWriter struct {
	buf    *bytes.Buffer
	header http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}
func (m *mockResponseWriter) Write(p []byte) (int, error) { return m.buf.Write(p) }
func (m *mockResponseWriter) WriteHeader(statusCode int)  {}

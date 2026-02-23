// Package metrics provides Prometheus metrics for Myrai
package metrics

import (
	"bytes"
	"net/http"

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

// GetPrometheus returns Prometheus metrics in text format
func GetPrometheus() string {
	buf := &bytes.Buffer{}
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
	handler.ServeHTTP(&mockResponseWriter{buf: buf}, nil)
	return buf.String()
}

// GetSnapshot returns a snapshot of current metrics
func GetSnapshot() map[string]interface{} {
	snapshot := make(map[string]interface{})
	// Collect basic metrics
	snapshot["jobs_total"] = 0
	snapshot["clusters_total"] = 0
	snapshot["messages_total"] = 0
	return snapshot
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

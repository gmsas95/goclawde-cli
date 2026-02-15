package metrics

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	startTime time.Time

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

	activeConnections atomic.Int64
	activeConversations atomic.Int64

	responseTimes     []time.Duration
	responseTimesLock sync.Mutex

	providerRequests map[string]*atomic.Int64
	providerLock     sync.Mutex

	skillCalls map[string]*atomic.Int64
	skillLock  sync.Mutex

	securityBlocks      atomic.Int64
	injectionBlocked    atomic.Int64
	secretsDetected     atomic.Int64
	pathTraversals      atomic.Int64
	dangerousCommands   atomic.Int64
}

var (
	defaultMetrics *Metrics
	once           sync.Once
)

func Default() *Metrics {
	once.Do(func() {
		defaultMetrics = New()
	})
	return defaultMetrics
}

func New() *Metrics {
	m := &Metrics{
		startTime:       time.Now(),
		responseTimes:   make([]time.Duration, 0, 1000),
		providerRequests: make(map[string]*atomic.Int64),
		skillCalls:      make(map[string]*atomic.Int64),
	}
	return m
}

func (m *Metrics) RecordRequest(success bool) {
	m.requestsTotal.Add(1)
	if success {
		m.requestsSuccess.Add(1)
	} else {
		m.requestsFailed.Add(1)
	}
}

func (m *Metrics) RecordRequestBlocked() {
	m.requestsBlocked.Add(1)
}

func (m *Metrics) RecordTokens(prompt, completion int64) {
	m.tokensUsed.Add(prompt + completion)
	m.tokensPrompt.Add(prompt)
	m.tokensCompletion.Add(completion)
}

func (m *Metrics) RecordMessage(blocked bool) {
	m.messagesProcessed.Add(1)
	if blocked {
		m.messagesBlocked.Add(1)
	}
}

func (m *Metrics) RecordToolCall(success bool) {
	m.toolCallsTotal.Add(1)
	if success {
		m.toolCallsSuccess.Add(1)
	} else {
		m.toolCallsFailed.Add(1)
	}
}

func (m *Metrics) RecordSkillCall(skill string) {
	m.skillLock.Lock()
	defer m.skillLock.Unlock()

	if m.skillCalls[skill] == nil {
		m.skillCalls[skill] = &atomic.Int64{}
	}
	m.skillCalls[skill].Add(1)
}

func (m *Metrics) RecordProviderRequest(provider string) {
	m.providerLock.Lock()
	defer m.providerLock.Unlock()

	if m.providerRequests[provider] == nil {
		m.providerRequests[provider] = &atomic.Int64{}
	}
	m.providerRequests[provider].Add(1)
}

func (m *Metrics) RecordResponseTime(d time.Duration) {
	m.responseTimesLock.Lock()
	defer m.responseTimesLock.Unlock()

	m.responseTimes = append(m.responseTimes, d)
	if len(m.responseTimes) > 1000 {
		m.responseTimes = m.responseTimes[1:]
	}
}

func (m *Metrics) SetActiveConnections(count int64) {
	m.activeConnections.Store(count)
}

func (m *Metrics) IncrementActiveConnections() {
	m.activeConnections.Add(1)
}

func (m *Metrics) DecrementActiveConnections() {
	m.activeConnections.Add(-1)
}

func (m *Metrics) SetActiveConversations(count int64) {
	m.activeConversations.Store(count)
}

func (m *Metrics) RecordSecurityBlock() {
	m.securityBlocks.Add(1)
}

func (m *Metrics) RecordInjectionBlocked() {
	m.injectionBlocked.Add(1)
}

func (m *Metrics) RecordSecretsDetected() {
	m.secretsDetected.Add(1)
}

func (m *Metrics) RecordPathTraversal() {
	m.pathTraversals.Add(1)
}

func (m *Metrics) RecordDangerousCommand() {
	m.dangerousCommands.Add(1)
}

type Snapshot struct {
	Uptime              time.Duration     `json:"uptime"`
	RequestsTotal       int64             `json:"requests_total"`
	RequestsSuccess     int64             `json:"requests_success"`
	RequestsFailed      int64             `json:"requests_failed"`
	RequestsBlocked     int64             `json:"requests_blocked"`
	TokensUsed          int64             `json:"tokens_used"`
	TokensPrompt        int64             `json:"tokens_prompt"`
	TokensCompletion    int64             `json:"tokens_completion"`
	MessagesProcessed   int64             `json:"messages_processed"`
	MessagesBlocked     int64             `json:"messages_blocked"`
	ToolCallsTotal      int64             `json:"tool_calls_total"`
	ToolCallsSuccess    int64             `json:"tool_calls_success"`
	ToolCallsFailed     int64             `json:"tool_calls_failed"`
	ActiveConnections   int64             `json:"active_connections"`
	ActiveConversations int64             `json:"active_conversations"`
	AvgResponseTime     time.Duration     `json:"avg_response_time"`
	P99ResponseTime     time.Duration     `json:"p99_response_time"`
	ProviderRequests    map[string]int64  `json:"provider_requests"`
	SkillCalls          map[string]int64  `json:"skill_calls"`
	SecurityBlocks      int64             `json:"security_blocks"`
	InjectionBlocked    int64             `json:"injection_blocked"`
	SecretsDetected     int64             `json:"secrets_detected"`
	PathTraversals      int64             `json:"path_traversals"`
	DangerousCommands   int64             `json:"dangerous_commands"`
	SuccessRate         float64           `json:"success_rate"`
}

func (m *Metrics) Snapshot() *Snapshot {
	s := &Snapshot{
		Uptime:              time.Since(m.startTime),
		RequestsTotal:       m.requestsTotal.Load(),
		RequestsSuccess:     m.requestsSuccess.Load(),
		RequestsFailed:      m.requestsFailed.Load(),
		RequestsBlocked:     m.requestsBlocked.Load(),
		TokensUsed:          m.tokensUsed.Load(),
		TokensPrompt:        m.tokensPrompt.Load(),
		TokensCompletion:    m.tokensCompletion.Load(),
		MessagesProcessed:   m.messagesProcessed.Load(),
		MessagesBlocked:     m.messagesBlocked.Load(),
		ToolCallsTotal:      m.toolCallsTotal.Load(),
		ToolCallsSuccess:    m.toolCallsSuccess.Load(),
		ToolCallsFailed:     m.toolCallsFailed.Load(),
		ActiveConnections:   m.activeConnections.Load(),
		ActiveConversations: m.activeConversations.Load(),
		SecurityBlocks:      m.securityBlocks.Load(),
		InjectionBlocked:    m.injectionBlocked.Load(),
		SecretsDetected:     m.secretsDetected.Load(),
		PathTraversals:      m.pathTraversals.Load(),
		DangerousCommands:   m.dangerousCommands.Load(),
		ProviderRequests:    make(map[string]int64),
		SkillCalls:          make(map[string]int64),
	}

	if s.RequestsTotal > 0 {
		s.SuccessRate = float64(s.RequestsSuccess) / float64(s.RequestsTotal) * 100
	}

	m.responseTimesLock.Lock()
	if len(m.responseTimes) > 0 {
		var total time.Duration
		for _, rt := range m.responseTimes {
			total += rt
		}
		s.AvgResponseTime = total / time.Duration(len(m.responseTimes))

		sorted := make([]time.Duration, len(m.responseTimes))
		copy(sorted, m.responseTimes)
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j] < sorted[i] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		p99Index := int(float64(len(sorted)) * 0.99)
		if p99Index >= len(sorted) {
			p99Index = len(sorted) - 1
		}
		s.P99ResponseTime = sorted[p99Index]
	}
	m.responseTimesLock.Unlock()

	m.providerLock.Lock()
	for k, v := range m.providerRequests {
		s.ProviderRequests[k] = v.Load()
	}
	m.providerLock.Unlock()

	m.skillLock.Lock()
	for k, v := range m.skillCalls {
		s.SkillCalls[k] = v.Load()
	}
	m.skillLock.Unlock()

	return s
}

func (m *Metrics) Prometheus() string {
	var sb strings.Builder

	sb.WriteString("# HELP goclawde_uptime_seconds Time since server start\n")
	sb.WriteString("# TYPE goclawde_uptime_seconds gauge\n")
	sb.WriteString("goclawde_uptime_seconds " + strconv.FormatInt(int64(time.Since(m.startTime).Seconds()), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_requests_total Total number of requests\n")
	sb.WriteString("# TYPE goclawde_requests_total counter\n")
	sb.WriteString("goclawde_requests_total " + strconv.FormatInt(m.requestsTotal.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_requests_success Successful requests\n")
	sb.WriteString("# TYPE goclawde_requests_success counter\n")
	sb.WriteString("goclawde_requests_success " + strconv.FormatInt(m.requestsSuccess.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_requests_failed Failed requests\n")
	sb.WriteString("# TYPE goclawde_requests_failed counter\n")
	sb.WriteString("goclawde_requests_failed " + strconv.FormatInt(m.requestsFailed.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_tokens_used Total tokens used\n")
	sb.WriteString("# TYPE goclawde_tokens_used counter\n")
	sb.WriteString("goclawde_tokens_used " + strconv.FormatInt(m.tokensUsed.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_tokens_prompt Prompt tokens used\n")
	sb.WriteString("# TYPE goclawde_tokens_prompt counter\n")
	sb.WriteString("goclawde_tokens_prompt " + strconv.FormatInt(m.tokensPrompt.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_tokens_completion Completion tokens used\n")
	sb.WriteString("# TYPE goclawde_tokens_completion counter\n")
	sb.WriteString("goclawde_tokens_completion " + strconv.FormatInt(m.tokensCompletion.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_tool_calls_total Total tool calls\n")
	sb.WriteString("# TYPE goclawde_tool_calls_total counter\n")
	sb.WriteString("goclawde_tool_calls_total " + strconv.FormatInt(m.toolCallsTotal.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_active_connections Active connections\n")
	sb.WriteString("# TYPE goclawde_active_connections gauge\n")
	sb.WriteString("goclawde_active_connections " + strconv.FormatInt(m.activeConnections.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_security_blocks_total Security blocks\n")
	sb.WriteString("# TYPE goclawde_security_blocks_total counter\n")
	sb.WriteString("goclawde_security_blocks_total " + strconv.FormatInt(m.securityBlocks.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_injection_blocked_total Prompt injections blocked\n")
	sb.WriteString("# TYPE goclawde_injection_blocked_total counter\n")
	sb.WriteString("goclawde_injection_blocked_total " + strconv.FormatInt(m.injectionBlocked.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_secrets_detected_total Secrets detected\n")
	sb.WriteString("# TYPE goclawde_secrets_detected_total counter\n")
	sb.WriteString("goclawde_secrets_detected_total " + strconv.FormatInt(m.secretsDetected.Load(), 10) + "\n\n")

	sb.WriteString("# HELP goclawde_dangerous_commands_blocked_total Dangerous commands blocked\n")
	sb.WriteString("# TYPE goclawde_dangerous_commands_blocked_total counter\n")
	sb.WriteString("goclawde_dangerous_commands_blocked_total " + strconv.FormatInt(m.dangerousCommands.Load(), 10) + "\n\n")

	m.providerLock.Lock()
	for provider, count := range m.providerRequests {
		sb.WriteString("# HELP goclawde_provider_requests_total Requests per provider\n")
		sb.WriteString("# TYPE goclawde_provider_requests_total counter\n")
		sb.WriteString("goclawde_provider_requests_total{provider=\"" + provider + "\"} " + strconv.FormatInt(count.Load(), 10) + "\n\n")
	}
	m.providerLock.Unlock()

	m.skillLock.Lock()
	for skill, count := range m.skillCalls {
		sb.WriteString("# HELP goclawde_skill_calls_total Calls per skill\n")
		sb.WriteString("# TYPE goclawde_skill_calls_total counter\n")
		sb.WriteString("goclawde_skill_calls_total{skill=\"" + skill + "\"} " + strconv.FormatInt(count.Load(), 10) + "\n\n")
	}
	m.skillLock.Unlock()

	return sb.String()
}

func RecordRequest(success bool) {
	Default().RecordRequest(success)
}

func RecordRequestBlocked() {
	Default().RecordRequestBlocked()
}

func RecordTokens(prompt, completion int64) {
	Default().RecordTokens(prompt, completion)
}

func RecordToolCall(success bool) {
	Default().RecordToolCall(success)
}

func RecordSkillCall(skill string) {
	Default().RecordSkillCall(skill)
}

func RecordProviderRequest(provider string) {
	Default().RecordProviderRequest(provider)
}

func RecordResponseTime(d time.Duration) {
	Default().RecordResponseTime(d)
}

func RecordSecurityBlock() {
	Default().RecordSecurityBlock()
}

func RecordInjectionBlocked() {
	Default().RecordInjectionBlocked()
}

func RecordSecretsDetected() {
	Default().RecordSecretsDetected()
}

func RecordDangerousCommand() {
	Default().RecordDangerousCommand()
}

func Snapshot() *Snapshot {
	return Default().Snapshot()
}

func Prometheus() string {
	return Default().Prometheus()
}

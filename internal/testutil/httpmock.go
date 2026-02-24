// Package testutil provides testing utilities for the myrai-cli project
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
)

// ResponseRecorder records HTTP responses for later verification
type ResponseRecorder struct {
	Responses []RecordedResponse
	mu        sync.Mutex
}

// RecordedResponse represents a captured HTTP response
type RecordedResponse struct {
	RequestURL      string
	RequestMethod   string
	RequestBody     []byte
	RequestHeaders  http.Header
	StatusCode      int
	ResponseBody    []byte
	ResponseHeaders http.Header
	Timestamp       time.Time
	Duration        time.Duration
}

// NewResponseRecorder creates a new response recorder
func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Responses: []RecordedResponse{},
	}
}

// Record records a request-response pair
func (r *ResponseRecorder) Record(req *http.Request, resp *http.Response, body []byte, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	r.Responses = append(r.Responses, RecordedResponse{
		RequestURL:      req.URL.String(),
		RequestMethod:   req.Method,
		RequestBody:     reqBody,
		RequestHeaders:  req.Header.Clone(),
		StatusCode:      resp.StatusCode,
		ResponseBody:    body,
		ResponseHeaders: resp.Header.Clone(),
		Timestamp:       time.Now(),
		Duration:        duration,
	})
}

// GetLast returns the most recent recorded response
func (r *ResponseRecorder) GetLast() *RecordedResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Responses) == 0 {
		return nil
	}
	return &r.Responses[len(r.Responses)-1]
}

// GetAll returns all recorded responses
func (r *ResponseRecorder) GetAll() []RecordedResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]RecordedResponse, len(r.Responses))
	copy(result, r.Responses)
	return result
}

// Count returns the number of recorded responses
func (r *ResponseRecorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Responses)
}

// Reset clears all recorded responses
func (r *ResponseRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Responses = []RecordedResponse{}
}

// HTTPMock provides HTTP mocking capabilities for testing
type HTTPMock struct {
	Server       *httptest.Server
	Handler      http.HandlerFunc
	Recorder     *ResponseRecorder
	Expectations []RequestExpectation
	mu           sync.Mutex
}

// RequestExpectation defines an expected request
type RequestExpectation struct {
	Method     string
	URLPath    string
	Headers    map[string]string
	BodyMatch  string
	Response   *http.Response
	ResponseFn func(*http.Request) *http.Response
}

// NewHTTPMock creates a new HTTP mock server
func NewHTTPMock() *HTTPMock {
	recorder := NewResponseRecorder()

	mock := &HTTPMock{
		Recorder:     recorder,
		Expectations: []RequestExpectation{},
	}

	mock.Handler = func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Check expectations
		for _, exp := range mock.Expectations {
			if exp.Method != "" && exp.Method != r.Method {
				continue
			}
			if exp.URLPath != "" && exp.URLPath != r.URL.Path {
				continue
			}

			// Found matching expectation
			var resp *http.Response
			if exp.ResponseFn != nil {
				resp = exp.ResponseFn(r)
			} else if exp.Response != nil {
				resp = exp.Response
			} else {
				resp = &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
					Header:     make(http.Header),
				}
			}

			// Copy response to response writer
			for k, v := range resp.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			w.Write(body)

			// Record
			recorder.Record(r, resp, body, time.Since(start))
			return
		}

		// Default response
		w.WriteHeader(200)
		w.Write([]byte("{}"))
	}

	mock.Server = httptest.NewServer(mock.Handler)
	return mock
}

// Close shuts down the mock server
func (m *HTTPMock) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// URL returns the mock server URL
func (m *HTTPMock) URL() string {
	return m.Server.URL
}

// AddExpectation adds a request expectation
func (m *HTTPMock) AddExpectation(exp RequestExpectation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Expectations = append(m.Expectations, exp)
}

// ExpectGET adds an expectation for a GET request
func (m *HTTPMock) ExpectGET(path string, response interface{}) {
	body, _ := json.Marshal(response)
	m.AddExpectation(RequestExpectation{
		Method:  "GET",
		URLPath: path,
		Response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		},
	})
}

// ExpectPOST adds an expectation for a POST request
func (m *HTTPMock) ExpectPOST(path string, response interface{}) {
	body, _ := json.Marshal(response)
	m.AddExpectation(RequestExpectation{
		Method:  "POST",
		URLPath: path,
		Response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		},
	})
}

// ExpectError adds an expectation that returns an error status
func (m *HTTPMock) ExpectError(path string, statusCode int, message string) {
	m.AddExpectation(RequestExpectation{
		URLPath: path,
		Response: &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewReader([]byte(message))),
			Header:     make(http.Header),
		},
	})
}

// MockLLM creates a mock LLM client
func MockLLM(t *testing.T, responses map[string]string) (*llm.Client, *HTTPMock) {
	mock := NewHTTPMock()

	mock.AddExpectation(RequestExpectation{
		Method:  "POST",
		URLPath: "/chat/completions",
		ResponseFn: func(req *http.Request) *http.Response {
			// Parse request to determine response
			var reqBody map[string]interface{}
			json.NewDecoder(req.Body).Decode(&reqBody)

			messages, _ := reqBody["messages"].([]interface{})
			var lastContent string
			if len(messages) > 0 {
				lastMsg := messages[len(messages)-1].(map[string]interface{})
				lastContent, _ = lastMsg["content"].(string)
			}

			// Find matching response
			responseText := "Mock LLM response"
			for pattern, resp := range responses {
				if contains(lastContent, pattern) {
					responseText = resp
					break
				}
			}

			respBody := map[string]interface{}{
				"id":      "mock-completion",
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   "mock-model",
				"choices": []interface{}{
					map[string]interface{}{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": responseText,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			}

			body, _ := json.Marshal(respBody)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}
		},
	})

	// Create LLM client pointing to mock server
	provider := config.Provider{
		BaseURL:   mock.URL(),
		APIKey:    "mock-api-key",
		Model:     "mock-model",
		MaxTokens: 1000,
		Timeout:   30,
	}
	client := llm.NewClient(provider)

	return client, mock
}

// MockGitHub creates a mock GitHub API server
func MockGitHub(t *testing.T) *HTTPMock {
	mock := NewHTTPMock()

	// Mock user endpoint
	mock.ExpectGET("/user", map[string]interface{}{
		"login":      "testuser",
		"id":         12345,
		"avatar_url": "https://example.com/avatar.png",
	})

	// Mock repos endpoint
	mock.ExpectGET("/user/repos", []interface{}{
		map[string]interface{}{
			"id":        1,
			"name":      "test-repo",
			"full_name": "testuser/test-repo",
			"private":   false,
		},
	})

	// Mock issues endpoint
	mock.ExpectGET("/repos/testuser/test-repo/issues", []interface{}{
		map[string]interface{}{
			"id":     1,
			"number": 1,
			"title":  "Test Issue",
			"state":  "open",
		},
	})

	return mock
}

// MockMCP creates a mock MCP server
func MockMCP(t *testing.T) *HTTPMock {
	mock := NewHTTPMock()

	// Mock initialize endpoint
	mock.AddExpectation(RequestExpectation{
		Method:    "POST",
		BodyMatch: "initialize",
		ResponseFn: func(req *http.Request) *http.Response {
			respBody := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"serverInfo": map[string]interface{}{
						"name":    "mock-mcp-server",
						"version": "1.0.0",
					},
				},
			}
			body, _ := json.Marshal(respBody)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}
		},
	})

	// Mock tools/list endpoint
	mock.AddExpectation(RequestExpectation{
		Method:    "POST",
		BodyMatch: "tools/list",
		ResponseFn: func(req *http.Request) *http.Response {
			respBody := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      2,
				"result": map[string]interface{}{
					"tools": []interface{}{
						map[string]interface{}{
							"name":        "mock_tool",
							"description": "A mock tool for testing",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"input": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
			}
			body, _ := json.Marshal(respBody)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}
		},
	})

	// Mock tools/call endpoint
	mock.AddExpectation(RequestExpectation{
		Method:    "POST",
		BodyMatch: "tools/call",
		ResponseFn: func(req *http.Request) *http.Response {
			respBody := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      3,
				"result": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Mock tool result",
						},
					},
				},
			}
			body, _ := json.Marshal(respBody)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}
		},
	})

	return mock
}

// AssertRequestCount verifies the number of requests made
func AssertRequestCount(t *testing.T, recorder *ResponseRecorder, expected int) {
	t.Helper()
	if recorder.Count() != expected {
		t.Errorf("Expected %d requests, got %d", expected, recorder.Count())
	}
}

// AssertRequestMade verifies a request was made with the given method and path
func AssertRequestMade(t *testing.T, recorder *ResponseRecorder, method, path string) {
	t.Helper()
	for _, resp := range recorder.GetAll() {
		if resp.RequestMethod == method && resp.RequestURL == path {
			return
		}
	}
	t.Errorf("Expected %s request to %s was not made", method, path)
}

// AssertJSONBody verifies the request body matches expected JSON
func AssertJSONBody(t *testing.T, recorder *ResponseRecorder, expected map[string]interface{}) {
	t.Helper()
	last := recorder.GetLast()
	if last == nil {
		t.Fatal("No requests recorded")
	}

	var actual map[string]interface{}
	if err := json.Unmarshal(last.RequestBody, &actual); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	// Compare key fields
	for key, expectedVal := range expected {
		actualVal, ok := actual[key]
		if !ok {
			t.Errorf("Expected key %q not found in request body", key)
			continue
		}
		if actualVal != expectedVal {
			t.Errorf("Expected %q=%v, got %v", key, expectedVal, actualVal)
		}
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

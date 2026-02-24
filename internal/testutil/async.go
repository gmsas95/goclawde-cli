// Package testutil provides testing utilities and helpers
package testutil

import (
	"context"
	"sync"
	"testing"
	"time"
)

// AsyncTestHelper provides utilities for testing async operations
type AsyncTestHelper struct {
	t         *testing.T
	wg        sync.WaitGroup
	mu        sync.Mutex
	errors    []error
	completed int
	maxWait   time.Duration
}

// NewAsyncTestHelper creates a new async test helper
func NewAsyncTestHelper(t *testing.T) *AsyncTestHelper {
	return &AsyncTestHelper{
		t:       t,
		errors:  make([]error, 0),
		maxWait: 5 * time.Second,
	}
}

// SetMaxWait sets the maximum wait time for operations
func (h *AsyncTestHelper) SetMaxWait(d time.Duration) {
	h.maxWait = d
}

// Go runs a function asynchronously and tracks it
func (h *AsyncTestHelper) Go(fn func() error) {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		if err := fn(); err != nil {
			h.mu.Lock()
			h.errors = append(h.errors, err)
			h.mu.Unlock()
		}
		h.mu.Lock()
		h.completed++
		h.mu.Unlock()
	}()
}

// Wait waits for all async operations to complete
func (h *AsyncTestHelper) Wait() {
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(h.maxWait):
		h.t.Fatalf("Async operations timed out after %v", h.maxWait)
	}
}

// WaitWithContext waits with a context
func (h *AsyncTestHelper) WaitWithContext(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetErrors returns all errors collected during async operations
func (h *AsyncTestHelper) GetErrors() []error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]error{}, h.errors...)
}

// GetCompletedCount returns the number of completed operations
func (h *AsyncTestHelper) GetCompletedCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.completed
}

// AssertNoErrors fails the test if any errors occurred
func (h *AsyncTestHelper) AssertNoErrors() {
	h.mu.Lock()
	errs := append([]error{}, h.errors...)
	h.mu.Unlock()

	if len(errs) > 0 {
		h.t.Fatalf("Expected no errors, got %d: %v", len(errs), errs)
	}
}

// Signal is a one-time signal for async coordination
type Signal struct {
	ch     chan struct{}
	closed bool
	mu     sync.Mutex
}

// NewSignal creates a new signal
func NewSignal() *Signal {
	return &Signal{ch: make(chan struct{})}
}

// Trigger triggers the signal (safe to call multiple times)
func (s *Signal) Trigger() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		close(s.ch)
		s.closed = true
	}
}

// Wait waits for the signal with timeout
func (s *Signal) Wait(timeout time.Duration) bool {
	select {
	case <-s.ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

// WaitContext waits for the signal with context
func (s *Signal) WaitContext(ctx context.Context) error {
	select {
	case <-s.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Triggered returns whether the signal has been triggered
func (s *Signal) Triggered() bool {
	select {
	case <-s.ch:
		return true
	default:
		return false
	}
}

// Eventually retries a condition until it succeeds or times out
func Eventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// EventuallyWithT is like Eventually but uses testing.T
func EventuallyWithT(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration) {
	if !Eventually(t, condition, timeout, interval) {
		t.Fatalf("Condition not met within %v", timeout)
	}
}

// WaitForCondition waits for a condition with timeout
func WaitForCondition(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// Retry retries an operation until it succeeds or max attempts reached
func Retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return err
}

// MockClock provides a mockable clock for testing
type MockClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*mockTimer
}

// NewMockClock creates a new mock clock
func NewMockClock(start time.Time) *MockClock {
	return &MockClock{now: start}
}

// Now returns the current mock time
func (c *MockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance advances the mock time
func (c *MockClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)

	// Trigger expired timers
	for _, t := range c.timers {
		if !t.triggered && c.now.After(t.deadline) {
			t.trigger()
		}
	}
}

// Set sets the mock time
func (c *MockClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

type mockTimer struct {
	deadline  time.Time
	triggered bool
	ch        chan time.Time
}

func (t *mockTimer) trigger() {
	if !t.triggered {
		t.triggered = true
		go func() {
			t.ch <- t.deadline
		}()
	}
}

// After returns a channel that receives after duration
func (c *MockClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	timer := &mockTimer{
		deadline: c.now.Add(d),
		ch:       make(chan time.Time, 1),
	}
	c.timers = append(c.timers, timer)

	// If already past deadline, trigger immediately
	if c.now.After(timer.deadline) {
		timer.trigger()
	}

	return timer.ch
}

// Counter is a thread-safe counter for testing
type Counter struct {
	mu    sync.Mutex
	value int64
}

// NewCounter creates a new counter
func NewCounter() *Counter {
	return &Counter{}
}

// Inc increments the counter
func (c *Counter) Inc() {
	c.mu.Lock()
	c.value++
	c.mu.Unlock()
}

// Add adds n to the counter
func (c *Counter) Add(n int64) {
	c.mu.Lock()
	c.value += n
	c.mu.Unlock()
}

// Get returns the current value
func (c *Counter) Get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// Reset resets the counter to 0
func (c *Counter) Reset() {
	c.mu.Lock()
	c.value = 0
	c.mu.Unlock()
}

// SafeBool is a thread-safe boolean for testing
type SafeBool struct {
	mu  sync.Mutex
	val bool
}

// Set sets the value
func (b *SafeBool) Set(v bool) {
	b.mu.Lock()
	b.val = v
	b.mu.Unlock()
}

// Get gets the value
func (b *SafeBool) Get() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.val
}

// Toggle toggles the value
func (b *SafeBool) Toggle() {
	b.mu.Lock()
	b.val = !b.val
	b.mu.Unlock()
}

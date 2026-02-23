package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ChainType represents the execution type of a tool chain
type ChainType string

const (
	ChainSequential  ChainType = "sequential"
	ChainParallel    ChainType = "parallel"
	ChainConditional ChainType = "conditional"
)

// ToolChain represents a chain of tool executions
type ToolChain struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Version     string            `json:"version" yaml:"version"`
	Author      string            `json:"author" yaml:"author"`
	Type        ChainType         `json:"type" yaml:"type"`
	Steps       []ChainStep       `json:"steps" yaml:"steps"`
	Variables   map[string]string `json:"variables" yaml:"variables"`
	IsBuiltIn   bool              `json:"is_builtin" yaml:"is_builtin,omitempty"`
	IsShared    bool              `json:"is_shared" yaml:"is_shared,omitempty"`
}

// ChainStep represents a single step in a chain
type ChainStep struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Tool        string                 `json:"tool" yaml:"tool"`
	Parameters  map[string]interface{} `json:"parameters" yaml:"parameters"`
	DependsOn   []string               `json:"depends_on" yaml:"depends_on,omitempty"`
	Condition   string                 `json:"condition" yaml:"condition,omitempty"`
	Timeout     time.Duration          `json:"timeout" yaml:"timeout,omitempty"`
	RetryCount  int                    `json:"retry_count" yaml:"retry_count,omitempty"`
	Optional    bool                   `json:"optional" yaml:"optional,omitempty"`
	OnFailure   string                 `json:"on_failure" yaml:"on_failure,omitempty"`
}

// ChainResult represents the result of executing a chain
type ChainResult struct {
	ChainID     string            `json:"chain_id"`
	Status      string            `json:"status"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	StepResults []StepResult      `json:"step_results"`
	Variables   map[string]string `json:"variables"`
	Error       string            `json:"error,omitempty"`
	TotalSteps  int               `json:"total_steps"`
	PassedSteps int               `json:"passed_steps"`
	FailedSteps int               `json:"failed_steps"`
}

// StepResult represents the result of a single step
type StepResult struct {
	StepID      string        `json:"step_id"`
	Status      string        `json:"status"`
	Output      interface{}   `json:"output,omitempty"`
	Error       string        `json:"error,omitempty"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration"`
	RetryCount  int           `json:"retry_count"`
}

// ToolExecutor is the interface for executing tools
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error)
}

// Execute runs the chain with the given context and executor
func (tc *ToolChain) Execute(ctx context.Context, executor ToolExecutor) (*ChainResult, error) {
	result := &ChainResult{
		ChainID:     tc.ID,
		Status:      "running",
		StartedAt:   time.Now(),
		StepResults: make([]StepResult, 0, len(tc.Steps)),
		Variables:   make(map[string]string),
		TotalSteps:  len(tc.Steps),
	}

	// Copy initial variables
	for k, v := range tc.Variables {
		result.Variables[k] = v
	}

	var err error
	switch tc.Type {
	case ChainSequential:
		err = tc.executeSequential(ctx, executor, result)
	case ChainParallel:
		err = tc.executeParallel(ctx, executor, result)
	case ChainConditional:
		err = tc.executeConditional(ctx, executor, result)
	default:
		err = tc.executeSequential(ctx, executor, result)
	}

	completedAt := time.Now()
	result.CompletedAt = &completedAt

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
	} else if result.FailedSteps > 0 {
		result.Status = "partial"
	} else {
		result.Status = "completed"
	}

	return result, err
}

// executeSequential executes steps one after another
func (tc *ToolChain) executeSequential(ctx context.Context, executor ToolExecutor, result *ChainResult) error {
	for i := range tc.Steps {
		step := &tc.Steps[i]

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("chain execution cancelled: %w", err)
		}

		stepResult := tc.executeStepWithRetry(ctx, executor, step, result.Variables)
		result.StepResults = append(result.StepResults, stepResult)

		if stepResult.Status == "success" {
			result.PassedSteps++
		} else {
			result.FailedSteps++
			if !step.Optional {
				if step.OnFailure == "abort" || step.OnFailure == "" {
					return fmt.Errorf("step %s failed: %s", step.ID, stepResult.Error)
				}
				// If on_failure is "continue", proceed to next step
			}
		}
	}

	return nil
}

// executeParallel executes independent steps in parallel
func (tc *ToolChain) executeParallel(ctx context.Context, executor ToolExecutor, result *ChainResult) error {
	// Group steps by dependency level
	levels := tc.groupStepsByLevel()

	for _, levelSteps := range levels {
		var wg sync.WaitGroup
		errChan := make(chan error, len(levelSteps))
		resultsChan := make(chan StepResult, len(levelSteps))

		for i := range levelSteps {
			wg.Add(1)
			go func(step *ChainStep) {
				defer wg.Done()

				stepResult := tc.executeStepWithRetry(ctx, executor, step, result.Variables)
				resultsChan <- stepResult

				if stepResult.Status != "success" && !step.Optional {
					errChan <- fmt.Errorf("step %s failed: %s", step.ID, stepResult.Error)
				}
			}(&levelSteps[i])
		}

		wg.Wait()
		close(resultsChan)
		close(errChan)

		// Collect results
		for sr := range resultsChan {
			result.StepResults = append(result.StepResults, sr)
			if sr.Status == "success" {
				result.PassedSteps++
			} else {
				result.FailedSteps++
			}
		}

		// Check for errors
		for err := range errChan {
			return err
		}
	}

	return nil
}

// executeConditional executes steps based on conditions
func (tc *ToolChain) executeConditional(ctx context.Context, executor ToolExecutor, result *ChainResult) error {
	stepMap := make(map[string]*ChainStep)
	for i := range tc.Steps {
		stepMap[tc.Steps[i].ID] = &tc.Steps[i]
	}

	executed := make(map[string]bool)

	for i := range tc.Steps {
		step := &tc.Steps[i]

		if executed[step.ID] {
			continue
		}

		// Check dependencies
		depsSatisfied := true
		for _, depID := range step.DependsOn {
			if !executed[depID] {
				depsSatisfied = false
				break
			}
		}

		if !depsSatisfied {
			continue
		}

		// Evaluate condition if present
		if step.Condition != "" {
			if !tc.evaluateCondition(step.Condition, result) {
				// Skip this step
				executed[step.ID] = true
				result.StepResults = append(result.StepResults, StepResult{
					StepID:    step.ID,
					Status:    "skipped",
					StartedAt: time.Now(),
				})
				continue
			}
		}

		stepResult := tc.executeStepWithRetry(ctx, executor, step, result.Variables)
		result.StepResults = append(result.StepResults, stepResult)
		executed[step.ID] = true

		if stepResult.Status == "success" {
			result.PassedSteps++
		} else {
			result.FailedSteps++
			if !step.Optional {
				if step.OnFailure == "abort" || step.OnFailure == "" {
					return fmt.Errorf("step %s failed: %s", step.ID, stepResult.Error)
				}
			}
		}
	}

	return nil
}

// executeStepWithRetry executes a single step with retry logic
func (tc *ToolChain) executeStepWithRetry(ctx context.Context, executor ToolExecutor, step *ChainStep, variables map[string]string) StepResult {
	startTime := time.Now()
	result := StepResult{
		StepID:    step.ID,
		Status:    "pending",
		StartedAt: startTime,
	}

	maxRetries := step.RetryCount
	if maxRetries < 0 {
		maxRetries = 0
	}

	// Substitute variables in parameters
	params := tc.substituteVariables(step.Parameters, variables)

	// Set timeout
	timeout := step.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			result.RetryCount = attempt
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
		}

		stepCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		output, err := executor.Execute(stepCtx, step.Tool, params)

		completedAt := time.Now()
		result.CompletedAt = &completedAt
		result.Duration = completedAt.Sub(startTime)

		if err == nil {
			result.Status = "success"
			result.Output = output

			// Update variables with output if it's a map
			if outputMap, ok := output.(map[string]interface{}); ok {
				for k, v := range outputMap {
					if str, ok := v.(string); ok {
						variables[fmt.Sprintf("%s.%s", step.ID, k)] = str
					}
				}
			}

			return result
		}

		result.Error = err.Error()
		result.Status = "failed"
	}

	return result
}

// substituteVariables replaces {{variable}} placeholders in parameters
func (tc *ToolChain) substituteVariables(params map[string]interface{}, variables map[string]string) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range params {
		switch val := v.(type) {
		case string:
			result[k] = tc.substituteString(val, variables)
		case map[string]interface{}:
			result[k] = tc.substituteVariables(val, variables)
		default:
			result[k] = v
		}
	}

	return result
}

// substituteString replaces {{variable}} in a string
func (tc *ToolChain) substituteString(s string, variables map[string]string) string {
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		s = replaceAllString(s, placeholder, value)
	}
	return s
}

// evaluateCondition evaluates a condition string
func (tc *ToolChain) evaluateCondition(condition string, result *ChainResult) bool {
	// Simple condition evaluation
	// Supports: step_id.status == "success", step_id.status != "failed", etc.

	if condition == "always" {
		return true
	}

	if condition == "never" {
		return false
	}

	// Check for step status conditions
	for _, stepResult := range result.StepResults {
		if contains(condition, fmt.Sprintf("%s.status", stepResult.StepID)) {
			if contains(condition, "== \"success\"") || contains(condition, "== 'success'") {
				return stepResult.Status == "success"
			}
			if contains(condition, "!= \"failed\"") || contains(condition, "!= 'failed'") {
				return stepResult.Status != "failed"
			}
		}
	}

	return true // Default to true if we can't evaluate
}

// groupStepsByLevel groups steps by their dependency level for parallel execution
func (tc *ToolChain) groupStepsByLevel() [][]ChainStep {
	// Build dependency map
	deps := make(map[string][]string)
	for _, step := range tc.Steps {
		deps[step.ID] = step.DependsOn
	}

	// Calculate levels
	levels := make([][]ChainStep, 0)
	executed := make(map[string]bool)

	for len(executed) < len(tc.Steps) {
		level := make([]ChainStep, 0)

		for _, step := range tc.Steps {
			if executed[step.ID] {
				continue
			}

			// Check if all dependencies are executed
			allDepsDone := true
			for _, dep := range deps[step.ID] {
				if !executed[dep] {
					allDepsDone = false
					break
				}
			}

			if allDepsDone {
				level = append(level, step)
			}
		}

		if len(level) == 0 {
			break // Prevent infinite loop
		}

		for _, step := range level {
			executed[step.ID] = true
		}
		levels = append(levels, level)
	}

	return levels
}

// Helper functions
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

func replaceAllString(s, old, new string) string {
	result := ""
	start := 0
	for {
		i := 0
		for i < len(s)-start-len(old)+1 {
			if s[start+i:start+i+len(old)] == old {
				break
			}
			i++
		}
		if i >= len(s)-start-len(old)+1 {
			result += s[start:]
			break
		}
		result += s[start : start+i]
		result += new
		start += i + len(old)
	}
	return result
}

// Validate checks if the chain is valid
func (tc *ToolChain) Validate() error {
	if tc.ID == "" {
		return fmt.Errorf("chain ID is required")
	}
	if tc.Name == "" {
		return fmt.Errorf("chain name is required")
	}
	if len(tc.Steps) == 0 {
		return fmt.Errorf("chain must have at least one step")
	}

	// Check for duplicate step IDs
	stepIDs := make(map[string]bool)
	for _, step := range tc.Steps {
		if step.ID == "" {
			return fmt.Errorf("step ID is required")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
	}

	// Validate dependencies
	for _, step := range tc.Steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				return fmt.Errorf("step %s depends on non-existent step %s", step.ID, dep)
			}
		}
	}

	return nil
}

// Clone creates a deep copy of the chain
func (tc *ToolChain) Clone() *ToolChain {
	clone := &ToolChain{
		ID:          tc.ID,
		Name:        tc.Name,
		Description: tc.Description,
		Version:     tc.Version,
		Author:      tc.Author,
		Type:        tc.Type,
		IsBuiltIn:   tc.IsBuiltIn,
		IsShared:    tc.IsShared,
		Steps:       make([]ChainStep, len(tc.Steps)),
		Variables:   make(map[string]string),
	}

	for i, step := range tc.Steps {
		clone.Steps[i] = ChainStep{
			ID:          step.ID,
			Name:        step.Name,
			Description: step.Description,
			Tool:        step.Tool,
			Parameters:  make(map[string]interface{}),
			DependsOn:   make([]string, len(step.DependsOn)),
			Condition:   step.Condition,
			Timeout:     step.Timeout,
			RetryCount:  step.RetryCount,
			Optional:    step.Optional,
			OnFailure:   step.OnFailure,
		}

		for k, v := range step.Parameters {
			clone.Steps[i].Parameters[k] = v
		}
		copy(clone.Steps[i].DependsOn, step.DependsOn)
	}

	for k, v := range tc.Variables {
		clone.Variables[k] = v
	}

	return clone
}

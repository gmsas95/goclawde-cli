// Package runtime provides the skill runtime engine
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gmsas95/myrai-cli/internal/circuitbreaker"
	"github.com/gmsas95/myrai-cli/internal/jobs"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/mcp"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"go.uber.org/zap"
)

// Config holds engine configuration
type Config struct {
	// Skill paths to search
	SkillPaths []string `yaml:"skill_paths"`

	// MCP server configurations
	MCPServers []mcp.MCPServerConfig `yaml:"mcp_servers"`

	// Sandboxing settings
	SandboxEnabled bool `yaml:"sandbox_enabled"`

	// LLM client for skill execution
	LLMProvider string `yaml:"llm_provider,omitempty"`

	// Job scheduler configuration
	EnableScheduler bool `yaml:"enable_scheduler"`

	// Circuit breaker configuration
	CircuitBreaker circuitbreaker.Config `yaml:"circuit_breaker"`
}

// DefaultConfig returns default engine configuration
func DefaultConfig() *Config {
	return &Config{
		SkillPaths:      []string{"./skills", "~/.myrai/skills"},
		MCPServers:      []mcp.MCPServerConfig{},
		SandboxEnabled:  true,
		EnableScheduler: true,
		CircuitBreaker:  circuitbreaker.DefaultConfig(),
	}
}

// ExecutionContext holds context for skill execution
type ExecutionContext struct {
	SkillName  string
	ToolName   string
	Input      map[string]interface{}
	StartTime  time.Time
	Timeout    time.Duration
	Sandboxed  bool
	SandboxCfg SandboxConfig
}

// ExecutionResult represents the result of skill execution
type ExecutionResult struct {
	Success   bool                   `json:"success"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	StartTime time.Time              `json:"start_time"`
	Duration  time.Duration          `json:"duration"`
	Logs      []string               `json:"logs,omitempty"`
}

// DependencyResolver handles skill dependencies
type DependencyResolver struct {
	loader *Loader
	logger *zap.Logger
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(loader *Loader, logger *zap.Logger) *DependencyResolver {
	return &DependencyResolver{
		loader: loader,
		logger: logger,
	}
}

// Resolve resolves dependencies for a skill
func (r *DependencyResolver) Resolve(skill *Skill) ([]*Skill, error) {
	if len(skill.Manifest.Dependencies) == 0 {
		return nil, nil
	}

	resolved := make([]*Skill, 0, len(skill.Manifest.Dependencies))

	for _, depName := range skill.Manifest.Dependencies {
		dep := r.loader.GetSkill(depName)
		if dep == nil {
			return nil, fmt.Errorf("dependency not found: %s", depName)
		}

		if dep.Status != SkillStatusValidated {
			if err := ValidateManifest(dep); err != nil {
				return nil, fmt.Errorf("dependency validation failed for %s: %w", depName, err)
			}
		}

		resolved = append(resolved, dep)
	}

	r.logger.Debug("Dependencies resolved",
		zap.String("skill", skill.Manifest.Name),
		zap.Int("count", len(resolved)),
	)

	return resolved, nil
}

// Engine is the skill runtime engine
type Engine struct {
	config     *Config
	loader     *Loader
	registry   *skills.Registry
	mcpManager *mcp.Manager
	scheduler  *jobs.Scheduler
	cbManager  *circuitbreaker.Manager
	llmClient  *llm.Client
	logger     *zap.Logger
	mu         sync.RWMutex
	running    bool

	// Dependency resolution
	resolver *DependencyResolver

	// Sandboxing
	sandboxEnabled bool
}

// NewEngine creates a new skill runtime engine
func NewEngine(cfg *Config, logger *zap.Logger) *Engine {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	loader := NewLoader(logger)
	for _, path := range cfg.SkillPaths {
		loader.AddPath(path)
	}

	engine := &Engine{
		config:         cfg,
		loader:         loader,
		logger:         logger,
		mcpManager:     mcp.NewManager(logger),
		cbManager:      circuitbreaker.NewDefaultManager(logger),
		sandboxEnabled: cfg.SandboxEnabled,
		resolver:       NewDependencyResolver(loader, logger),
	}

	// Initialize scheduler if enabled
	if cfg.EnableScheduler {
		engine.scheduler = jobs.NewScheduler(logger)
	}

	return engine
}

// SetRegistry sets the skills registry
func (e *Engine) SetRegistry(registry *skills.Registry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.registry = registry
}

// SetLLMClient sets the LLM client
func (e *Engine) SetLLMClient(client *llm.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.llmClient = client
}

// Start initializes and starts the engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	e.logger.Info("Starting skill runtime engine")

	// Scan for skills
	if err := e.loader.ScanDirectories(); err != nil {
		e.logger.Error("Failed to scan skill directories", zap.Error(err))
	}

	// Register MCP servers
	for _, serverCfg := range e.config.MCPServers {
		if err := e.mcpManager.RegisterClient(serverCfg); err != nil {
			e.logger.Error("Failed to register MCP server",
				zap.String("name", serverCfg.Name),
				zap.Error(err),
			)
			continue
		}

		// Connect to MCP server
		if err := e.mcpManager.ConnectClient(ctx, serverCfg.Name); err != nil {
			e.logger.Error("Failed to connect to MCP server",
				zap.String("name", serverCfg.Name),
				zap.Error(err),
			)
		}
	}

	// Start scheduler
	if e.scheduler != nil {
		e.scheduler.Start()
	}

	e.running = true
	e.logger.Info("Skill runtime engine started",
		zap.Int("skills_loaded", len(e.loader.ListSkills())),
		zap.Int("mcp_clients", len(e.mcpManager.ListClients())),
	)

	return nil
}

// Stop shuts down the engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.logger.Info("Stopping skill runtime engine")

	// Stop scheduler
	if e.scheduler != nil {
		e.scheduler.Stop()
	}

	// Disconnect all MCP clients
	e.mcpManager.DisconnectAll()

	e.running = false
	e.logger.Info("Skill runtime engine stopped")

	return nil
}

// IsRunning returns true if the engine is running
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// LoadSkill loads a skill from the given path
func (e *Engine) LoadSkill(path string) (*Skill, error) {
	skill, err := e.loader.LoadSkill(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load skill: %w", err)
	}

	// Validate manifest
	if err := ValidateManifest(skill); err != nil {
		return nil, fmt.Errorf("skill validation failed: %w", err)
	}

	// Resolve dependencies
	if _, err := e.resolver.Resolve(skill); err != nil {
		e.logger.Warn("Dependency resolution failed",
			zap.String("skill", skill.Manifest.Name),
			zap.Error(err),
		)
	}

	// Register with registry if available
	if e.registry != nil {
		skillHandler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			result, err := e.ExecuteSkill(skill.Manifest.Name, ctx, args)
			return result, err
		}

		registrySkill := skill.ToRegistrySkill(skillHandler)
		if err := e.registry.Register(registrySkill); err != nil {
			e.logger.Warn("Failed to register skill with registry",
				zap.String("skill", skill.Manifest.Name),
				zap.Error(err),
			)
		}
	}

	return skill, nil
}

// ExecuteSkill executes a skill with the given input
func (e *Engine) ExecuteSkill(name string, ctx context.Context, input map[string]interface{}) (*ExecutionResult, error) {
	start := time.Now()

	result := &ExecutionResult{
		Success:   false,
		StartTime: start,
		Logs:      []string{},
	}

	// Get skill
	skill := e.loader.GetSkill(name)
	if skill == nil {
		result.Error = fmt.Sprintf("skill not found: %s", name)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Check status
	if skill.Status != SkillStatusValidated {
		result.Error = fmt.Sprintf("skill not validated: %s", name)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("%s", result.Error)
	}

	e.logger.Info("Executing skill",
		zap.String("skill", name),
		zap.Int("input_keys", len(input)),
	)

	// Prepare execution context
	execCtx := &ExecutionContext{
		SkillName:  name,
		Input:      input,
		StartTime:  start,
		Timeout:    30 * time.Second,
		Sandboxed:  e.sandboxEnabled && skill.Manifest.Sandbox.Enabled,
		SandboxCfg: skill.Manifest.Sandbox,
	}

	// Execute with timeout
	execCtxWithTimeout, cancel := context.WithTimeout(ctx, execCtx.Timeout)
	defer cancel()

	// Determine execution method
	var output map[string]interface{}
	var execErr error

	if skill.Manifest.EntryPoint != "" {
		// Execute via entry point
		output, execErr = e.executeEntryPoint(execCtxWithTimeout, skill, execCtx)
	} else {
		// Execute as inline skill
		output, execErr = e.executeInline(execCtxWithTimeout, skill, execCtx)
	}

	result.Duration = time.Since(start)

	if execErr != nil {
		result.Error = execErr.Error()
		e.logger.Error("Skill execution failed",
			zap.String("skill", name),
			zap.Error(execErr),
			zap.Duration("duration", result.Duration),
		)
	} else {
		result.Success = true
		result.Output = output
		e.logger.Info("Skill execution completed",
			zap.String("skill", name),
			zap.Duration("duration", result.Duration),
		)
	}

	return result, execErr
}

// executeEntryPoint executes a skill via its entry point
func (e *Engine) executeEntryPoint(ctx context.Context, skill *Skill, execCtx *ExecutionContext) (map[string]interface{}, error) {
	entryPoint := skill.Manifest.EntryPoint

	// Check if it's a command
	if strings.HasPrefix(entryPoint, "$") {
		// Shell command
		cmd := strings.TrimPrefix(entryPoint, "$")
		return e.executeSandboxed(ctx, skill, execCtx, cmd)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(entryPoint))
	skillDir := filepath.Dir(skill.Path)
	fullPath := filepath.Join(skillDir, entryPoint)

	// Verify file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("entry point not found: %s", fullPath)
	}

	switch ext {
	case ".js", ".ts":
		return e.executeSandboxed(ctx, skill, execCtx, "node", fullPath)
	case ".py":
		return e.executeSandboxed(ctx, skill, execCtx, "python3", fullPath)
	case ".go":
		return e.executeSandboxed(ctx, skill, execCtx, "go", "run", fullPath)
	case ".sh":
		return e.executeSandboxed(ctx, skill, execCtx, "bash", fullPath)
	default:
		return nil, fmt.Errorf("unsupported entry point type: %s", ext)
	}
}

// executeInline executes a skill inline
func (e *Engine) executeInline(ctx context.Context, skill *Skill, execCtx *ExecutionContext) (map[string]interface{}, error) {
	// For inline skills, use LLM to process
	if e.llmClient == nil {
		return nil, fmt.Errorf("LLM client not available for inline skill execution")
	}

	// Build prompt from skill content
	prompt := fmt.Sprintf("You are executing the skill '%s'.\n\n", skill.Manifest.Name)
	prompt += fmt.Sprintf("Description: %s\n\n", skill.Manifest.Description)
	prompt += fmt.Sprintf("Skill Content:\n%s\n\n", skill.Content)
	prompt += fmt.Sprintf("Input: %s\n\n", mustMarshalJSON(execCtx.Input))
	prompt += "Please process this input according to the skill instructions and return a JSON response."

	response, err := e.llmClient.SimpleChat(ctx, "", prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM execution failed: %w", err)
	}

	// Try to parse as JSON
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		// Return as text output
		output = map[string]interface{}{
			"text": response,
		}
	}

	return output, nil
}

// executeSandboxed executes a command in sandbox mode
func (e *Engine) executeSandboxed(ctx context.Context, skill *Skill, execCtx *ExecutionContext, command string, args ...string) (map[string]interface{}, error) {
	if !execCtx.Sandboxed {
		// Run without sandbox
		cmd := exec.CommandContext(ctx, command, args...)

		// Set working directory to skill directory
		skillDir := filepath.Dir(skill.Path)
		cmd.Dir = skillDir

		// Pass input as environment variable
		inputJSON, _ := json.Marshal(execCtx.Input)
		cmd.Env = append(os.Environ(), fmt.Sprintf("SKILL_INPUT=%s", string(inputJSON)))

		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("execution failed: %w, output: %s", err, string(output))
		}

		// Try to parse output as JSON
		var result map[string]interface{}
		if err := json.Unmarshal(output, &result); err != nil {
			result = map[string]interface{}{
				"output": string(output),
			}
		}

		return result, nil
	}

	// Sandboxed execution
	return e.executeInSandbox(ctx, skill, execCtx, command, args...)
}

// executeInSandbox runs a command in a restricted sandbox environment
func (e *Engine) executeInSandbox(ctx context.Context, skill *Skill, execCtx *ExecutionContext, command string, args ...string) (map[string]interface{}, error) {
	// Build sandbox command based on platform
	// For now, use a basic chroot-like approach with restrictions

	cmd := exec.CommandContext(ctx, command, args...)

	// Set working directory to skill directory
	skillDir := filepath.Dir(skill.Path)
	cmd.Dir = skillDir

	// Prepare restricted environment
	env := []string{
		fmt.Sprintf("SKILL_NAME=%s", skill.Manifest.Name),
		fmt.Sprintf("SKILL_VERSION=%s", skill.Manifest.Version),
	}

	// Add input as environment variable
	inputJSON, _ := json.Marshal(execCtx.Input)
	env = append(env, fmt.Sprintf("SKILL_INPUT=%s", string(inputJSON)))

	// Add allowed environment variables
	if execCtx.SandboxCfg.AllowFS {
		env = append(env, "SKILL_ALLOW_FS=1")
	}
	if execCtx.SandboxCfg.AllowNet {
		env = append(env, "SKILL_ALLOW_NET=1")
	}

	cmd.Env = env

	// Restrict file system access
	if !execCtx.SandboxCfg.AllowFS {
		// For Unix systems, we could use chroot or namespaces
		// For now, just restrict to skill directory
		cmd.Dir = skillDir
	}

	// Execute
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sandboxed execution failed: %w, output: %s", err, string(output))
	}

	// Parse output
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		result = map[string]interface{}{
			"output": string(output),
		}
	}

	return result, nil
}

// ListSkills returns all loaded skills
func (e *Engine) ListSkills() []*Skill {
	return e.loader.ListSkills()
}

// GetSkill returns a skill by name
func (e *Engine) GetSkill(name string) *Skill {
	return e.loader.GetSkill(name)
}

// RegisterMCPClient registers an MCP client
func (e *Engine) RegisterMCPClient(config mcp.MCPServerConfig) error {
	return e.mcpManager.RegisterClient(config)
}

// GetMCPClient returns an MCP client by name
func (e *Engine) GetMCPClient(name string) (*mcp.Client, bool) {
	return e.mcpManager.GetClient(name)
}

// ScheduleSkill schedules a skill for periodic execution
func (e *Engine) ScheduleSkill(name string, schedule string, input map[string]interface{}) error {
	if e.scheduler == nil {
		return fmt.Errorf("scheduler not enabled")
	}

	job := &jobs.Job{
		ID:       fmt.Sprintf("skill-%s-%d", name, time.Now().Unix()),
		Name:     fmt.Sprintf("Execute skill: %s", name),
		Schedule: schedule,
		Enabled:  true,
		Func: func(ctx context.Context) error {
			_, err := e.ExecuteSkill(name, ctx, input)
			return err
		},
	}

	return e.scheduler.RegisterJob(job)
}

// mustMarshalJSON marshals v to JSON, returning empty string on error
func mustMarshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

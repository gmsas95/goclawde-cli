package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// ChainExecutor manages the execution of tool chains
type ChainExecutor struct {
	chainsDir    string
	chains       map[string]*ToolChain
	toolExecutor ToolExecutor
	store        *store.Store
}

// ChainExecution represents a recorded chain execution
type ChainExecution struct {
	ID           string            `json:"id"`
	ChainID      string            `json:"chain_id"`
	Status       string            `json:"status"`
	InputParams  map[string]string `json:"input_params"`
	Results      *ChainResult      `json:"results"`
	ErrorMessage string            `json:"error_message,omitempty"`
	StartedAt    time.Time         `json:"started_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// NewChainExecutor creates a new chain executor
func NewChainExecutor(cfg *config.Config, store *store.Store, toolExecutor ToolExecutor) (*ChainExecutor, error) {
	chainsDir := filepath.Join(cfg.Storage.DataDir, "chains")
	if err := os.MkdirAll(chainsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chains directory: %w", err)
	}

	executor := &ChainExecutor{
		chainsDir:    chainsDir,
		chains:       make(map[string]*ToolChain),
		toolExecutor: toolExecutor,
		store:        store,
	}

	// Load built-in chains
	if err := executor.LoadBuiltInChains(); err != nil {
		return nil, fmt.Errorf("failed to load built-in chains: %w", err)
	}

	// Load user chains
	if err := executor.LoadUserChains(); err != nil {
		return nil, fmt.Errorf("failed to load user chains: %w", err)
	}

	return executor, nil
}

// LoadBuiltInChains loads pre-built chains from the embedded chains directory
func (ce *ChainExecutor) LoadBuiltInChains() error {
	builtInChainsPath := filepath.Join("internal", "tools", "chains")

	// Check if directory exists
	if _, err := os.Stat(builtInChainsPath); os.IsNotExist(err) {
		// Create the directory and default chains
		if err := ce.createDefaultChains(builtInChainsPath); err != nil {
			return err
		}
	}

	return ce.loadChainsFromDir(builtInChainsPath, true)
}

// LoadUserChains loads user-created chains
func (ce *ChainExecutor) LoadUserChains() error {
	return ce.loadChainsFromDir(ce.chainsDir, false)
}

// loadChainsFromDir loads all chains from a directory
func (ce *ChainExecutor) loadChainsFromDir(dir string, isBuiltIn bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		chainPath := filepath.Join(dir, name)
		data, err := os.ReadFile(chainPath)
		if err != nil {
			continue // Skip files that can't be read
		}

		chain, err := ce.ParseChain(data)
		if err != nil {
			continue // Skip invalid chains
		}

		chain.IsBuiltIn = isBuiltIn
		if chain.ID == "" {
			chain.ID = strings.TrimSuffix(name, filepath.Ext(name))
		}

		ce.chains[chain.ID] = chain
	}

	return nil
}

// createDefaultChains creates default built-in chains
func (ce *ChainExecutor) createDefaultChains(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Deploy Application Chain
	deployChain := `name: deploy-application
description: Build, push, and deploy application
version: 1.0.0
author: myrai-team
type: sequential

steps:
  - id: verify-dockerfile
    name: Verify Dockerfile
    description: Check if Dockerfile exists
    tool: read_file
    parameters:
      path: "./Dockerfile"
    on_failure: abort
    
  - id: build-image
    name: Build Docker Image
    description: Build the Docker image
    tool: exec
    parameters:
      command: "docker build -t {{image_tag}} ."
    depends_on: [verify-dockerfile]
    timeout: 300s
    retry_count: 1
    
  - id: verify-registry
    name: Verify Registry Login
    description: Check Docker Hub login status
    tool: exec
    parameters:
      command: "docker info | grep Username"
    depends_on: [build-image]
    optional: true
    
  - id: push-image
    name: Push to Registry
    description: Push image to Docker registry
    tool: exec
    parameters:
      command: "docker push {{image_tag}}"
    depends_on: [build-image, verify-registry]
    timeout: 180s
    retry_count: 2
    
  - id: deploy-k8s
    name: Deploy to Kubernetes
    description: Apply Kubernetes manifests
    tool: exec
    parameters:
      command: "kubectl apply -f k8s/"
    depends_on: [push-image]
    timeout: 120s
    
  - id: verify-deployment
    name: Verify Deployment
    description: Check deployment status
    tool: exec
    parameters:
      command: "kubectl rollout status deployment/{{app_name}}"
    depends_on: [deploy-k8s]
    optional: true
    timeout: 180s

variables:
  image_tag: "myapp:latest"
  app_name: "myapp"
`

	// Setup CI/CD Chain
	cicdChain := `name: setup-ci-cd
description: Setup CI/CD pipeline configuration
version: 1.0.0
author: myrai-team
type: sequential

steps:
  - id: analyze-project
    name: Analyze Project
    description: Analyze project structure to determine CI/CD needs
    tool: list_dir
    parameters:
      path: "."
      recursive: false
    
  - id: detect-language
    name: Detect Language
    description: Detect primary programming language
    tool: exec
    parameters:
      command: "find . -maxdepth 1 -name '*.go' -o -name '*.py' -o -name '*.js' -o -name '*.ts' -o -name 'package.json' -o -name 'go.mod' | head -5"
    depends_on: [analyze-project]
    
  - id: create-github-workflow
    name: Create GitHub Workflow
    description: Create GitHub Actions workflow
    tool: write_file
    parameters:
      path: ".github/workflows/ci.yml"
      content: |
        name: CI
        on: [push, pull_request]
        jobs:
          build:
            runs-on: ubuntu-latest
            steps:
              - uses: actions/checkout@v3
              - name: Build
                run: echo "Build steps here"
    depends_on: [detect-language]
    
  - id: verify-workflow
    name: Verify Workflow
    description: Verify workflow file syntax
    tool: exec
    parameters:
      command: "cat .github/workflows/ci.yml"
    depends_on: [create-github-workflow]
    optional: true

variables:
  workflow_type: "github"
`

	// Database Migration Chain
	migrationChain := `name: database-migration
description: Run database migrations safely
version: 1.0.0
author: myrai-team
type: sequential

steps:
  - id: backup-database
    name: Backup Database
    description: Create database backup before migration
    tool: exec
    parameters:
      command: "pg_dump {{database_url}} > backup_{{timestamp}}.sql"
    timeout: 300s
    on_failure: abort
    
  - id: check-pending
    name: Check Pending Migrations
    description: Check for pending migrations
    tool: exec
    parameters:
      command: "{{migration_tool}} status"
    depends_on: [backup-database]
    
  - id: run-migrations
    name: Run Migrations
    description: Execute pending migrations
    tool: exec
    parameters:
      command: "{{migration_tool}} up"
    depends_on: [check-pending]
    timeout: 300s
    retry_count: 1
    
  - id: verify-migrations
    name: Verify Migrations
    description: Verify migration success
    tool: exec
    parameters:
      command: "{{migration_tool}} version"
    depends_on: [run-migrations]
    optional: true

variables:
  database_url: "postgres://localhost:5432/mydb"
  migration_tool: "migrate"
  timestamp: "backup"
`

	// Code Review Chain
	reviewChain := `name: code-review
description: Automated code review workflow
version: 1.0.0
author: myrai-team
type: parallel

steps:
  - id: lint
    name: Run Linter
    description: Run code linting
    tool: exec
    parameters:
      command: "{{lint_command}}"
    timeout: 120s
    optional: true
    
  - id: test
    name: Run Tests
    description: Execute test suite
    tool: exec
    parameters:
      command: "{{test_command}}"
    timeout: 300s
    optional: true
    
  - id: security-scan
    name: Security Scan
    description: Run security vulnerability scan
    tool: exec
    parameters:
      command: "{{security_command}}"
    timeout: 180s
    optional: true
    
  - id: analyze-changes
    name: Analyze Changes
    description: Analyze git changes
    tool: exec
    parameters:
      command: "git diff --stat HEAD~1"
    timeout: 30s

variables:
  lint_command: "echo 'No linter configured'"
  test_command: "echo 'No tests configured'"
  security_command: "echo 'No security scanner configured'"
`

	chains := map[string]string{
		"deploy-application.yaml": deployChain,
		"setup-ci-cd.yaml":        cicdChain,
		"database-migration.yaml": migrationChain,
		"code-review.yaml":        reviewChain,
	}

	for filename, content := range chains {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// ParseChain parses a YAML chain definition
func (ce *ChainExecutor) ParseChain(data []byte) (*ToolChain, error) {
	var chain ToolChain
	if err := yaml.Unmarshal(data, &chain); err != nil {
		return nil, fmt.Errorf("failed to parse chain YAML: %w", err)
	}

	// Set defaults
	if chain.Type == "" {
		chain.Type = ChainSequential
	}
	if chain.Version == "" {
		chain.Version = "1.0.0"
	}

	// Parse timeout strings
	for i := range chain.Steps {
		if chain.Steps[i].Timeout == 0 {
			chain.Steps[i].Timeout = 60 * time.Second
		}
	}

	if err := chain.Validate(); err != nil {
		return nil, err
	}

	return &chain, nil
}

// GetChain retrieves a chain by ID
func (ce *ChainExecutor) GetChain(id string) (*ToolChain, bool) {
	chain, ok := ce.chains[id]
	if !ok {
		return nil, false
	}
	return chain.Clone(), true
}

// ListChains returns all available chains
func (ce *ChainExecutor) ListChains() []*ToolChain {
	chains := make([]*ToolChain, 0, len(ce.chains))
	for _, chain := range ce.chains {
		chains = append(chains, chain.Clone())
	}
	return chains
}

// ExecuteChain executes a chain with the given variables
func (ce *ChainExecutor) ExecuteChain(ctx context.Context, chainID string, variables map[string]string) (*ChainResult, error) {
	chain, ok := ce.GetChain(chainID)
	if !ok {
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}

	// Merge variables
	if variables != nil {
		for k, v := range variables {
			chain.Variables[k] = v
		}
	}

	// Add timestamp variable
	chain.Variables["timestamp"] = time.Now().Format("20060102_150405")

	// Record execution start
	execution := &ChainExecution{
		ID:          uuid.New().String(),
		ChainID:     chainID,
		Status:      "running",
		InputParams: variables,
		StartedAt:   time.Now(),
	}

	// Execute chain
	result, err := chain.Execute(ctx, ce.toolExecutor)
	execution.Results = result

	completedAt := time.Now()
	execution.CompletedAt = &completedAt

	if err != nil {
		execution.Status = "failed"
		execution.ErrorMessage = err.Error()
	} else {
		execution.Status = result.Status
	}

	// Save execution record if store is available
	if ce.store != nil {
		ce.saveExecution(execution)
	}

	return result, err
}

// saveExecution saves an execution record to the store
func (ce *ChainExecutor) saveExecution(execution *ChainExecution) error {
	// This would save to the database
	// For now, just log it
	return nil
}

// CreateChain creates a new chain from a YAML definition
func (ce *ChainExecutor) CreateChain(name string, definition []byte) (*ToolChain, error) {
	chain, err := ce.ParseChain(definition)
	if err != nil {
		return nil, err
	}

	chain.ID = name
	chain.IsBuiltIn = false

	// Save to user chains directory
	chainPath := filepath.Join(ce.chainsDir, name+".yaml")
	if err := os.WriteFile(chainPath, definition, 0644); err != nil {
		return nil, fmt.Errorf("failed to save chain: %w", err)
	}

	ce.chains[name] = chain

	return chain, nil
}

// UpdateChain updates an existing chain
func (ce *ChainExecutor) UpdateChain(name string, definition []byte) (*ToolChain, error) {
	chain, err := ce.ParseChain(definition)
	if err != nil {
		return nil, err
	}

	chain.ID = name

	// Don't allow updating built-in chains
	if existing, ok := ce.chains[name]; ok && existing.IsBuiltIn {
		return nil, fmt.Errorf("cannot modify built-in chain: %s", name)
	}

	chain.IsBuiltIn = false

	// Save to user chains directory
	chainPath := filepath.Join(ce.chainsDir, name+".yaml")
	if err := os.WriteFile(chainPath, definition, 0644); err != nil {
		return nil, fmt.Errorf("failed to save chain: %w", err)
	}

	ce.chains[name] = chain

	return chain, nil
}

// DeleteChain deletes a user chain
func (ce *ChainExecutor) DeleteChain(name string) error {
	chain, ok := ce.chains[name]
	if !ok {
		return fmt.Errorf("chain not found: %s", name)
	}

	if chain.IsBuiltIn {
		return fmt.Errorf("cannot delete built-in chain: %s", name)
	}

	chainPath := filepath.Join(ce.chainsDir, name+".yaml")
	if err := os.Remove(chainPath); err != nil {
		return fmt.Errorf("failed to delete chain file: %w", err)
	}

	delete(ce.chains, name)
	return nil
}

// GetChainPath returns the file path for a chain
func (ce *ChainExecutor) GetChainPath(name string) (string, error) {
	chain, ok := ce.chains[name]
	if !ok {
		return "", fmt.Errorf("chain not found: %s", name)
	}

	if chain.IsBuiltIn {
		return filepath.Join("internal", "tools", "chains", name+".yaml"), nil
	}

	return filepath.Join(ce.chainsDir, name+".yaml"), nil
}

// substituteVariables replaces {{variable}} placeholders in a string
func substituteVariables(s string, variables map[string]string) string {
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		s = strings.ReplaceAll(s, placeholder, value)
	}
	return s
}

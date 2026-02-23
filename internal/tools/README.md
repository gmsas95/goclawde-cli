# Phase 4: Tool Orchestration

This package implements Phase 4 of the Myrai 2.0 implementation plan, providing intelligent task decomposition and execution capabilities.

## Components

### 1. Intent Classification (`intent.go`)
- **IntentClassifier**: Uses LLM to classify user intents
- **Intent Types**: command, question, task, conversation
- **Categories**: devops, coding, research, general
- **Complexity Levels**: simple, compound, complex
- Fallback to rule-based classification when LLM unavailable

### 2. Task Decomposition (`decomposer.go`)
- **TaskDecomposer**: Breaks complex intents into executable steps
- **Task Structure**: ID, description, dependencies, tools, estimated time
- Dependency resolution and execution ordering
- LLM-based and rule-based decomposition strategies

### 3. Tool Chain Builder (`chain.go`)
- **Chain Types**: Sequential, Parallel, Conditional
- **ToolChain**: Reusable workflow definitions
- **ChainStep**: Individual steps with conditions, timeouts, retries
- Variable substitution support (`{{variable_name}}`)
- Error handling with abort/continue on failure

### 4. Chain Execution Engine (`executor.go`)
- Load chains from YAML files
- Execute chains with context support
- Progress tracking and result recording
- User chain management (create, edit, delete)

### 5. Tool Inventory (`inventory.go`)
- Categorized registry of 50+ built-in tools
- Categories: System, File, Browser, DevOps, Git, API, Database, Cloud, Communication, Cron, Memory, MCP
- Dynamic tool registration for MCP tools

## Pre-built Chains

### deploy-application.yaml
Build, push, and deploy a containerized application:
1. Verify Dockerfile exists
2. Build Docker image
3. Verify registry login
4. Push image to registry
5. Deploy to Kubernetes
6. Verify deployment status

### setup-ci-cd.yaml
Setup CI/CD pipeline configuration:
1. Analyze project structure
2. Detect programming language
3. Create GitHub Actions workflow
4. Verify workflow configuration

### database-migration.yaml
Run database migrations safely:
1. Backup database
2. Check pending migrations
3. Run migrations
4. Verify migration success

### code-review.yaml
Automated code review (parallel execution):
1. Run linter
2. Execute tests
3. Security vulnerability scan
4. Analyze git changes

## CLI Commands

```bash
# List available chains
myrai chain list

# Execute a chain with variables
myrai chain run deploy-application --var image_tag=myapp:v1.0 --var app_name=myapp

# Create a new chain interactively
myrai chain create my-custom-chain

# Edit a chain
myrai chain edit my-custom-chain

# Show chain details
myrai chain show deploy-application

# Delete a user chain
myrai chain delete my-custom-chain

# Validate chain syntax
myrai chain validate deploy-application

# Publish chain to marketplace (Phase 6)
myrai chain publish my-custom-chain
```

## Tool Inventory Commands

```bash
# List all tools by category
myrai tools list

# Show tools in a category
myrai tools category DevOps

# Search tools
myrai tools search docker

# Get tool information
myrai tools info docker_build
```

## Intent Classification Commands

```bash
# Classify user intent
myrai intent "Deploy my application to Kubernetes"
```

## Integration with Skills System

The orchestration system integrates with the Phase 2 Skills system:
- Tools from skills are available in the inventory
- Chains can use tools from any skill
- Task decomposition considers available skills

## Usage Example

```go
// Create orchestrator
orchestrator, err := NewOrchestratorIntegration(cfg, store, llmClient, toolExecutor)

// Classify intent
intent, err := orchestrator.Classifier.Classify(ctx, "Deploy my app")

// Decompose into tasks
if intent.NeedsDecomposition() {
    tasks, err := orchestrator.Decomposer.Decompose(ctx, intent, availableTools)
}

// Execute a chain
result, err := orchestrator.ChainExec.ExecuteChain(ctx, "deploy-application", variables)
```

## Database Schema

```sql
CREATE TABLE tool_chains (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    version TEXT,
    author TEXT,
    type TEXT NOT NULL,
    definition JSON NOT NULL,
    is_builtin BOOLEAN DEFAULT false,
    is_shared BOOLEAN DEFAULT false,
    install_count INTEGER DEFAULT 0,
    rating FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chain_executions (
    id TEXT PRIMARY KEY,
    chain_id TEXT REFERENCES tool_chains(id),
    status TEXT,
    input_params JSON,
    results JSON,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);
```

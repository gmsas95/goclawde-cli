# Myrai 2.0 Implementation Plan

## Executive Summary

Myrai 2.0 is a comprehensive upgrade to transform Myrai into an adaptive, autonomous AI assistant with dynamic memory management, extensive tool capabilities, and community-driven marketplace.

**Key Innovation**: Neural Clusters - semantic memory compression inspired by human brain neural networks, enabling efficient context management at scale.

**Status**: Implementation Ready  
**Timeline**: 12 weeks  
**Target**: Production-ready autonomous AI assistant

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Phase 0: Foundation](#phase-0-foundation)
3. [Phase 1: Neural Clusters](#phase-1-neural-clusters)
4. [Phase 2: Skill Runtime + MCP](#phase-2-skill-runtime--mcp)
5. [Phase 3: Adaptive Persona](#phase-3-adaptive-persona)
6. [Phase 4: Tool Orchestration](#phase-4-tool-orchestration)
7. [Phase 5: Reflection Engine](#phase-5-reflection-engine)
8. [Phase 6: Marketplace](#phase-6-marketplace)
9. [API Specifications](#api-specifications)
10. [Database Schema](#database-schema)
11. [Configuration](#configuration)

---

## Architecture Overview

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Myrai 2.0 Architecture                       │
└─────────────────────────────────────────────────────────────────────┘

┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Telegram   │     │  WhatsApp    │     │   Discord    │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
       └────────────────────┼────────────────────┘
                            │
       ┌────────────────────▼────────────────────┐
       │              Gateway Layer               │
       │  (WebSocket API / REST API / CLI)       │
       └────────────────────┬────────────────────┘
                            │
       ┌────────────────────▼────────────────────┐
       │              Agent Core                  │
       │  ┌──────────────┐  ┌──────────────┐    │
       │  │   Neural     │  │   Persona    │    │
       │  │   Clusters   │  │   Manager    │    │
       │  └──────────────┘  └──────────────┘    │
       │  ┌──────────────┐  ┌──────────────┐    │
       │  │   Tool       │  │   Reflection │    │
       │  │Orchestrator  │  │   Engine     │    │
       │  └──────────────┘  └──────────────┘    │
       └────────────────────┬────────────────────┘
                            │
       ┌────────────────────▼────────────────────┐
       │              Skill Layer                 │
       │  ┌──────────┐ ┌──────────┐ ┌──────────┐ │
       │  │ Built-in │ │  Runtime │ │   MCP    │ │
       │  │  Skills  │ │  Skills  │ │  Tools   │ │
       │  └──────────┘ └──────────┘ └──────────┘ │
       └─────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                     Background Processes                            │
│  • Neural Cluster Formation (Weekly)                                │
│  • Persona Evolution Analysis (Weekly)                              │
│  • Memory Reflection Audit (Weekly)                                 │
│  • Skill Hot-Reload (Continuous)                                    │
└─────────────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. Neural Clusters System
**Purpose**: Semantic memory compression for efficient context retrieval

**How it Works**:
1. Raw memories stored in database
2. Weekly background job analyzes memory patterns
3. Related memories clustered by semantic similarity
4. AI generates "essence" (compressed summary) for each cluster
5. Clusters used for context retrieval instead of raw memories

**Benefits**:
- 10x more efficient context loading
- Scales to millions of memories
- Human-readable themes
- Enables long-term memory retention

#### 2. Skill Runtime
**Purpose**: Dynamic tool loading without restart

**Components**:
- **Built-in Skills**: Compiled Go code (fastest)
- **Runtime Skills**: Hot-loaded from GitHub or local files
- **MCP Tools**: External Model Context Protocol servers

**Hot-Reload**: File watcher detects changes, reloads skills instantly

#### 3. Adaptive Persona
**Purpose**: AI personality evolves based on user patterns

**Evolution Triggers**:
- Frequency analysis (topic mentions)
- Temporal patterns (time-based behaviors)
- Skill usage patterns
- Context switching behaviors

**User Control**: All changes proposed, user approves/rejects/customizes

#### 4. Tool Orchestrator
**Purpose**: Intelligent task breakdown and execution

**Capabilities**:
- Intent classification
- Task decomposition
- Tool chain building (sequential/parallel)
- Error recovery and retries

#### 5. Reflection Engine
**Purpose**: Self-auditing memory system

**Functions**:
- Contradiction detection
- Redundancy identification
- Gap analysis
- Health scoring
- Optimization suggestions

---

## Phase 0: Foundation

**Duration**: Week 0  
**Goal**: Establish infrastructure for MCP and hot-reload

### 0.1 MCP Client Implementation

**Location**: `internal/mcp/`

**Components**:
```go
// client.go
package mcp

type Client struct {
    serverConfig MCPServerConfig
    transport    Transport
    tools        []Tool
}

type MCPServerConfig struct {
    Name    string
    Command string
    Args    []string
    Env     map[string]string
}

func (c *Client) Connect() error
func (c *Client) ListTools() ([]Tool, error)
func (c *Client) CallTool(name string, args map[string]interface{}) (interface{}, error)
```

**Transport Types**:
- **stdio**: Communicate via stdin/stdout (most common)
- **sse**: Server-Sent Events for remote servers

**Configuration** (`config/mcp.yaml`):
```yaml
mcp:
  servers:
    - name: filesystem
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user"]
      enabled: true
      
    - name: github
      command: docker
      args: ["run", "-i", "--rm", "-e", "GITHUB_TOKEN", "mcp/github"]
      env:
        GITHUB_TOKEN: "${GITHUB_TOKEN}"
      enabled: true
      
    - name: playwright
      command: npx
      args: ["-y", "@modelcontextprotocol/server-playwright"]
      enabled: false
```

### 0.2 Skill Runtime Infrastructure

**Location**: `internal/skills/runtime/`

**Components**:
```go
// loader.go
package runtime

type SkillLoader struct {
    registry *Registry
    watchers map[string]*fsnotify.Watcher
}

func (sl *SkillLoader) LoadFromGitHub(repo string) error
func (sl *SkillLoader) LoadFromLocal(path string) error
func (sl *SkillLoader) Watch(path string) error
func (sl *SkillLoader) HotReload(skillName string) error
```

### 0.3 Docker Integration for MCP

**Purpose**: Run MCP servers in isolated containers

**Implementation**:
```go
// internal/mcp/docker.go
func StartMCPServerInDocker(config MCPServerConfig) (*DockerContainer, error) {
    // Pull image if needed
    // Create container with isolated filesystem
    // Start container
    // Return container handle for communication
}
```

**Security**:
- Each MCP server runs in separate container
- Read-only filesystem where possible
- Resource limits (CPU, memory)
- Network isolation

---

## Phase 1: Neural Clusters

**Duration**: Week 1-2  
**Goal**: Implement semantic memory compression system

### 1.1 Database Schema

**New Tables**:

```sql
-- Neural Clusters: Compressed memory themes
CREATE TABLE neural_clusters (
    id TEXT PRIMARY KEY,
    theme TEXT NOT NULL,              -- Human-readable theme name
    essence TEXT NOT NULL,            -- AI-generated summary
    memory_ids JSON NOT NULL,         -- Array of source memory IDs
    embedding VECTOR(1536),           -- Vector for similarity search
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP,
    access_count INTEGER DEFAULT 0,
    confidence_score FLOAT DEFAULT 0.0,  -- AI confidence (0.0-1.0)
    cluster_size INTEGER DEFAULT 0,      -- Number of memories
    metadata JSON                        -- Additional cluster metadata
);

-- Cluster Formation Log: Track cluster generation
CREATE TABLE cluster_formation_logs (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    formation_type TEXT NOT NULL,     -- 'auto', 'manual', 'merge'
    memory_count INTEGER,
    formed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    formation_metadata JSON
);

-- Create indexes
CREATE INDEX idx_clusters_theme ON neural_clusters(theme);
CREATE INDEX idx_clusters_created ON neural_clusters(created_at);
CREATE INDEX idx_clusters_accessed ON neural_clusters(last_accessed);
CREATE INDEX idx_clusters_embedding ON neural_clusters 
    USING ivfflat (embedding vector_cosine_ops);
```

### 1.2 Neural Cluster Formation

**Algorithm**:

```go
// internal/neural/cluster.go

func FormClusters(memories []Memory) ([]NeuralCluster, error) {
    // Step 1: Generate embeddings for all memories
    embeddings := generateEmbeddings(memories)
    
    // Step 2: Cluster by semantic similarity
    clusters := clusterBySimilarity(embeddings, threshold=0.85)
    
    // Step 3: For each cluster, generate essence
    for _, cluster := range clusters {
        cluster.Essence = generateEssence(cluster.Memories)
        cluster.Theme = extractTheme(cluster.Memories)
        cluster.ConfidenceScore = calculateConfidence(cluster)
    }
    
    // Step 4: Store clusters
    return storeClusters(clusters)
}

func generateEssence(memories []Memory) string {
    // Use LLM to generate summary
    prompt := fmt.Sprintf(
        "Summarize these %d related memories into a concise paragraph:\n%s",
        len(memories),
        formatMemories(memories),
    )
    return llm.Generate(prompt)
}
```

**Clustering Strategy**:
- **Hierarchical clustering** for related topics
- **Similarity threshold**: 0.85 (adjustable)
- **Minimum cluster size**: 3 memories
- **Maximum cluster size**: 50 memories (split if exceeded)

### 1.3 Context Retrieval

**Process**:

```go
// internal/neural/retrieval.go

func RetrieveContext(query string, limit int) (*ContextResult, error) {
    // Step 1: Generate query embedding
    queryEmbedding := generateEmbedding(query)
    
    // Step 2: Find relevant clusters via vector search
    clusters := vectorSearch(queryEmbedding, limit=5)
    
    // Step 3: From each cluster, get top 2-3 most relevant raw memories
    var contextMemories []Memory
    for _, cluster := range clusters {
        relevantMemories := getMostRelevant(cluster, query, n=3)
        contextMemories = append(contextMemories, relevantMemories...)
    }
    
    // Step 4: Return cluster essences + raw memories
    return &ContextResult{
        Clusters:  clusters,
        Memories:  contextMemories,
        TotalTokens: estimateTokens(clusters, contextMemories),
    }
}
```

### 1.4 Background Jobs

**Formation Job** (Weekly):
```go
// cmd/jobs/cluster-formation.go

func RunClusterFormation() {
    // Get unclustered memories (older than 1 week)
    memories := store.GetUnclusteredMemories(age=7*24*time.Hour)
    
    // Form clusters
    clusters, err := neural.FormClusters(memories)
    if err != nil {
        logger.Error("Cluster formation failed", zap.Error(err))
        return
    }
    
    // Store clusters
    for _, cluster := range clusters {
        store.CreateNeuralCluster(cluster)
    }
    
    logger.Info("Cluster formation complete", 
        zap.Int("clusters_created", len(clusters)))
}
```

**Daily Maintenance** (Lightweight):
- Update access counts
- Detect new contradictions in recent memories
- Quick redundancy check

### 1.5 User Interface

**CLI Commands**:
```bash
# View clusters
myrai memory clusters
myrai memory clusters --detailed

# View specific cluster
myrai memory cluster show <cluster-id>

# Force refresh
myrai memory cluster refresh <cluster-id>

# Search within clusters
myrai memory search "docker" --in-clusters
```

**API Endpoints**:
```
GET   /api/v1/memory/clusters              # List all clusters
GET   /api/v1/memory/clusters/:id          # Get cluster details
GET   /api/v1/memory/clusters/:id/memories # Get cluster's raw memories
POST  /api/v1/memory/clusters/:id/refresh  # Regenerate cluster
GET   /api/v1/memory/clusters/search       # Search clusters
```

**Telegram/Discord Command**:
```
/memory - Show your neural clusters summary
```

---

## Phase 2: Skill Runtime + MCP

**Duration**: Week 3-4  
**Goal**: Dynamic skill loading and MCP integration

### 2.1 Skill Manifest Format

**SKILL.md Specification**:

```markdown
---
name: docker-helper
version: 1.2.0
description: Docker container management operations
author: myrai-team
tags: [devops, containers, docker]
min_myrai_version: "2.0.0"

mcp:
  server: docker
  required: false

tools:
  - name: docker_ps
    description: List running Docker containers
    parameters:
      - name: format
        type: string
        enum: [table, json]
        default: table
        description: Output format
      - name: all
        type: boolean
        default: false
        description: Show all containers (including stopped)
    
  - name: docker_build
    description: Build Docker image from Dockerfile
    parameters:
      - name: path
        type: string
        required: true
        description: Path to Dockerfile
      - name: tag
        type: string
        required: true
        description: Image tag
---

# Docker Helper Skill

## Overview
This skill provides Docker container management capabilities...

## Usage Examples

### List Containers
```
docker_ps --format json
```

### Build Image
```
docker_build --path ./ --tag myapp:latest
```
```

### 2.2 Skill Registry

**Implementation**:

```go
// internal/skills/registry.go

type Registry struct {
    mu        sync.RWMutex
    skills    map[string]*Skill
    mcpTools  map[string]*MCPTool
}

type Skill struct {
    Manifest    *SkillManifest
    Tools       []Tool
    Status      SkillStatus  // enabled, disabled, error
    Source      SkillSource  // builtin, github, local, mcp
    LoadedAt    time.Time
    LastUpdated time.Time
}

func (r *Registry) Register(skill *Skill) error
func (r *Registry) Enable(name string) error
func (r *Registry) Disable(name string) error
func (r *Registry) GetTool(name string) (Tool, error)
func (r *Registry) ListSkills() []*Skill
```

### 2.3 Hot-Reload System

**File Watcher**:

```go
// internal/skills/watcher.go

type Watcher struct {
    watcher *fsnotify.Watcher
    loader  *SkillLoader
}

func (w *Watcher) WatchDirectory(path string) error {
    // Watch SKILL.md files
    // On change: reload skill
    // On create: load new skill
    // On delete: unregister skill
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
    switch event.Op {
    case fsnotify.Write:
        w.loader.ReloadSkill(event.Name)
    case fsnotify.Create:
        w.loader.LoadSkill(event.Name)
    case fsnotify.Remove:
        w.loader.UnloadSkill(event.Name)
    }
}
```

**Usage**:
```bash
# Development mode with hot-reload
myrai skills watch ./my-custom-skills/

# Edit SKILL.md in ./my-custom-skills/
# Changes apply instantly without restart
```

### 2.4 GitHub Integration

**Install from GitHub**:

```go
// internal/skills/github.go

func InstallFromGitHub(repo string) error {
    // Parse repo: github.com/user/repo
    // Download release or raw SKILL.md
    // Validate manifest
    // Install to ~/.myrai/skills/
    // Register in registry
}
```

**CLI**:
```bash
# Install from GitHub
myrai skills install github.com/myrai-skills/docker

# Install specific version
myrai skills install github.com/myrai-skills/docker@v1.2.0

# Update skill
myrai skills update docker

# List available skills on GitHub
myrai skills search docker
```

### 2.5 MCP Integration

**MCP Tool Adapter**:

```go
// internal/mcp/adapter.go

func AdaptMCPTool(mcpTool MCPTool) Tool {
    return Tool{
        Name:        mcpTool.Name,
        Description: mcpTool.Description,
        Parameters:  convertSchema(mcpTool.InputSchema),
        Execute: func(args map[string]interface{}) (interface{}, error) {
            // Call MCP server
            return mcpClient.CallTool(mcpTool.Name, args)
        },
    }
}
```

**Auto-Discovery**:

```go
// internal/mcp/discovery.go

func AutoDiscoverMCPServers() []MCPServerInfo {
    // Check official MCP registry
    // Check GitHub topic: mcp-server
    // Check community curations
    return []MCPServerInfo{
        {
            Name:        "filesystem",
            Description: "File system operations",
            Command:     "npx -y @modelcontextprotocol/server-filesystem",
            Category:    "system",
        },
        // ...
    }
}
```

**User Commands**:
```bash
# List MCP servers
myrai mcp list

# Add MCP server
myrai mcp add --name filesystem --command "npx -y @modelcontextprotocol/server-filesystem /home"

# Start MCP server
myrai mcp start filesystem

# List MCP tools
myrai mcp tools

# Auto-discover available MCP servers
myrai mcp discover
```

### 2.6 Skill Validation

**Validation Rules**:
- Manifest must be valid YAML
- Required fields: name, version, description
- Tool names must be unique
- Parameters must have valid types
- MCP server must be reachable (if specified)

**Security**:
- Skills run in sandbox (restricted filesystem access)
- Network policies for MCP servers
- Resource limits (CPU, memory)

---

## Phase 3: Adaptive Persona

**Duration**: Week 5-6  
**Goal**: Persona evolution based on usage patterns

### 3.1 Evolution Engine

**Pattern Detection**:

```go
// internal/persona/evolution.go

type EvolutionEngine struct {
    store          *store.Store
    crystalAnalyzer *neural.Analyzer
}

type PatternType string

const (
    PatternFrequency    PatternType = "frequency"    // Topic mentions
    PatternTemporal     PatternType = "temporal"     // Time-based
    PatternSkillUsage   PatternType = "skill_usage"  // Tool usage
    PatternContextSwitch PatternType = "context"     // Project switching
)

type DetectedPattern struct {
    Type       PatternType
    Subject    string
    Frequency  float64       // 0.0-1.0
    Evidence   []string      // Memory IDs
    Confidence float64       // 0.0-1.0
}

func (ee *EvolutionEngine) AnalyzePatterns() ([]DetectedPattern, error) {
    patterns := []DetectedPattern{}
    
    // Analyze Neural Clusters
    clusters := ee.store.GetNeuralClusters()
    for _, cluster := range clusters {
        if cluster.AccessCount > 20 {
            patterns = append(patterns, DetectedPattern{
                Type:       PatternFrequency,
                Subject:    cluster.Theme,
                Frequency:  float64(cluster.AccessCount) / 100.0,
                Evidence:   cluster.MemoryIDs,
                Confidence: cluster.ConfidenceScore,
            })
        }
    }
    
    // Analyze skill usage
    skillStats := ee.store.GetSkillUsageStats(lookback=30*24*time.Hour)
    for skill, count := range skillStats {
        if count > 50 {
            patterns = append(patterns, DetectedPattern{
                Type:      PatternSkillUsage,
                Subject:   skill,
                Frequency: float64(count) / 100.0,
            })
        }
    }
    
    return patterns, nil
}
```

**Proposal Generation**:

```go
func (ee *EvolutionEngine) GenerateProposals(patterns []DetectedPattern) ([]EvolutionProposal, error) {
    proposals := []EvolutionProposal{}
    
    for _, pattern := range patterns {
        switch pattern.Type {
        case PatternFrequency:
            if pattern.Frequency > 0.5 {
                proposals = append(proposals, EvolutionProposal{
                    ID:          generateID(),
                    Type:        "expertise",
                    Title:       fmt.Sprintf("Add '%s' expertise", pattern.Subject),
                    Description: fmt.Sprintf("You've mentioned %s in %d conversations", 
                        pattern.Subject, len(pattern.Evidence)),
                    Change: PersonaChange{
                        Field:   "expertise",
                        Action:  "add",
                        Value:   pattern.Subject,
                    },
                    Confidence: pattern.Confidence,
                    CreatedAt:  time.Now(),
                })
            }
            
        case PatternSkillUsage:
            // Propose skill prioritization
            
        case PatternTemporal:
            // Propose schedule-aware behaviors
        }
    }
    
    return proposals, nil
}
```

### 3.2 User Approval Flow

**Proposal Structure**:

```go
type EvolutionProposal struct {
    ID          string
    Type        string              // "expertise", "voice", "values"
    Title       string
    Description string
    Rationale   string              // Why this change is suggested
    Change      PersonaChange       // Specific change details
    Confidence  float64             // 0.0-1.0
    Status      ProposalStatus      // pending, approved, rejected
    CreatedAt   time.Time
    RespondedAt *time.Time
}
```

**User Interface**:

```
🧬 Persona Evolution Proposal

📊 Pattern Detected:
   • You frequently work with Kubernetes and Docker
   • 70% of your questions are DevOps-related
   • You use the Docker skill 53 times this week

💡 Suggested Evolution:
   • Add "DevOps Specialist" to my expertise
   • Adjust communication style to be more technical
   • Prioritize containerization tools in suggestions
   
📝 Rationale:
   Based on analysis of 47 conversations over the past 
   30 days, DevOps topics dominate your interactions.
   Specializing in this area will improve relevance.

Confidence: 92%

❓ Would you like me to:
   [✅ Yes] Apply these changes
   [📖 Details] See detailed analysis
   [❌ No] Ignore this suggestion
   [✏️ Customize] Adjust specific aspects
```

**CLI Commands**:
```bash
# View pending proposals
myrai persona proposals

# View specific proposal
myrai persona proposal show <id>

# Apply proposal
myrai persona proposal apply <id>

# Reject proposal
myrai persona proposal reject <id>

# View evolution history
myrai persona history

# Rollback to previous persona
myrai persona rollback <version>
```

### 3.3 Persona Versions

**Versioning**:

```go
// Store persona snapshots
type PersonaVersion struct {
    ID          string
    Timestamp   time.Time
    Identity    Identity
    UserProfile UserProfile
    ChangeType  string      // "evolution", "manual", "rollback"
    ChangeDescription string
}

// Save version on each change
func (pm *PersonaManager) SaveVersion(changeType, description string) error {
    version := &PersonaVersion{
        ID:                  generateID(),
        Timestamp:           time.Now(),
        Identity:            *pm.identity,
        UserProfile:         *pm.user,
        ChangeType:          changeType,
        ChangeDescription:   description,
    }
    return pm.store.SavePersonaVersion(version)
}
```

---

## Phase 4: Tool Orchestration

**Duration**: Week 7-8  
**Goal**: Intelligent task decomposition and execution

### 4.1 Intent Classification

```go
// internal/tools/intent.go

type IntentClassifier struct {
    llmClient *llm.Client
}

type Intent struct {
    Type        string      // "command", "question", "task", "conversation"
    Category    string      // "devops", "coding", "research", "general"
    Complexity  string      // "simple", "compound", "complex"
    Confidence  float64
    Keywords    []string
}

func (ic *IntentClassifier) Classify(input string) (*Intent, error) {
    // Use LLM to classify intent
    prompt := fmt.Sprintf(`
Classify the following user input:
"%s"

Provide:
1. Type: command, question, task, or conversation
2. Category: devops, coding, research, or general
3. Complexity: simple, compound, or complex
4. Keywords: key terms extracted
`, input)

    response := ic.llmClient.Complete(prompt)
    return parseIntentResponse(response)
}
```

### 4.2 Task Decomposition

```go
// internal/tools/decomposer.go

type TaskDecomposer struct {
    llmClient *llm.Client
}

type Task struct {
    ID          string
    Description string
    Dependencies []string      // Task IDs this depends on
    Tools       []string       // Suggested tools
    EstimatedTime time.Duration
}

func (td *TaskDecomposer) Decompose(intent string, availableTools []Tool) ([]Task, error) {
    prompt := fmt.Sprintf(`
Break down the following task into steps:
Task: "%s"

Available tools:
%s

Provide a list of steps with:
- Step description
- Required tools
- Dependencies on previous steps
`, intent, formatTools(availableTools))

    response := td.llmClient.Complete(prompt)
    return parseTasks(response)
}
```

### 4.3 Tool Chain Builder

**Chain Types**:

1. **Sequential**: Step 1 → Step 2 → Step 3
2. **Parallel**: Step 1 & Step 2 simultaneously → Step 3
3. **Conditional**: If Step 1 succeeds → Step 2, else → Step 3

```go
// internal/tools/chain.go

type ChainType string

const (
    ChainSequential  ChainType = "sequential"
    ChainParallel    ChainType = "parallel"
    ChainConditional ChainType = "conditional"
)

type ToolChain struct {
    ID          string
    Name        string
    Description string
    Type        ChainType
    Steps       []ChainStep
    Variables   map[string]string  // Shared variables
}

type ChainStep struct {
    ID          string
    Tool        string
    Parameters  map[string]interface{}
    DependsOn   []string
    Condition   string      // For conditional chains
    Timeout     time.Duration
    RetryCount  int
}

func (tc *ToolChain) Execute(ctx context.Context, orchestrator *Orchestrator) (*ChainResult, error) {
    switch tc.Type {
    case ChainSequential:
        return tc.executeSequential(ctx, orchestrator)
    case ChainParallel:
        return tc.executeParallel(ctx, orchestrator)
    case ChainConditional:
        return tc.executeConditional(ctx, orchestrator)
    }
}
```

### 4.4 Pre-built Tool Chains

**Example Chains**:

```yaml
# chains/deploy-application.yaml
name: deploy-application
description: Build, push, and deploy application
version: 1.0.0

type: sequential

steps:
  - id: verify-dockerfile
    tool: read_file
    parameters:
      path: "./Dockerfile"
    on_failure: abort
    
  - id: build-image
    tool: exec
    parameters:
      command: "docker build -t {{image_tag}} ."
    depends_on: [verify-dockerfile]
    timeout: 300s
    
  - id: verify-registry-login
    tool: browser
    parameters:
      action: "verify_element"
      url: "https://hub.docker.com"
      selector: "[data-testid='username']"
    depends_on: [build-image]
    optional: true
    
  - id: push-image
    tool: exec
    parameters:
      command: "docker push {{image_tag}}"
    depends_on: [build-image, verify-registry-login]
    
  - id: deploy-k8s
    tool: exec
    parameters:
      command: "kubectl apply -f k8s/"
    depends_on: [push-image]
    
  - id: verify-deployment
    tool: browser
    parameters:
      action: "screenshot"
      url: "{{app_url}}"
    depends_on: [deploy-k8s]
    optional: true
```

**Chain Execution**:

```bash
# Execute pre-built chain
myrai chain run deploy-application --var image_tag=myapp:v1.0

# Create custom chain
myrai chain create my-custom-chain

# Edit chain (opens editor)
myrai chain edit my-custom-chain

# List available chains
myrai chain list

# Share chain to marketplace
myrai chain publish my-custom-chain
```

### 4.5 Tool Inventory

**Categories**:

| Category | Tools | Count |
|----------|-------|-------|
| **System** | exec, process_list, process_kill, system_info | 4 |
| **File** | read_file, write_file, edit_file, search_files, list_dir, diff_files | 6 |
| **Browser** | navigate, click, type, screenshot, pdf, scroll, form_fill | 7 |
| **DevOps** | docker_ps, docker_build, docker_run, kubectl, helm, terraform | 6 |
| **Git** | git_status, git_commit, git_push, git_branch, git_diff, git_log | 6 |
| **API** | http_request, curl, websocket | 3 |
| **Database** | sql_query, mongo_query, redis_command | 3 |
| **Cloud** | aws_cli, gcloud, azure_cli | 3 |
| **Communication** | telegram_send, discord_send, email_send | 3 |
| **Cron** | schedule_job, list_jobs, delete_job | 3 |
| **Memory** | add_memory, search_memories, get_neural_cluster | 3 |
| **MCP** | (Dynamic from MCP servers) | Varies |

**Total**: 50+ built-in tools + unlimited MCP tools

---

## Phase 5: Reflection Engine

**Duration**: Week 9-10  
**Goal**: Self-auditing memory system

### 5.1 Contradiction Detection

```go
// internal/reflection/contradiction.go

type ContradictionDetector struct {
    llmClient *llm.Client
}

type Contradiction struct {
    ID          string
    MemoryA     string      // Memory ID
    MemoryB     string      // Memory ID
    Severity    string      // "high", "medium", "low"
    Description string      // "User said they like X, but later said they hate X"
    SuggestedResolution string
    DetectedAt  time.Time
    Status      string      // "open", "resolved", "ignored"
}

func (cd *ContradictionDetector) Detect(memories []Memory) ([]Contradiction, error) {
    // Use embeddings to find semantically similar memories
    // Then use LLM to check for contradictions
    
    contradictions := []Contradiction{}
    
    for i, memA := range memories {
        for _, memB := range memories[i+1:] {
            // Check if memories are about same topic
            similarity := cosineSimilarity(memA.Embedding, memB.Embedding)
            if similarity > 0.8 {
                // Check for contradiction
                prompt := fmt.Sprintf(`
Check if these two memories contradict each other:

Memory 1: "%s"
Memory 2: "%s"

Do they contradict? If yes, explain how.
`, memA.Content, memB.Content)

                result := cd.llmClient.Complete(prompt)
                if contains(result, "contradiction") {
                    contradictions = append(contradictions, Contradiction{
                        MemoryA:  memA.ID,
                        MemoryB:  memB.ID,
                        Severity: extractSeverity(result),
                        Description: extractDescription(result),
                    })
                }
            }
        }
    }
    
    return contradictions, nil
}
```

**User Interface**:

```
⚠️ Contradiction Found

Memory A (Jan 15, 2026): 
"I prefer Python for scripting tasks"

Memory B (Feb 20, 2026):
"I dislike Python, prefer Go for everything"

🤔 Resolution Options:
   [A is correct] I changed preference (keep A, archive B)
   [B is correct] I changed preference (keep B, archive A)
   [Context-dependent] Both true in different contexts
   [Neither] Both are incorrect
   [Ask me] Clarify before deciding

Detected: Feb 23, 2026
Severity: Medium
```

### 5.2 Redundancy Detection

```go
// internal/reflection/redundancy.go

type RedundancyGroup struct {
    ID       string
    Theme    string
    Memories []string  // Memory IDs
    Reason   string    // Why these are redundant
    SuggestedAction string  // "consolidate", "archive", "keep"
}

func FindRedundancies(clusters []NeuralCluster) ([]RedundancyGroup, error) {
    groups := []RedundancyGroup{}
    
    // Find clusters with same theme
    themeMap := make(map[string][]NeuralCluster)
    for _, cluster := range clusters {
        themeMap[cluster.Theme] = append(themeMap[cluster.Theme], cluster)
    }
    
    for theme, clusters := range themeMap {
        if len(clusters) > 1 {
            groups = append(groups, RedundancyGroup{
                Theme:    theme,
                Memories: flattenClusterMemories(clusters),
                Reason:   fmt.Sprintf("%d clusters with same theme", len(clusters)),
                SuggestedAction: "consolidate",
            })
        }
    }
    
    return groups, nil
}
```

### 5.3 Gap Analysis

```go
// internal/reflection/gaps.go

type Gap struct {
    Topic          string
    MentionCount   int
    MemoryCount    int
    GapRatio       float64  // mentions / memories
    SuggestedAction string
}

func IdentifyGaps(conversations []Conversation, memories []Memory) ([]Gap, error) {
    gaps := []Gap{}
    
    // Extract topics from conversations
    topicCounts := extractTopics(conversations)
    
    // Check which topics have few memories
    for topic, count := range topicCounts {
        if count > 10 {  // Mentioned frequently
            memoryCount := countMemoriesAboutTopic(memories, topic)
            if memoryCount < 3 {  // But few memories
                gaps = append(gaps, Gap{
                    Topic:          topic,
                    MentionCount:   count,
                    MemoryCount:    memoryCount,
                    GapRatio:       float64(memoryCount) / float64(count),
                    SuggestedAction: "ask_user",
                })
            }
        }
    }
    
    return gaps, nil
}
```

### 5.4 Health Report

**Report Structure**:

```go
type HealthReport struct {
    GeneratedAt     time.Time
    OverallScore    int         // 0-100
    
    // Metrics
    TotalMemories       int
    NeuralClusters      int
    Contradictions      int
    Redundancies        int
    Gaps                int
    
    // Breakdown
    ContradictionList   []Contradiction
    RedundancyList      []RedundancyGroup
    GapList             []Gap
    
    // Actions
    SuggestedActions    []Action
}

type Action struct {
    Type        string      // "resolve_contradiction", "consolidate", "fill_gap"
    Priority    string      // "high", "medium", "low"
    Description string
    Impact      string      // What will improve
}
```

**Sample Report**:

```
📊 Memory Health Report
Generated: February 23, 2026

Overall Score: 85/100 ✅

📈 Statistics:
• Total Memories: 247
• Neural Clusters: 18
• Contradictions: 3 (Medium priority)
• Redundancies: 12 memories (can consolidate)
• Knowledge Gaps: 1

⚠️ Issues Requiring Attention:

1. Contradictions (3 found)
   [View details] / [Auto-resolve]
   
2. Redundancies (12 memories)
   • 5 Docker-related memories → Consolidate to 1 cluster
   • 4 API-related memories → Consolidate to 1 cluster
   • 3 Python-related memories → Consolidate to 1 cluster
   [Consolidate all] / [Review individually]
   
3. Knowledge Gap (1 found)
   • Topic: "Microservices"
   • Mentioned 45 times, but only 2 memories
   • Suggestion: Ask user about microservices knowledge
   [Ask now] / [Dismiss]

💡 Optimization Impact:
   • Consolidating redundancies: Save ~8KB storage
   • Resolving contradictions: Improve accuracy by ~12%
   • Filling knowledge gap: Better responses on microservices

📊 Weekly Evolution:
   • 47 new memories added
   • 5 neural clusters formed
   • 1 contradiction resolved
   • Memory health: +5% vs last week
```

**CLI Commands**:

```bash
# Generate health report
myrai memory health

# Run full reflection
myrai memory reflect

# View contradictions
myrai memory contradictions

# Resolve specific contradiction
myrai memory resolve <contradiction-id> --keep A

# Consolidate redundancies
myrai memory consolidate --all

# View gaps
myrai memory gaps
```

### 5.5 Background Schedule

```
Daily (Lightweight):
  02:00 AM - Check new memories for contradictions
  02:30 AM - Quick redundancy scan
  Duration: <5 minutes

Weekly (Deep Analysis):
  Sunday 03:00 AM - Full reflection audit
  Duration: 15-30 minutes
  Actions:
    • Generate health report
    • Detect all contradictions
    • Find redundancies
    • Identify gaps
    • Create evolution proposals
    • Optimize neural clusters

Monthly:
  1st of month - Archive old clusters
  • Move clusters with no access in 90 days to cold storage
  • Generate monthly evolution report
```

---

## Phase 6: Marketplace

**Duration**: Week 11-12  
**Goal**: Community-driven agent and skill marketplace

### 6.1 GitHub Organization Structure

**Org**: `github.com/myrai-agents`

**Repository Structure**:
```
myrai-agents/
├── README.md                    # Marketplace overview
├── agents/
│   ├── devops-agent/
│   │   ├── AGENT.yaml
│   │   ├── README.md
│   │   └── icon.png
│   ├── developer-agent/
│   ├── researcher-agent/
│   └── personal-assistant/
├── skills/
│   ├── docker/
│   │   ├── SKILL.md
│   │   └── README.md
│   ├── kubernetes/
│   ├── aws/
│   └── github/
└── chains/
    ├── deploy-application/
    ├── setup-ci-cd/
    └── database-migration/
```

### 6.2 Agent Package Format

**AGENT.yaml**:

```yaml
agent:
  name: DevOps Assistant
  version: 2.0.0
  author: myrai-team
  description: |
    Complete DevOps automation agent specializing in 
    containerization, orchestration, and CI/CD pipelines.
  
  tags: [devops, docker, kubernetes, ci-cd, automation]
  
  icon: icon.png
  
  requirements:
    min_myrai_version: "2.0.0"
    recommended_memory: 4GB
    
  # Knowledge configuration
  knowledge:
    neural_clusters:
      - docker-best-practices
      - kubernetes-patterns
      - ci-cd-workflows
      - terraform-modules
      
  # Skills included
  skills:
    builtin:
      - docker
      - kubectl
      - helm
      - terraform
      - github-actions
    external:
      - name: aws-cli
        source: github.com/myrai-agents/skills/aws
      - name: gcp-cli
        source: github.com/myrai-agents/skills/gcp
        
  # Tool chains included
  chains:
    - deploy-application
    - rollback-deployment
    - scale-service
    - setup-monitoring
    
  # MCP servers required
  mcp:
    - filesystem
    - github
    - docker
    
  # Default persona
  persona:
    identity:
      name: DevOps Assistant
      personality: |
        Technical, efficient, and focused on automation.
        Prefers Infrastructure as Code and containerization.
      voice: concise, technical, solution-oriented
      expertise: 
        - DevOps
        - Cloud Infrastructure
        - CI/CD
        - Containerization
        - Kubernetes
      values:
        - automation
        - reliability
        - efficiency
        - best-practices
        
    # Learning goals
    learning:
      - Infrastructure as Code
      - GitOps workflows
      - Security best practices
      
  # Pricing (Early adoption: free)
  pricing:
    model: free
    
  # Support
  support:
    documentation: https://docs.myrai.dev/agents/devops
    issues: https://github.com/myrai-agents/devops-agent/issues
    
  # Stats
  stats:
    downloads: 15420
    rating: 4.8
    reviews: 127
    last_updated: 2026-02-23
```

### 6.3 Marketplace CLI

```bash
# Search marketplace
myrai marketplace search "devops"
myrai marketplace search --category devops --sort downloads

# View agent details
myrai marketplace info myrai-agents/devops-agent

# Install agent
myrai marketplace install myrai-agents/devops-agent

# Install specific version
myrai marketplace install myrai-agents/devops-agent@v2.0.0

# List installed agents
myrai marketplace list --installed

# Update agent
myrai marketplace update devops-agent

# Remove agent
myrai marketplace remove devops-agent

# Submit agent to marketplace
myrai marketplace publish ./my-agent/

# Rate/review agent
myrai marketplace review devops-agent --rating 5 --comment "Excellent!"
```

### 6.4 Verification System

**Quality Checks**:
- YAML validation
- Security scan
- Dependency check
- Test execution
- Documentation completeness

**Badges**:
- ✅ Verified by Myrai Team
- 🔒 Security Audited
- 🧪 Tested
- 📚 Well Documented
- ⭐ Community Favorite

### 6.5 Community Contributions

**Contributor Levels**:
- **User**: Install and rate agents
- **Creator**: Publish agents/skills
- **Maintainer**: Help maintain official agents
- **Core Team**: Manage marketplace

**Monetization (Future)**:
- Free: All current features
- Pro: Premium agents (optional)
- Creator earnings: 70% to creator, 30% to platform

---

## API Specifications

### REST API Endpoints

#### Neural Clusters
```
GET   /api/v1/memory/clusters
      Query: ?limit=20&offset=0&sort=access_count
      Response: { clusters: [...], total: 150 }

GET   /api/v1/memory/clusters/:id
      Response: { id, theme, essence, memories: [...], metadata }

GET   /api/v1/memory/clusters/:id/memories
      Response: { memories: [...] }

POST  /api/v1/memory/clusters/:id/refresh
      Response: { success: true, cluster: {...} }

GET   /api/v1/memory/clusters/search?q=docker
      Response: { clusters: [...] }
```

#### Skills
```
GET   /api/v1/skills
      Response: { skills: [...] }

POST  /api/v1/skills/install
      Body: { source: "github.com/user/repo" }
      Response: { success: true, skill: {...} }

POST  /api/v1/skills/:name/enable
      Response: { success: true }

POST  /api/v1/skills/:name/disable
      Response: { success: true }

DELETE /api/v1/skills/:name
       Response: { success: true }
```

#### Persona
```
GET   /api/v1/persona
      Response: { identity: {...}, user_profile: {...} }

PUT   /api/v1/persona
      Body: { identity: {...}, user_profile: {...} }
      Response: { success: true }

GET   /api/v1/persona/evolution
      Response: { proposals: [...] }

POST  /api/v1/persona/evolution/:id/apply
      Response: { success: true }

POST  /api/v1/persona/evolution/:id/reject
      Response: { success: true }

GET   /api/v1/persona/history
      Response: { versions: [...] }
```

#### Memory Health
```
GET   /api/v1/memory/health
      Response: { score: 85, issues: {...}, metrics: {...} }

POST  /api/v1/memory/reflect
      Response: { report: {...} }

GET   /api/v1/memory/contradictions
      Response: { contradictions: [...] }

POST  /api/v1/memory/contradictions/:id/resolve
      Body: { resolution: "keep_a" }
      Response: { success: true }
```

#### MCP
```
GET   /api/v1/mcp/servers
      Response: { servers: [...] }

POST  /api/v1/mcp/servers
      Body: { name, command, args, env }
      Response: { success: true }

DELETE /api/v1/mcp/servers/:name
       Response: { success: true }

GET   /api/v1/mcp/tools
      Response: { tools: [...] }
```

#### Marketplace
```
GET   /api/v1/marketplace/agents
      Query: ?q=devops&category=devops&sort=downloads
      Response: { agents: [...] }

GET   /api/v1/marketplace/agents/:id
      Response: { agent: {...} }

POST  /api/v1/marketplace/agents/:id/install
      Response: { success: true }

GET   /api/v1/marketplace/installed
      Response: { agents: [...] }
```

---

## Database Schema

### Complete Schema

```sql
-- Core tables (existing)
-- users, conversations, messages, memories, tasks, scheduled_jobs

-- ============================================
-- Phase 1: Neural Clusters
-- ============================================

CREATE TABLE neural_clusters (
    id TEXT PRIMARY KEY,
    theme TEXT NOT NULL,
    essence TEXT NOT NULL,
    memory_ids JSON NOT NULL,
    embedding VECTOR(1536),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP,
    access_count INTEGER DEFAULT 0,
    confidence_score FLOAT DEFAULT 0.0,
    cluster_size INTEGER DEFAULT 0,
    metadata JSON
);

CREATE TABLE cluster_formation_logs (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL REFERENCES neural_clusters(id),
    formation_type TEXT NOT NULL,
    memory_count INTEGER,
    formed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    formation_metadata JSON
);

-- ============================================
-- Phase 2: Skills & MCP
-- ============================================

CREATE TABLE skills (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    author TEXT,
    source TEXT NOT NULL,  -- 'builtin', 'github', 'local', 'mcp'
    source_url TEXT,
    manifest JSON NOT NULL,
    status TEXT DEFAULT 'disabled',  -- 'enabled', 'disabled', 'error'
    error_message TEXT,
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_loaded_at TIMESTAMP
);

CREATE TABLE mcp_servers (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    command TEXT NOT NULL,
    args JSON,
    env JSON,
    enabled BOOLEAN DEFAULT false,
    status TEXT DEFAULT 'stopped',  -- 'running', 'stopped', 'error'
    error_message TEXT,
    container_id TEXT,
    started_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Phase 3: Persona Evolution
-- ============================================

CREATE TABLE persona_versions (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    identity JSON NOT NULL,
    user_profile JSON NOT NULL,
    change_type TEXT NOT NULL,  -- 'evolution', 'manual', 'rollback'
    change_description TEXT,
    triggered_by TEXT  -- 'user', 'system', 'evolution_engine'
);

CREATE TABLE evolution_proposals (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,  -- 'expertise', 'voice', 'values'
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    rationale TEXT,
    change JSON NOT NULL,  -- PersonaChange struct
    confidence FLOAT NOT NULL,
    status TEXT DEFAULT 'pending',  -- 'pending', 'approved', 'rejected'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    responded_at TIMESTAMP,
    response TEXT  -- user feedback
);

-- ============================================
-- Phase 4: Tool Chains
-- ============================================

CREATE TABLE tool_chains (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    version TEXT,
    author TEXT,
    type TEXT NOT NULL,  -- 'sequential', 'parallel', 'conditional'
    definition JSON NOT NULL,  -- Chain definition
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
    status TEXT,  -- 'running', 'completed', 'failed'
    input_params JSON,
    results JSON,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- ============================================
-- Phase 5: Reflection Engine
-- ============================================

CREATE TABLE contradictions (
    id TEXT PRIMARY KEY,
    memory_a_id TEXT NOT NULL REFERENCES memories(id),
    memory_b_id TEXT NOT NULL REFERENCES memories(id),
    severity TEXT NOT NULL,  -- 'high', 'medium', 'low'
    description TEXT NOT NULL,
    suggested_resolution TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'open',  -- 'open', 'resolved', 'ignored'
    resolved_at TIMESTAMP,
    resolution TEXT
);

CREATE TABLE redundancy_groups (
    id TEXT PRIMARY KEY,
    theme TEXT,
    memory_ids JSON NOT NULL,
    cluster_ids JSON,
    reason TEXT,
    suggested_action TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'open',
    consolidated_at TIMESTAMP
);

CREATE TABLE knowledge_gaps (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    mention_count INTEGER,
    memory_count INTEGER,
    gap_ratio FLOAT,
    suggested_action TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'open',
    filled_at TIMESTAMP
);

CREATE TABLE reflection_reports (
    id TEXT PRIMARY KEY,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    overall_score INTEGER,
    total_memories INTEGER,
    neural_clusters INTEGER,
    contradictions INTEGER,
    redundancies INTEGER,
    gaps INTEGER,
    details JSON,
    suggested_actions JSON
);

-- ============================================
-- Phase 6: Marketplace
-- ============================================

CREATE TABLE marketplace_agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    author TEXT NOT NULL,
    description TEXT,
    repository_url TEXT,
    manifest JSON NOT NULL,
    download_count INTEGER DEFAULT 0,
    rating FLOAT,
    review_count INTEGER DEFAULT 0,
    is_verified BOOLEAN DEFAULT false,
    is_official BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, version)
);

CREATE TABLE installed_agents (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES marketplace_agents(id),
    installed_version TEXT,
    config JSON,
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE TABLE agent_reviews (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES marketplace_agents(id),
    user_id TEXT,
    rating INTEGER,
    review TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_clusters_theme ON neural_clusters(theme);
CREATE INDEX idx_clusters_accessed ON neural_clusters(last_accessed);
CREATE INDEX idx_clusters_embedding ON neural_clusters USING ivfflat (embedding vector_cosine_ops);

CREATE INDEX idx_skills_status ON skills(status);
CREATE INDEX idx_skills_source ON skills(source);

CREATE INDEX idx_proposals_status ON evolution_proposals(status);

CREATE INDEX idx_contradictions_status ON contradictions(status);
CREATE INDEX idx_redundancies_status ON redundancy_groups(status);

CREATE INDEX idx_marketplace_downloads ON marketplace_agents(download_count);
CREATE INDEX idx_marketplace_rating ON marketplace_agents(rating);
```

---

## Configuration

### Main Config (`~/.myrai/config.yaml`)

```yaml
# Myrai 2.0 Configuration

version: "2.0"

# Server settings
server:
  host: "0.0.0.0"
  port: 8080
  
# LLM Configuration
llm:
  default_provider: "anthropic"
  providers:
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"
      model: "claude-sonnet-4.6"
    openai:
      api_key: "${OPENAI_API_KEY}"
      model: "gpt-5.2"

# Storage Configuration
storage:
  data_dir: "~/.myrai/data"
  sqlite_path: "~/.myrai/data/myrai.db"
  vector_dim: 1536

# Neural Cluster Settings
neural_clusters:
  enabled: true
  formation_schedule: "weekly"  # daily, weekly, monthly
  formation_day: "sunday"
  formation_time: "03:00"
  min_cluster_size: 3
  max_cluster_size: 50
  similarity_threshold: 0.85
  
# Persona Evolution
persona:
  evolution_enabled: true
  analysis_schedule: "weekly"
  auto_apply_threshold: 0.95  # Auto-apply if confidence > 95%
  require_approval: true
  
# Reflection Engine
reflection:
  enabled: true
  daily_check: true
  weekly_audit: true
  daily_time: "02:00"
  weekly_day: "sunday"
  weekly_time: "03:00"
  
# Skills Configuration
skills:
  hot_reload: true
  builtin_dir: "~/.myrai/skills/builtin"
  custom_dir: "~/.myrai/skills/custom"
  auto_update: false
  
# MCP Configuration
mcp:
  enabled: true
  docker_network: "myrai-mcp"
  servers:
    - name: filesystem
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-filesystem", "~"]
      enabled: true
      
    - name: github
      command: "docker"
      args: ["run", "-i", "--rm", "-e", "GITHUB_TOKEN", "mcp/github"]
      env:
        GITHUB_TOKEN: "${GITHUB_TOKEN}"
      enabled: false

# Channels (Telegram, WhatsApp, Discord)
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    
  discord:
    enabled: false
    bot_token: "${DISCORD_BOT_TOKEN}"

# Marketplace
marketplace:
  registry_url: "https://github.com/myrai-agents"
  auto_check_updates: true
  check_interval: "daily"

# Logging
logging:
  level: "info"  # debug, info, warn, error
  file: "~/.myrai/logs/myrai.log"
  max_size: 100  # MB
  max_backups: 5
```

---

## Success Metrics

### Phase Completion Criteria

| Phase | Success Metric |
|-------|---------------|
| **Phase 0** | MCP client can connect to 5+ servers, hot-reload works |
| **Phase 1** | Neural Clusters reduce context tokens by 50% |
| **Phase 2** | Install skill from GitHub in <30 seconds |
| **Phase 3** | Persona evolution proposals accepted >70% of time |
| **Phase 4** | Tool chains execute 10+ step workflows |
| **Phase 5** | Health score >80% for active users |
| **Phase 6** | 10+ agents in marketplace |

### User Experience Goals

- **Onboarding**: New user productive in <5 minutes
- **Skill Install**: <30 seconds from search to usage
- **Persona Evolution**: Proposals relevant >80% of time
- **Memory Health**: Users resolve issues within 1 week
- **Marketplace**: Find and install agent in <2 minutes

---

## Next Steps

1. **Review**: Please review this implementation plan
2. **Feedback**: Any changes or additions needed?
3. **Priority**: Which phase to start with?
4. **Resources**: Confirm timeline and resources

**Recommended Start**: Phase 1 (Neural Clusters) - core innovation  
**Alternative**: Phase 0 (Foundation) - establish infrastructure first

Ready to proceed with implementation?
# Myrai v0.3 - OpenClaw-Style Features

This release brings comprehensive OpenClaw-style capabilities to Myrai, transforming it from a simple chatbot into a powerful AI agent with skills, multi-channel support, and scheduled automation.

## 🎯 New Features Overview

### 1. Skills Architecture (Plugin System)
A modular skill system that allows extending Myrai's capabilities without modifying core code.

**Built-in Skills:**

| Skill | Description | Example Usage |
|-------|-------------|---------------|
| `github` | GitHub repository operations | "Search for Go repositories about web frameworks" |
| `weather` | Weather forecasts | "What's the weather in New York?" |
| `notes` | Personal note management | "Take a note: Remember to call mom" |
| `time` | Time utilities | "What time is it in Tokyo?" |

**Architecture:**
```go
// Skills are self-contained modules
type Skill interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}
```

**Configuration:**
```yaml
# config.yaml
skills:
  enabled:
    - github
    - weather
    - notes
  
  github:
    token: "${GITHUB_TOKEN}"  # GitHub personal access token
    default_owner: "myorg"
  
  weather:
    api_key: "${WEATHER_API_KEY}"
    default_city: "Beijing"
```

### 2. Telegram Bot Integration
Chat with Myrai directly from Telegram on any device.

**Features:**
- 💬 Direct message support
- 👥 Group chat support (with @mention)
- 🔄 Multi-turn conversations with context
- 📋 Commands: `/start`, `/help`, `/new`, `/status`

**Setup:**
```yaml
# config.yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    allowed_users: []  # Optional: restrict to specific users
```

**Commands:**
- `/start` - Welcome message and instructions
- `/help` - List available commands
- `/new` - Start a new conversation
- `/status` - Check bot and service status

### 3. Cron Job Scheduler (Planned)
Schedule tasks to run automatically at specified intervals.

**Use Cases:**
- Daily weather reports at 8 AM
- Weekly GitHub issue summaries
- Periodic health checks
- Automated backups

**Example (Future Implementation):**
```yaml
jobs:
  - name: "morning-weather"
    schedule: "0 8 * * *"
    skill: "weather"
    action: "get_forecast"
    args:
      city: "Beijing"
    notify:
      channel: "telegram"
      message: "Good morning! Here's today's weather: {{result}}"
  
  - name: "github-summary"
    schedule: "0 9 * * 1"
    skill: "github"
    action: "list_issues"
    args:
      owner: "myorg"
      repo: "myproject"
```

### 4. Vector Memory with RAG (Planned)
Long-term memory with semantic search for context-aware responses.

**Features:**
- Document ingestion (PDF, Markdown, Text)
- Semantic search using embeddings
- Conversation memory persistence
- Knowledge base integration

**Architecture:**
```
User Query → Embedding → Vector Search → Relevant Docs → LLM Response
```

## 🏗️ Architecture Changes

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Myrai v0.3                                 │
├─────────────────────────────────────────────────────────────────────────┤
│  Channels Layer                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────────────────┐  │
│  │   Web UI     │  │  Telegram    │  │    Future: Discord, Slack   │  │
│  └──────────────┘  └──────────────┘  └─────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────────┤
│  Agent Layer                                                            │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │  Agent (Orchestrator)                                           │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │  │
│  │  │  LLM Client  │  │    Tools     │  │      Skills          │  │  │
│  │  │  (Kimi K2.5) │  │  (Internal)  │  │  (External Plugins)  │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────────────┘  │  │
│  └─────────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────────┤
│  Skills Registry                                                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────────────┐ │
│  │  github  │ │  weather │ │  notes   │ │    Future: Custom Skills  │ │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────────────────┘ │
├─────────────────────────────────────────────────────────────────────────┤
│  Data Layer                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────────┐ │
│  │  SQLite      │  │   BadgerDB   │  │    Vector DB (Future)        │ │
│  │  (GORM)      │  │   (KV Store) │  │    (chromem-go)              │ │
│  └──────────────┘  └──────────────┘  └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

## 🚀 Getting Started

### Installation
```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/myrai.ai.git
cd myrai.ai

# Build
make build

# Run
./bin/myrai
```

### Configuration
```bash
# Copy example config
cp config.example.yaml config.yaml

# Edit with your settings
vim config.yaml
```

### Environment Variables
```bash
# Create .env file
cat > .env << 'EOF'
# LLM Configuration
GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY=your_kimi_api_key

# Skills Configuration
GITHUB_TOKEN=your_github_token
WEATHER_API_KEY=your_weather_api_key

# Telegram Configuration
TELEGRAM_BOT_TOKEN=your_bot_token
EOF
```

## 📝 API Endpoints

### Skills Management
```bash
# List available skills
GET /api/skills

# Get skill details
GET /api/skills/:name

# Execute skill directly
POST /api/skills/:name/execute
{
  "action": "search_repositories",
  "args": {
    "query": "language:go web framework"
  }
}
```

### Cron Jobs (Future)
```bash
# List jobs
GET /api/jobs

# Create job
POST /api/jobs
{
  "name": "daily-weather",
  "schedule": "0 8 * * *",
  "skill": "weather",
  "action": "get_forecast"
}

# Delete job
DELETE /api/jobs/:id
```

## 🔧 Creating Custom Skills

```go
package myskill

import "context"

type Skill struct {
    config map[string]interface{}
}

func (s *Skill) Name() string {
    return "my_custom_skill"
}

func (s *Skill) Description() string {
    return "Does something awesome"
}

func (s *Skill) GetToolDefinition() llm.ToolDefinition {
    return llm.ToolDefinition{
        Type: "function",
        Function: llm.FunctionDefinition{
            Name:        s.Name(),
            Description: s.Description(),
            Parameters: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "param1": map[string]interface{}{
                        "type": "string",
                        "description": "Description of param1",
                    },
                },
                "required": []string{"param1"},
            },
        },
    }
}

func (s *Skill) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    // Your implementation here
    return map[string]string{"status": "success"}, nil
}
```

## 📊 Feature Comparison

| Feature | Myrai v0.2 | Myrai v0.3 | OpenClaw |
|---------|---------------|---------------|----------|
| LLM Support | ✅ Kimi | ✅ Kimi | ✅ Multiple |
| Web UI | ✅ | ✅ | ✅ |
| Tools | ✅ | ✅ | ✅ |
| **Skills System** | ❌ | ✅ | ✅ |
| **Telegram Bot** | ❌ | ✅ | ✅ |
| **Discord/Slack** | ❌ | 🔄 Planned | ✅ |
| **Cron Jobs** | ❌ | 🔄 Planned | ✅ |
| **Vector Memory** | ❌ | 🔄 Planned | ✅ |
| **Web Search** | ❌ | ✅ via skill | ✅ |
| **MCP Protocol** | ❌ | 🔄 Planned | ✅ |

## 🗺️ Roadmap

### v0.3.x (Current)
- ✅ Skills architecture
- ✅ Built-in skills (GitHub, Weather, Notes)
- ✅ Telegram bot integration
- ✅ Improved tool streaming

### v0.4.x (Next)
- 🔄 Cron job scheduler
- 🔄 Discord/Slack integration
- 🔄 More built-in skills
- 🔄 Skill marketplace

### v0.5.x (Future)
- 🔄 Vector memory (RAG)
- 🔄 MCP protocol support
- 🔄 Web browsing skill
- 🔄 Multi-LLM support

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

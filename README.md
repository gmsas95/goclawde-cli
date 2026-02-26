# Myrai 2.0 (未来) - Production Ready

> **Myrai** (未来) means "future" in Japanese.  
> **Myrai** (My + AI) means "my personal AI".  
> **Myrai 2.0** is the future of autonomous personal assistance.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue)](https://golang.org)
[![Tests](https://img.shields.io/badge/tests-120%2B%20passing-brightgreen)](https://github.com/gmsas95/goclawde-cli)
[![Coverage](https://img.shields.io/badge/coverage-87%25%20core-brightgreen)](https://github.com/gmsas95/goclawde-cli)
[![Status](https://img.shields.io/badge/status-production%20ready-green.svg)](https://github.com/gmsas95/goclawde-cli)

**Myrai** is a lightweight, local-first, autonomous AI assistant that adapts to you.

Not just a chatbot. Not just a CLI tool. **A life assistant that learns and evolves.**

---

## ✨ What's New in 2.0

### 🧠 Neural Memory System
- **Neural Clusters** - Semantic memory compression inspired by human brain
- **Persistent Context** - Remembers conversations, preferences, and facts indefinitely
- **Smart Retrieval** - Contextually relevant memory surfacing

### 🎭 Adaptive Persona Evolution
- **Self-Improving AI** - Analyzes conversations to improve its personality
- **Evolution Proposals** - Suggests persona updates based on your feedback
- **Multiple Personas** - Context-aware personalities for different scenarios

### 🛠️ Skill Runtime + MCP
- **Dynamic Skills** - Install skills from GitHub with hot-reload
- **MCP Protocol** - Native Model Context Protocol support
- **Sandboxed Execution** - Secure skill runtime environment
- **Auto-Discovery** - Automatically find and configure MCP servers

### 🔍 Reflection Engine
- **Self-Monitoring** - Detects contradictions and knowledge gaps
- **Health Reports** - Analyzes conversation quality
- **Continuous Improvement** - Learns from its own mistakes

---

## 🚀 Features

| Feature | Description |
|---------|-------------|
| 🤖 **20+ LLM Providers** | OpenAI, Anthropic, Google, Groq, DeepSeek, Ollama, OpenRouter, and more |
| 🧠 **Neural Memory** | Semantic clustering for intelligent context management |
| 🎭 **Adaptive Persona** | AI personality that evolves with you |
| 🛠️ **18+ Skills** | Tasks, Calendar, Health, Shopping, Documents, GitHub, Browser, Voice |
| 💎 **Beautiful TUI** | Bubble Tea-based terminal UI with markdown rendering |
| 🔌 **MCP Protocol** | Native Model Context Protocol for tool integration |
| 💬 **Multi-Channel** | CLI, Web UI, Telegram, Discord |
| 🌐 **Real-time Search** | Brave Search, DuckDuckGo integration |
| 🔒 **Privacy First** | Local-first, SQLite + BadgerDB, no data sharing |
| 🐳 **Docker Ready** | Single container deployment |
| 📦 **Single Binary** | ~45MB Go binary, no dependencies |
| ⚡ **High Performance** | Circuit breakers, job queues, concurrent processing |
| 🧪 **Well Tested** | 120+ integration tests, 87%+ coverage |

---

## 🆚 Comparison

| Feature | Myrai 2.0 | OpenClaw | ChatGPT | Siri/Alexa |
|---------|-----------|----------|---------|------------|
| **Neural Memory** | ✅ Semantic clusters | ✅ Basic | ❌ Session-only | ❌ Limited |
| **Adaptive Persona** | ✅ Self-evolving | ❌ Static | ❌ Static | ❌ Static |
| **MCP Protocol** | ✅ Native | ⚠️ Partial | ❌ No | ❌ No |
| **Local-First** | ✅ Full | ✅ Yes | ❌ Cloud | ⚠️ Hybrid |
| **Self-Hosted** | ✅ Easy | ✅ Moderate | ❌ No | ❌ No |
| **Open Source** | ✅ MIT | ✅ MIT | ❌ No | ❌ No |
| **Multi-LLM** | ✅ 20+ providers | ✅ Multiple | ❌ Locked | ❌ Locked |
| **Skills System** | ✅ 18+ + Runtime | ✅ Many | ⚠️ Limited | ⚠️ Basic |
| **Mobile Apps** | ❌ No | ✅ iOS/Android | ✅ Yes | ✅ Native |
| **Voice** | ⚠️ Basic TTS/STT | ✅ Advanced | ⚠️ Limited | ✅ Native |

---

## 🚀 Quick Start (5 Minutes)

### Prerequisites

- **Go 1.24+** (for building from source)
- **At least one LLM API key**:
  - [OpenAI](https://platform.openai.com) - Most popular
  - [Anthropic](https://console.anthropic.com) - Great reasoning
  - [Groq](https://console.groq.com) - Fast & affordable
  - [Ollama](https://ollama.com) - **Free**, runs locally

### Installation

**Docker (Recommended for VPS)**
```bash
docker run -d \
  --name myrai \
  -p 8080:8080 \
  -v ~/.myrai:/app/data \
  -e OPENAI_API_KEY=your-key \
  ghcr.io/gmsas95/myrai:latest
```

**Docker Compose**
```bash
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/docker-compose.yml -o docker-compose.yml
docker-compose up -d
```

**Build from Source**
```bash
git clone https://github.com/gmsas95/goclawde-cli.git
cd goclawde-cli
go build -o myrai ./cmd/myrai
```

### First Run

```bash
# Run the interactive setup wizard
./myrai onboard

# Start the server (Web UI + API + Channels)
./myrai server

# Or use the beautiful TUI mode (recommended for local use)
./myrai --tui

# Or use CLI mode
./myrai --cli
```

---

## 💰 Cost Considerations

**Myrai is FREE to use**. You only pay for LLM API calls:

| Provider | Cost | Best For |
|----------|------|----------|
| **Ollama** | **FREE** | Privacy, unlimited local use |
| **Groq** | ~$0.0001/1K tokens | Speed, cost-effective |
| **DeepSeek** | Very cheap | Coding tasks |
| **OpenAI** | Standard | General reliability |
| **Anthropic** | Standard | Complex reasoning |

**Typical usage**: $1-5/month for casual use, $10-20/month for heavy use.

---

## 🛠️ Skills System

Myrai comes with 18 built-in skills and a runtime system for custom skills:

### Built-in Skills

**Productivity:**
- ✅ **tasks** - Todo management with scheduling
- ✅ **calendar** - Event management, Google Calendar integration
- ✅ **notes** - Note taking with search
- ✅ **documents** - PDF processing, OCR, image analysis

**Personal:**
- ✅ **health** - Health tracking, medication reminders
- ✅ **shopping** - Shopping lists, inventory
- ✅ **expenses** - Budget tracking, expense analysis

**Development:**
- ✅ **github** - Repository management, PR reviews
- ✅ **browser** - Web automation with ChromeDP
- ✅ **agentic** - Git automation, code analysis

**Information:**
- ✅ **search** - Web search (Brave, DuckDuckGo)
- ✅ **weather** - Weather forecasts
- ✅ **knowledge** - Knowledge base with semantic search
- ✅ **intelligence** - Smart suggestions

**System:**
- ✅ **voice** - STT/TTS integration
- ✅ **vision** - Image analysis, OCR
- ✅ **system** - System commands

### Custom Skills

Install skills from GitHub:
```bash
# Install a skill
myrai skills install github.com/user/skill-name

# Enable it
myrai skills enable skill-name

# Watch for changes during development
myrai skills watch ./my-skills/
```

Create custom skills with SKILL.md:
```yaml
---
name: my-custom-skill
version: 1.0.0
description: Does something awesome
author: your-name
tools:
  - name: do_something
    description: Perform the action
    parameters:
      - name: input
        type: string
        description: Input parameter
---

# Your skill documentation here
```

### MCP Integration

Connect external MCP servers:
```bash
# Discover available MCP servers
myrai mcp discover

# Add a server
myrai mcp discover-add filesystem

# Start the server
myrai mcp start filesystem

# List available tools
myrai mcp tools
```

---

## 🎭 Persona System

Myrai uses markdown-based personas that adapt over time:

```bash
# View current persona
myrai persona

# Edit manually
myrai persona edit

# View evolution proposals
myrai persona proposals

# Apply an evolution
myrai persona apply <proposal-id>
```

Persona files are stored in `~/.myrai/`:
- `IDENTITY.md` - AI personality
- `USER.md` - Your preferences
- `TOOLS.md` - Tool descriptions
- `AGENTS.md` - Agent behavior

---

## 🧠 Memory System

Myrai features a sophisticated neural memory system:

**Neural Clusters:**
- Automatically groups related memories
- Semantic compression reduces token usage
- Contextual retrieval based on conversation

**Types of Memory:**
- **Facts** - User preferences, important information
- **Preferences** - Communication style, likes/dislikes
- **Tasks** - Active and completed tasks
- **Conversations** - Archived conversation summaries

**Memory Commands:**
```bash
# Search memories
myrai memory search "python projects"

# Add a memory
myrai memory add "I prefer dark mode in all apps"

# View memory health
myrai memory health
```

---

## 🔒 Security & Privacy

- ✅ **Local-First**: All data stays on your device in SQLite/BadgerDB
- ✅ **No Cloud**: No data sent to external servers (except LLM APIs)
- ✅ **Encrypted Storage**: Sensitive data encrypted at rest
- ✅ **Sandboxed Skills**: Custom skills run in restricted environment
- ✅ **No Telemetry**: No analytics, no tracking
- ✅ **Self-Hosted**: You control everything
- ✅ **Open Source**: MIT License, auditable code

**API Keys**: Stored locally in `~/.myrai/.env`, never transmitted except to your chosen LLM provider.

---

## 🏗️ Architecture

```
myrai/
├── cmd/myrai/              # Entry point
├── internal/
│   ├── agent/              # AI agent with tool use
│   ├── api/                # HTTP API + WebSocket
│   ├── app/                # Application lifecycle
│   ├── channels/           # Telegram, Discord
│   ├── circuitbreaker/     # Fault tolerance
│   ├── cli/                # CLI commands
│   ├── config/             # Configuration
│   ├── cron/               # Scheduled jobs
│   ├── errors/             # Error handling
│   ├── jobs/               # Background job scheduler
│   ├── llm/                # LLM client (20+ providers)
│   ├── mcp/                # Model Context Protocol
│   ├── metrics/            # Prometheus metrics
│   ├── neural/             # Neural memory clusters
│   ├── persona/            # Persona evolution
│   ├── reflection/         # Self-monitoring
│   ├── security/           # Security tools
│   ├── skills/             # 18+ skill implementations
│   ├── store/              # SQLite + BadgerDB
│   └── testutil/           # Test utilities
├── web/                    # Web UI
├── config/                 # Configuration templates
└── docs/                   # Documentation
```

---

## 📊 Testing & Quality

**Test Coverage:**
- 120+ integration tests
- 87.5% coverage on store package
- 99.4% coverage on metrics package
- 61.7% coverage on circuit breaker
- All tests passing ✅

**Run Tests:**
```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 🛠️ Commands Reference

```bash
# Setup
myrai onboard              # Interactive setup wizard
myrai doctor               # System health check

# Chat & Server
myrai --cli                # Interactive CLI chat
myrai -m "message"         # One-shot message
myrai server               # Start server (Web UI + API)
myrai server --port 3000   # Custom port

# Skills
myrai skills list          # List installed skills
myrai skills install <repo> # Install from GitHub
myrai skills enable <name>  # Enable skill
myrai skills watch <path>   # Hot-reload development

# MCP
myrai mcp discover         # Discover MCP servers
myrai mcp list             # List configured servers
myrai mcp start <name>     # Start MCP server

# Persona
myrai persona              # View current persona
myrai persona edit         # Edit persona
myrai persona proposals    # View evolution proposals
myrai persona apply <id>   # Apply evolution

# Memory
myrai memory search <query> # Search memories
myrai memory health        # Memory system health

# Configuration
myrai config get <key>     # Get config value
myrai config set <key> <val> # Set config value
myrai config edit          # Edit config file

# System
myrai status               # Show system status
myrai version              # Show version
myrai --help               # Show all commands
```

---

## 🐳 Deployment

### Docker

```bash
# Basic deployment
docker run -d \
  --name myrai \
  -p 8080:8080 \
  -v ~/.myrai:/app/data \
  -e OPENAI_API_KEY=your-key \
  ghcr.io/gmsas95/myrai:latest

# With multiple API keys
docker run -d \
  --name myrai \
  -p 8080:8080 \
  -v ~/.myrai:/app/data \
  -e OPENAI_API_KEY=sk-... \
  -e ANTHROPIC_API_KEY=sk-... \
  -e BRAVE_API_KEY=... \
  ghcr.io/gmsas95/myrai:latest
```

### Dokploy (VPS)

See [docs/DEPLOY_DOKPLOY.md](docs/DEPLOY_DOKPLOY.md) for detailed VPS deployment instructions.

Quick setup:
```bash
# On your VPS
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash
myrai onboard
myrai server
```

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Areas for contribution:**
- 🐛 Bug fixes
- 📱 Mobile app (Flutter/React Native)
- 🌍 Additional channels (WhatsApp, Signal, iMessage)
- 🏠 Smart home integrations
- 📧 Email skill
- 🎙️ Advanced voice features

---

## 🗺️ Roadmap

### ✅ Completed (v2.0)
- [x] Neural memory with clustering
- [x] Adaptive persona evolution
- [x] Skill runtime with hot-reload
- [x] MCP protocol support
- [x] Reflection engine
- [x] 18+ built-in skills
- [x] Multi-channel (CLI, Web, Telegram, Discord)
- [x] Circuit breaker & job scheduler
- [x] Comprehensive test suite

### 🚧 In Progress
- [ ] Bug fixes and polish
- [ ] Performance optimizations
- [ ] Documentation improvements

### 📅 Upcoming
- [ ] WhatsApp channel
- [ ] Email skill
- [ ] Mobile companion app
- [ ] Advanced voice (wake word, continuous)
- [ ] Visual Canvas (A2UI-style)
- [ ] Skills marketplace

---

## 📚 Documentation

- [Quick Start Guide](docs/QUICKSTART.md)
- [Usage Guide](docs/USAGE.md)
- [VPS Deployment](docs/DEPLOY_DOKPLOY.md)
- [Persona System](PERSONA_SYSTEM.md)
- [API Documentation](docs/QUICK_REFERENCE.md)

---

## 📝 License

MIT License - See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

Built with inspiration from:
- [OpenClaw](https://github.com/openclaw/openclaw) - Multi-channel AI assistant patterns
- [MCP](https://modelcontextprotocol.io) - Model Context Protocol standard
- **MemoryCore** - Knowledge retention concepts

---

> *"未来はここにある"*  
> *"The future is here"*

**Built with ❤️ for everyone.** 🚀

---

<p align="center">
  <a href="https://github.com/gmsas95/goclawde-cli">⭐ Star on GitHub</a> •
  <a href="https://github.com/gmsas95/goclawde-cli/issues">🐛 Report Bug</a> •
  <a href="https://github.com/gmsas95/goclawde-cli/discussions">💬 Discuss</a>
</p>

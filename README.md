# 🦞 GoClawde CLI

> A blazing-fast, self-hosted AI assistant in Go. Inspired by OpenClaw and PicoClaw, with a MemoryCore-inspired persona system.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue)](https://golang.org)
[![GitHub Release](https://img.shields.io/github/v/release/gmsas95/goclawde-cli)](https://github.com/gmsas95/goclawde-cli/releases)

**GoClawde** (Go + Claw + Claude) is an ultra-lightweight, self-hosted AI assistant that combines the best of:
- **[PicoClaw](https://github.com/gmsas95/picoclaw)** - Extreme efficiency (<10MB RAM, $10 hardware)
- **[OpenClaw](https://github.com/openclaw/openclaw)** - Rich features and multi-channel support
- **[MemoryCore](https://github.com/Kiyoraka/Project-AI-MemoryCore)** - Persistent persona and memory system

All in a **single 50MB binary** with **zero dependencies**.

---

## ✨ What Makes GoClawde Different?

| Feature | GoClawde | PicoClaw | OpenClaw |
|---------|----------|----------|----------|
| **Binary Size** | ~50MB | ~10MB | ~2GB (Node) |
| **Memory** | ~25MB | <10MB | >1GB |
| **Startup** | 50ms | 1s | Minutes |
| **Multi-LLM Support** | ✅ Kimi, OpenAI, Anthropic, Ollama | ❌ | ⚠️ Limited |
| **Persona System** | ✅ Markdown-based + Caching | ❌ | ✅ SOUL.md, IDENTITY.md |
| **Time Awareness** | ✅ Built-in | ❌ | ❌ |
| **Project Context** | ✅ LRU Management | ❌ | ⚠️ Sessions |
| **Persistent Memory** | ✅ SQLite + BadgerDB | ⚠️ Files | ✅ Markdown-based |
| **Vector Search** | ✅ Multi-provider embeddings | ❌ | ❌ |
| **Multi-Channel** | ✅ Web, CLI, Discord, Telegram | ⚠️ CLI | ✅ 10+ channels |
| **Cron Jobs** | ✅ Scheduled automation | ❌ | ❌ |
| **MCP Server** | ✅ SSE streaming | ❌ | ❌ |
| **Batch Processing** | ✅ Tier 3 optimized (200 concurrent) | ❌ | ❌ |
| **CLI Commands** | ✅ OpenClaw-style (status, doctor, skills) | ❌ | ✅ |

---

## 🚀 Installation

### Option 1: Pre-built Binary (Recommended)

```bash
# Linux/macOS
curl -L https://github.com/gmsas95/goclawde-cli/releases/latest/download/goclawde-$(uname -s)-$(uname -m) -o goclawde
chmod +x goclawde
sudo mv goclawde /usr/local/bin/

# Run onboarding wizard
goclawde onboard
```

### Option 2: Build from Source

```bash
# Prerequisites: Go 1.23+
git clone https://github.com/gmsas95/goclawde-cli.git
cd goclawde-cli

# Build
make build

# Install locally (~/.local/bin, no sudo needed)
make install-local

# Or install system-wide (requires sudo)
sudo make install
```

### Option 3: Docker

```bash
docker run -d \
  --name goclawde \
  -p 8080:8080 \
  -v ~/.goclawde:/app/data \
  -e GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY="sk-..." \
  ghcr.io/gmsas95/goclawde-cli:latest
```

---

## 🎮 Quick Start

### 1. First Run (Onboarding)

```bash
$ goclawde

🤖 Welcome to GoClawde!
It looks like this is your first time running GoClawde.
Let's set up your personal AI assistant.

Run onboarding wizard? (Y/n): Y
```

The wizard will guide you through:
1. **Workspace setup** - Create your data directory (`~/.goclawde/`)
2. **User profile** - Your preferences and communication style
3. **LLM Provider selection** - Choose from Kimi, OpenAI, Anthropic, or Ollama
4. **API configuration** - Enter your API key for the selected provider
5. **Integrations** - Optional Telegram/Discord setup
6. **Persona setup** - AI personality and behavior

#### Supported LLM Providers

| Provider | Setup | Models |
|----------|-------|--------|
| **Kimi (Moonshot)** | API key from [platform.moonshot.cn](https://platform.moonshot.cn) | kimi-k2.5, kimi-k2.5-long |
| **OpenAI** | API key from [platform.openai.com](https://platform.openai.com) | gpt-4o, gpt-4o-mini, gpt-4-turbo |
| **Anthropic** | API key from [console.anthropic.com](https://console.anthropic.com) | claude-3-5-sonnet, claude-3-opus, claude-3-haiku |
| **Ollama** | Local installation, no API key needed | llama3.2, llama3.1, mistral, codellama |

### 2. Check Status & Diagnostics

```bash
# Quick status overview
goclawde status

# Run diagnostics
goclawde doctor
```

### 3. Interactive CLI Mode

```bash
$ goclawde --cli

🤖 GoClawde - Interactive Mode
Type 'exit' or 'quit' to exit, 'help' for commands

👤 You: Hello!
🤖 GoClawde: Good evening! How can I help you today?
```

### 4. One-shot Mode

```bash
$ goclawde -m "Explain Go channels in simple terms"

🤖 GoClawde is thinking...

Go channels are like pipes that let goroutines (lightweight threads) 
communicate and synchronize with each other...
```

### 5. Start Server

```bash
# Start server in foreground
goclawde gateway run

# Or run in background
goclawde gateway run &

# Access web UI at http://localhost:8080
```

---

## 🛠️ CLI Commands

GoClawde provides an OpenClaw-style command interface:

### Setup & Configuration
```bash
goclawde onboard                  # Run setup wizard
goclawde config get <key>         # Get config value (e.g., llm.default_provider)
goclawde config set <key> <val>   # Set configuration value
goclawde config edit              # Open config in $EDITOR
goclawde config path              # Show config file location
goclawde config show              # Display full config
```

### Server Management
```bash
goclawde gateway run              # Start server (foreground)
goclawde gateway status           # Show gateway configuration
goclawde channels status          # Show Telegram/Discord status
```

### System & Diagnostics
```bash
goclawde status                   # Show current status
goclawde doctor                   # Run diagnostics
goclawde version                  # Show version
```

### Skills
```bash
goclawde skills                   # List available skills
goclawde skills info <skill>      # Show skill details (e.g., goclawde skills info weather)
```

### Project Management
```bash
goclawde project new <name> <type>   # Create new project (coding, writing, research, business)
goclawde project list                # List all projects
goclawde project switch <name>       # Switch to project
goclawde project archive <name>      # Archive a project
goclawde project delete <name>       # Delete a project
```

### Batch Processing
```bash
goclawde batch -i <file>             # Process prompts from file
goclawde batch -i in.txt -o out.json # Process and save results
```

### Persona & User
```bash
goclawde persona            # Show current AI identity
goclawde persona edit       # Edit AI identity
goclawde user               # Show your profile
goclawde user edit          # Edit your profile
```

---

## 🧠 Persona System

GoClawde features a markdown-based persona system (originally created by MemoryCore, popularized by OpenClaw's SOUL.md) that makes your AI assistant truly personal:

### Core Files

Your workspace (`~/.goclawde/`) contains:

```
~/.goclawde/
├── goclawde.yaml        # Main configuration
├── .env                 # Environment variables
├── IDENTITY.md          # AI personality, voice, values
├── USER.md              # Your preferences and profile
├── TOOLS.md             # Tool descriptions
├── AGENTS.md            # Behavior guidelines
└── projects/            # Project contexts
```

### Time Awareness

GoClawde automatically includes time context:

```
Current time: Saturday, February 15, 2026 8:45 PM
Time of day: evening
Context: It's evening. The user is likely focused on personal projects.
```

### Project Management

Create projects to maintain isolated contexts:

```bash
# Create a coding project
goclawde project new "API Refactor" coding

# List projects
goclawde project list

# Switch to a project
goclawde project switch "API Refactor"

# Archive completed projects
goclawde project archive "Old Project"
```

**LRU Management**: Automatically keeps 10 active projects, archives oldest.

### Editing Persona

```bash
# Edit AI identity
goclawde persona edit

# Edit your profile
goclawde user edit
```

---

## 💬 Multi-Channel Support

### Discord Bot

```yaml
channels:
  discord:
    enabled: true
    token: "${DISCORD_BOT_TOKEN}"
    allow_dm: true
    # Optional: restrict to specific channels
    # channels: ["channel-id-1", "channel-id-2"]
```

Features:
- Responds to mentions (@GoClawde) and DMs
- Typing indicators while processing
- Automatic message splitting for long responses
- Commands: `/help`, `/new`, `/status`, `/ping`

### Telegram Bot

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    # Optional: allowlist specific users
    # allowlist: [123456789]
```

### Web UI

Access the web interface at `http://localhost:8080` when running in server mode.

---

## 🤖 MCP Server

GoClawde includes an MCP (Model Context Protocol) server for external tool integration:

```yaml
mcp:
  enabled: true
  host: "0.0.0.0"
  port: 8081
```

The MCP server exposes tools via SSE streaming at `http://localhost:8081/mcp`.

---

## ⏰ Cron Jobs

Schedule automated tasks:

```yaml
cron:
  enabled: true
  interval_minutes: 5
  max_concurrent: 3
```

Create scheduled jobs via API:
```bash
curl -X POST http://localhost:8080/api/cron/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily-report",
    "prompt": "Generate daily summary",
    "schedule": "@daily"
  }'
```

Supported schedules: `30m`, `1h`, `@hourly`, `@daily`, `@weekly`, cron expressions

---

## 🔍 Vector Search

Semantic memory search using embeddings:

```yaml
vector:
  enabled: true
  provider: "openai"  # or "ollama" for local
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-3-small"
```

API endpoints:
- `POST /api/search` - Semantic search across memories
- `POST /api/memories/:id/index` - Index a memory

---

## ⚡ Batch Processing

High-throughput batch processing optimized for Tier 3 rate limits:

```bash
# Process with Tier 3 limits (200 concurrent, 5000 RPM)
goclawde batch -i prompts.jsonl --tier 3 -o results.json

# Available tiers: 3, 4, 5
```

Features:
- Token bucket rate limiting
- Progress tracking with ETA
- Checkpoint/resume (survives crashes)
- Supports .txt and .jsonl formats

---

## 🔧 Configuration

### Minimal Config (`~/.goclawde/goclawde.yaml`)

```yaml
llm:
  default_provider: kimi
  providers:
    kimi:
      api_key: "sk-your-api-key"
      model: "kimi-k2.5"
      base_url: "https://api.moonshot.cn/v1"

storage:
  data_dir: "~/.goclawde"
```

### Multi-Provider Configuration

```yaml
llm:
  default_provider: openai
  providers:
    openai:
      api_key: "sk-..."
      model: "gpt-4o"
      base_url: "https://api.openai.com/v1"
    anthropic:
      api_key: "sk-ant-..."
      model: "claude-3-5-sonnet-20241022"
      base_url: "https://api.anthropic.com/v1"
    ollama:
      api_key: "ollama"
      model: "llama3.2"
      base_url: "http://localhost:11434/v1"
```

### Environment Variables

```bash
# Provider-specific API keys
export GOCLAWDE_LLM_PROVIDERS_KIMI_API_KEY="sk-..."
export GOCLAWDE_LLM_PROVIDERS_OPENAI_API_KEY="sk-..."
export GOCLAWDE_LLM_PROVIDERS_ANTHROPIC_API_KEY="sk-ant-..."

# Channel tokens
export TELEGRAM_BOT_TOKEN="..."
export DISCORD_BOT_TOKEN="..."
```

### Full Configuration

See [CONFIGURATION.md](docs/CONFIGURATION.md) for all options.

---

## 🧩 Skills System

Built-in skills:

| Skill | Description | Example |
|-------|-------------|---------|
| `github` | Repository operations | "Search for Go web frameworks" |
| `weather` | Weather forecasts (wttr.in + Open-Meteo) | "What's the weather in Tokyo?" |
| `notes` | Note management | "Take a note: call mom tomorrow" |
| `vision` | Image analysis | "Describe this image" |
| `system` | File operations, shell commands | Built-in |
| `browser` | Web automation (requires Chrome) | "Navigate to example.com" |

### Weather Skill

The weather skill uses multiple sources for reliability:
- **Primary**: wttr.in (simple text format)
- **Fallback**: Open-Meteo API (JSON, no API key needed)

```bash
# Check weather
goclawde -m "What's the weather in Kuala Lumpur?"
```

---

## ⚡ Performance

Optimized for speed:

| Metric | Value |
|--------|-------|
| Binary Size | ~50MB |
| Memory (idle) | ~25MB |
| Startup Time | ~50ms |
| Response Time | <1s (API dependent) |
| Concurrent Chats | 100+ |

### Optimizations

- ✅ WAL mode SQLite with connection pooling
- ✅ Database indexes on hot paths
- ✅ System prompt caching (no file I/O per request)
- ✅ Optimized BadgerDB (disabled logging, reduced versions)
- ✅ Memory preallocation for messages
- ✅ Pure Go SQLite driver (no CGO)

---

## 🗺️ Roadmap

- ✅ **v0.3** - Discord bot, MCP server, cron jobs, vector search, batch processing, multi-LLM support
- 🔄 **v0.4** - Slack integration, web browsing skill, memory visualization
- 🔄 **v0.5** - Multi-LLM routing, skill marketplace, agent workflows
- 🔄 **v1.0** - Mobile apps, team collaboration, enterprise features

---

## 🏗️ Architecture

```
GoClawde (~50MB single binary)
├── Persona System (IDENTITY.md, USER.md)
├── Project Manager (LRU, context switching)
├── Time Awareness (dynamic greetings)
├── SQLite (conversations, WAL mode)
├── BadgerDB (sessions, optimized)
├── HTTP API + WebSocket (Go/Fiber)
├── MCP Server (SSE streaming)
├── Agent Runtime (goroutines)
├── Tool System (file, shell, web)
├── Skills Registry (extensible)
│   ├── GitHub
│   ├── Weather (multi-source)
│   ├── Notes
│   ├── Browser (ChromeDP)
│   └── Vision (image analysis)
├── Cron Runner (scheduled jobs)
├── Vector Search (semantic memory)
├── Batch Processor (high-throughput)
├── Multi-Channel (Discord, Telegram, Web)
└── Multi-LLM (Kimi, OpenAI, Anthropic, Ollama)
```

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Clone
git clone https://github.com/gmsas95/goclawde-cli.git
cd goclawde-cli

# Build
make build

# Test
make test

# Run
goclawde --help
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file.

---

## 🙏 Acknowledgments

**History:** The markdown-based persona system was originally created by a Malaysian developer (MemoryCore) for local use before being publicized. OpenClaw brought the concept to wider attention with their implementation.

- **[MemoryCore](https://github.com/Kiyoraka/Project-AI-MemoryCore)** - Original creator of the markdown-based persona system concept (local/private use, not commercialized)
- **[OpenClaw](https://github.com/openclaw/openclaw)** - First to publicize and popularize the markdown-based persona system (SOUL.md, IDENTITY.md, USER.md, AGENTS.md)
- **[PicoClaw](https://github.com/gmsas95/picoclaw)** - Inspiration for performance optimization and Go implementation
- **[nanobot](https://github.com/HKUDS/nanobot)** - Original concept by HKUDS

Built with Go, SQLite, BadgerDB, and ❤️

---

**Star ⭐ this repo if you find it useful!**

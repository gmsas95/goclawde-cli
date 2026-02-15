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
| **Persona System** | ✅ Markdown-based + Caching | ❌ | ✅ SOUL.md, IDENTITY.md |
| **Time Awareness** | ✅ Built-in | ❌ | ❌ |
| **Project Context** | ✅ LRU Management | ❌ | ⚠️ Sessions |
| **Persistent Memory** | ✅ SQLite + BadgerDB | ⚠️ Files | ✅ Markdown-based |
| **Multi-Channel** | ✅ Web, CLI, Telegram | ⚠️ CLI | ✅ 10+ channels |

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

# Or with Go directly
go build -o bin/goclawde ./cmd/jimmy

# Install
sudo cp bin/goclawde /usr/local/bin/
```

### Option 3: Docker

```bash
docker run -d \
  --name goclawde \
  -p 8080:8080 \
  -v ~/.goclawde:/app/data \
  -e JIMMY_LLM_PROVIDERS_KIMI_API_KEY="sk-..." \
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

The wizard will:
1. Create your workspace (`~/.goclawde/`)
2. Configure API keys
3. Set up your AI persona
4. Create your user profile

### 2. Interactive CLI Mode

```bash
$ goclawde --cli

🤖 GoClawde - Interactive Mode
Type 'exit' or 'quit' to exit, 'help' for commands

👤 You: Hello!
🤖 GoClawde: Good evening! How can I help you today?
```

### 3. One-shot Mode

```bash
$ goclawde -m "Explain Go channels in simple terms"

🤖 GoClawde is thinking...

Go channels are like pipes that let goroutines (lightweight threads) 
communicate and synchronize with each other...
```

### 4. Web Interface

```bash
$ goclawde --server

🌐 Server started at http://localhost:8080
```

---

## 🧠 Persona System

GoClawde features a markdown-based persona system (inspired by OpenClaw's SOUL.md and MemoryCore) that makes your AI assistant truly personal:

### Core Files

Your workspace (`~/.goclawde/`) contains:

```
~/.goclawde/
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

### Telegram Bot

```bash
# 1. Get token from @BotFather
# 2. Add to config:
```

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
```

### Web UI

Access the web interface at `http://localhost:8080` when running in server mode.

---

## 🔧 Configuration

### Minimal Config (`~/.goclawde/jimmy.yaml`)

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

### Environment Variables

```bash
export JIMMY_LLM_PROVIDERS_KIMI_API_KEY="sk-..."
export TELEGRAM_BOT_TOKEN="..."
```

### Full Configuration

See [CONFIGURATION.md](docs/CONFIGURATION.md) for all options.

---

## 🧩 Skills System

Built-in skills:

| Skill | Description | Example |
|-------|-------------|---------|
| `github` | Repository operations | "Search for Go web frameworks" |
| `weather` | Weather forecasts | "What's the weather in Tokyo?" |
| `notes` | Note management | "Take a note: call mom tomorrow" |
| `system` | File operations, shell commands | Built-in |

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

- ✅ **v0.3** - Persona system, project management, Telegram bot
- 🔄 **v0.4** - Discord/Slack integration, cron jobs, more skills
- 🔄 **v0.5** - Vector memory (RAG), MCP protocol, web browsing
- 🔄 **v1.0** - Multi-LLM support, skill marketplace, mobile apps

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
├── Agent Runtime (goroutines)
├── Tool System (file, shell, web)
├── Skills Registry (extensible)
└── Telegram Bot (multi-channel)
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
./bin/goclawde --help
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file.

---

## 🙏 Acknowledgments

- **[OpenClaw](https://github.com/openclaw/openclaw)** - Inspiration for features, UX, and the markdown-based persona system (SOUL.md, IDENTITY.md, USER.md, AGENTS.md)
- **[PicoClaw](https://github.com/gmsas95/picoclaw)** - Inspiration for performance optimization and Go implementation
- **[MemoryCore](https://github.com/Kiyoraka/Project-AI-MemoryCore)** - Inspiration for time-awareness and project context features
- **[nanobot](https://github.com/HKUDS/nanobot)** - Original concept by HKUDS

Built with Go, SQLite, BadgerDB, and ❤️

---

**Star ⭐ this repo if you find it useful!**

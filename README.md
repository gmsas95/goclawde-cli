# Myrai (未来)

> **Myrai** (未来) means "future" in Japanese.  
> **Myrai** (My + AI) means "my personal AI".  
> **Myrai** is the future of personal assistance.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://golang.org)
[![GitHub release](https://img.shields.io/github/v/release/gmsas95/goclawde-cli?include_prereleases)](https://github.com/gmsas95/goclawde-cli/releases)

**Myrai** is a lightweight, local-first personal AI assistant for everyone.

Not just a coding assistant. Not just a terminal tool. **A life assistant.**

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🤖 **20+ LLM Providers** | OpenAI, Anthropic, Google, Groq, DeepSeek, Ollama, and more |
| 💬 **Multi-Channel** | CLI, Web UI, Telegram, Discord |
| 🧠 **15+ Skills** | Tasks, Calendar, Notes, Health, Shopping, Documents, Weather, GitHub |
| 🔒 **Privacy First** | Local-first, your data stays on your device |
| 🚀 **Easy Setup** | One-command installation via curl or npm |
| 🐳 **Docker Ready** | Deploy with a single command |
| 📦 **Single Binary** | ~50MB, no dependencies |

---

## 🆚 Comparison

| Feature | Myrai | Siri/Alexa | ChatGPT |
|---------|-------|------------|---------|
| **Target** | Everyone | Consumers | Everyone |
| **Privacy** | ✅ Local-first | Cloud | Cloud |
| **Memory** | ✅ Knowledge graph | Session-only | Session-only |
| **Multi-LLM** | ✅ 20+ providers | Locked | Locked |
| **Self-host** | ✅ Easy | No | No |
| **Open Source** | ✅ MIT | No | No |
| **Setup** | ✅ 2 minutes | Easy | Easy |
| **Size** | ✅ ~50MB | N/A | N/A |

---

## 🚀 Quick Start

### Installation

**curl (Recommended)**
```bash
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash
```

**npm**
```bash
npm install -g myrai
# or use without installing
npx myrai --help
```

**Docker**
```bash
docker run -d \
  --name myrai \
  -p 8080:8080 \
  -v ~/.myrai:/app/data \
  -e OPENAI_API_KEY=your-key \
  ghcr.io/gmsas95/myrai:latest
```

**Build from Source**
```bash
git clone https://github.com/gmsas95/goclawde-cli.git
cd goclawde-cli
make build
sudo make install
```

### First Run

```bash
# Interactive setup wizard
myrai onboard

# Start the server
myrai server

# Or use CLI mode
myrai --cli
```

---

## 🤖 Supported LLM Providers

### Cloud (Recommended)
| Provider | Models | API Key Env |
|----------|--------|-------------|
| OpenAI | GPT-4o, GPT-4-turbo, o1 | `OPENAI_API_KEY` |
| Anthropic | Claude 3.5 Sonnet, Claude 3 Opus | `ANTHROPIC_API_KEY` |
| Google | Gemini 2.0 Flash, Gemini Pro | `GOOGLE_API_KEY` |
| Kimi/Moonshot | kimi-k2.5 | `KIMI_API_KEY` |

### Fast & Affordable
| Provider | Models | Notes |
|----------|--------|-------|
| Groq | Llama 3.3 70B | Ultra-fast inference |
| DeepSeek | DeepSeek Chat, Reasoner | Great for coding |
| Together AI | Llama, Mistral, Qwen | Many open models |
| Cerebras | Llama 3.1 | Fastest inference |

### Model Aggregators
| Provider | Models | Notes |
|----------|--------|-------|
| OpenRouter | 100+ models | One API for all |
| Fireworks | Llama, Qwen | Serverless inference |

### Chinese Providers
| Provider | Models | Notes |
|----------|--------|-------|
| Zhipu (智谱) | GLM-4 Plus | Chinese support |
| SiliconFlow | Qwen, DeepSeek | Affordable |
| Novita | Llama, Mistral | Budget-friendly |

### Local/Self-Hosted
| Provider | Models | Notes |
|----------|--------|-------|
| Ollama | Llama, Mistral, Qwen | No API key needed |
| LocalAI | Any GGUF model | OpenAI-compatible |
| vLLM | Any HF model | High performance |

---

## 💬 Channels

### CLI Mode
```bash
# Interactive chat
myrai --cli

# One-shot message
myrai -m "What's the weather in Tokyo?"
```

### Web UI
```bash
# Start server with web interface
myrai server
# Open http://localhost:8080
```

### Telegram
```bash
# Set bot token
export TELEGRAM_BOT_TOKEN=your-token

# Start server (Telegram auto-enabled)
myrai server
```

### Discord
```bash
# Set bot token
export DISCORD_BOT_TOKEN=your-token

# Start server
myrai server
```

---

## 🧠 Skills (15+ Built-in)

| Skill | Description |
|-------|-------------|
| **Tasks** | Task management with priorities and due dates |
| **Calendar** | Google Calendar integration |
| **Notes** | Personal notes with search |
| **Documents** | PDF/DOCX processing with OCR |
| **Health** | Health tracking and reminders |
| **Shopping** | Shopping lists with categories |
| **Expenses** | Expense tracking and analysis |
| **Weather** | Weather forecasts and alerts |
| **GitHub** | Repository management |
| **Browser** | Web browsing and scraping |
| **Intelligence** | AI-powered analysis |
| **Knowledge** | Knowledge base management |
| **Voice** | Speech-to-text and text-to-speech |
| **Vision** | Image analysis and OCR |
| **Agentic** | Code analysis, Git operations |

---

## 📖 CLI Commands

```bash
# Setup & Configuration
myrai onboard              # Run setup wizard
myrai config get <key>     # Get config value
myrai config set <key> <val>  # Set config value
myrai config edit          # Edit config file

# Project Management
myrai project new <name> <type>  # Create project
myrai project list               # List projects
myrai project switch <name>      # Switch project

# Server
myrai server               # Start server
myrai gateway status       # Show gateway status

# Persona & User
myrai persona              # Show AI identity
myrai persona edit         # Edit AI personality
myrai user                 # Show your profile
myrai user edit            # Edit your profile

# System
myrai doctor               # Run diagnostics
myrai status               # Show system status
myrai version              # Show version
```

---

## 🏗️ Architecture

```
myrai-cli/
├── cmd/myrai/           # Entry point (~170 lines)
├── internal/
│   ├── app/             # App lifecycle, server, CLI
│   ├── api/             # HTTP API + WebSocket
│   ├── agent/           # AI agent with tool calling
│   ├── llm/             # LLM client (20+ providers)
│   ├── skills/          # 15+ skill implementations
│   ├── channels/        # Telegram, Discord, etc.
│   ├── store/           # SQLite + BadgerDB
│   ├── vector/          # Vector search
│   ├── config/          # Configuration management
│   ├── onboarding/      # Setup wizard
│   ├── cli/             # CLI command handlers
│   ├── interfaces/      # Public interfaces
│   └── errors/          # Error types
├── web/                 # Web UI (HTML/CSS/JS)
├── npm/                 # npm package wrapper
└── scripts/             # Install scripts
```

---

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Node.js 14+ (for npm package)
- Docker (optional)

### Build & Test

```bash
# Install dependencies
make deps

# Build
make build

# Run tests
make test-all

# Run in development
make dev
```

### Run Tests

```bash
make test-unit         # Unit tests
make test-smoke        # Smoke tests
make test-integration  # Integration tests
make test-all          # All tests
```

---

## 📦 Release

```bash
# Build release binaries
make release VERSION=0.1.0

# Create GitHub release
make release-gh VERSION=0.1.0

# Publish to npm
make publish-npm
```

See [RELEASE.md](RELEASE.md) for details.

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas We Need Help
- 📱 Mobile app (Flutter)
- 🌐 Better Web UI
- 📧 Email integration
- 🏠 Smart home connectors
- 🌍 Internationalization

---

## 🗺️ Roadmap

### Phase 1: Foundation ✅
- [x] Multi-provider LLM support (20+ providers)
- [x] CLI and server modes
- [x] Telegram and Discord channels
- [x] 15+ skills
- [x] Setup wizard

### Phase 2: Memory ✅
- [x] SQLite storage
- [x] Vector search
- [x] Knowledge graph

### Phase 3: Intelligence 🚧
- [ ] Proactive suggestions
- [ ] Pattern recognition
- [ ] Automated workflows

### Phase 4: Mobile 📱
- [ ] Flutter mobile app
- [ ] iOS and Android

---

## 🔒 Privacy

- ✅ **Local-First**: Data stays on your device
- ✅ **No Cloud Required**: Works offline with Ollama
- ✅ **Open Source**: Audit the code yourself
- ✅ **Encrypted Storage**: Your data is encrypted
- ✅ **Export Anytime**: You own your data

---

## 📞 Connect

- **GitHub**: [github.com/gmsas95/goclawde-cli](https://github.com/gmsas95/goclawde-cli)
- **Issues**: [Report a bug](https://github.com/gmsas95/goclawde-cli/issues)
- **Discussions**: [Join the conversation](https://github.com/gmsas95/goclawde-cli/discussions)

---

## 📝 License

MIT License - See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

Inspired by:
- [OpenClaude](https://github.com/openclaw/openclaw) - Agentic patterns
- [PicoClaw](https://github.com/gmsas95/picoclaw) - Lightweight philosophy
- [Anthropic](https://www.anthropic.com) - Claude AI
- [OpenAI](https://openai.com) - GPT models

---

> *"未来はここにある"*  
> *"The future is here"*

**Built with ❤️ for everyone.** 🚀

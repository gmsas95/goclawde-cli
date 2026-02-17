# Myrai (未来) - Public Beta

> **Myrai** (未来) means "future" in Japanese.  
> **Myrai** (My + AI) means "my personal AI".  
> **Myrai** is the future of personal assistance.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://golang.org)
[![GitHub release](https://img.shields.io/github/v/release/gmsas95/goclawde-cli?include_prereleases)](https://github.com/gmsas95/goclawde-cli/releases)
[![Beta](https://img.shields.io/badge/status-beta-orange.svg)](https://github.com/gmsas95/goclawde-cli)

**⚠️ PUBLIC BETA**: This software is in beta. Expect bugs and rough edges. Your feedback helps us improve!

**Myrai** is a lightweight, local-first personal AI assistant for everyone.

Not just a coding assistant. Not just a terminal tool. **A life assistant.**

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🤖 **20+ LLM Providers** | OpenAI, Anthropic, Google, Groq, DeepSeek, Ollama, and more |
| 🌐 **Real-time Web Search** | Get current news, weather, stock prices - never out of date |
| 💬 **Multi-Channel** | CLI, Web UI, Telegram, Discord |
| 🧠 **15+ Skills** | Tasks, Calendar, Notes, Health, Shopping, Documents, Weather, GitHub |
| 🔒 **Privacy First** | Local-first, your data stays on your device |
| 🚀 **Easy Setup** | One-command installation via curl or npm |
| 🐳 **Docker Ready** | Deploy with a single command |
| 📦 **Single Binary** | ~50MB, no dependencies |
| 🧠 **Smart Memory** | Remembers your preferences and facts across conversations |

---

## 🆚 Comparison

| Feature | Myrai | Siri/Alexa | ChatGPT |
|---------|-------|------------|---------|
| **Target** | Everyone | Consumers | Everyone |
| **Privacy** | ✅ Local-first | ❌ Cloud | ❌ Cloud |
| **Memory** | ✅ Persistent | ❌ Session-only | ❌ Session-only |
| **Multi-LLM** | ✅ 20+ providers | ❌ Locked | ❌ Locked |
| **Self-host** | ✅ Easy | ❌ No | ❌ No |
| **Open Source** | ✅ MIT | ❌ No | ❌ No |
| **Web Search** | ✅ Built-in | ⚠️ Limited | ⚠️ ChatGPT Plus only |
| **API Costs** | ✅ You control | N/A | N/A |

---

## 🚀 Quick Start (5 Minutes)

### What You Need

Before installing, you'll need:
- **At least one LLM API key** (pick one):
  - [OpenAI](https://platform.openai.com) - Most popular, reliable
  - [Anthropic](https://console.anthropic.com) - Claude, great reasoning
  - [Groq](https://console.groq.com) - Fast & affordable
  - [DeepSeek](https://platform.deepseek.com) - Great for coding
  - [Ollama](https://ollama.com) - **Free**, runs locally (no API key!)
- Optional: [Brave Search API](https://api.search.brave.com) key for web search

### Installation

**macOS/Linux (curl)**
```bash
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash
```

**npm (cross-platform)**
```bash
npm install -g myrai
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

**Windows**
Download the `.exe` from [GitHub Releases](https://github.com/gmsas95/goclawde-cli/releases)

### First Run

```bash
# Run the interactive setup wizard
myrai onboard

# Start chatting!
myrai --cli
```

The **onboarding wizard** will guide you through:
1. ✨ Choosing your LLM provider
2. 🔑 Entering your API key
3. 🌐 Setting up web search (optional)
4. 👤 Creating your profile

---

## 💰 Cost Considerations

**Myrai is free to use**, but you pay for LLM API calls:

| Provider | Cost | Free Tier |
|----------|------|-----------|
| **Ollama** | **FREE** | Unlimited (runs locally) |
| **Groq** | ~$0.0001/1K tokens | $10-50 credits |
| **DeepSeek** | Very cheap | $10 credits |
| **OpenAI** | Standard | $5 credits |
| **Anthropic** | Standard | $5 credits |

**Typical usage**: $1-5/month for casual use, $10-20/month for heavy use.

**Web Search**:
- **Brave Search**: 2,000 queries/month FREE
- **DuckDuckGo**: FREE (less reliable)
- **Serper**: 2,500 queries FREE

---

## 🤖 Supported LLM Providers

### Cloud (Easy Setup)
| Provider | Best For | Get API Key |
|----------|----------|-------------|
| **OpenAI** | General use, reliable | [Get Key](https://platform.openai.com) |
| **Anthropic** | Reasoning, analysis | [Get Key](https://console.anthropic.com) |
| **Google** | Multilingual | [Get Key](https://aistudio.google.com) |
| **Kimi** | Long context | [Get Key](https://platform.moonshot.cn) |

### Fast & Affordable
| Provider | Best For | Get API Key |
|----------|----------|-------------|
| **Groq** | Speed | [Get Key](https://console.groq.com) |
| **DeepSeek** | Coding | [Get Key](https://platform.deepseek.com) |
| **Together AI** | Open models | [Get Key](https://api.together.xyz) |
| **Cerebras** | Fast inference | [Get Key](https://cerebras.ai) |

### Free / Self-Hosted
| Provider | Best For | Notes |
|----------|----------|-------|
| **Ollama** | Privacy | [Download](https://ollama.com) - Completely FREE |
| **LocalAI** | Flexibility | Run any model locally |
| **vLLM** | Performance | For advanced users |

---

## 🌐 Web Search Feature

Myrai can search the web for **real-time information**:

```
You: "What's the latest news about AI?"
Myrai: *searches web* "Here are the latest AI developments..."

You: "Current Bitcoin price"
Myrai: *searches web* "Bitcoin is currently at $67,450..."

You: "Weather in Tokyo tomorrow"
Myrai: *searches web* "Tomorrow in Tokyo: 22°C, partly cloudy..."
```

**Setup**: During `myrai onboard`, choose to enable web search and enter your API key.

**Benefits**:
- ✅ Never outdated information
- ✅ Current events and news
- ✅ Live data (weather, stocks, sports)
- ✅ Recent developments

---

## 💬 How to Use

### CLI Mode (Recommended for daily use)
```bash
# Interactive chat
myrai --cli

# One-shot question
myrai -m "What's the weather in Tokyo?"

# Pipe input
echo "Explain quantum computing" | myrai
```

### Web UI
```bash
# Start server
myrai server

# Open http://localhost:8080 in your browser
```

### Telegram Bot
```bash
# 1. Message @BotFather on Telegram, create a bot, copy the token
# 2. Run setup:
myrai onboard
# 3. Start server
myrai server
# 4. Chat with your bot on Telegram!
```

---

## 🧠 What Can Myrai Do?

### Personal Assistant
- ✅ Manage tasks and reminders
- ✅ Track your health and medications
- ✅ Create shopping lists
- ✅ Take notes and documents
- ✅ Remember your preferences

### Knowledge Worker
- ✅ Analyze documents (PDF, images)
- ✅ Search the web for current info
- ✅ Manage GitHub repositories
- ✅ Track expenses
- ✅ Calendar integration

### Developer Tools
- ✅ Read and write files
- ✅ Execute commands (safely)
- ✅ Git operations
- ✅ Code analysis
- ✅ Web scraping

---

## 🔒 Privacy & Security

- ✅ **Local-First**: All your data stays on your device
- ✅ **No Data Sharing**: We don't collect or sell your data
- ✅ **Encrypted Storage**: Your data is encrypted locally
- ✅ **Open Source**: You can audit the code
- ✅ **Self-Hosted**: You control everything
- ✅ **No Lock-in**: Export your data anytime

**Your API keys** are stored locally in `~/.myrai/.env` and never leave your machine.

---

## 🛠️ Commands Reference

```bash
# Setup
myrai onboard              # Run setup wizard
myrai doctor               # Check system health

# Chat
myrai --cli                # Interactive chat
myrai -m "message"         # One-shot message

# Server
myrai server               # Start server (Web UI + API)
myrai server --port 3000   # Custom port

# Configuration
myrai config get <key>     # Get config value
myrai config set <key> <val>  # Set config value
myrai config edit          # Edit config file

# Personalization
myrai persona              # View AI personality
myrai persona edit         # Customize AI personality
myrai user                 # View your profile
myrai user edit            # Edit your preferences

# Projects
myrai project new <name> <type>  # Create project
myrai project list               # List projects
myrai project switch <name>      # Switch project

# System
myrai status               # Show system status
myrai version              # Show version
myrai --help               # Show all commands
```

---

## 🚨 Troubleshooting

### "No API key configured"
Run `myrai onboard` to set up your LLM provider and API key.

### "Web search not working"
You need a search API key:
1. Get free key from [Brave Search](https://api.search.brave.com) (2,000 queries/month)
2. Run `myrai onboard` and enable web search
3. Or set: `export MYRAI_SEARCH_API_KEY=your_key`

### "Permission denied" errors
Myrai respects your system permissions. Use `sudo` only if necessary, or adjust file permissions.

### Outdated information
Enable web search during onboarding to get real-time information.

### High API costs
- Use **Ollama** for free local inference
- Switch to cheaper providers (Groq, DeepSeek)
- Set spending limits in your provider dashboard

### Getting Help
```bash
myrai doctor          # Run diagnostics
myrai --help          # Show help
```

Or [open an issue](https://github.com/gmsas95/goclawde-cli/issues) on GitHub.

---

## 🏗️ Architecture

```
myrai-cli/
├── cmd/myrai/           # Entry point
├── internal/
│   ├── app/             # App lifecycle
│   ├── api/             # HTTP API + WebSocket
│   ├── agent/           # AI agent with tools
│   ├── llm/             # LLM client (20+ providers)
│   ├── skills/          # 15+ skill implementations
│   ├── channels/        # Telegram, Discord, etc.
│   ├── store/           # SQLite + BadgerDB storage
│   ├── vector/          # Vector search (memory)
│   ├── config/          # Configuration
│   ├── onboarding/      # Setup wizard
│   └── cli/             # CLI commands
├── web/                 # Web UI
└── npm/                 # npm package
```

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Current Priorities**:
- 🐛 Bug fixes
- 📱 Mobile app (Flutter)
- 🌍 Internationalization
- 🏠 Smart home integrations
- 📧 Email channel

---

## 🗺️ Roadmap

### Phase 1: Foundation ✅ (COMPLETE)
- [x] Multi-provider LLM support (20+ providers)
- [x] CLI and server modes
- [x] Web UI
- [x] Telegram/Discord channels
- [x] 15+ skills
- [x] Setup wizard

### Phase 2: Intelligence ✅ (COMPLETE)
- [x] SQLite storage
- [x] Vector search
- [x] Knowledge graph
- [x] Web search

### Phase 3: Beta 🚧 (CURRENT)
- [ ] Bug fixes and polish
- [ ] Performance improvements
- [ ] Better error handling
- [ ] Documentation

### Phase 4: Mobile 📱 (UPCOMING)
- [ ] Flutter mobile app
- [ ] iOS and Android support

---

## 📞 Support & Community

- **🐛 Bug Reports**: [GitHub Issues](https://github.com/gmsas95/goclawde-cli/issues)
- **💬 Discussions**: [GitHub Discussions](https://github.com/gmsas95/goclawde-cli/discussions)
- **⭐ Star us**: If you like Myrai, please star the repo!

---

## 📝 License

MIT License - See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

Built with inspiration from:
- [OpenClaude](https://github.com/openclaw/openclaw) - Agentic patterns
- [Anthropic](https://www.anthropic.com) - Claude AI
- [OpenAI](https://openai.com) - GPT models

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

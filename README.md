# nanobot 🤖

> Your personal AI assistant that runs entirely on your own computer.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue)](https://golang.org)

**nanobot** is an open-source, self-hosted AI assistant inspired by [OpenClaw](https://github.com/openclaw/openclaw). It keeps your data local while providing powerful AI capabilities through your choice of LLM providers.

## 🌟 Features

- 🔒 **Privacy First** - Your data never leaves your machine
- 💻 **Self-Hosted** - Single binary, zero external dependencies
- 🧠 **Persistent Memory** - Remembers conversations and facts
- 🔧 **Tool Use** - File operations, web search, shell commands
- 🤖 **Background Tasks** - Spawn subagents for complex work
- 📱 **Multi-Channel** - Web UI, CLI, Telegram, WhatsApp
- ⏰ **Scheduled Jobs** - Automate recurring tasks

## 🚀 Quick Start

### Option 1: Binary Download (Easiest)

```bash
# Download latest release
curl -L https://github.com/YOUR_USERNAME/nanobot/releases/latest/download/nanobot-linux-amd64 -o nanobot
chmod +x nanobot

# Run
./nanobot

# Open http://localhost:8080
```

### Option 2: Docker Compose

```bash
# Clone repo
git clone https://github.com/YOUR_USERNAME/nanobot.git
cd nanobot

# Configure
cp config.example.yaml config.yaml
# Edit config.yaml with your API keys

# Run
docker-compose up -d
```

### Option 3: Build from Source

```bash
# Prerequisites: Go 1.23+, Node.js 20+
git clone https://github.com/YOUR_USERNAME/nanobot.git
cd nanobot

# Build
make build

# Run
./bin/nanobot
```

## 📝 Configuration

Create `config.yaml` in your data directory (`~/.nanobot/`):

```yaml
llm:
  default_model: "kimi-k2.5"
  providers:
    kimi:
      api_key: "your-kimi-api-key"
    openrouter:
      api_key: "your-openrouter-key"

tools:
  enabled:
    - read_file
    - write_file
    - web_search
    - exec_shell
```

## 🏗️ Architecture

```
nanobot/
├── cmd/nanobot/          # Entry point
├── internal/
│   ├── api/              # HTTP API handlers
│   ├── agent/            # Core AI agent logic
│   ├── llm/              # LLM provider integrations
│   ├── tools/            # Built-in tools
│   ├── store/            # SQLite & vector storage
│   ├── worker/           # Background job processing
│   └── channels/         # Telegram, WhatsApp, etc.
├── web/                  # Next.js web UI
├── plugins/              # Plugin system
└── docs/                 # Documentation
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) file.

## 🙏 Acknowledgments

- Inspired by [OpenClaw](https://github.com/openclaw/openclaw)
- Built with Go, SQLite, and ❤️

---

**Star ⭐ this repo if you find it useful!**

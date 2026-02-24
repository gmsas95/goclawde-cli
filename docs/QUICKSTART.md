# Quick Start Guide

Get Myrai running in under 5 minutes.

---

## Prerequisites

- **Go 1.24+** (if building from source)
- **At least one LLM API key**:
  - [OpenAI](https://platform.openai.com) - Most popular
  - [Anthropic](https://console.anthropic.com) - Great reasoning
  - [Groq](https://console.groq.com) - Fast & affordable
  - [Ollama](https://ollama.com) - **Free**, runs locally

---

## Installation

### Option 1: Docker (Recommended)

```bash
# Run with Docker
docker run -d \
  --name myrai \
  -p 8080:8080 \
  -v ~/.myrai:/app/data \
  -e OPENAI_API_KEY=sk-your-key \
  ghcr.io/gmsas95/myrai:latest

# Or use Docker Compose
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/docker-compose.yml -o docker-compose.yml
docker-compose up -d
```

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/gmsas95/goclawde-cli.git
cd goclawde-cli

# Build
go build -o myrai ./cmd/myrai

# Or use Make
make build
```

### Option 3: Download Binary

Download the latest release from [GitHub Releases](https://github.com/gmsas95/goclawde-cli/releases).

---

## First Run

### 1. Run the Onboarding Wizard

```bash
./myrai onboard
```

This interactive wizard will:
1. ✨ Help you choose an LLM provider
2. 🔑 Configure your API key
3. 🌐 Set up web search (optional)
4. 👤 Create your user profile
5. 🤖 Configure your AI persona

### 2. Start the Server

```bash
# Start server (Web UI + API + Channels)
./myrai server

# With custom port
./myrai server --port 3000

# With verbose logging
./myrai server --verbose
```

The server will start:
- **Web UI**: http://localhost:8080
- **API**: http://localhost:8080/api
- **WebSocket**: ws://localhost:8080/ws

### 3. Access the Web UI

Open http://localhost:8080 in your browser.

---

## Using the CLI

### Interactive Mode

```bash
./myrai --cli
```

### One-Shot Messages

```bash
./myrai -m "What's the weather in Tokyo?"
```

### Pipe Input

```bash
echo "Explain quantum computing" | ./myrai
```

---

## Project Structure

After running, Myrai creates this structure:

```
~/.myrai/
├── myrai.yaml          # Configuration file
├── .env                # Environment variables (API keys)
├── myrai.db            # SQLite database
├── badger/             # BadgerDB (sessions, queue, KV)
├── files/              # Uploaded files
├── IDENTITY.md         # AI personality
├── USER.md             # Your preferences
├── TOOLS.md            # Tool descriptions
├── AGENTS.md           # Agent behavior
├── projects/           # Project contexts
├── diary/              # Conversation archives
└── skills/             # Custom skills
```

---

## Configuration

### Environment Variables

```bash
# Required: LLM API Key
export OPENAI_API_KEY=sk-your-key
# OR
export ANTHROPIC_API_KEY=sk-your-key
# OR
export GROQ_API_KEY=gsk-your-key

# Optional: Web Search
export BRAVE_API_KEY=your-key

# Optional: Server settings
export MYRAI_SERVER_PORT=8080
export MYRAI_STORAGE_DATA_DIR=/path/to/data
```

### Configuration File

Edit `~/.myrai/myrai.yaml`:

```yaml
server:
  port: 8080
  address: 0.0.0.0

llm:
  default_provider: openai
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
      model: gpt-4
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"
      model: claude-3-opus-4-6

channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
  discord:
    enabled: true
    token: "${DISCORD_BOT_TOKEN}"

search:
  enabled: true
  provider: brave
  api_key: "${BRAVE_API_KEY}"
```

---

## Next Steps

- 📖 Read the [Usage Guide](USAGE.md)
- 🤖 Learn about [Personas](../PERSONA_SYSTEM.md)
- 🛠️ Explore [Skills](../README.md#skills-system)
- 🚀 Deploy to [VPS](DEPLOY_DOKPLOY.md)

---

## Troubleshooting

### "No API key configured"
Run `./myrai onboard` to configure your LLM provider.

### "Port already in use"
Use a different port: `./myrai server --port 3000`

### Permission errors
Ensure `~/.myrai` directory is writable:
```bash
mkdir -p ~/.myrai
chmod 755 ~/.myrai
```

---

## Support

- 🐛 [Report bugs](https://github.com/gmsas95/goclawde-cli/issues)
- 💬 [Discussions](https://github.com/gmsas95/goclawde-cli/discussions)
- ⭐ [Star the repo](https://github.com/gmsas95/goclawde-cli)

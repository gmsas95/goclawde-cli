# Quick Start Guide

## Installation

### Option 1: Download Binary
Download the latest release for your platform:
```bash
# Linux/macOS
curl -L https://github.com/YOUR_USERNAME/nanobot/releases/latest/download/nanobot-linux-amd64 -o nanobot
chmod +x nanobot

# Run
./nanobot
```

### Option 2: Build from Source
Requirements: Go 1.23+, Node.js 20+

```bash
# Clone
git clone https://github.com/YOUR_USERNAME/nanobot.git
cd nanobot

# Build (with embedded web UI)
make build-prod

# Run
./bin/nanobot
```

### Option 3: Docker
```bash
docker run -d \
  -p 8080:8080 \
  -v nanobot-data:/app/data \
  -e NANOBOT_API_KEY=sk-... \
  ghcr.io/YOUR_USERNAME/nanobot:latest
```

## Configuration

Create `config.yaml` in your data directory:

```yaml
server:
  address: 0.0.0.0
  port: 8080

llm:
  provider: kimi
  api_key: sk-your-key-here
  model: kimi-k2.5
  base_url: https://api.moonshot.cn/v1

# Optional: Enable channels
channels:
  telegram:
    enabled: true
    bot_token: "your-bot-token"
  
  whatsapp:
    enabled: false
```

Or use environment variables:
```bash
export NANOBOT_API_KEY=sk-...
export NANOBOT_SERVER_PORT=8080
```

## First Run

1. Start the server:
```bash
./nanobot
```

2. Open http://localhost:8080

3. The default admin user is created automatically. Check logs for the initial password.

## Project Structure

```
~/nanobot-data/
├── nanobot.db          # SQLite database (chats, agents, config)
├── badger/             # BadgerDB (sessions, queue, vectors)
├── files/              # Uploaded files
└── config.yaml         # Configuration file
```

## Next Steps

- Read [Architecture Overview](./ARCHITECTURE.md)
- Learn about [Agents](./AGENTS.md)
- Create custom [Plugins](./PLUGINS.md)

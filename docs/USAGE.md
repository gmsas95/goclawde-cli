# Myrai Usage Guide

## Table of Contents
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Web UI](#web-ui)
- [CLI Mode](#cli-mode)
- [Tools](#tools)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)

## Quick Start

### 1. Installation

```bash
# Download binary
curl -L https://github.com/YOUR_USERNAME/jimmy.ai/releases/latest/download/jimmy-linux-amd64 -o jimmy
chmod +x jimmy

# Or build from source
git clone https://github.com/YOUR_USERNAME/jimmy.ai.git
cd jimmy.ai
make build
```

### 2. First Run

```bash
# Set your LLM API key
export JIMMY_LLM_PROVIDERS_KIMI_API_KEY="your-api-key"

# Run the server
./jimmy

# Open http://localhost:8080 in your browser
```

## Configuration

### Environment Variables

All config options can be set via environment variables:

```bash
# Server
export JIMMY_SERVER_PORT=8080

# LLM Provider
export JIMMY_LLM_PROVIDERS_KIMI_API_KEY="sk-..."
export JIMMY_LLM_PROVIDERS_KIMI_MODEL="kimi-k2.5"

# OpenRouter alternative
export JIMMY_LLM_PROVIDERS_OPENROUTER_API_KEY="sk-or-..."
export JIMMY_LLM_DEFAULT_PROVIDER="openrouter"

# Data directory
export JIMMY_STORAGE_DATA_DIR="/path/to/data"
```

### Configuration File

Create `~/.local/share/jimmy/jimmy.yaml`:

```yaml
server:
  port: 8080

llm:
  default_provider: kimi
  providers:
    kimi:
      api_key: "your-key"
      model: "kimi-k2.5"
```

## Web UI

The web interface is available at `http://localhost:8080` when the server is running.

### Features
- **Chat Interface**: Send messages and receive streaming responses
- **Conversation History**: View and manage past conversations
- **File Uploads**: Attach files to conversations
- **Memory**: View and manage stored memories

### Keyboard Shortcuts
- `Enter` - Send message
- `Shift + Enter` - New line
- `Ctrl + N` - New conversation

## CLI Mode

### One-Shot Mode

Send a single message and get a response:

```bash
./jimmy -m "Explain quantum computing"

# With file input
./jimmy -m "Summarize this" < document.txt

# In pipelines
git diff | ./jimmy -m "Review these changes"
```

### Interactive Mode

```bash
./jimmy --cli

# Output:
# 🤖 Myrai - Interactive Mode
# Type 'exit' or 'quit' to exit, 'help' for commands
#
# 👤 You: _
```

### CLI Commands

| Command | Description |
|---------|-------------|
| `help`, `h` | Show help |
| `new`, `n` | Start new conversation |
| `clear`, `cls` | Clear screen |
| `exit`, `quit` | Exit the program |

## Tools

Myrai comes with built-in tools that the AI can use:

### File Operations

**read_file** - Read file contents
```
Myrai: Read the contents of /etc/hosts
→ Uses read_file tool
```

**write_file** - Write to files
```
Myrai: Create a file hello.txt with "Hello World"
→ Uses write_file tool
```

**list_dir** - List directory contents
```
Myrai: What's in the current directory?
→ Uses list_dir tool
```

### System Operations

**exec_command** - Execute shell commands (configurable allowlist)
```
Myrai: Show me the disk usage
→ Uses exec_command with "df -h"
```

### Web Operations

**web_search** - Search the web (requires search API config)
```
Myrai: Search for recent Go programming news
→ Uses web_search tool
```

**fetch_url** - Fetch and extract webpage content
```
Myrai: Summarize https://example.com/article
→ Uses fetch_url tool
```

### Utility

**thinking** - Chain-of-thought reasoning
```
Myrai uses thinking tool to show step-by-step reasoning before responding
```

## API Reference

### Authentication

All API endpoints except `/api/health` and `/api/auth/login` require authentication.

```bash
# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"password": ""}'

# Use the returned token
curl http://localhost:8080/api/conversations \
  -H "Authorization: Bearer <token>"
```

### Endpoints

#### Conversations

**List conversations**
```bash
GET /api/conversations?limit=20&offset=0
```

**Create conversation**
```bash
POST /api/conversations
Content-Type: application/json

{
  "title": "My Chat",
  "model": "kimi-k2.5"
}
```

**Get conversation**
```bash
GET /api/conversations/:id
```

**Delete conversation**
```bash
DELETE /api/conversations/:id
```

**Get messages**
```bash
GET /api/conversations/:id/messages?limit=50
```

#### Chat

**Send message (non-streaming)**
```bash
POST /api/chat
Content-Type: application/json

{
  "conversation_id": "conv_xxx",
  "message": "Hello!",
  "system_prompt": "You are a helpful assistant."
}
```

**Send message (streaming)**
```bash
POST /api/chat/stream
Content-Type: application/json

{
  "conversation_id": "conv_xxx",
  "message": "Hello!"
}

# Response: text/event-stream
```

#### Memories

**List memories**
```bash
GET /api/memories
```

**Create memory**
```bash
POST /api/memories
Content-Type: application/json

{
  "content": "User likes dark mode",
  "type": "preference",
  "importance": 8
}
```

**Delete memory**
```bash
DELETE /api/memories/:id
```

#### Tools

**List tools**
```bash
GET /api/tools
```

**Execute tool**
```bash
POST /api/tools/execute
Content-Type: application/json

{
  "name": "read_file",
  "args": {
    "path": "/etc/hosts"
  }
}
```

#### WebSocket

Connect to `/ws` for real-time chat:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  ws.send(JSON.stringify({
    conversation_id: "conv_xxx",
    message: "Hello!"
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // data.type: 'chunk' | 'done' | 'error'
  // data.content: string
};
```

## Troubleshooting

### Common Issues

**"Failed to load config"**
- Check that your API key is set
- Verify the config file syntax

**"LLM error"**
- Verify your API key is valid
- Check your internet connection
- Ensure you have API credits

**"Port already in use"**
```bash
# Change port
export JIMMY_SERVER_PORT=8081
./jimmy
```

**"Permission denied"**
```bash
chmod +x jimmy
```

### Data Location

Myrai stores data in:
- **Linux/macOS**: `~/.local/share/jimmy/`
- **Windows**: `%APPDATA%/jimmy/`
- **Custom**: Set `JIMMY_STORAGE_DATA_DIR`

### Logs

Enable debug logging:
```bash
export JIMMY_LOG_LEVEL=debug
./jimmy
```

### Reset Everything

```bash
# Stop Myrai
rm -rf ~/.local/share/jimmy/
# Restart and reconfigure
```

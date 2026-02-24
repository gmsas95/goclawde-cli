# Myrai Usage Guide

Complete guide to using Myrai 2.0.

---

## Table of Contents

- [Quick Start](#quick-start)
- [CLI Commands](#cli-commands)
- [Configuration](#configuration)
- [Web UI](#web-ui)
- [Skills](#skills)
- [MCP Integration](#mcp-integration)
- [Persona System](#persona-system)
- [Memory](#memory)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

```bash
# 1. Run onboarding
./myrai onboard

# 2. Start server
./myrai server

# 3. Open http://localhost:8080
```

---

## CLI Commands

### Core Commands

```bash
# Interactive chat
myrai --cli

# One-shot message
myrai -m "Explain quantum computing"

# Pipe input
cat file.txt | myrai

# Start server
myrai server
myrai server --port 3000 --verbose

# System health check
myrai doctor

# Show version
myrai version
```

### Skills Commands

```bash
# List installed skills
myrai skills list

# Install skill from GitHub
myrai skills install github.com/user/skill-name
myrai skills install github.com/user/skill-name@v1.2.0

# Enable/disable skill
myrai skills enable skill-name
myrai skills disable skill-name

# Uninstall skill
myrai skills uninstall skill-name

# Watch directory for hot-reload
myrai skills watch ./my-skills/

# Validate SKILL.md
myrai skills validate ./SKILL.md

# Search for skills
myrai skills search docker

# Show skill statistics
myrai skills stats
```

### MCP Commands

```bash
# Discover available MCP servers
myrai mcp discover

# List configured servers
myrai mcp list

# Add a discovered server
myrai mcp discover-add filesystem

# Start/stop MCP server
myrai mcp start filesystem
myrai mcp stop filesystem

# Remove server
myrai mcp remove filesystem

# List available tools
myrai mcp tools
```

### Persona Commands

```bash
# View current persona
myrai persona

# Edit persona manually
myrai persona edit

# View evolution proposals
myrai persona proposals
myrai persona proposals --pending

# Apply/reject proposals
myrai persona apply <proposal-id>
myrai persona reject <proposal-id>

# View persona history
myrai persona history

# Rollback to previous version
myrai persona rollback <version-id>

# Show evolution configuration
myrai persona config
```

### Memory Commands

```bash
# Search memories
myrai memory search "python projects"
myrai memory search --type fact --limit 10

# Add memory
myrai memory add "I prefer dark mode in all apps"
myrai memory add --type preference "My favorite color is blue"

# View memory health
myrai memory health

# List recent memories
myrai memory list --limit 20
```

### Configuration Commands

```bash
# Get config value
myrai config get server.port
myrai config get llm.default_provider

# Set config value
myrai config set server.port 3000
myrai config set llm.default_provider anthropic

# Edit config file directly
myrai config edit

# Show all config
myrai config list
```

---

## Configuration

### Environment Variables

All configuration can be set via environment variables:

```bash
# Server
export MYRAI_SERVER_PORT=8080
export MYRAI_SERVER_ADDRESS=0.0.0.0

# LLM Providers (pick one or more)
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-...
export GROQ_API_KEY=gsk-...
export DEEPSEEK_API_KEY=sk-...
export GOOGLE_API_KEY=...

# Default provider
export MYRAI_LLM_DEFAULT_PROVIDER=openai

# Web Search
export BRAVE_API_KEY=...

# Channels
export TELEGRAM_BOT_TOKEN=...
export DISCORD_BOT_TOKEN=...

# Storage
export MYRAI_STORAGE_DATA_DIR=/path/to/data
```

### Configuration File

Location: `~/.myrai/myrai.yaml`

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
      max_tokens: 4096
      timeout: 60
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"
      model: claude-3-opus-4-6
      max_tokens: 4096
    groq:
      api_key: "${GROQ_API_KEY}"
      model: llama-3.1-70b

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

storage:
  data_dir: ~/.myrai

persona:
  auto_evolve: true
  evolution_threshold: 0.7
```

---

## Web UI

The web interface is available at `http://localhost:8080`.

### Features

- **Chat Interface**: Send messages with streaming responses
- **Conversation History**: View and search past conversations
- **File Uploads**: Attach documents, images, PDFs
- **Memory Viewer**: Browse and search stored memories
- **Skills Panel**: View installed skills and their status
- **Settings**: Configure providers, channels, and preferences

### Keyboard Shortcuts

- `Enter` - Send message
- `Shift + Enter` - New line in message
- `Ctrl + K` - Search conversations
- `Ctrl + N` - New conversation
- `Ctrl + /` - Show keyboard shortcuts

---

## Skills

### Built-in Skills

Myrai includes 18 built-in skills:

**Productivity:**
- `tasks` - Todo management
- `calendar` - Event scheduling
- `notes` - Note taking
- `documents` - PDF/image processing

**Personal:**
- `health` - Health tracking
- `shopping` - Shopping lists
- `expenses` - Budget tracking

**Development:**
- `github` - Repository management
- `browser` - Web automation
- `agentic` - Git automation

**Information:**
- `search` - Web search
- `weather` - Weather forecasts
- `knowledge` - Knowledge base
- `intelligence` - Smart suggestions

**System:**
- `voice` - STT/TTS
- `vision` - Image analysis
- `system` - System commands

### Using Skills

```bash
# In CLI mode
myrai --cli
You: "Add task: Buy groceries"
Myrai: *uses tasks skill* "Added task: Buy groceries"

# Or directly
myrai -m "What's the weather in Tokyo?"
```

### Creating Custom Skills

Create a `SKILL.md` file:

```yaml
---
name: my-skill
version: 1.0.0
description: Does something awesome
author: your-name
tags: [utility, automation]
tools:
  - name: do_thing
    description: Perform the action
    parameters:
      - name: input
        type: string
        required: true
        description: Input to process
---

# My Skill

This skill does awesome things.

## Usage

Ask me to "do something awesome" and I'll use this skill.
```

---

## MCP Integration

Model Context Protocol (MCP) allows connecting external tools.

### Auto-Discovery

```bash
# Discover available MCP servers
myrai mcp discover

# Output:
# Available MCP servers:
# - filesystem: File system operations
# - github: GitHub integration
# - postgres: PostgreSQL database
# - brave-search: Web search
```

### Adding MCP Servers

```bash
# Add from discovery
myrai mcp discover-add filesystem

# Or manually add
myrai mcp add --name my-server --url http://localhost:3000
```

### Using MCP Tools

Once added, MCP tools appear in `myrai mcp tools` and can be used by the AI:

```
You: "List files in /tmp"
Myrai: *uses filesystem tool* "Files in /tmp: ..."
```

---

## Persona System

Myrai's personality is defined in markdown files.

### Persona Files

Located in `~/.myrai/`:

- **IDENTITY.md** - AI personality, voice, values
- **USER.md** - Your preferences and profile
- **TOOLS.md** - Tool descriptions and usage
- **AGENTS.md** - Agent behavior guidelines

### Evolution

Myrai can evolve its persona based on interactions:

```bash
# View pending proposals
myrai persona proposals

# Example proposal:
# ID: prop-123
# Type: communication_style
# Description: User prefers concise responses
# Confidence: 85%

# Apply proposal
myrai persona apply prop-123

# Or reject
myrai persona reject prop-123
```

---

## Memory

Myrai has three types of memory:

### 1. Facts
Important information about you:
- "I work as a software engineer"
- "My favorite programming language is Go"

### 2. Preferences
Your likes/dislikes:
- "I prefer dark mode"
- "I like concise responses"

### 3. Tasks
Active and completed tasks:
- "Buy groceries (due tomorrow)"
- "Submit report (completed)"

### Managing Memory

```bash
# Search
myrai memory search "project"

# Add
myrai memory add "I'm allergic to peanuts"

# View health
myrai memory health
```

---

## Troubleshooting

### "No API key configured"

Run `./myrai onboard` to configure your LLM provider and API key.

### "Connection refused" to LLM provider

Check:
1. API key is correct
2. Provider service is up
3. Network connectivity
4. Try a different provider

### Web search not working

1. Get free key from [Brave Search](https://api.search.brave.com)
2. Run `./myrai onboard` and enable web search
3. Or set: `export BRAVE_API_KEY=your_key`

### High API costs

- Use **Ollama** for free local inference
- Switch to cheaper providers (Groq, DeepSeek)
- Set spending limits in provider dashboard
- Enable caching in config

### Skills not loading

Check skill manifest:
```bash
myrai skills validate ./path/to/SKILL.md
```

### Memory/search not working

Check vector storage:
```bash
myrai doctor
```

### Telegram/Discord not responding

1. Verify bot tokens are correct
2. Check bot is added to channel/DM
3. Check channel configuration in `myrai.yaml`
4. Restart server

### Getting Help

```bash
# Run diagnostics
myrai doctor

# Show help
myrai --help
myrai skills --help
myrai mcp --help
```

Or [open an issue](https://github.com/gmsas95/goclawde-cli/issues).

---

## Advanced Usage

### Multiple LLM Providers

Configure multiple providers and switch between them:

```yaml
llm:
  default_provider: openai
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
      model: gpt-4
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"
      model: claude-3-opus-4-6
```

Switch at runtime:
```
You: "Use Claude for this conversation"
```

### Custom Skills Directory

```bash
# Create skills directory
mkdir -p ~/.myrai/custom-skills

# Watch for changes
myrai skills watch ~/.myrai/custom-skills
```

### Backup and Restore

```bash
# Backup
rsync -av ~/.myrai ~/myrai-backup

# Restore
rsync -av ~/myrai-backup ~/.myrai
```

---

## Support

- 🐛 [Bug reports](https://github.com/gmsas95/goclawde-cli/issues)
- 💬 [Discussions](https://github.com/gmsas95/goclawde-cli/discussions)
- ⭐ [Star the repo](https://github.com/gmsas95/goclawde-cli)

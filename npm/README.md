---
type: readme
title: Myrai - npm Package
resource: goclawde-cli
description: "This is the npm wrapper for [Myrai](https://github.com/gmsas95/goclawde-cli), your personal AI assistant."
tags: []
updated: 2026-06-18
---

# Myrai - npm Package

This is the npm wrapper for [Myrai](https://github.com/gmsas95/goclawde-cli), your personal AI assistant.

## Installation

```bash
# Install globally
npm install -g myrai

# Or use with npx (no installation required)
npx myrai --help
```

## Usage

```bash
# First time setup
myrai onboard

# Start interactive CLI
myrai --cli

# Start server
myrai server

# Get help
myrai --help
```

## Features

- 🤖 Multi-provider LLM support (OpenAI, Anthropic, Google, Groq, etc.)
- 💬 Interactive CLI and server modes
- 📱 Telegram and Discord integrations
- 🔧 Extensible skills system
- 🔒 Local-first privacy

## Requirements

- Node.js 14+
- Supported platforms: macOS (x64/arm64), Linux (x64/arm64), Windows (x64)

## License

MIT

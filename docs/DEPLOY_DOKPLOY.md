# Deploying Myrai on Dokploy

Complete guide to deploying Myrai 2.0 on Dokploy VPS.

---

## Quick Start

### Option 1: Docker Compose (Recommended)

1. **Create a new application in Dokploy**
   - Go to your Dokploy dashboard
   - Create a new **"Compose"** application
   - Connect your GitHub repository: `https://github.com/gmsas95/goclawde-cli`
   - Select the **"dev"** branch (or "main" for stable)

2. **Set environment variables**
   
   In Dokploy's environment variables section, add at minimum:
   ```bash
   # Required: At least one LLM provider
   OPENAI_API_KEY=sk-your-key-here
   ```
   
   See [Complete Environment Variables](#complete-environment-variables) below for all options.

3. **Deploy**
   - Click **"Deploy"**
   - Your app will be available at the assigned domain

### Option 2: Dockerfile

1. **Create a new application in Dokploy**
   - Choose **"Application"** type
   - Select **"Dockerfile"** as the build pack
   - Connect your GitHub repo: `https://github.com/gmsas95/goclawde-cli`
   - Select branch: **"dev"**

2. **Configure build settings**
   - Build path: `/`
   - Dockerfile path: `Dockerfile`

3. **Set environment variables** (see below)

4. **Configure volumes** (for data persistence)
   - Create a persistent volume
   - Mount path: `/app/data`

5. **Deploy**

---

## Complete Environment Variables

### Required (At least one LLM provider)

| Variable | Description | Get Key From |
|----------|-------------|--------------|
| `OPENAI_API_KEY` | OpenAI GPT-4, GPT-3.5 | [platform.openai.com](https://platform.openai.com) |
| `ANTHROPIC_API_KEY` | Claude 3 Opus/Sonnet/Haiku | [console.anthropic.com](https://console.anthropic.com) |
| `KIMI_API_KEY` | Moonshot AI (Kimi K2.5) | [platform.moonshot.cn](https://platform.moonshot.cn) |
| `GOOGLE_API_KEY` | Google Gemini | [aistudio.google.com](https://aistudio.google.com) |
| `GROQ_API_KEY` | Groq (fast inference) | [console.groq.com](https://console.groq.com) |
| `DEEPSEEK_API_KEY` | DeepSeek (coding) | [platform.deepseek.com](https://platform.deepseek.com) |
| `OPENROUTER_API_KEY` | OpenRouter (100+ models) | [openrouter.ai/keys](https://openrouter.ai/keys) |

**Note:** You only need to set ONE of the above. Setting multiple allows switching between providers.

### Optional: AI Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MYRAI_LLM_DEFAULT_PROVIDER` | `openai` | Default LLM provider to use |
| `MYRAI_LLM_DEFAULT_MODEL` | - | Default model (e.g., `gpt-4`, `claude-3-opus-4-6`) |

### Optional: Channels (Bots)

| Variable | Description | How to Get |
|----------|-------------|------------|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot token | Message [@BotFather](https://t.me/BotFather) |
| `DISCORD_BOT_TOKEN` | Discord Bot token | [Discord Developer Portal](https://discord.com/developers/applications) |
| `DISCORD_TOKEN` | Alias for DISCORD_BOT_TOKEN | - |
| `SLACK_BOT_TOKEN` | Slack Bot token (xoxb-...) | [Slack API](https://api.slack.com/apps) |
| `SLACK_APP_TOKEN` | Slack App-level token (xapp-...) | [Slack API](https://api.slack.com/apps) |

### Optional: Skills & Tools

| Variable | Description | Get Key From |
|----------|-------------|--------------|
| `GITHUB_TOKEN` | GitHub Personal Access Token | [github.com/settings/tokens](https://github.com/settings/tokens) |
| `BRAVE_API_KEY` | Brave Search API (2,000 queries/month free) | [api.search.brave.com](https://api.search.brave.com) |
| `WEATHER_API_KEY` | Weather API (optional, wttr.in used by default) | [openweathermap.org](https://openweathermap.org) |

### Optional: Voice & Media

| Variable | Description | Get Key From |
|----------|-------------|--------------|
| `ELEVENLABS_API_KEY` | ElevenLabs TTS | [elevenlabs.io](https://elevenlabs.io) |
| `DEEPGRAM_API_KEY` | Deepgram STT | [deepgram.com](https://deepgram.com) |

### Optional: Security (Recommended for Production)

| Variable | Description | How to Generate |
|----------|-------------|-----------------|
| `MYRAI_JWT_SECRET` | JWT signing secret | `openssl rand -hex 32` |
| `MYRAI_SECURITY_JWT_SECRET` | Alias for above | - |
| `MYRAI_ADMIN_PASSWORD` | Admin password for web UI | Choose strong password |
| `MYRAI_SECURITY_ADMIN_PASSWORD` | Alias for above | - |

### Optional: Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MYRAI_SERVER_PORT` | `8080` | Server port (usually don't change in Docker) |
| `MYRAI_STORAGE_DATA_DIR` | `/app/data` | Data directory path |
| `MYRAI_CONFIG_PATH` | - | Custom config file path |

---

## Example Environment Configuration

### Minimal Setup (Just Chat)
```bash
OPENAI_API_KEY=sk-your-openai-key
```

### With Web Search
```bash
OPENAI_API_KEY=sk-your-openai-key
BRAVE_API_KEY=your-brave-key
```

### With Telegram Bot
```bash
OPENAI_API_KEY=sk-your-openai-key
TELEGRAM_BOT_TOKEN=123456:ABCDEF...
```

### Production Setup
```bash
# LLM
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=sk-your-anthropic-key
MYRAI_LLM_DEFAULT_PROVIDER=anthropic

# Channels
TELEGRAM_BOT_TOKEN=123456:ABCDEF...
DISCORD_BOT_TOKEN=your-discord-token

# Tools
GITHUB_TOKEN=ghp_...
BRAVE_API_KEY=your-brave-key

# Security
MYRAI_JWT_SECRET=your-64-char-hex-secret
MYRAI_ADMIN_PASSWORD=your-strong-admin-password
```

---

## Port Configuration

Myrai runs on port **8080** by default.

In Dokploy:
- **Container port**: `8080`
- **Health check endpoint**: `/api/health`
- **Health check interval**: 30 seconds

---

## Data Persistence

Myrai stores data in `/app/data`:
- SQLite database (`myrai.db`)
- BadgerDB files (`badger/`)
- Uploaded files (`files/`)
- Persona files (`IDENTITY.md`, `USER.md`, etc.)
- Skills (`skills/`)
- Configuration (`myrai.yaml`, `.env`)

**Important:** Create a persistent volume for `/app/data` or use Docker Compose which handles this automatically.

---

## First Time Setup

After deployment:

1. **Access the Web UI**
   - Open your Dokploy domain
   - Default: `https://your-app.your-domain.com`

2. **Run Onboarding** (if not using env vars)
   ```bash
   # SSH into container or use Dokploy console
   ./myrai onboard
   ```
   
   Or configure via environment variables (recommended for Dokploy).

3. **Configure Channels** (optional)
   - Add bot tokens to environment variables
   - Restart the container
   - Your bots should come online automatically

---

## Updating

To update to the latest version:

1. **If using Docker Compose:**
   - Pull latest changes in Dokploy
   - Click "Redeploy"

2. **If using Dockerfile:**
   - Dokploy will automatically rebuild on push
   - Or click "Redeploy" to force rebuild

---

## Troubleshooting

### Container won't start
- ✅ Check that at least one `*_API_KEY` is set
- ✅ Check logs in Dokploy dashboard
- ✅ Verify environment variables are saved correctly

### "No API key configured" error
- ✅ Set one of the LLM provider API keys
- ✅ Restart the container after adding env vars

### Database errors
- ✅ Ensure `/app/data` volume is mounted
- ✅ Check volume has write permissions
- ✅ Verify sufficient disk space

### Channels not working
- ✅ Verify bot tokens are correct
- ✅ Check bot is added to channel/DM
- ✅ Ensure bots have proper permissions
- ✅ Check logs for connection errors

### Can't access the app
- ✅ Verify port 8080 is exposed
- ✅ Check Dokploy's domain/SSL configuration
- ✅ Try accessing `/api/health` for health check

### High memory usage
- ✅ Myrai uses ~100-200MB RAM normally
- ✅ SQLite + BadgerDB may use more with large data
- ✅ Consider setting memory limits in Dokploy

---

## Security Best Practices

1. **Use strong passwords**
   ```bash
   MYRAI_ADMIN_PASSWORD=your-very-strong-password-here
   ```

2. **Generate secure JWT secret**
   ```bash
   openssl rand -hex 32
   ```

3. **Rotate API keys regularly**
   - Don't share API keys
   - Use environment variables, never commit to git
   - Set spending limits on LLM provider dashboards

4. **Enable HTTPS**
   - Dokploy handles this automatically
   - Always use HTTPS in production

5. **Restrict access**
   - Use firewall rules if needed
   - Set up authentication (JWT secret + admin password)

---

## Performance Tips

1. **Choose the right LLM**
   - Groq: Fastest, cheapest
   - OpenAI: Most reliable
   - Ollama: Free if you have GPU

2. **Enable caching**
   - Myrai caches LLM responses automatically
   - Use `MYRAI_LLM_CACHE_ENABLED=true` (if supported)

3. **Monitor usage**
   - Check LLM provider dashboards for costs
   - Use `myrai doctor` to check system health

---

## Support

- 🐛 [Report issues](https://github.com/gmsas95/goclawde-cli/issues)
- 💬 [Discussions](https://github.com/gmsas95/goclawde-cli/discussions)
- 📖 [Documentation](https://github.com/gmsas95/goclawde-cli/tree/dev/docs)

---

**Ready to deploy!** 🚀

Minimum required: Just set `OPENAI_API_KEY` and click Deploy.

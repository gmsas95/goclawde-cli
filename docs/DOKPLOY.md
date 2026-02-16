# Deploying GoClawde on Dokploy

[Dokploy](https://dokploy.com) is an open-source deployment platform that makes it easy to deploy Docker applications. This guide shows you how to deploy GoClawde on Dokploy.

## Quick Deploy

### 1. Create a New Project

1. Log in to your Dokploy dashboard
2. Click "Create Project"
3. Name it "goclawde"

### 2. Add a Service

1. Click "Add Service" → "Application"
2. Select "Git" as the source
3. Configure:
   - **Repository**: `https://github.com/gmsas95/goclawde-cli`
   - **Branch**: `main`
   - **Build Path**: `/` (root)
   - **Dockerfile**: `Dockerfile`

### 3. Configure Environment Variables

Go to the "Environment" tab and add your LLM provider:

#### Option A: Kimi (Moonshot) - Recommended
```bash
KIMI_API_KEY=sk-your-kimi-api-key
GOCLAWDE_LLM_DEFAULT_PROVIDER=kimi
```

#### Option B: OpenAI
```bash
OPENAI_API_KEY=sk-your-openai-api-key
GOCLAWDE_LLM_DEFAULT_PROVIDER=openai
```

#### Option C: Anthropic
```bash
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
GOCLAWDE_LLM_DEFAULT_PROVIDER=anthropic
```

#### Option D: Ollama (Local/External)
```bash
GOCLAWDE_LLM_DEFAULT_PROVIDER=ollama
GOCLAWDE_LLM_PROVIDERS_OLLAMA_BASE_URL=http://your-ollama-server:11434/v1
```

### 4. Optional: Enable Channels

#### Telegram Bot
```bash
GOCLAWDE_CHANNELS_TELEGRAM_BOT_TOKEN=your-telegram-bot-token
```

#### Discord Bot
```bash
GOCLAWDE_CHANNELS_DISCORD_TOKEN=your-discord-bot-token
```

### 5. Configure Storage

1. Go to the "Volumes" tab
2. Add a volume:
   - **Host Path**: `/var/lib/dokploy/volumes/goclawde/data`
   - **Container Path**: `/app/data`
   - **Type**: Bind Mount

### 6. Deploy

1. Go to "General" tab
2. Click "Deploy"
3. Wait for the build to complete
4. Access your app at the provided domain

## Complete Environment Variables Reference

```bash
# Required: LLM Provider (choose one)
KIMI_API_KEY=sk-...
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Optional: Default provider (auto-detected if only one key is set)
GOCLAWDE_LLM_DEFAULT_PROVIDER=kimi

# Optional: Channels
GOCLAWDE_CHANNELS_TELEGRAM_BOT_TOKEN=...
GOCLAWDE_CHANNELS_DISCORD_TOKEN=...

# Optional: Skills
GITHUB_TOKEN=ghp_...
BRAVE_API_KEY=...

# Optional: Security
GOCLAWDE_JWT_SECRET=your-secret
GOCLAWDE_ADMIN_PASSWORD=admin-password
GOCLAWDE_GATEWAY_TOKEN=gateway-token

# Optional: Server (Dokploy sets PORT automatically)
GOCLAWDE_SERVER_PORT=8080
GOCLAWDE_SERVER_ADDRESS=0.0.0.0
```

## Health Checks

Dokploy will automatically use the health check endpoint:

```
GET /api/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2026-02-16T10:30:00Z"
}
```

## Updating

To update GoClawde:

1. Go to your service in Dokploy
2. Click "Deploy" again
3. Dokploy will pull the latest changes and rebuild

Or enable "Auto Deploy" to automatically deploy on git push.

## Troubleshooting

### Container won't start

Check logs in Dokploy:
1. Go to "Logs" tab
2. Look for configuration errors
3. Ensure required API keys are set

### Health check failing

1. Ensure port 8080 is exposed
2. Check that the container is running: `docker ps`
3. Check logs for startup errors

### Data not persisting

1. Verify volume is mounted at `/app/data`
2. Check volume permissions
3. Ensure `VOLUME ["/app/data"]` is in Dockerfile

## Advanced Configuration

### Using a Custom Domain

1. Go to "Domains" tab
2. Add your domain
3. Configure DNS to point to your Dokploy server
4. Enable HTTPS

### Scaling

Dokploy supports horizontal scaling:

1. Go to "Settings" tab
2. Increase "Replicas"
3. Note: GoClawde uses SQLite, so multiple replicas should share the same volume

### Backup

Backup your data volume:

```bash
# On the Dokploy host
tar -czf goclawde-backup-$(date +%Y%m%d).tar.gz /var/lib/dokploy/volumes/goclawde/data
```

## Alternative: Docker Compose

You can also use Dokploy's Docker Compose support:

```yaml
version: '3.8'
services:
  goclawde:
    build:
      context: https://github.com/gmsas95/goclawde-cli#main
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - KIMI_API_KEY=${KIMI_API_KEY}
      - GOCLAWDE_LLM_DEFAULT_PROVIDER=kimi
    volumes:
      - goclawde-data:/app/data
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  goclawde-data:
```

## Links

- [GoClawde GitHub](https://github.com/gmsas95/goclawde-cli)
- [Dokploy Documentation](https://docs.dokploy.com)
- [Dokploy GitHub](https://github.com/Dokploy/dokploy)

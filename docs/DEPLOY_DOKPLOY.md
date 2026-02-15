# Deploying on Dokploy

GoClawde works great on Dokploy! Here's how to deploy it.

## Option 1: Docker Compose (Recommended)

1. **Create a new application in Dokploy**
   - Go to your Dokploy dashboard
   - Create a new "Compose" application
   - Connect your GitHub repository: `https://github.com/gmsas95/goclawde-cli`

2. **Set environment variables**
   
   In Dokploy's environment variables section, add:
   ```bash
   # Required: At least one LLM provider
   OPENAI_API_KEY=sk-your-key-here
   # OR
   ANTHROPIC_API_KEY=sk-ant-your-key-here
   # OR  
   KIMI_API_KEY=your-kimi-key
   
   # Optional: Channels
   TELEGRAM_BOT_TOKEN=123456:ABCDEF...
   DISCORD_BOT_TOKEN=your-discord-token
   
   # Optional: Security
   GOCLAWDE_JWT_SECRET=random-32-char-string
   GOCLAWDE_GATEWAY_TOKEN=random-32-char-string
   ```

3. **Deploy**
   - Click "Deploy"
   - Your app will be available at the assigned domain

## Option 2: Dockerfile

1. **Create a new application in Dokploy**
   - Choose "Application" type
   - Select "Dockerfile" as the build pack
   - Connect your GitHub repo

2. **Configure build settings**
   - Build path: `/`
   - Dockerfile path: `Dockerfile`

3. **Set environment variables** (same as above)

4. **Configure volumes** (for data persistence)
   - Mount path: `/app/data`

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | One of these | OpenAI API key |
| `ANTHROPIC_API_KEY` | | Anthropic Claude API key |
| `KIMI_API_KEY` | | Kimi/Moonshot API key |
| `GOOGLE_API_KEY` | | Google Gemini API key |
| `OPENROUTER_API_KEY` | | OpenRouter API key |
| `TELEGRAM_BOT_TOKEN` | No | Telegram bot token |
| `DISCORD_BOT_TOKEN` | No | Discord bot token |
| `GITHUB_TOKEN` | No | GitHub personal access token |
| `GOCLAWDE_JWT_SECRET` | Auto-generated | JWT secret for auth |
| `GOCLAWDE_GATEWAY_TOKEN` | Auto-generated | Gateway auth token |

## Port Configuration

GoClawde runs on port **8080** by default. In Dokploy:
- Container port: `8080`
- The health check endpoint is `/api/health`

## Data Persistence

The container stores data in `/app/data`. Make sure to:
1. Create a persistent volume for this path
2. Or use Docker Compose which handles this automatically

## Health Checks

The Dockerfile includes health checks. Dokploy will automatically:
- Check `/api/health` every 30 seconds
- Restart the container if unhealthy

## Troubleshooting

### Container won't start
- Check that at least one API key is set
- Check logs for error messages

### Database errors
- Ensure the `/app/data` volume is properly mounted
- Check file permissions

### Can't access the app
- Verify port 8080 is exposed
- Check Dokploy's domain/SSL configuration

# Myrai VPS Deployment Guide

Deploy Myrai to your own VPS for 24/7 uptime.

## Quick Start

### 1. Get a VPS

Recommended providers:
- **DigitalOcean**: $6-12/month (1-2GB RAM)
- **Hetzner**: €4-8/month (2-4GB RAM)
- **Linode**: $5-10/month (1-2GB RAM)

Minimum requirements:
- 1 CPU core
- 2GB RAM
- 20GB SSD
- Ubuntu 22.04 LTS

### 2. Run Installation

```bash
# SSH into your VPS
ssh root@your-server-ip

# Download and run installer
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/deploy/install.sh | bash

# Edit configuration
nano /opt/myrai/.env

# Start Myrai
systemctl start myrai
```

### 3. Configure Domain (Optional)

```bash
# Point your domain to VPS IP
# Then run SSL setup
cd /opt/myrai
./deploy/setup-ssl.sh your-domain.com

# Start with HTTPS
docker-compose -f docker-compose.prod.yml --profile with-caddy up -d
```

## Management Commands

```bash
# Check status
systemctl status myrai

# View logs
journalctl -u myrai -f

# Restart
systemctl restart myrai

# Update to latest
cd /opt/myrai && ./update.sh

# Backup data
./backup.sh

# Monitor health
./deploy/monitor.sh
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | Yes* | OpenAI API key |
| `ANTHROPIC_API_KEY` | Yes* | Anthropic API key |
| `KIMI_API_KEY` | Yes* | Moonshot AI API key |
| `MYRAI_JWT_SECRET` | Yes | Random secret for tokens |
| `MYRAI_ADMIN_PASSWORD` | Yes | Admin dashboard password |
| `TELEGRAM_BOT_TOKEN` | No | Telegram bot integration |
| `GOOGLE_CLIENT_ID` | No | Google Calendar OAuth |

*At least one LLM provider required

## Troubleshooting

### Container won't start
```bash
# Check logs
docker logs myrai

# Check disk space
df -h

# Check port availability
netstat -tlnp | grep 8080
```

### Database issues
```bash
# Repair SQLite
docker exec myrai sqlite3 /app/data/myrai.db ".recover" | sqlite3 /app/data/myrai_fixed.db
```

### Out of memory
```bash
# Add swap
fallocate -l 2G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
```

## Security

- Change default passwords in `.env`
- Use strong JWT secret (32+ random characters)
- Enable firewall: `ufw allow 80,443,8080/tcp`
- Set up automated backups
- Keep system updated: `apt update && apt upgrade`

## Monitoring

Set up cron for health checks:
```bash
# Edit crontab
crontab -e

# Add every 5 minutes
*/5 * * * * /opt/myrai/deploy/monitor.sh
```

#!/bin/bash
#
# Myrai VPS Installation Script
# Run this on your VPS to install Myrai
#

set -e

MYRAI_VERSION="${MYRAI_VERSION:-latest}"
MYRAI_USER="${MYRAI_USER:-myrai}"
MYRAI_DIR="${MYRAI_DIR:-/opt/myrai}"
MYRAI_DATA="${MYRAI_DATA:-/var/lib/myrai}"

echo "🚀 Installing Myrai (未来) Personal AI Assistant..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root (use sudo)"
    exit 1
fi

# Install dependencies
echo "📦 Installing dependencies..."
apt-get update
apt-get install -y \
    curl \
    wget \
    git \
    docker.io \
    docker-compose \
    sqlite3 \
    ca-certificates \
    gnupg \
    lsb-release

# Enable Docker
systemctl enable docker
systemctl start docker

# Create myrai user
if ! id "$MYRAI_USER" &>/dev/null; then
    echo "👤 Creating myrai user..."
    useradd -r -s /bin/false -d "$MYRAI_DATA" -m "$MYRAI_USER"
    usermod -aG docker "$MYRAI_USER"
fi

# Create directories
echo "📁 Creating directories..."
mkdir -p "$MYRAI_DIR"
mkdir -p "$MYRAI_DATA"
mkdir -p /var/log/myrai

# Clone or download
echo "⬇️ Downloading Myrai..."
if [ -d "$MYRAI_DIR/.git" ]; then
    cd "$MYRAI_DIR"
    git pull
else
    git clone https://github.com/gmsas95/goclawde-cli.git "$MYRAI_DIR"
    cd "$MYRAI_DIR"
fi

# Set permissions
chown -R "$MYRAI_USER:$MYRAI_USER" "$MYRAI_DATA"
chown -R "$MYRAI_USER:$MYRAI_USER" /var/log/myrai

# Create environment file
echo "⚙️ Creating environment configuration..."
cat > "$MYRAI_DIR/.env" << EOF
# Myrai Environment Configuration
# Edit this file with your API keys and settings

# Server
MYRAI_PORT=8080
MYRAI_DOMAIN=your-domain.com

# LLM Provider (Required - choose one)
OPENAI_API_KEY=your-openai-key
# ANTHROPIC_API_KEY=your-anthropic-key
# KIMI_API_KEY=your-kimi-key
# GOOGLE_API_KEY=your-google-key

# Default Model
MYRAI_LLM_MODEL=gpt-4o-mini
MYRAI_LLM_PROVIDER=openai

# Channels (Optional)
# TELEGRAM_BOT_TOKEN=your-telegram-token
# DISCORD_BOT_TOKEN=your-discord-token

# Security (Change in production!)
MYRAI_JWT_SECRET=$(openssl rand -hex 32)
MYRAI_ADMIN_PASSWORD=change-me-now

# Google Calendar (Optional)
# GOOGLE_CLIENT_ID=your-client-id
# GOOGLE_CLIENT_SECRET=your-client-secret

# Logging
MYRAI_LOG_LEVEL=info
EOF

chown "$MYRAI_USER:$MYRAI_USER" "$MYRAI_DIR/.env"

# Create systemd service
echo "🔧 Creating systemd service..."
cat > /etc/systemd/system/myrai.service << EOF
[Unit]
Description=Myrai (未来) Personal AI Assistant
Documentation=https://myr.ai
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$MYRAI_DIR
User=root
Group=root

# Build and start
ExecStart=/usr/bin/docker-compose -f docker-compose.prod.yml up -d --build
ExecStop=/usr/bin/docker-compose -f docker-compose.prod.yml down
ExecReload=/usr/bin/docker-compose -f docker-compose.prod.yml up -d --build

[Install]
WantedBy=multi-user.target
EOF

# Create update script
cat > "$MYRAI_DIR/update.sh" << 'EOF'
#!/bin/bash
# Myrai Update Script

set -e

cd /opt/myrai

echo "🔄 Updating Myrai..."

# Pull latest code
git pull

# Rebuild and restart
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d --build

echo "✅ Myrai updated successfully!"
EOF

chmod +x "$MYRAI_DIR/update.sh"

# Create backup script
cat > "$MYRAI_DIR/backup.sh" << 'EOF'
#!/bin/bash
# Myrai Backup Script

set -e

BACKUP_DIR="/var/backups/myrai"
DATA_DIR="/var/lib/myrai"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

echo "💾 Creating backup..."

# Backup SQLite database
sqlite3 "$DATA_DIR/myrai.db" ".backup '$BACKUP_DIR/myrai_$TIMESTAMP.db'"

# Backup data directory
tar -czf "$BACKUP_DIR/myrai_data_$TIMESTAMP.tar.gz" -C "$DATA_DIR" .

# Keep only last 7 backups
ls -t "$BACKUP_DIR"/*.db | tail -n +8 | xargs -r rm
ls -t "$BACKUP_DIR"/*.tar.gz | tail -n +8 | xargs -r rm

echo "✅ Backup completed: $BACKUP_DIR/myrai_$TIMESTAMP.db"
EOF

chmod +x "$MYRAI_DIR/backup.sh"

# Enable and start service
systemctl daemon-reload
systemctl enable myrai.service

echo ""
echo "✅ Myrai installation complete!"
echo ""
echo "Next steps:"
echo "1. Edit configuration: nano $MYRAI_DIR/.env"
echo "2. Start Myrai: systemctl start myrai"
echo "3. Check status: systemctl status myrai"
echo "4. View logs: journalctl -u myrai -f"
echo ""
echo "Access Myrai at: http://your-server-ip:8080"
echo ""

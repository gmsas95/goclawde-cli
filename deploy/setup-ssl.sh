#!/bin/bash
#
# Setup SSL with Let's Encrypt using Caddy
#

set -e

MYRAI_DIR="${MYRAI_DIR:-/opt/myrai}"
DOMAIN="${1:-}"

if [ -z "$DOMAIN" ]; then
    echo "Usage: $0 <your-domain.com>"
    echo "Example: $0 myai.mydomain.com"
    exit 1
fi

echo "🔒 Setting up SSL for $DOMAIN..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root (use sudo)"
    exit 1
fi

# Update environment
cd "$MYRAI_DIR"
sed -i "s/MYRAI_DOMAIN=.*/MYRAI_DOMAIN=$DOMAIN/" .env

# Create Caddyfile
cat > Caddyfile << EOF
$DOMAIN {
    encode gzip
    
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    reverse_proxy myrai:8080

    log {
        output file /var/log/caddy/access.log {
            roll_size 10MB
            roll_keep 5
        }
    }
}
EOF

echo "✅ Caddyfile created for $DOMAIN"
echo ""
echo "Make sure your domain points to this server's IP address."
echo "Then start Myrai with Caddy:"
echo "  docker-compose -f docker-compose.prod.yml --profile with-caddy up -d"

#!/bin/bash
# Migration script to fix Myrai data persistence issue
# This script backs up your current data and prepares for the fix

echo "🔍 Myrai Data Persistence Fix"
echo "=============================="
echo ""

# Check if container is running
if ! docker ps | grep -q myrai; then
    echo "❌ Myrai container is not running"
    echo "   Start it first: docker-compose up -d"
    exit 1
fi

echo "✅ Myrai container found"
echo ""

# Find where the database actually is
echo "🔍 Checking current database location..."
DB_LOCATION=$(docker exec myrai find /home -name "myrai.db" 2>/dev/null | head -1)

if [ -z "$DB_LOCATION" ]; then
    echo "⚠️  No existing database found in container"
    echo "   This is normal for first-time setup"
else
    echo "📁 Found database at: $DB_LOCATION"
    
    # Backup the database
    BACKUP_DIR="/tmp/myrai-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    echo "💾 Backing up database to: $BACKUP_DIR/"
    docker cp "myrai:$DB_LOCATION" "$BACKUP_DIR/myrai.db"
    
    # Also backup any other data
    echo "💾 Backing up additional data..."
    docker cp myrai:/app/data "$BACKUP_DIR/data" 2>/dev/null || true
    
    echo "✅ Backup complete: $BACKUP_DIR/"
    echo ""
fi

# Check volume
echo "🔍 Checking Docker volume..."
docker volume ls | grep myrai-data || echo "⚠️  Volume not found"
echo ""

echo "📝 Next steps:"
echo "=============="
echo ""
echo "1. Rebuild the Docker image with the fix:"
echo "   docker-compose build --no-cache"
echo ""
echo "2. Stop and remove the old container:"
echo "   docker-compose down"
echo ""
echo "3. Start the new container:"
echo "   docker-compose up -d"
echo ""

if [ -n "$DB_LOCATION" ]; then
    echo "4. Restore your data (optional):"
    echo "   docker cp $BACKUP_DIR/myrai.db myrai:/app/data/myrai.db"
    echo ""
fi

echo "5. Verify the database is in the right place:"
echo "   docker exec myrai ls -la /app/data/"
echo ""

echo "✨ The fix ensures database is stored in /app/data (persisted volume)"
echo "   instead of /home/myrai/.local/share (not persisted)"
echo ""

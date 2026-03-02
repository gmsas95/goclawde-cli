# PostgreSQL Migration Guide

## Overview
Myrai has been migrated from SQLite to PostgreSQL for better performance, scalability, and JSON support.

## Changes Made

### 1. Docker Compose
- Added PostgreSQL 15 service
- Configured with persistent volume
- Added health checks
- Optional pgAdmin for development

### 2. Database Schema
- Uses PostgreSQL native UUID type
- JSONB columns for content blocks and metadata
- GIN index for JSONB queries
- Automatic updated_at triggers

### 3. Configuration
- Removed SQLitePath from config
- Database URL from DATABASE_URL environment variable
- Defaults to: `postgres://myrai:myrai_secret@localhost:5432/myrai?sslmode=disable`

### 4. Dependencies
- Removed: `github.com/glebarez/go-sqlite`
- Removed: `gorm.io/driver/sqlite`
- Added: `gorm.io/driver/postgres v1.5.11`

## Quick Start

### Using Docker Compose

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f myrai

# Access pgAdmin (optional, dev profile)
docker-compose --profile dev up -d
# Then visit http://localhost:5050
# Login: admin@myrai.local / admin
```

### Manual Setup

```bash
# 1. Install PostgreSQL 15
# 2. Create database
createdb myrai

# 3. Set environment variable
export DATABASE_URL="postgres://user:password@localhost:5432/myrai?sslmode=disable"

# 4. Run migrations
psql -d myrai -f migrations/001_init.sql

# 5. Start application
go run .
```

## Environment Variables

```bash
# Required
export DATABASE_URL="postgres://myrai:myrai_secret@localhost:5432/myrai?sslmode=disable"

# Optional - for Docker
export TELEGRAM_BOT_TOKEN="your_token"
export OPENAI_API_KEY="your_key"
```

## Benefits of PostgreSQL

1. **JSONB Support**: Native JSON with indexing
2. **Concurrency**: Handles multiple connections better
3. **Scalability**: Easy to add read replicas
4. **Data Integrity**: ACID compliance
5. **Extensions**: Full-text search, vector operations, etc.

## Migration from SQLite (Optional)

If you have existing SQLite data:

```bash
# Use the migration tool
go run ./cmd/migrate
```

Or start fresh - the v2 architecture is designed to work best with PostgreSQL from the start.

## Troubleshooting

### Connection refused
```bash
# Check if PostgreSQL is running
docker-compose ps

# Check logs
docker-compose logs postgres
```

### Authentication failed
```bash
# Verify credentials match docker-compose.yml
# Default: myrai / myrai_secret
```

### Database does not exist
```bash
# Create database manually
docker-compose exec postgres createdb -U myrai myrai
```

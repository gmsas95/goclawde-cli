# Myrai Documentation Index

Welcome to the Myrai documentation. This index helps you navigate all available documentation.

## 🚀 Getting Started

- **[README.md](../README.md)** - Project overview and quick start
- **[v2-README.md](v2-README.md)** - Complete v2 architecture guide
- **[QUICKSTART.md](QUICKSTART.md)** - Quick start tutorial

## 🏗️ Architecture

### Core v2 Components
- **[architecture-redesign.md](architecture-redesign.md)** - Design document with technical details
- **[PostgreSQL Migration](POSTGRESQL_MIGRATION.md)** - Migration guide from SQLite

### Component Documentation

| Component | Description | Location |
|-----------|-------------|----------|
| **types** | Content block model | `internal/types/` |
| **pipeline** | Sanitization pipeline | `internal/pipeline/` |
| **providers** | LLM adapters | `internal/providers/` |
| **storev2** | PostgreSQL storage | `internal/storev2/` |
| **tools** | Tool registry | `internal/tools/` |
| **streaming** | Event streaming | `internal/streaming/` |
| **compaction** | Context management | `internal/compaction/` |
| **conversation** | Conversation API | `internal/conversation/` |

## 🧪 Testing

- **[TESTING.md](TESTING.md)** - Comprehensive testing guide
- **[TEST_SUMMARY.md](../TEST_SUMMARY.md)** - Test coverage summary

### Running Tests

```bash
# All tests
make test

# With coverage
make test-cover

# Specific package
go test ./internal/types/... -v
```

## 🗄️ Database

- **[PostgreSQL Migration](POSTGRESQL_MIGRATION.md)** - Setup and migration
- **Migrations**: `migrations/001_init.sql`

## 💻 Development

- **[Makefile](../Makefile)** - Build and test commands
- **Examples**: `examples/v2demo/main.go`

### Key Commands

```bash
# Build
make build

# Test
make test

# Run
export DATABASE_URL="postgres://..."
go run ./cmd/myrai

# Docker
docker-compose up -d
```

## 📦 Deployment

- **[DEPLOY_DOKPLOY.md](DEPLOY_DOKPLOY.md)** - VPS deployment guide
- **[docker-compose.yml](../docker-compose.yml)** - Docker setup

## 🔄 Migration from v1

If you're migrating from v1 to v2:

1. Read [PostgreSQL Migration](POSTGRESQL_MIGRATION.md)
2. Review [architecture-redesign.md](architecture-redesign.md)
3. Check out the [migration package](../internal/migration/)

## 📚 Additional Resources

- **[LICENSE](../LICENSE)** - MIT License
- **Examples**: `examples/` directory
- **Tests**: `*_test.go` files in each package

## 🔍 Quick Reference

### Architecture Overview
```
┌─────────────────────────────────────┐
│           Myrai v2                  │
├─────────────────────────────────────┤
│  Content Block Model (types)        │
│  - TextBlock, ToolCallBlock, etc.   │
├─────────────────────────────────────┤
│  Sanitization Pipeline              │
│  - EmptyAssistantFilter             │
│  - ConsecutiveAssistantMerger       │
│  - Provider-specific pipelines      │
├─────────────────────────────────────┤
│  Tool Registry                      │
│  - Dynamic registration             │
│  - Thread-safe operations           │
├─────────────────────────────────────┤
│  Streaming Architecture             │
│  - 12 event types                   │
│  - Real-time processing             │
├─────────────────────────────────────┤
│  Context Compaction                 │
│  - Sliding window                   │
│  - Summarization                    │
├─────────────────────────────────────┤
│  PostgreSQL Storage                 │
│  - JSONB content blocks             │
│  - Scalable, ACID                   │
└─────────────────────────────────────┘
```

### Test Coverage

| Package | Lines | Status |
|---------|-------|--------|
| types | 328 | ✅ |
| pipeline | 379 | ✅ |
| tools | 383 | ✅ |
| streaming | 327 | ✅ |
| compaction | 274 | ✅ |
| **Total** | **1,691** | **✅** |

## 🤝 Contributing

See individual package documentation for:
- Code structure
- Testing patterns
- Extension points

## 📞 Support

- 🐛 [Report Issues](https://github.com/gmsas95/goclawde-cli/issues)
- 💬 [Discussions](https://github.com/gmsas95/goclawde-cli/discussions)
- ⭐ [Star on GitHub](https://github.com/gmsas95/goclawde-cli)

---

**Quick Links**:
- [Architecture](v2-README.md)
- [Testing](TESTING.md)
- [PostgreSQL](POSTGRESQL_MIGRATION.md)
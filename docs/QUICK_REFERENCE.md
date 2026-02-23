# Myrai 2.0 - Quick Reference Card

## 🎯 Key Concepts

### Neural Clusters
Semantic memory compression. Like human brain clustering related memories.
- **Theme**: Human-readable category (e.g., "Docker Workflows")
- **Essence**: AI-generated summary of cluster
- **Formation**: Automatic (weekly) or manual
- **Visibility**: User can view via `/memory` command

### Skills
Modular tools that can be added/removed dynamically.
- **Built-in**: Compiled Go code (fastest)
- **Runtime**: Hot-loaded from GitHub/local files
- **MCP**: External Model Context Protocol servers

### Persona Evolution
AI personality adapts based on usage patterns.
- **Detection**: Patterns in conversations
- **Proposal**: Suggested changes with rationale
- **Approval**: User approves/rejects/customizes
- **Versioning**: Rollback to previous personas

### Tool Chains
Pre-built workflows combining multiple tools.
- **Sequential**: Step 1 → Step 2 → Step 3
- **Parallel**: Step 1 & Step 2 → Step 3
- **Sharing**: Users can publish custom chains

## 🚀 Common Commands

### Neural Clusters
```bash
myrai memory clusters                 # View all clusters
myrai memory cluster show <id>       # View specific cluster
myrai memory health                   # Memory health report
```

### Skills
```bash
myrai skills list                     # List installed skills
myrai skills install github.com/...   # Install from GitHub
myrai skills enable <name>           # Enable skill
myrai skills watch ./local/           # Dev mode with hot-reload
```

### MCP
```bash
myrai mcp list                        # List MCP servers
myrai mcp add --name fs --command "..."  # Add MCP server
myrai mcp tools                       # List MCP tools
```

### Persona
```bash
myrai persona evolution               # View evolution proposals
myrai persona apply <id>             # Apply proposal
myrai persona history                 # View evolution history
```

### Marketplace
```bash
myrai marketplace search devops       # Search agents
myrai marketplace install myrai-team/devops-agent  # Install
myrai marketplace list                # List installed agents
```

### Tool Chains
```bash
myrai chain list                      # List available chains
myrai chain run deploy-app            # Execute chain
myrai chain create my-chain           # Create new chain
myrai chain publish my-chain          # Share to marketplace
```

## 📊 System Health

### Weekly Processes
- **Neural Cluster Formation**: Sundays 3:00 AM
- **Persona Evolution Analysis**: Sundays 3:30 AM
- **Memory Reflection Audit**: Sundays 4:00 AM

### Daily Processes
- **Contradiction Check**: Daily 2:00 AM
- **Quick Redundancy Scan**: Daily 2:30 AM

## 🔧 Configuration

### Key Files
- `~/.myrai/config.yaml` - Main configuration
- `~/.myrai/data/myrai.db` - SQLite database
- `~/.myrai/skills/custom/` - Custom skills
- `~/.myrai/workspace/` - AI workspace

### Important Settings
```yaml
neural_clusters:
  enabled: true
  formation_schedule: "weekly"  # daily, weekly, monthly

persona:
  evolution_enabled: true
  require_approval: true  # Always ask before evolving

reflection:
  enabled: true
  weekly_audit: true
```

## 🐛 Troubleshooting

### Issue: Skills not loading
```bash
myrai skills list --verbose          # Check status
myrai logs --follow                  # View logs
myrai doctor                         # Health check
```

### Issue: MCP server not starting
```bash
myrai mcp logs <name>              # View MCP logs
docker ps | grep mcp                 # Check containers
myrai mcp restart <name>           # Restart server
```

### Issue: Neural clusters not forming
```bash
myrai memory clusters --debug        # Debug info
myrai jobs run cluster-formation     # Run manually
myrai logs --component neural        # View neural logs
```

## 📈 Performance Metrics

### Target Metrics
- **Memory Reduction**: 50% fewer tokens vs raw memories
- **Skill Install**: <30 seconds
- **Persona Relevance**: >80% accepted proposals
- **Health Score**: >80% for active users

### Monitoring
```bash
myrai metrics                        # View metrics
myrai metrics --export               # Export to file
myrai performance                    # Performance report
```

## 🔐 Security

### Best Practices
- Run MCP servers in Docker (isolated)
- Review persona evolution proposals
- Check memory contradictions weekly
- Verify skills before installing

### Commands
```bash
myrai security audit                 # Run security audit
myrai memory contradictions          # Check for contradictions
myrai skills verify <name>          # Verify skill signature
```

## 🆘 Getting Help

### Documentation
- Full docs: `docs/` directory
- API spec: `docs/API.md`
- Architecture: `docs/ARCHITECTURE.md`

### Commands
```bash
myrai help                           # General help
myrai help <command>                # Command-specific help
myrai doctor                         # Diagnose issues
myrai version                        # Version info
```

## 🎓 Learning Path

### Beginner
1. Install Myrai 2.0
2. Configure one LLM provider
3. Use built-in skills
4. View neural clusters

### Intermediate
1. Install skills from marketplace
2. Create custom tool chains
3. Review persona evolution proposals
4. Configure MCP servers

### Advanced
1. Create custom skills
2. Publish to marketplace
3. Optimize neural cluster settings
4. Contribute to core

## 📞 Support

- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Discord**: [Join Server]
- **Email**: support@myrai.dev

---

**Version**: Myrai 2.0  
**Last Updated**: 2026-02-23  
**Status**: Implementation Phase

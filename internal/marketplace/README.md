# Myrai Agent Marketplace (Phase 6)

This package implements Phase 6 of the Myrai 2.0 roadmap - the Agent Marketplace. It provides a complete system for discovering, installing, managing, and publishing AI agents.

## Features

- **Agent Package Format**: AGENT.yaml specification with full validation
- **GitHub Integration**: Direct integration with github.com/myrai-agents
- **Search & Discovery**: Search agents by name, category, tags, or author
- **Installation Management**: Install, update, and remove agents with version control
- **Security Verification**: Automated security scanning and quality checks
- **Review System**: User ratings and reviews for agents
- **Badge System**: Verified, Security Audited, Tested, and Well Documented badges

## Directory Structure

```
internal/marketplace/
├── agent.go           # AgentPackage struct and AGENT.yaml parser
├── client.go          # MarketplaceClient for search and discovery
├── manager.go         # AgentManager for install/update/remove
├── verify.go          # Verification system and badge management
├── reviews.go         # Reviews and ratings system
├── github.go          # GitHub API integration
├── schema.go          # Database models (GORM)
├── migrate.go         # Database migration functions
└── AGENT.yaml.example # Example agent configuration
```

## CLI Commands

```bash
# Search for agents
myrai marketplace search <query>
myrai marketplace search productivity --sort installs
myrai marketplace search --category development --verified

# View agent details
myrai marketplace info <agent-name>

# Install an agent
myrai marketplace install <agent-name>
myrai marketplace install <agent-name>@1.2.0

# List installed agents
myrai marketplace list --installed

# Update an agent
myrai marketplace update <agent-name>
myrai marketplace update --all

# Remove an agent
myrai marketplace remove <agent-name>

# Publish an agent
myrai marketplace publish <path-to-agent>

# Submit a review
myrai marketplace review <agent-name> --rating 5 --comment "Great agent!"

# Sync with GitHub
myrai marketplace sync
```

## Agent Package Format (AGENT.yaml)

```yaml
name: my-agent
version: 1.0.0
author: my-name
description: A helpful agent
tags: [productivity, tasks]
icon: ./icon.png
license: MIT

requirements:
  min_myrai_version: 0.3.0
  memory: 256MB
  dependencies: [nodejs >= 14]

knowledge:
  neural_clusters:
    - name: patterns
      documents: [./docs/patterns.md]

skills:
  builtin:
    - name: tasks
      enabled: true
  external:
    - name: custom-skill
      version: 1.0.0
      source: github.com/user/skill

chains:
  - name: morning-routine
    steps: [step1, step2]

mcp_servers:
  - name: my-server
    command: node
    args: [./server.js]

persona:
  name: MyAgent
  personality: Helpful and efficient
  values: [productivity, clarity]

pricing:
  model: free  # or paid/subscription

support:
  email: support@example.com
  url: https://example.com/support
```

## Verification Badges

Agents are automatically verified and can earn badges:

- **Verified** ✓: Passed all validation checks
- **Security Audited** 🔒: Score ≥ 80 on security checks
- **Tested** 🧪: Has automated tests
- **Well Documented** 📚: Score ≥ 70 on documentation checks

## Security Features

The verification system checks for:
- YAML validation and schema compliance
- Hardcoded secrets or API keys
- Suspicious commands (sudo, shell scripts)
- Network access patterns
- Dangerous file operations

## Database Schema

The marketplace uses the following tables:

- `marketplace_agents`: Catalog of all available agents
- `installed_agents`: User's installed agent instances
- `agent_reviews`: User ratings and reviews
- `agent_versions`: Version history for each agent
- `agent_categories`: Category definitions

## GitHub Integration

The marketplace integrates with the `github.com/myrai-agents` organization:

1. **Discovery**: Automatically fetches repositories from the org
2. **Parsing**: Reads AGENT.yaml from each repository
3. **Releases**: Downloads release packages for installation
4. **Sync**: Periodic sync to update agent metadata

## Future Monetization

The system includes hooks for future monetization:

- Pricing model support (free/paid/subscription)
- Trial period configuration
- Feature gating through pricing tiers
- License validation framework

## Usage Example

```go
import (
    "context"
    "github.com/gmsas95/myrai-cli/internal/marketplace"
)

// Initialize components
logger, _ := zap.NewDevelopment()
githubClient := marketplace.NewGitHubClient(logger)
manager := marketplace.NewManager(store, githubClient, "/path/to/install", logger)
client := marketplace.NewClient(db, githubClient, manager, reviewsManager, logger)

// Search for agents
result, _ := client.Search(ctx, marketplace.SearchOptions{
    Query: "calendar",
    Sort:  marketplace.SortByRating,
    Limit: 10,
})

// Install an agent
installed, _ := manager.Install(ctx, "taskmaster", "latest", "user-id")

// Submit a review
review, _ := reviewsManager.SubmitReview(ctx, agentID, userID, 5, "Great!", "Works well", "1.0.0")
```

## Testing

Run tests for the marketplace package:

```bash
go test ./internal/marketplace/...
```

## Contributing

To publish an agent:

1. Create an AGENT.yaml file
2. Verify with: `myrai marketplace publish ./`
3. Create a GitHub repository at `github.com/myrai-agents/<agent-name>`
4. Push your code
5. Create a release with semantic version tag

Your agent will be discoverable through the marketplace search!

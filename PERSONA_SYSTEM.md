# GoClawde Persona & Memory System

**History:** The markdown-based persona system was originally created by a Malaysian developer as MemoryCore for local private use before being publicized. OpenClaw later brought the concept to wider attention with their implementation. GoClawde builds on both with a Go implementation featuring performance optimizations (caching, SQLite persistence).

GoClawde features a powerful persona and memory system using markdown files (IDENTITY.md, USER.md, TOOLS.md, AGENTS.md) adapted for a self-hosted Go application.

## 🧠 Overview

The persona system allows Jimmy to:
- Maintain a consistent personality and voice
- Learn and remember user preferences
- Track time context for appropriate responses
- Manage multiple projects with isolated contexts
- Persist memory across conversations

## 📁 File Structure

Your Jimmy workspace (`~/.jimmy/` by default) contains:

```
~/.jimmy/
├── jimmy.yaml              # Main configuration
├── .env                    # Environment variables
├── IDENTITY.md             # AI personality & characteristics
├── USER.md                 # Your profile & preferences
├── TOOLS.md                # Tool descriptions
├── AGENTS.md               # Agent behavior guidelines
├── projects/               # Project contexts
│   ├── active/            # Active projects (max 10)
│   ├── archived/          # Archived projects
│   └── project-index.json # Project registry
├── diary/                  # Conversation archives
└── memory/                 # Additional memory storage
```

## 🎭 IDENTITY.md - AI Personality

Defines Jimmy's personality, voice, values, and expertise:

```markdown
# Identity

Name: Jimmy

## Personality
You are Jimmy, a helpful and capable AI assistant. You are:
- Friendly but professional
- Concise yet thorough in your responses
- Proactive in suggesting solutions

## Voice
You communicate in a clear, approachable manner:
- Use natural, conversational language
- Be encouraging and supportive

## Values
- Privacy first
- Transparency
- Efficiency

## Expertise
- Software development
- Writing and content creation
- Data analysis
```

### Editing Identity

```bash
jimmy persona edit    # Opens in $EDITOR
jimmy persona show    # Display full file
```

## 👤 USER.md - Your Profile

Stores information about you that Jimmy learns:

```markdown
# User Profile

Name: Alice

## Communication Style
Concise and direct - get to the point quickly

## Expertise
- Go programming
- Kubernetes
- System architecture

## Goals
- Build scalable systems
- Learn Rust
- Write better documentation

## Preferences
- Response format: Bullet points preferred
- Code style: Follow Go conventions
- Documentation: Include examples

Updated: 2026-02-15T10:30:00Z
```

### Editing Profile

```bash
jimmy user edit       # Opens in $EDITOR
jimmy user show       # Display full file
```

## 📦 Project Management

Projects allow Jimmy to maintain separate contexts for different work:

### Creating a Project

```bash
jimmy project new "Web API" coding
# Description: Building a REST API for user management
```

### Available Project Types

| Type | Best For |
|------|----------|
| `coding` | Software development, scripts, algorithms |
| `writing` | Blog posts, documentation, creative writing |
| `research` | Academic research, market analysis |
| `business` | Strategy, planning, operations |

### Project Commands

```bash
jimmy project list                    # List all projects
jimmy project switch "Web API"       # Switch active project
jimmy project archive "Old Project"  # Archive a project
jimmy project delete "Test Project"  # Delete permanently
```

### How Projects Work

- **LRU Management**: Up to 10 active projects; oldest auto-archived
- **Context Isolation**: Each project maintains its own context
- **Auto-Switching**: Jimmy knows which project you're working on
- **Smart Loading**: Recently used projects load faster

## ⏰ Time Awareness

Jimmy automatically includes time context in conversations:

```
Current time: Saturday, February 15, 2026 8:45 PM
Time of day: evening
Context: It's evening. The user is likely focused on personal projects.
```

This enables:
- Time-appropriate greetings
- Context-aware suggestions
- Schedule-sensitive responses

## 🔧 System Prompt Construction

Jimmy builds the system prompt dynamically:

1. **Time Context** - Current time and appropriate guidance
2. **Identity** - Who Jimmy is (from IDENTITY.md)
3. **User Profile** - Who you are (from USER.md)
4. **Project Context** - Current project details (if any)
5. **Tools** - Available tools and capabilities
6. **Guidelines** - Behavior guidelines (from AGENTS.md)

## 🚀 Getting Started

### First-Time Setup

```bash
jimmy onboard
```

This interactive wizard will:
1. Create your workspace
2. Set up your profile
3. Configure API keys
4. Create default persona files

### Manual Setup

If you prefer manual configuration:

```bash
# Create workspace
mkdir -p ~/.jimmy/{projects,diary,memory}

# Create config
cat > ~/.jimmy/jimmy.yaml << 'EOF'
server:
  address: 0.0.0.0
  port: 8080

llm:
  default_provider: kimi
  providers:
    kimi:
      api_key: "your-api-key"
      model: "kimi-k2.5"
      base_url: "https://api.moonshot.cn/v1"

storage:
  data_dir: "~/.jimmy"
EOF

# Create default persona files
jimmy persona edit
jimmy user edit
```

## 💡 Tips

1. **Be Specific**: When editing IDENTITY.md or USER.md, specific details help Jimmy adapt better

2. **Update Regularly**: Your USER.md updates automatically, but you can manually add preferences

3. **Use Projects**: Create projects for distinct work contexts to keep conversations focused

4. **Archive Completed Work**: Archive projects you're not actively working on to keep the list manageable

5. **Review Growth**: Periodically check your USER.md to see what Jimmy has learned about you

## 🔄 How Memory Updates

### Automatic Updates
- User preferences (extracted from conversation)
- Project context (accumulated during work)
- Conversation history (stored in SQLite)

### Manual Updates
- Edit IDENTITY.md to change Jimmy's personality
- Edit USER.md to update your profile
- Use CLI commands to manage projects

## 🛡️ Privacy

All persona data is stored locally:
- Markdown files in your workspace
- SQLite database for conversations
- No cloud storage or external services

You have complete control over your data.

## 📝 Example Workflow

```bash
# 1. Start Jimmy
jimmy

# 2. Create a new coding project
jimmy project new "API Refactor" coding

# 3. Chat with project context
jimmy --cli
> Let's refactor the authentication middleware
# Jimmy knows you're working on the API Refactor project

# 4. Switch to writing project
jimmy project switch "Blog Post"

# 5. Jimmy now has different context
jimmy --cli  
> Help me outline this article
# Jimmy knows you're writing a blog post

# 6. Archive completed project
jimmy project archive "API Refactor"
```

## 🎓 Advanced: Custom Templates

You can create custom project templates by adding markdown files to your workspace:

```
~/.jimmy/templates/
├── frontend-dev.md
├── backend-api.md
├── technical-writing.md
└── research-paper.md
```

These will be used when creating new projects of matching types.

---

**Note**: This system is inspired by [AI MemoryCore](https://github.com/Kiyoraka/Project-AI-MemoryCore) but adapted for the Go-based Jimmy.ai architecture with persistent storage and CLI integration.

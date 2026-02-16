# Myrai (未来) 

> **Myrai** (未来) means "future" in Japanese.  
> **Myrai** (My + AI) means "my personal AI".  
> **Myrai** is the future of personal assistance.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue)](https://golang.org)
[![Discord](https://img.shields.io/discord/YOUR_DISCORD_ID?color=7289da&label=discord)](https://discord.gg/myrai)

**Myrai** is a lightweight, local-first personal AI assistant for the 99%.

Not a coding assistant. Not a terminal tool. **A life assistant.**

---

## ✨ What Makes Myrai Different?

| Feature | Myrai | Siri/Alexa | OpenClaude |
|---------|-------|------------|------------|
| **Target** | Everyone | Consumers | Developers |
| **Privacy** | ✅ Local-first | Cloud | Cloud/Complex |
| **Memory** | ✅ Personal knowledge graph | Session-only | Project-only |
| **Voice** | ✅ Natural conversation | Commands | None |
| **Vision** | ✅ Camera-enabled | Limited | None |
| **Documents** | ✅ OCR + storage | No | Code only |
| **Setup** | ✅ One binary | Easy | Technical |
| **Size** | ✅ ~50MB | N/A | ~2GB |

---

## 🎯 For the 99%

**Talk naturally. No commands to learn.**

```
🎙️ "Remind me to call mom Sunday at 3pm"
🎙️ "I spent $45 at Whole Foods"
🎙️ "What's on my schedule today?"
🎙️ "I'm at the store, what's on my list?"
```

**Show, don't type.**

```
📸 [Photo of receipt]
Myrai: "I see $45.50 from Whole Foods. Added to groceries."

📸 [Photo of document]
Myrai: "This is your car insurance renewal. Due March 15th."

👁️ "What am I looking at?"
Myrai: "That's a Fiddle Leaf Fig. Water it weekly."
```

**Myrai remembers.**

```
You: "I met Sarah at the coffee shop on Tuesday"

Later: "Who did I meet last week?"
Myrai: "You met Sarah at Blue Bottle on Tuesday."
```

---

## 🚀 Quick Start

### Installation

```bash
# Download binary (macOS/Linux)
curl -L https://myr.ai/download/latest -o myrai
chmod +x myrai

# Or install via Homebrew
brew install myrai

# Run
./myrai server
```

### Docker

```bash
docker run -d \
  --name myrai \
  -p 8080:8080 \
  -v ~/.myrai:/app/data \
  myrai/myrai:latest
```

### Mobile App

Coming soon! Join the waitlist at [myr.ai](https://myr.ai)

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        MYRAI (未来)                          │
├─────────────────────────────────────────────────────────────┤
│  INTERFACES                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Mobile  │  │  Voice   │  │  Vision  │  │   Web    │    │
│  │   App    │  │ (Speak)  │  │ (Camera) │  │Dashboard │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
├─────────────────────────────────────────────────────────────┤
│  MULTI-MODAL PROCESSING                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │    STT      │  │    TTS      │  │   Vision    │          │
│  │  whisper    │  │   piper     │  │  Moondream  │          │
│  │  (local)    │  │  (local)    │  │  (local)    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│  MYRAI CORE (Go Binary ~50MB)                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Personal Knowledge Graph                           │    │
│  │  • Entities: People, Places, Events, Documents     │    │
│  │  • Relations: "met at", "works at", "paid for"     │    │
│  │  • Temporal: "happened on", "due by"               │    │
│  └─────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Life Skills (50+)                                  │    │
│  │  • Tasks & Reminders  • Calendar & Scheduling      │    │
│  │  • Documents & OCR    • Expense Tracking           │    │
│  │  • Shopping Lists     • Health Tracking            │    │
│  └─────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│  STORAGE (Local-First, Private)                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   SQLite     │  │  Vector DB   │  │  Filesystem  │      │
│  │ (structured) │  │ (embeddings) │  │ (documents)  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

---

## 📖 Documentation

- [Vision](docs/myrai/MYRAI_VISION.md) - The Myrai manifesto
- [Roadmap](docs/myrai/MYRAI_ROADMAP.md) - 1-year development plan
- [Action Plan](docs/myrai/MYRAI_ACTION_PLAN.md) - Week-by-week tasks
- [Architecture](docs/myrai/MYRAI_ARCHITECTURE.md) - System design

---

## 🛠️ Development

### Build from Source

```bash
# Prerequisites: Go 1.23+
git clone https://github.com/myrai/myrai.git
cd myrai

# Build
make build

# Run
./bin/myrai server

# Or install locally
make install-local
```

### Development Mode

```bash
# Hot reload
make dev

# Run tests
make test

# Format code
make fmt
```

---

## 🤝 Contributing

We welcome contributors who share our vision of personal AI for everyone.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas We Need Help

- 📱 Mobile app (Flutter) - In progress
- 💰 Expense tracking with receipt OCR
- 🛒 Shopping lists with location reminders
- 📧 Email integration (Gmail, Outlook)
- 🏠 Smart home connectors (Home Assistant)

---

## 🗺️ Roadmap

### Phase 1: Foundation ✅ (Complete)
- [x] Voice interface (STT/TTS) - whisper.cpp + piper
- [x] Document processing (PDF/OCR) - with vision AI
- [x] Task & reminder system
- [x] Agent loop with tool calling
- [ ] Mobile app MVP - In progress

### Phase 2: Memory ✅ (Complete)
- [x] Personal knowledge graph
- [x] Entity extraction from conversations
- [x] Long-term memory with compression
- [x] Semantic search & Q&A

### Phase 3: Life Tools 🚧 (In Progress)
- [x] Calendar integration (Google Calendar)
- [x] Natural language event parsing
- [ ] Expense tracking with receipt OCR
- [ ] Shopping lists with smart reminders
- [ ] Health tracking

### Phase 4: Intelligence (Planned)
- [ ] Proactive suggestions
- [ ] Pattern recognition
- [ ] Automated workflows
- [ ] Life dashboard

See [MYRAI_ROADMAP.md](docs/myrai/MYRAI_ROADMAP.md) for details.

---

## 💡 Use Cases

### Busy Parent
- "Remind me to pick up milk when I'm near the store"
- "What time is soccer practice?"
- "Track my spending this month"

### Remote Worker
- "Summarize this PDF contract"
- "Add this receipt to expenses"
- "What's my schedule today?"

### Student
- "Track my assignment deadlines"
- "Explain this research paper simply"
- "How much did I spend this week?"

### Retiree
- "Remind me to take medication at 8am"
- "Find photos from last Christmas"
- "Call my daughter"

---

## 🔒 Privacy First

- ✅ **Local-First**: Data stays on your device
- ✅ **No Cloud**: No sending your life to big tech
- ✅ **Open Source**: You can audit the code
- ✅ **Encrypted**: Your data is encrypted at rest
- ✅ **Exportable**: You own your data, always

---

## 🌟 Why Myrai?

**The 99% deserves AI too.**

OpenClaude dominates the developer space (160k stars) because it's a **coding assistant**.

Myrai will dominate the personal space by being a **life assistant**.

> "The future is not more complex dev tools.  
> The future is simple AI that helps with daily life."

---

## 📞 Connect

- **Website**: [myr.ai](https://myr.ai)
- **GitHub**: [github.com/myrai/myrai](https://github.com/myrai/myrai)
- **Discord**: [discord.gg/myrai](https://discord.gg/myrai)
- **Twitter**: [@MyraiAI](https://twitter.com/MyraiAI)

---

## 📝 License

MIT License - See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

Inspired by:
- [OpenClaude](https://github.com/openclaw/openclaw) - For agentic patterns
- [VisionClaw](https://github.com/sseanliu/VisionClaw) - For wearable AI vision
- [PicoClaw](https://github.com/gmsas95/picoclaw) - For lightweight philosophy
- [MemoryCore](https://github.com/Kiyoraka/Project-AI-MemoryCore) - For memory systems

---

> *"未来はここにある"*  
> *"The future is here"*

**Let's build the future together.** 🚀

---

Built with ❤️ for the 99%.

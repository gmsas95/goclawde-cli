---
type: guide
title: Myrai (未来) - Your Personal AI for the Future
resource: goclawde-cli
description: "> **Myrai** (未来) means 'future' in Japanese."
tags: [go]
updated: 2026-06-18
---

# Myrai (未来) - Your Personal AI for the Future

> **Myrai** (未来) means "future" in Japanese.  
> **Myrai** (My + AI) means "my personal AI".  
> **Myrai** is the future of personal assistance.

---

## The Name

```
Myrai = My + AI + Mirai (未来)
        ↓    ↓      ↓
      Personal  Artificial  Future
       (My)    Intelligence (未来)
```

**Pronunciation**: "MY-rye"  
**Domain**: Myr.ai  
**Tagline**: *"Your future, organized"*

---

## Vision

**For the 99% who aren't developers.**

Not a coding assistant.  
Not a terminal tool.  
Not just for techies.

**Myrai is your personal life companion.**

- Remembers everything about you
- Helps with daily tasks
- Works hands-free (voice + vision)
- Keeps your data private (local-first)
- Available on all your devices

---

## What Myrai Does

### 🎙️ **Voice-First Interface**
Talk naturally. No commands to learn.

```
"Remind me to call mom Sunday at 3pm"
"I spent $45 at Whole Foods"
"What's on my schedule today?"
"I'm at the store, what's on my list?"
```

### 👁️ **Vision-Enabled**
Sees what you see through camera.

```
"What am I looking at?" [points camera]
"Read this receipt" [photo]
"Is this document important?" [scan]
"What plant is this?" [garden]
```

### 🧠 **Personal Memory**
Remembers facts, not just chats.

```
"You met Sarah at Blue Bottle last Tuesday"
"Your dentist appointment is next week"
"You spent $342 on groceries this month"
"Your mom's birthday is March 15th"
```

### 📱 **Works Everywhere**
Mobile app, smart glasses, watch, home speaker.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        MYRAI (未来)                          │
│                   "Your Future, Organized"                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  INTERFACES                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Mobile  │  │  Voice   │  │  Vision  │  │   Web    │    │
│  │   App    │  │ (Speak)  │  │ (Camera) │  │Dashboard │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       └─────────────┴─────────────┴─────────────┘            │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  MULTI-MODAL PROCESSING                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │    STT      │  │    TTS      │  │   Vision    │          │
│  │  whisper    │  │   piper     │  │  Moondream  │          │
│  │  (local)    │  │  (local)    │  │  (local)    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
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
│  │  • Smart Home         • Travel & Bookings          │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  CONNECTORS                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ Calendar │ │  Email   │ │ Banking  │ │  Smart   │       │
│  │ Google   │ │  IMAP    │ │  Plaid   │ │  Home    │       │
│  │ Apple    │ │  SMTP    │ │  Open    │ │  Home    │       │
│  │ Outlook  │ │          │ │ Banking  │ │Assistant │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  STORAGE (Local-First, Private)                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   SQLite     │  │  Vector DB   │  │  Filesystem  │      │
│  │ (structured) │  │ (embeddings) │  │ (documents)  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Target Users (The 99%)

### 👩‍💼 "Busy Parent Sarah"
**Pain**: Keeping track of kids, shopping, bills  
**Myrai helps**:
- "Remind me to pick up milk when I'm near the store"
- "What time is soccer practice?"
- "Track my spending this month"

### 👨‍💻 "Remote Worker Mike"
**Pain**: Meeting notes, expenses, travel  
**Myrai helps**:
- "Summarize this PDF contract"
- "Add this receipt to expenses"
- "What's my schedule today?"

### 👵 "Retiree Linda"
**Pain**: Medication, photos, family updates  
**Myrai helps**:
- "Remind me to take medication at 8am"
- "Find photos from last Christmas"
- "Call my daughter"

### 🎓 "Student Alex"
**Pain**: Assignments, budget, research  
**Myrai helps**:
- "Track my assignment deadlines"
- "Explain this research paper simply"
- "How much did I spend this week?"

---

## Competitive Position

```
                        DEV FOCUSED
                              ↑
                              │
         OpenClaude ←─────────┼────────→ GitHub Copilot
         (160k stars)         │
                              │
    ←─────────────────────────┼──────────────────────────→
    PERSONAL USE              │         WORK USE
                              │
         Siri/Alexa ←─────────┼────────→ Myrai (未来)
         (Basic)              │         (Comprehensive)
                              │
                              ↓
                         LIFE FOCUSED
```

**Myrai vs Others**:

| Feature | Siri/Alexa | Myrai | OpenClaude |
|---------|-----------|-------|------------|
| **Privacy** | Cloud | ✅ Local-first | Cloud/Complex |
| **Memory** | Session | ✅ Persistent graph | Project-only |
| **Vision** | Limited | ✅ Full camera AI | None |
| **Documents** | No | ✅ OCR + storage | Code only |
| **Customizable** | No | ✅ Open source | Limited |
| **Setup** | Easy | ✅ One binary | Technical |

---

## Development Phases

### Phase 1: Foundation (Weeks 1-4)
- Voice interface (whisper.cpp + piper)
- Document processing (PDF + OCR)
- Basic task system
- Mobile app scaffold

### Phase 2: Memory (Weeks 5-8)
- Personal knowledge graph
- Entity extraction
- Long-term memory
- Fact confirmation

### Phase 3: Life Tools (Weeks 9-12)
- Calendar integration
- Expense tracking
- Shopping lists
- Health metrics

### Phase 4: Intelligence (Weeks 13-16)
- Proactive suggestions
- Pattern recognition
- Automated workflows
- Life dashboard

### Phase 5: Scale (Year 2)
- Wearable integration
- Smart home control
- Family sharing
- Enterprise features

---

## Technical Principles

1. **Local-First**: Your data stays on your device
2. **Lightweight**: <100MB, runs on any device
3. **Offline-First**: Works without internet
4. **Private**: No cloud processing of personal data
5. **Open**: Open source, extensible skills

---

## Success Metrics

### Year 1 Goals
- [ ] 10,000+ downloads
- [ ] 4.5+ app store rating
- [ ] 50%+ daily active users
- [ ] 100+ community skills
- [ ] 1,000+ GitHub stars

### Year 3 Goals
- [ ] 1M+ users
- [ ] #1 personal AI app
- [ ] Hardware partnerships
- [ ] Sustainable business model
- [ ] Changing how people organize life

---

## The Future (未来)

**2025**: Myrai helps with daily tasks  
**2026**: Myrai manages your entire life  
**2027**: Myrai integrates with all your devices  
**2028**: Myrai knows you better than you know yourself  
**2029**: Myrai is your digital twin  
**2030**: Everyone has a Myrai

---

## Call to Action

> "Let's build Myrai - the personal AI that the 99% deserve.  
> Not just for developers. For everyone.  
> The future is personal. The future is Myrai."

**未来はここにある (The future is here)**

---

## Resources

- **Website**: Myr.ai
- **GitHub**: github.com/myrai/myrai
- **Discord**: discord.gg/myrai
- **Twitter**: @MyraiAI

---

*Built with ❤️ for the future.*

---
type: architecture
title: Architecture Comparison: Current vs Target
resource: goclawde-cli
description: "┌────────────────────────────────────────────────────────────────┐"
tags: [architecture, go, react]
updated: 2026-06-18
---

# Architecture Comparison: Current vs Target

## Current State: Dev Assistant

```
┌────────────────────────────────────────────────────────────────┐
│                     Myrai (Current)                          │
│                     "Dev Assistant"                             │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  USER: Developer typing commands                               │
│     ↓                                                          │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  INTERFACES                                             │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐                 │  │
│  │  │   CLI   │  │   Web   │  │Telegram │                 │  │
│  │  │ (typed) │  │ (typed) │  │ (typed) │                 │  │
│  │  └────┬────┘  └────┬────┘  └────┬────┘                 │  │
│  │       └─────────────┴──────────┘                        │  │
│  │                    │                                    │  │
│  │              TEXT ONLY                                  │  │
│  └────────────────────┬────────────────────────────────────┘  │
│                       ↓                                         │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  SKILLS (Dev-focused)                                   │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │ Code Analysis│  │   Git Ops   │  │ System Info │     │  │
│  │  │  (AST parse) │  │ (status/diff)│  │(ps/df/net) │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │ File Operations│  │  Execute    │  │   Web       │     │  │
│  │  │ (read/write) │  │  Commands   │  │  Search     │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                       ↓                                         │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  MEMORY                                                 │  │
│  │  ┌─────────┐  ┌─────────┐                              │  │
│  │  │  Chat   │  │ Vector  │                              │  │
│  │  │ History │  │ Search  │                              │  │
│  │  │(20 msgs)│  │(semantic)│                             │  │
│  │  └─────────┘  └─────────┘                              │  │
│  │                                                          │  │
│  │  ❌ No personal facts                                    │  │
│  │  ❌ No knowledge graph                                   │  │
│  │  ❌ No document storage                                  │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  USE CASE: "Analyze this codebase"                             │
│  USE CASE: "Git status"                                        │
│  USE CASE: "List processes"                                    │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Target State: Personal Life Assistant

```
┌────────────────────────────────────────────────────────────────┐
│                     Myrai (Target)                           │
│                  "Personal Life Assistant"                      │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  USER: Anyone talking naturally                                │
│     ↓                                                          │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  INTERFACES                                             │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │  │
│  │  │  MOBILE │  │  VOICE  │  │   Web   │  │  Chat   │    │  │
│  │  │  (App)  │  │ (Speak) │  │(Dashboard)│ │(Telegram)│   │  │
│  │  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘    │  │
│  │       └─────────────┴───────────┴────────────┘          │  │
│  │                         │                               │  │
│  │              MULTI-MODAL                                │  │
│  │         Voice + Touch + Text + Camera                   │  │
│  └────────────────────────┬────────────────────────────────┘  │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  MULTI-MODAL PROCESSING                                 │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │    STT      │  │    TTS      │  │    OCR      │     │  │
│  │  │ (whisper)   │  │  (piper)    │  │ (tesseract) │     │  │
│  │  │  "speech    │  │  "speaks    │  │  "reads     │     │  │
│  │  │   to text"  │  │   back"     │  │   text"     │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │   Vision    │  │  Document   │  │  Location   │     │  │
│  │  │  (LLaVA)    │  │  (PDF/Doc)  │  │   (GPS)     │     │  │
│  │  │ "sees photo"│  │"extracts    │  │ "knows      │     │  │
│  │  │             │  │  data"      │  │  where"     │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  PERSONAL KNOWLEDGE GRAPH                               │  │
│  │                                                         │  │
│  │     Sarah ──met at──→ Coffee Shop                       │  │
│  │       │                  │                              │  │
│  │       │                  └── on ──→ Tuesday             │  │
│  │       │                     │                           │  │
│  │       └── works at ──→ Tech Corp                        │  │
│  │              │                                          │  │
│  │              └── with ──→ John                          │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │  Entities   │  │  Relations  │  │   Facts     │     │  │
│  │  │  (People,   │  │  (met, works│  │  (birthdays,│     │  │
│  │  │   Places,   │  │   lives,    │  │   prefs)    │     │  │
│  │  │   Events)   │  │   paid)     │  │             │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  │  "You met Sarah at Blue Bottle last Tuesday"            │  │
│  │  "Your dentist appointment is next week"                │  │
│  │  "You spent $450 on groceries this month"               │  │
│  │                                                         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  LIFE MANAGEMENT SKILLS                                 │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │   Tasks &   │  │  Documents  │  │   Calendar  │     │  │
│  │  │  Reminders  │  │  (Receipts, │  │ (Schedule)  │     │  │
│  │  │             │  │   Forms)    │  │             │     │  │
│  │  │ • Location  │  │             │  │ • Events    │     │  │
│  │  │   reminders │  │ • Expense   │  │ • Meetings  │     │  │
│  │  │ • Recurring │  │   tracking  │  │ • Deadlines │     │  │
│  │  │ • Smart     │  │ • Search    │  │ • Travel    │     │  │
│  │  │   alerts    │  │ • Organize  │  │             │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │   Health    │  │  Shopping   │  │  Smart Home │     │  │
│  │  │  Tracking   │  │    Lists    │  │  Control    │     │  │
│  │  │             │  │             │  │             │     │  │
│  │  │ • Water     │  │ • Multi-list│  │ • Lights    │     │  │
│  │  │ • Medication│  │ • Store     │  │ • Thermostat│     │  │
│  │  │ • Exercise  │  │   aisles    │  │ • Locks     │     │  │
│  │  │ • Symptoms  │  │ • Price     │  │ • Scenes    │     │  │
│  │  │             │  │   tracking  │  │             │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │    Email    │  │   Banking   │  │   Travel    │     │  │
│  │  │  (Summarize)│  │ (Expenses)  │  │ (Bookings)  │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │                                                         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  PROACTIVE INTELLIGENCE                                 │  │
│  │                                                         │  │
│  │  "You usually buy milk on Sundays - add to list?"       │  │
│  │  "Traffic is heavy, leave 15 min early for meeting"     │  │
│  │  "Don't forget Sarah's birthday is tomorrow"            │  │
│  │  "You haven't logged water today"                       │  │
│  │  "Your Amazon package arrives today"                    │  │
│  │                                                         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  USE CASE: "Remind me to call mom Sunday 3pm"                  │
│  USE CASE: "I spent $45 at Whole Foods" [photo of receipt]     │
│  USE CASE: "Find that document about car insurance"            │
│  USE CASE: "What's my schedule today?"                         │
│  USE CASE: "I'm at the store, what's on my list?"              │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Key Differences

| Aspect | Current (Dev) | Target (Personal) |
|--------|---------------|-------------------|
| **Primary Input** | Typed commands | Voice + Touch |
| **Target User** | Developers | Everyone |
| **Main Use Case** | Code editing | Life management |
| **Memory** | Chat history | Personal knowledge graph |
| **Documents** | Code files | Receipts, forms, photos |
| **Skills** | Git, AST, terminal | Tasks, calendar, health |
| **Interface** | Terminal/Web | Mobile app |
| **Availability** | When coding | All day, everywhere |
| **Proactivity** | Reactive | Proactive suggestions |

---

## The Transformation

### What We Keep (Foundation)
- ✅ Single binary architecture
- ✅ Skills system (extensible)
- ✅ Local-first (privacy)
- ✅ Multi-channel support
- ✅ Context management
- ✅ Agent loop

### What We Remove (Dev-only)
- ❌ Complex code analysis (make optional)
- ❌ Git integration (make optional)
- ❌ System introspection focus
- ❌ Terminal command emphasis

### What We Add (Life-focused)
- 🆕 Voice interface (STT/TTS)
- 🆕 Document processing (OCR, PDF)
- 🆕 Personal knowledge graph
- 🆕 Task & reminder system
- 🆕 Calendar integration
- 🆕 Mobile app
- 🆕 Health tracking
- 🆕 Shopping lists
- 🆕 Proactive intelligence

---

## User Journey Comparison

### Current: Developer Workflow
```
User: [types] "analyze_project_structure"
AI:   Here's the Go project structure...
      - 12 packages
      - main.go entry point
      
User: [types] "git_status"
AI:   3 files modified, 1 untracked...

User: [types] "search_code pattern=TODO"
AI:   Found 5 TODOs...
```

### Target: Personal Assistant Workflow
```
User: [voice] "I met Sarah at the coffee shop on Tuesday"
AI:   [speaks] Got it. I've noted that you met Sarah at 
      the coffee shop on Tuesday.

User: [voice] "Remind me to call mom Sunday at 3pm"
AI:   [speaks] I'll remind you to call mom this Sunday 
      at 3pm. Should I prepare a summary of what you've 
      been up to this week?

User: [voice] "What was that restaurant we went to last month?"
AI:   [speaks] You went to The Italian Place with Mike 
      on January 15th. You rated it 4 stars and said 
      the pasta was excellent.

User: [photo of receipt] 
AI:   [speaks] I see a receipt from Whole Foods for $87.50. 
      I'll categorize this under groceries. You've spent 
      $342 on groceries this month.

User: [voice] "I'm at the grocery store"
AI:   [speaks] You're at Whole Foods. Your shopping list 
      has: milk, eggs, bread, and avocados. You're in 
      the produce section now - avocados are to your right.
```

---

## Market Position

```
                    HIGH COMPLEXITY
                           ↑
                           │
         OpenClaude   ←────┼────→  GitHub Copilot
         (160k stars)      │       (dev tools)
                           │
    ←──────────────────────┼──────────────────────→
    PERSONAL USE           │           WORK USE
                           │
         Siri/Alexa   ←────┼────→  Myrai Target
         (basic)           │       (comprehensive)
                           │
                           │
                    LOW COMPLEXITY
                           ↓
                           
    Myrai Current →-----┼----→  Mid-journey
    (dev assistant)        │       (transition)
                           ↓
                    
    ╔═══════════════════════════════════════════════════╗
    ║  THE OPPORTUNITY:                                 ║
    ║                                                   ║
    ║  No one owns the personal AI space yet.          ║
    ║  Siri/Alexa are too basic.                       ║
    ║  OpenClaude is too technical.                    ║
    ║                                                   ║
    ║  We can own the middle:                          ║
    ║  "Smart enough to be helpful,                    ║
    ║   simple enough for everyone"                    ║
    ╚═══════════════════════════════════════════════════╝
```

---

## Conclusion

**Current**: A great dev tool (competing with 160k star project)
**Target**: The personal AI for everyone else (99% of market)

**The pivot**: Remove dev complexity, add life management.
**The opportunity**: Own the personal AI space.

---

**Ready to build the future?**

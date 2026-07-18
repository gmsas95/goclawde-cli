---
type: plan
title: Agent Runtime Iteration Plan
resource: goclawde-cli
description: "Build an experimental but real agent runtime that can:"
tags: [planning]
updated: 2026-06-18
---

# Agent Runtime Iteration Plan

## North Star

Build an experimental but real agent runtime that can:

- accept a task
- plan and execute with tools safely
- persist execution state
- resume after failure or restart
- require approval for risky actions
- expose task history and logs
- complete a narrow set of real workflows reliably

## What We Are Building First

- Not an OpenClaw-scale platform
- Not a PicoClaw-style broad channel product
- Not a ZeptoClaw-style full security and runtime stack on day one
- We are building `v1 runtime core`: durable, observable, restart-safe, permission-aware, and testable

## Recommended Scope For Iteration 1

- Single-user
- Local-first
- CLI plus existing dashboard only
- Narrow tool surface:
  - file read
  - file write and edit
  - shell command
  - search, grep, list
  - web fetch
- Narrow task types:
  - inspect repo
  - modify files
  - run tests and builds
  - summarize results
- No heavy multi-agent orchestration yet
- No broad public channel integrations as a priority

## Current Architecture Reality

Good base already exists in:

- `internal/agent/agent.go:18`
- `internal/agent/loop.go:16`
- `internal/agent/context.go:19`
- `internal/skills/runtime/engine.go:121`
- `internal/jobs/scheduler.go:32`

Missing runtime spine:

- durable task lifecycle
- task state machine
- checkpoint, restart, and resume
- real approval flow
- hardened tool permissions
- end-to-end runtime tests
- clear runtime telemetry

## Execution Strategy

- Deliver in 4 iterations
- End every iteration with something demoable and testable
- Do not redesign everything at once
- Build a separate runtime core path alongside the existing chat flow, then gradually converge

## Iteration 1: Runtime Core MVP

### Goal

Make the agent loop durable, inspectable, and restart-safe for one task at a time.

### Deliverables

- `Task` entity with lifecycle:
  - `pending`
  - `running`
  - `blocked`
  - `failed`
  - `completed`
  - `cancelled`
- `TaskStep` or `TaskEvent` persistence for:
  - prompt
  - action type
  - tool call
  - tool result
  - error
  - timestamps
- Task runner service:
  - start task
  - continue task
  - cancel task
  - resume interrupted task on process restart
- Loop state model:
  - iteration number
  - current plan
  - last tool result
  - stop reason
- Runtime API and CLI commands:
  - `task create`
  - `task list`
  - `task show`
  - `task resume`
  - `task cancel`
- Dashboard page for task timeline

### Code Direction

- Keep `internal/agent/loop.go:16` logic conceptually, but stop treating it as transient in-memory control flow
- Add a runtime package, for example:
  - `internal/runtime/task.go`
  - `internal/runtime/runner.go`
  - `internal/runtime/store.go`
  - `internal/runtime/events.go`

### Data Model

- `tasks`
- `task_steps`
- `task_checkpoints`
- `task_approvals` if approvals are added early

### Exit Criteria

- Create a task, let it call tools, kill the process, restart, and resume
- Full task timeline visible from CLI and dashboard
- At least 3 end-to-end tasks succeed repeatedly

## Iteration 2: Safety and Permission Model

### Goal

Make actions governable enough for internal testers.

### Deliverables

- Approval gate system for:
  - write file
  - delete file
  - shell exec outside safe allowlist
  - network access if desired
- Permission model:
  - `safe`
  - `write`
  - `exec`
  - `network`
  - `admin`
- Policy evaluation before tool execution
- Tool metadata:
  - risk level
  - required approval
  - allowed paths
  - allowed commands
  - timeout
- Approval UX:
  - task enters `blocked`
  - pending approval shows exact action
  - user can approve or deny
- Command allowlist for shell tool
- Workspace boundary enforcement for all file operations

### Code Direction

- Replace ad hoc destructive detection in `internal/agent/loop.go` with a real policy evaluator
- Stop relying on permissive behavior in `pkg/tools/registry.go:19` as good enough

### Exit Criteria

- Dangerous actions cannot run silently
- Denied approvals fail gracefully and task can continue, replan, or stop cleanly
- Safe tasks run uninterrupted

## Iteration 3: Reliable Planning and Tool Execution

### Goal

Make the runtime dependable, not just persistent.

### Deliverables

- Strong action schema:
  - structured plan
  - structured next action
  - structured completion criteria
- Better loop contract:
  - `plan`
  - `act`
  - `observe`
  - `reflect`
  - `finish`
- Malformed output recovery:
  - retry parse
  - fallback repair prompt
  - bounded retries
- Tool execution wrappers:
  - retries
  - timeout handling
  - typed errors
  - normalized result envelopes
- Idempotency strategy:
  - avoid duplicate tool execution on resume
  - persist tool call fingerprint and result
- Loop guardrails:
  - repeated action detection
  - max identical tool retries
  - runaway loop detection

### Architecture Change

- Today the loop in `internal/agent/loop.go:218` asks for the next JSON action
- Evolve that into a smaller runtime protocol with explicit states and validators
- This is the point where the project starts feeling like a runtime, not a chat wrapper

### Exit Criteria

- The same task runs consistently across repeated attempts
- Malformed LLM output no longer kills the task path
- Resume does not duplicate side effects

## Iteration 4: Beta Readiness

### Goal

Make the runtime usable by a small group of testers.

### Deliverables

- Observability:
  - task duration
  - step counts
  - tool success and failure rates
  - blocked approvals
  - token and cost estimate if available
- Runtime health endpoints and dashboard widgets
- Better operator controls:
  - pause and resume
  - cancel
  - retry failed step
  - inspect tool input and output
- Evaluation suite:
  - benchmark tasks
  - success rate tracking
  - regression suite
- Docs:
  - supported task types
  - safety model
  - failure modes
  - beta caveats
- Packaging:
  - stable startup path
  - migration path for task tables
  - seed and test fixtures

### Exit Criteria

- 5 to 10 internal testers can run tasks with minimal hand-holding
- Failures are understandable, inspectable, and recoverable
- We can honestly call it an experimental agent runtime beta

## Workstreams

Run these in parallel once Iteration 1 starts.

### A. Runtime Core

- task lifecycle
- runner
- persistence
- resume logic

### B. Tool Safety

- tool metadata
- permission checks
- approval flow
- workspace boundaries

### C. Loop Reliability

- structured planning
- retry and repair
- repeated-loop detection
- deterministic envelopes

### D. UX

- CLI task commands
- dashboard task inspector
- approval interface
- task logs

### E. Validation

- end-to-end tests
- replay tests
- crash and resume tests
- task benchmarks

## Concrete Backlog By Priority

### P0

- Introduce persistent task model
- Introduce task step and event log
- Refactor autonomous loop into resumable runner
- Add CLI task commands
- Add dashboard task timeline
- Add crash and restart recovery
- Add approval blocking state
- Add tool risk metadata
- Add workspace and path restrictions
- Add end-to-end runtime tests

### P1

- Add typed tool result envelopes
- Add retry policies and loop guards
- Add idempotent tool replay protection
- Add operator actions: retry, cancel, resume
- Add metrics and health reporting
- Add benchmark tasks

### P2

- Add subagents
- Add budgets and cost caps
- Add richer sandboxing
- Add policy profiles
- Add multi-tenant or remote worker support

## Suggested File and Module Plan

Do not jam everything into `internal/agent`.

- `internal/runtime/`
  - `models.go`
  - `runner.go`
  - `planner.go`
  - `executor.go`
  - `policy.go`
  - `approvals.go`
  - `recovery.go`
  - `events.go`
- `internal/store/` or migrations
  - task tables
  - step tables
  - approval tables
  - checkpoint tables
- `internal/agent/`
  - keep chat-specific orchestration
  - delegate autonomous execution into runtime layer
- `internal/tools/` or `pkg/tools/`
  - add tool metadata
  - add policy hooks
  - add normalized execution result types

## Testing Plan

We need runtime tests, not just unit tests.

### Unit Tests

- task state transitions
- policy evaluation
- approval rules
- checkpoint serialization
- repeated-step detection

### Integration Tests

- create task -> run -> complete
- create task -> block for approval -> approve -> continue
- create task -> crash mid-run -> restart -> resume
- tool timeout -> retry -> fail cleanly
- malformed model output -> repair path works

### Benchmark Tasks

- inspect repo and summarize architecture
- edit file and run tests
- search logs and explain failures

## Six-Week Suggested Schedule

### Week 1

- task schema
- migrations
- runner skeleton
- CLI task create, show, and list

### Week 2

- persistent step log
- resumable loop
- dashboard timeline
- first end-to-end task

### Week 3

- tool metadata
- policy engine
- approval flow
- blocked and resume UX

### Week 4

- retry and repair logic
- idempotency
- loop guards
- crash recovery tests

### Week 5

- observability
- benchmark suite
- operator controls
- docs

### Week 6

- beta hardening
- fix regressions
- internal tester rollout
- evaluate success and failure data

## Definition Of Done For Agent Runtime MVP

We can say we have an MVP when all are true:

- A task has its own durable identity and lifecycle
- The runtime survives restart and resumes work
- Risky actions require explicit approval
- Tool execution is policy-checked and logged
- Failures are inspectable step by step
- At least 3 real task classes succeed reliably
- There is a CLI and dashboard surface for operators

## Definition Of Done For Beta

- 5 or more internal users can use it
- success rate is measurable
- unsafe defaults are removed
- blocked tasks are explainable
- crashes do not lose task progress
- docs match reality

## What Not To Do Right Now

- Do not chase dozens of channels
- Do not overbuild multi-agent swarms yet
- Do not market production-grade runtime yet
- Do not rewrite the whole repo around a grand architecture
- Do not broaden tools before permissions and durability are solved

## Strong Recommendation

Start with this exact milestone:

### Milestone 1: Durable Task Runtime

If we finish only one thing this month, it should be:

- persistent tasks
- resumable execution
- approval blocking
- task timeline UI
- 3 passing end-to-end runtime scenarios

That milestone will change the repo's identity more than any number of new features.

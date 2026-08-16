# Architecture Documentation

Welcome to the architecture documentation of Payment Sandbox.

This directory explains **why** the system is designed the way it is.

The goal is not only to document implementation details, but also to capture the architectural thinking behind the project.

If you are new to the codebase, start here before diving into the source code.

---

# Reading Guide

The documentation is organised into three complementary parts.

```
Architecture Principles
        │
        ▼
Architectural Decision Records (ADRs)
        │
        ▼
Implementation
```

- **Architecture Principles** describe the long-term philosophy of the project.
- **Architectural Decision Records** explain specific design decisions.
- **The source code** demonstrates how those decisions are implemented.

---

# Architecture at a Glance 

```
                    Payment Sandbox

                  Payment Provider Simulator

                             │
         ┌───────────────────┴───────────────────┐
         │                                       │
         Deterministic                        Local First
         │                                       │
         └───────────────┬───────────────────────┘
         ▼
         Modular Monolith
         │
         ┌────────────────┴────────────────┐
         ▼                                 ▼
         Domain Model                   Hexagonal Architecture
         │                                 │
         └────────────────┬────────────────┘
         ▼
         SQLite (State)
         │
         ▼
         Durable Async Engine
         │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
         Event Log        Virtual Clock     Replay Engine
         │
         ▼
         Provider Plugins
         │
         ▼
         Observability
```

---

# Architecture Principles

Start by reading:

- [Architecture Principles](principles.md)

This document describes the values that guide every architectural decision.

Examples include:

- Local First
- Determinism
- Explicitness
- Inspectability
- Reliability
- Pragmatism

These principles should remain stable even if implementation details evolve.

---

# Bounded contexts

Read:

- [Bounded Contexts](bounded-contexts.md)

This document describes the bounded contexts.

---

# Architectural Decision Records

The ADRs should be read sequentially.

Each decision builds upon previous ones and introduces concepts that are reused throughout the rest of the architecture.

| ADR | Topic | Why it matters |
|------|-------|----------------|
| ADR-0001 | Modular Monolith | Defines the overall system structure and architectural boundaries. |
| ADR-0002 | Domain Model & Bounded Contexts | Organises the payment domain into cohesive business modules. |
| ADR-0003 | Targeted Hexagonal Architecture | Defines dependency boundaries and interaction patterns. |
| ADR-0004 | SQLite as the Default Persistence Engine | Explains why SQLite is the canonical persistence layer. |
| ADR-0005 | Persist Asynchronous Work | Defines durable asynchronous execution as part of the application state. |
| ADR-0006 | Payment State Machine | Defines the lifecycle of payments and valid state transitions. |
| ADR-0007 | Event Log & Audit Trail | Explains how business history is recorded and audited. |
| ADR-0008 | Observability & Diagnostics | Defines how system behaviour is inspected and diagnosed. |
| ADR-0009 | Deterministic Scenario Replay | Explains reproducible execution and scenario replay. |
| ADR-0010 | Virtual Clock | Defines deterministic control of time during execution. |
| ADR-0011 | Provider Plugin Model | Explains how payment providers extend the platform without modifying the core. |
---

# Relationships Between ADRs

```
                    Architecture Principles
                             │
                             ▼
                  ADR-0001 Modular Monolith
                             │
              ┌──────────────┴──────────────┐
              ▼                             ▼
 ADR-0002 Domain Model        ADR-0003 Hexagonal Architecture
              │                             │
              └──────────────┬──────────────┘
                             ▼
          ADR-0004 SQLite Persistence Engine
                             │
                             ▼
       ADR-0005 Durable Asynchronous Execution
                             │
                             ▼
          ADR-0006 Payment State Machine
                             │
                             ▼
            ADR-0007 Event Log & Audit Trail
                             │
                             ▼
       ADR-0008 Observability & Diagnostics
                             │
                             ▼
       ADR-0009 Deterministic Scenario Replay
                             │
                             ▼
               ADR-0010 Virtual Clock
                             │
                             ▼
            ADR-0011 Provider Plugin Model
```

The ADRs are intentionally ordered from foundational architectural concepts toward increasingly specialised capabilities.

Each decision introduces concepts that are reused by subsequent ADRs, allowing the documentation to progressively build a complete mental model of the system.
---

# Design Philosophy

Payment Sandbox intentionally optimises for:

- correctness before performance;
- determinism before concurrency;
- simplicity before infrastructure;
- observability before abstraction;
- developer experience before operational sophistication.

These priorities influence every architectural decision documented in this directory.

---

# About the Code

The implementation may evolve over time.

Libraries may change.

Internal structures may be refactored.

The architectural principles and ADRs describe the intended behaviour of the system rather than its current implementation details.

Whenever implementation and documentation diverge, the discrepancy should be treated as technical debt and resolved accordingly.
# Architecture Principles

Payment Sandbox is more than a payment API simulator.

It is an execution engine designed to reproduce the observable behaviour of real-world payment providers while remaining deterministic, inspectable and easy to understand.

These principles guide every architectural decision made throughout the project.

Architectural Decision Records (ADRs) explain individual decisions.

This document explains the philosophy behind those decisions.

---

# Principles at a Glance

| Principle | Why it Matters |
|------------|----------------|
| Local First | The project should work immediately after cloning. |
| Determinism | The same scenario should always produce the same behaviour. |
| Simplicity over Complexity | Avoid infrastructure unless it provides clear value. |
| Explicitness | Hidden behaviour is harder to understand and debug. |
| Inspectability | Every important state should be observable. |
| Reliability | Failure should not corrupt system behaviour. |
| Reproducibility | Bugs should be reproducible from persisted state. |
| Testability | Every architectural choice should improve automated testing. |
| Evolvability | New providers should integrate without changing the core. |
| Pragmatism | Choose the simplest solution that satisfies the requirements. |

---

# Local First

Payment Sandbox is designed to run locally without external infrastructure.

A developer should be able to execute:

```bash
git clone ...

go build

./payment-sandbox serve
```

without installing additional services.

Reducing setup friction encourages experimentation, learning and contribution.

Infrastructure should remain optional whenever possible.

---

# Determinism

Determinism is a fundamental property of the project.

Given:

- identical configuration;
- identical persisted state;
- identical virtual time;
- identical inputs;

the simulator should always produce identical observable behaviour.

Deterministic execution improves:

- debugging;
- testing;
- documentation;
- reproducibility.

Whenever multiple valid implementations exist, deterministic behaviour should be preferred.

---

# Simplicity over Complexity

Complexity has a long-term maintenance cost.

New technologies should not be introduced because they are fashionable or technically impressive.

Every dependency should solve an actual architectural problem.

The project intentionally favours:

- fewer moving parts;
- fewer runtime dependencies;
- fewer deployment requirements.

Simple systems are easier to understand, maintain and evolve.

---

# Explicitness

Hidden behaviour creates hidden bugs.

Whenever possible, important behaviour should be represented explicitly.

Examples include:

- persisted asynchronous jobs;
- retry schedules;
- provider configuration;
- execution history;
- state transitions.

If the system performs an important action, developers should be able to observe it.

---

# Inspectability

A simulator is also a debugging tool.

Developers should be able to answer questions such as:

- Why did this payment fail?
- Why has a webhook not been delivered?
- Which retries have already occurred?
- What will happen next?

The architecture should make these answers easy to obtain.

System behaviour should be visible rather than inferred.

---

# Reliability

Failures are expected.

The architecture should behave predictably in the presence of:

- process crashes;
- network failures;
- retries;
- duplicate requests;
- unexpected interruptions.

Recoverability is more valuable than attempting to avoid every possible failure.

---

# Reproducibility

Every execution should be reproducible.

Given a persisted database and the same configuration, developers should be able to replay scenarios and obtain identical behaviour.

Reproducibility significantly reduces the cost of investigating defects.

---

# Testability

Architecture should improve testing rather than complicate it.

The project should encourage:

- isolated unit tests;
- deterministic integration tests;
- scenario replay;
- automated verification.

Features that cannot be tested reliably should be reconsidered.

---

# Evolvability

Payment providers differ significantly.

The core architecture should remain stable while allowing provider-specific behaviour to evolve independently.

Adding support for a new provider should require extending the system rather than modifying existing core behaviour.

Whenever possible, the Open/Closed Principle should be respected.

---

# Pragmatism

Architecture exists to solve problems.

Not every pattern deserves to be implemented.

Not every abstraction deserves to exist.

The project intentionally prefers practical solutions over architectural purity.

Design decisions should be justified by measurable benefits rather than theoretical elegance.

---

# How These Principles Influence Decisions

The following ADRs demonstrate how these architectural principles are applied throughout the project.

| ADR | Primary Principles |
|------|--------------------|
| ADR-0001 | Simplicity, Evolvability |
| ADR-0002 | Explicitness, Evolvability |
| ADR-0003 | Testability, Pragmatism |
| ADR-0004 | Local First, Reliability |
| ADR-0005 | Determinism, Reliability |
| ADR-0006 | Explicitness, Determinism |
| ADR-0007 | Reliability, Inspectability |
| ADR-0008 | Inspectability, Testability |
| ADR-0009 | Reproducibility, Determinism |
| ADR-0010 | Determinism, Testability |
| ADR-0011 | Evolvability, Pragmatism |

Architectural principles should remain relatively stable over time.

Implementation details may evolve.

Technologies may change.

Patterns may be be replaced.

The principles described in this document should continue to guide future architectural decisions.

---

# Architectural Themes

Although the ADRs are intended to be read sequentially, they naturally group into a small number of architectural themes.

## Foundations

These ADRs establish the structural rules of the system.

- ADR-0001 — Modular Monolith
- ADR-0002 — Domain Model & Bounded Contexts
- ADR-0003 — Targeted Hexagonal Architecture

Together, they define how the codebase is organised and how components interact.

---

## Execution Engine

These ADRs define how Payment Sandbox behaves at runtime.

- ADR-0004 — SQLite as the Default Persistence Engine
- ADR-0005 — Persist Asynchronous Work
- ADR-0006 — Payment State Machine
- ADR-0007 — Event Log & Audit Trail

Together, they describe how business state, asynchronous execution and historical information are represented.

---

## Platform Capabilities

These ADRs build upon the execution engine to provide higher-level capabilities.

- ADR-0008 — Observability & Diagnostics
- ADR-0009 — Deterministic Scenario Replay
- ADR-0010 — Virtual Clock
- ADR-0011 — Provider Plugin Model

Together, they explain how the platform becomes observable, reproducible and extensible.

---

# Non-Goals

Payment Sandbox does not aim to become:

- a production payment gateway;
- a distributed payment platform;
- a high-throughput transaction processor;
- a workflow orchestration engine;
- a generic message broker.

The project intentionally focuses on accurately simulating payment provider behaviour while remaining simple to operate and understand.

Architectural decisions should reinforce this scope rather than expand it unnecessarily.
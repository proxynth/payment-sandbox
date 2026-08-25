# ADR-0010: Virtual Clock

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox Maintainers
- **Tags:** Time, Determinism, Simulation

---

# Executive Summary

Payment Sandbox treats time as an explicit architectural concept.

Rather than relying directly on the operating system clock, the simulator executes against a virtual clock whose progression is fully controlled by the platform.

The virtual clock enables deterministic execution of time-dependent business behaviour, including:

- delayed webhook delivery;
- retry scheduling;
- payment expiration;
- timeout processing;
- provider-specific delays.

By making time an explicit input rather than an external dependency, Payment Sandbox guarantees reproducible behaviour across executions, environments and replay scenarios.

---

# Context

Time influences nearly every payment provider.

Business operations frequently depend on temporal conditions.

Examples include:

- webhook retries after a delay;
- payment authorisation expiration;
- delayed settlement;
- scheduled captures;
- provider cooldown periods;
- timeout policies.

In most applications, these behaviours depend directly on the system clock.

For example:

```go
time.Now()
```

or

```go
time.After(...)
```

Although convenient, this approach tightly couples business behaviour to real-world time.

Execution therefore becomes dependent upon factors outside the control of the simulator.

---

# Problem Statement

Real time introduces unavoidable non-determinism.

For example:

- different execution speeds;
- different machines;
- different time zones;
- operating system scheduling;
- test execution delays;
- CI performance variations.

Consider a webhook scheduled five minutes after payment capture.

During one execution:

```text
Capture

↓

5 minutes later

↓

Webhook Delivered
```

During another execution, the webhook may be delayed by:

- machine load;
- scheduler latency;
- debugger pauses.

Although business behaviour should remain identical, observable execution differs.

Such variability makes deterministic replay significantly more difficult.

It also complicates automated testing and provider validation.

---

# Decision

Payment Sandbox introduces a virtual clock that represents the canonical source of time throughout the simulator.

Business components must obtain the current time exclusively from this virtual clock.

The operating system clock must not directly influence business behaviour.

The virtual clock controls:

- current simulated time;
- scheduled execution;
- retry delays;
- expiration policies;
- provider timing rules.

Time therefore becomes part of the simulator's business context rather than an implicit property of the execution environment.

---

# Architectural Principles

Payment Sandbox adopts the following principles.

> **Time is an input.**

Business behaviour should depend upon explicit temporal information rather than hidden environmental state.

---

> **Time must be controllable.**

The simulator must be able to:

- pause time;
- advance time;
- reproduce time;
- inspect time.

Business execution should never depend upon waiting for real time.

---

> **Business time is more important than wall-clock time.**

The objective is to reproduce provider behaviour faithfully.

Real elapsed time is irrelevant if identical business behaviour can be achieved through deterministic virtual time.

---

# Scope

This ADR applies to every business capability whose behaviour depends on time.

Examples include:

- retry scheduling;
- webhook delivery;
- delayed provider operations;
- payment expiration;
- timeout processing;
- deterministic replay;
- scenario execution.

Infrastructure concerns such as:

- process uptime;
- operating system clocks;
- log timestamps;
- deployment monitoring;

are intentionally outside the scope of this decision.

The virtual clock governs business behaviour.

It does not replace the system clock for infrastructure concerns.

# Why Time Is Part of the Domain

Many business behaviours in payment providers depend on time.

Examples include:

- payment authorisation expiration;
- delayed captures;
- webhook retries;
- settlement windows;
- timeout policies;
- scheduled callbacks.

Time is therefore not merely an implementation detail.

It directly influences business decisions.

---

# Real Time Is Non-Deterministic

The operating system clock continuously progresses independently of the simulator.

During one execution:

```text
Capture Payment

↓

Wait 5 Minutes

↓

Retry Webhook
```

During another execution, the exact same business scenario may execute differently because:

- the machine is slower;
- the scheduler is busy;
- a debugger pauses execution;
- CI executes under heavier load.

The business rules remain identical.

The observed behaviour does not.

---

# Waiting Is Not Simulation

Many systems implement delayed behaviour by literally waiting.

For example:

```go
time.Sleep(5 * time.Minute)
```

or

```go
time.After(...)
```

This approach is suitable for production infrastructure.

It is inappropriate for a deterministic simulator.

Business behaviour should not require developers to wait for real time to pass.

Simulation should advance because the simulator decides to advance time.

---

# Time Becomes Business State

The virtual clock makes current time part of the simulator itself.

Conceptually:

```text
Business State

+

Virtual Time

↓

Business Decision
```

A retry scheduled for fifteen minutes later is no longer interpreted relative to the operating system.

Instead, it is evaluated relative to the current virtual time.

Time therefore becomes another explicit business input.

---

# Scheduled Work Depends on Time

Durable jobs are evaluated against the virtual clock.

For example:

```text
Current Virtual Time

15:00

↓

Webhook Scheduled

15:05

↓

Not Ready
```

Later:

```text
Current Virtual Time

15:05

↓

Webhook Scheduled

15:05

↓

Ready
```

The scheduler therefore evaluates business time rather than wall-clock time.

---

# Replay Requires Stable Time

Replay reproduces business behaviour under identical execution conditions.

Without controlling time, replay could produce different outcomes despite identical business inputs.

For example:

```text
Scenario

↓

Capture

↓

Retry Scheduled

↓

Replay One Minute Later
```

would naturally produce different behaviour from:

```text
Scenario

↓

Capture

↓

Retry Scheduled

↓

Replay Thirty Minutes Later
```

The virtual clock removes this uncertainty.

Replay always begins from the same temporal context.

---

# Testing Without Waiting

Virtual time dramatically simplifies automated testing.

Instead of waiting for delays to expire, tests advance time explicitly.

Conceptually:

```text
Current Time

15:00

↓

Advance Time

+5 Minutes

↓

Execute Scheduler

↓

Webhook Delivered
```

Tests therefore remain:

- fast;
- deterministic;
- reproducible.

Business rules are verified without introducing artificial delays.

---

# Provider Behaviour Becomes Predictable

Different payment providers implement different temporal rules.

Examples include:

- retry intervals;
- payment expiration;
- settlement delays;
- asynchronous notifications.

Because every provider uses the same virtual clock, these behaviours become completely reproducible.

Provider implementations differ only in business rules.

They do not depend on the execution environment.

---

# Time Can Be Observed

The virtual clock also improves observability.

Developers can immediately inspect:

- current simulated time;
- scheduled execution time;
- remaining delay;
- expired timers.

For example:

```text
Virtual Time

15:02 UTC

↓

Next Retry

15:05 UTC

↓

Remaining

3 Minutes
```

Time therefore becomes another observable business concept.

---

# A Foundation Built by Previous ADRs

The virtual clock completes the deterministic architecture established by previous decisions.

- **ADR-0004** provides deterministic persisted state.
- **ADR-0005** schedules future work explicitly.
- **ADR-0006** defines deterministic business transitions.
- **ADR-0007** records immutable business history.
- **ADR-0008** exposes temporal behaviour through diagnostics.
- **ADR-0009** reproduces complete business scenarios.

The virtual clock removes the final major source of environmental variability.

Business behavior no longer depends on the operating system clock.

It depends entirely on explicit business state and controlled simulated time.

# Architectural Model

The virtual clock acts as the single authoritative source of business time throughout the simulator.

Every component whose behaviour depends on time must obtain temporal information from the virtual clock rather than from the operating system.

Conceptually:

```text
                 Virtual Clock
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
 Payment Domain   Job Scheduler   Provider Adapter
        │              │              │
        └──────────────┼──────────────┘
                       ▼
              Time-Dependent Behaviour
```

Business time therefore becomes an explicit architectural dependency.

---

# A Single Source of Time

The architecture intentionally defines only one notion of business time.

Whether evaluating:

- payment expiration;
- webhook scheduling;
- retry policies;
- timeout processing;
- delayed captures;

every component observes the same virtual time.

This eliminates inconsistencies caused by components consulting different clocks.

---

# Time Is Injected, Never Retrieved

Business components must not retrieve time from the execution environment.

Instead, time is provided by the architecture.

Conceptually:

```text
Business Operation

↓

Virtual Clock

↓

Current Time

↓

Business Decision
```

The domain never depends directly upon:

- operating system clocks;
- local machine time;
- UTC conversion logic;
- execution timing.

Business logic consumes time.

It never owns time.

---

# Scheduler Execution

The scheduler evaluates pending jobs against the virtual clock.

Conceptually:

```text
Pending Job

↓

Execute At

15:05

↓

Virtual Clock

15:03

↓

Not Ready
```

Later:

```text
Pending Job

↓

Execute At

15:05

↓

Virtual Clock

15:05

↓

Execute
```

The scheduler does not determine when time progresses.

It merely evaluates whether execution conditions have been satisfied.

---

# Time Progression

Advancing time is an explicit architectural operation.

Conceptually:

```text
15:00

↓

Advance Time (+5 min)

↓

15:05

↓

Evaluate Pending Jobs

↓

Execute Eligible Jobs
```

Time never progresses implicitly as a consequence of execution.

The simulator controls temporal progression explicitly.

---

# Provider Behaviour

Provider implementations may define different temporal rules.

Examples include:

- retry intervals;
- settlement delays;
- payment expiration windows;
- asynchronous callback delays.

However, every provider evaluates these rules using the same virtual clock.

For example:

```text
Provider Rule

Retry After 15 Minutes

↓

Virtual Clock

15:15

↓

Retry Eligible
```

Provider behaviour remains deterministic because temporal evaluation is consistent across all implementations.

---

# Interaction with Durable Jobs

The virtual clock directly complements the durable job model introduced in ADR-0005.

Jobs already contain their execution conditions.

For example:

```text
Webhook Delivery

Execute At

15:30 UTC
```

The virtual clock determines whether those conditions are currently satisfied.

Conceptually:

```text
Virtual Clock

↓

Current Time

↓

Pending Jobs

↓

Eligible Jobs

↓

Worker Execution
```

Time determines eligibility.

The scheduler determines execution.

Workers perform the work.

Each responsibility remains clearly separated.

---

# Interaction with Replay

Replay restores the virtual clock before business execution begins.

Conceptually:

```text
Recorded Scenario

↓

Restore Virtual Time

↓

Execute Commands

↓

Advance Time

↓

Execute Jobs
```

Every replay therefore begins from the same temporal conditions.

No behaviour depends upon the actual date or time at which replay is executed.

---

# Interaction with Observability

The current virtual time forms part of the platform's observable business state.

Diagnostic interfaces may expose information such as:

- current virtual time;
- scheduled execution time;
- remaining delay;
- expired timers.

For example:

```text
Current Time

15:20 UTC

↓

Next Job

15:25 UTC

↓

Remaining

5 Minutes
```

Temporal behaviour becomes visible without inspecting implementation details.

---

# Isolation from Infrastructure

The virtual clock intentionally isolates business behaviour from infrastructure concerns.

Infrastructure may continue using the operating system clock for purposes such as:

- process startup;
- log timestamps;
- monitoring;
- deployment metrics.

These concerns remain outside the business model.

Only business behaviour depends upon virtual time.

---

# Interaction with Other ADRs

The virtual clock integrates naturally with previous architectural decisions.

- **ADR-0004** persists business state independently from wall-clock time.
- **ADR-0005** evaluates durable jobs using virtual time.
- **ADR-0006** applies temporal business rules through deterministic lifecycle transitions.
- **ADR-0007** records time-dependent business events consistently.
- **ADR-0008** exposes temporal information through diagnostic models.
- **ADR-0009** restores virtual time to reproduce complete business scenarios.

Together, these decisions make temporal behaviour deterministic, reproducible and observable throughout the simulator.

# Alternatives Considered

Several approaches were evaluated before introducing a virtual clock as the canonical source of business time.

---

## Operating System Clock

The simplest solution consists of relying directly on the operating system clock.

For example:

```go
time.Now()
```

or

```go
time.After(...)
```

This approach is appropriate for many production applications.

However, it tightly couples business behaviour to the execution environment.

Business execution becomes dependent upon factors such as:

- machine performance;
- scheduler latency;
- execution timing;
- current date and time.

Payment Sandbox instead requires deterministic business behaviour.

---

## Injected Clock Interface

Another common approach consists of introducing an abstract clock interface.

For example:

```text
Business Logic

↓

Clock Interface

↓

System Clock
```

Although this improves testability, it does not define how time behaves throughout the architecture.

Different components may still observe different temporal contexts.

Payment Sandbox therefore elevates time from a technical abstraction to an architectural concept.

The virtual clock represents a shared business timeline rather than merely an injectable dependency.

---

## Waiting for Real Time

Delayed operations could simply wait until sufficient real time has elapsed.

For example:

```text
Capture

↓

Wait Five Minutes

↓

Retry
```

While simple, this approach is unsuitable for deterministic simulation.

Long-running delays slow development, complicate automated testing and prevent efficient replay.

Business execution should never depend upon waiting for wall-clock time.

---

## Mocking Time in Tests

Many applications replace the system clock with mocks during testing.

Although useful for isolated unit tests, mocked clocks do not provide a consistent temporal model across the entire platform.

Payment Sandbox instead defines a single virtual timeline shared by every architectural component.

Testing naturally benefits from this decision, but it is not its primary objective.

---

# Accepted Trade-offs

Introducing a virtual clock influences the entire architecture.

These consequences are intentionally accepted.

---

## Explicit Time Management

Business components must obtain temporal information from the virtual clock.

Direct access to the operating system clock is no longer acceptable within business logic.

This introduces additional architectural discipline while significantly improving determinism.

---

## Additional Architectural Component

The simulator now owns a dedicated temporal model.

This increases architectural complexity slightly.

However, time becomes explicit, inspectable and reproducible throughout the platform.

---

## Existing Code Requires Adaptation

Any component currently depending upon:

- `time.Now()`;
- `time.After()`;
- `time.Sleep()`;

must instead rely upon the virtual clock.

Although this requires additional implementation effort, it removes hidden temporal dependencies.

---

## Time Progression Must Be Controlled

Advancing time becomes an explicit operation.

Developers must intentionally decide when simulated time progresses.

This differs from traditional applications where time advances automatically.

The resulting execution model is more predictable and easier to reason about.

---

# Implementation Guidelines

The following architectural rules apply.

---

## The Virtual Clock Is the Only Business Clock

Business components should obtain temporal information exclusively from the virtual clock.

Direct calls to the operating system clock should be avoided within business logic.

---

## Business Logic Must Not Sleep

Business execution should never depend upon:

- `time.Sleep()`;
- blocking delays;
- waiting for real time.

Delayed behaviour should instead be modelled using scheduled work evaluated against virtual time.

---

## Time Progression Must Be Explicit

Advancing simulated time is an explicit architectural operation.

Business execution must never advance time implicitly.

This guarantees predictable execution.

---

## Scheduler Decisions Depend on Virtual Time

Schedulers determine execution eligibility by comparing scheduled execution time with the current virtual time.

The scheduler does not own time.

Lease recovery is an operational concern rather than business behaviour. The
runtime therefore supplies a separate wall-clock source when checking whether
a persisted lease has expired, while the virtual clock remains the source for
job eligibility, retry scheduling and provider decisions. This separation
keeps crash recovery practical without making deterministic business execution
depend on elapsed wall-clock time.

It only evaluates temporal conditions.

---

## Replay Must Restore Time

Replay execution should restore the original virtual time before business execution begins.

Temporal context therefore forms part of every deterministic scenario.

---

## Time Must Be Observable

Diagnostic interfaces should expose:

- current virtual time;
- scheduled execution time;
- remaining delays;
- expired work.

Temporal behaviour should be inspectable without examining implementation details.

---

# What This Decision Is Not

This ADR does not require:

- replacing infrastructure timestamps;
- deterministic operating system scheduling;
- replacing the system clock for logging;
- custom timezone management;
- distributed logical clocks;
- Lamport clocks;
- vector clocks;
- synchronised cluster clocks.

The virtual clock exists exclusively to govern business behaviour within the simulator.

Infrastructure continues using conventional system time where appropriate.

---

# Revisit Conditions

This ADR should be revisited if:

- the simulator evolves into a distributed system requiring multiple coordinated timelines;
- provider integrations require multiple independent business clocks;
- replay introduces temporal requirements incompatible with a single virtual timeline;
- future architectural capabilities require richer temporal semantics.

Future changes should preserve the principle that business time remains deterministic and explicitly controlled.

---

# Consequences

## Positive

- Time-dependent behaviour becomes deterministic.
- Replay scenarios become fully reproducible.
- Automated tests execute without waiting.
- Scheduled work becomes predictable.
- Temporal behaviour becomes observable.
- Provider implementations remain consistent across environments.

## Negative

- Business code can no longer depend directly upon the operating system clock.
- Time progression must be managed explicitly.
- Existing implementations require adaptation.
- Developers must understand the distinction between business time and infrastructure time.

These costs are accepted because deterministic temporal behaviour is one of the defining characteristics of Payment Sandbox.

---

# Decision Summary

Payment Sandbox models time as an explicit architectural capability through a virtual clock.

Business components no longer depend upon the operating system clock.

Instead, temporal behaviour is governed by a shared, controllable and observable virtual timeline.

This decision enables deterministic scheduling, reproducible replay, efficient automated testing and predictable provider behaviour.

The virtual clock therefore completes the architectural foundation required for a deterministic payment simulation platform.

---

# References

- Eric Evans — *Domain-Driven Design*
- Martin Fowler — *Time Patterns*
- Martin Fowler — *Temporal Patterns*
- Michael Feathers — *Working Effectively with Legacy Code*
- ADR-0005 — Persist Asynchronous Work
- ADR-0008 — Observability & Diagnostics
- ADR-0009 — Deterministic Scenario Replay
- ADR-0011 — Provider Plugin Model

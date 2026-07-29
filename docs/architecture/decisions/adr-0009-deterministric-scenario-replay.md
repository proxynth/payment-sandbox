# ADR-0009: Deterministic Scenario Replay

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox Maintainers
- **Tags:** Replay, Determinism, Testing, Diagnostics

---

# Executive Summary

Payment Sandbox provides deterministic replay of payment scenarios.

Given the same:

- initial business state;
- provider configuration;
- virtual time;
- business inputs;

the simulator must always reproduce the same observable behaviour.

Replay enables developers to reproduce complex payment scenarios without relying on external providers or attempting to manually recreate asynchronous execution.

Deterministic replay is therefore considered a core architectural capability rather than a testing convenience.

---

# Context

Modern payment providers execute numerous asynchronous operations.

A single payment may involve:

- multiple lifecycle transitions;
- webhook deliveries;
- retry policies;
- delayed operations;
- provider-specific behaviour;
- virtual time progression.

Many of these interactions occur over extended periods.

When unexpected behaviour occurs, developers often attempt to reproduce the issue.

Unfortunately, reproduction is rarely straightforward.

External providers evolve.

Network conditions change.

Timing differs.

Retries occur differently.

Even small variations may produce completely different execution paths.

For a simulator whose primary objective is predictability, this level of uncertainty is unacceptable.

---

# Problem Statement

Without deterministic replay, understanding complex scenarios becomes significantly more difficult.

Consider a payment that eventually reaches the following state:

```text
Captured
```

The current state alone provides no explanation.

Even the Event Log only records what happened during the original execution.

Developers still need a reliable way to answer questions such as:

- Can this behaviour be reproduced?
- Will this bug occur again?
- Which sequence of operations produced this outcome?
- Does a recent code change alter provider behaviour?
- Is the simulator still behaving identically?

Without deterministic replay, these questions often require rebuilding the scenario manually.

Manual reproduction is:

- slow;
- error-prone;
- difficult to automate;
- rarely identical to the original execution.

---

# Decision

Payment Sandbox supports deterministic replay of complete business scenarios.

Replay executes the same sequence of business operations under the same execution conditions.

Provided that the following remain unchanged:

- business inputs;
- provider configuration;
- virtual time;
- simulator version;

replay must produce the same:

- payment transitions;
- business events;
- durable jobs;
- webhook behaviour;
- observable outcomes.

Replay is therefore expected to be reproducible rather than approximate.

---

# Architectural Principles

Payment Sandbox adopts the following principles.

> **Identical inputs produce identical observable behaviour.**

Randomness, infrastructure timing and implementation details must not influence replay results.

---

> **Replay reproduces business behaviour, not infrastructure behaviour.**

The objective is not to recreate HTTP requests, SQL execution order or thread scheduling.

Instead, replay reproduces the business decisions that define provider behaviour.

---

> **Reproducibility is a feature.**

Replay is not reserved for debugging.

It supports:

- automated testing;
- regression detection;
- provider validation;
- documentation;
- developer education.

Deterministic behaviour is therefore part of the product itself.

---

# Scope

This ADR applies to every capability involved in reproducing payment scenarios.

This includes:

- payment lifecycle execution;
- business events;
- durable asynchronous jobs;
- virtual time progression;
- provider-specific behaviour;
- observable business outcomes.

Infrastructure implementation details such as:

- SQL execution plans;
- operating system scheduling;
- network latency;
- thread execution order;

are intentionally outside the scope of replay.

Replay focuses exclusively on business behavior.

# Why Deterministic Replay Is Possible

Deterministic replay is not implemented through a dedicated replay engine alone.

Instead, it emerges from the architectural decisions that define how Payment Sandbox models business behaviour.

Replay is possible because the simulator intentionally eliminates unnecessary sources of non-determinism.

---

# Determinism Begins with the Domain

Business behaviour is deterministic by construction.

The payment aggregate defines explicit transition rules.

Given:

- the same aggregate state;
- the same command;
- the same business rules;

the aggregate always produces the same outcome.

For example:

```text
Authorized

↓

Capture(100)

↓

Captured
```

The aggregate does not depend on:

- execution timing;
- infrastructure;
- operating system scheduling.

Its behaviour is therefore reproducible.

---

# Durable State Eliminates Uncertainty

Current business state is persisted explicitly.

Replay therefore starts from a well-defined initial condition.

There is no need to reconstruct the system from partially available information.

The aggregate represents the canonical business state from which replay begins.

---

# Business Events Explain Previous Decisions

The Event Log preserves the sequence of business transitions.

For example:

```text
PaymentCreated

↓

PaymentAuthorized

↓

PaymentCaptured
```

These events explain how the original execution evolved.

Replay can therefore validate that the reproduced execution follows the same observable business history.

Historical events become a reference for expected behaviour rather than the execution mechanism itself.

---

# Durable Jobs Preserve Future Behaviour

Future work is explicitly represented through durable jobs.

Examples include:

- webhook deliveries;
- retries;
- delayed captures;
- timeout processing.

Because future work is persisted, replay knows which business actions must occur.

Nothing depends on transient in-memory scheduling.

---

# Virtual Time Removes Real Time

Real clocks introduce uncertainty.

For example:

- different execution speeds;
- different machines;
- different time zones;
- different scheduling delays.

Payment Sandbox instead relies on virtual time.

Time progresses according to the simulator rather than the operating system.

Temporal behaviour therefore becomes reproducible.

---

# Provider Behaviour Is Explicit

Replay requires provider behaviour to be deterministic as well.

Provider implementations should therefore expose:

- retry policies;
- payment lifecycle;
- timeout behaviour;
- webhook strategy;
- asynchronous operations.

These behaviours become part of the simulated business model.

Replay never depends on undocumented provider implementation details.

---

# Replay Focuses on Observable Behaviour

Replay does not attempt to reproduce every internal implementation detail.

For example, it does not require identical:

- memory allocation;
- SQL execution order;
- goroutine scheduling;
- HTTP connection reuse.

Instead, replay verifies that the externally observable business behaviour remains identical.

Observable behaviour includes:

- payment state;
- business events;
- durable jobs;
- webhook behaviour;
- provider responses.

---

# Stable Inputs Produce Stable Outputs

Replay assumes identical inputs.

Conceptually:

```text
Initial State

+

Commands

+

Provider Configuration

+

Virtual Time

↓

Replay

↓

Identical Observable Behaviour
```

Changing any of these inputs may legitimately produce different results.

Determinism therefore applies only under equivalent execution conditions.

---

# Regression Detection

One of the primary objectives of replay is regression detection.

Suppose a previously recorded scenario produced:

```text
Authorized

↓

Captured

↓

Webhook Delivered
```

After modifying the simulator, replay should produce the same sequence.

If the resulting behaviour changes unexpectedly, replay immediately reveals the regression.

Deterministic replay therefore provides confidence that architectural evolution has not altered established business behaviour.

---

# A Consequence of Previous Decisions

Replay is made possible by the architectural foundations established in earlier ADRs.

- **ADR-0004** provides deterministic persisted state.
- **ADR-0005** makes future work explicit through durable jobs.
- **ADR-0006** defines deterministic lifecycle transitions.
- **ADR-0007** preserves immutable business history.
- **ADR-0008** exposes the information necessary to understand replay results.

Replay is therefore not an isolated subsystem.

It is the natural consequence of an architecture designed for determinism from the beginning.

# Architectural Model

Replay reproduces business execution rather than restoring historical execution.

Instead of replaying persisted events directly, the simulator executes the domain model under the same conditions as the original scenario.

Conceptually:

```text
Recorded Scenario

↓

Restore Initial State

↓

Restore Provider Configuration

↓

Restore Virtual Time

↓

Execute Business Commands

↓

Generate New Behaviour

↓

Compare Observable Results
```

Replay therefore validates behaviour instead of reproducing implementation details.

---

# Scenario Definition

A replay scenario represents everything required to reproduce business behaviour.

Conceptually, a scenario consists of:

- initial aggregate state;
- business commands;
- provider configuration;
- virtual time;
- deterministic configuration.

The scenario intentionally excludes implementation-specific information such as:

- SQL execution plans;
- thread scheduling;
- memory layout;
- HTTP connection reuse.

Replay focuses exclusively on business execution.

---

# Restoring Initial Conditions

Replay begins by restoring the original business context.

Conceptually:

```text
Scenario

↓

Initial Payment State

↓

Provider Configuration

↓

Virtual Clock

↓

Ready for Execution
```

The simulator therefore begins execution from exactly the same business conditions.

No previous runtime state is reused.

Every replay starts from a clean environment.

---

# Re-executing the Domain

Once the initial context has been restored, replay executes the original business commands.

For example:

```text
Create Payment

↓

Authorize

↓

Capture

↓

Refund
```

Each command follows the same execution pipeline used during normal operation.

Conceptually:

```text
Command

↓

Aggregate

↓

State Transition

↓

Business Event

↓

Durable Jobs
```

Replay therefore exercises the production domain model rather than a simplified replay engine.

---

# Deterministic Asynchronous Execution

Replay must also reproduce asynchronous behaviour.

Durable jobs are executed according to the same business rules as during the original execution.

For example:

```text
Webhook Scheduled

↓

Virtual Time Advances

↓

Webhook Delivered
```

Execution order is determined by business semantics rather than operating system scheduling.

Asynchronous execution therefore remains deterministic.

---

# Virtual Time Drives Progress

The replay engine never waits for real time.

Instead, execution progresses according to the virtual clock.

Conceptually:

```text
Current Virtual Time

↓

Next Scheduled Job

↓

Advance Time

↓

Execute Job

↓

Repeat
```

Time becomes another deterministic input rather than an external dependency.

---

# Observable Behaviour Is Compared

Replay evaluates observable behaviour instead of implementation details.

Typical comparison points include:

- payment lifecycle;
- monetary progression;
- business events;
- durable jobs;
- webhook outcomes;
- provider responses.

For example:

```text
Original

Captured

↓

Replay

Captured

✓
```

or

```text
Original

Webhook Delivered

↓

Replay

Webhook Delivered

✓
```

The objective is behavioural equivalence.

---

# Behavioural Differences

A replay may legitimately produce different results.

For example:

- provider configuration changed;
- business rules evolved intentionally;
- scenario inputs differ.

Unexpected differences indicate potential regressions.

Replay should therefore make behavioural differences explicit.

For example:

```text
Original

Captured

↓

Replay

Failed

↓

Regression Detected
```

The purpose of replay is not merely to execute scenarios.

It is to explain whether behaviour has changed.

---

# Repeatability

Replay should be repeatable without limitation.

Executing the same scenario multiple times under identical conditions should always produce the same observable behaviour.

Conceptually:

```text
Scenario

↓

Replay #1

↓

Captured

Replay #2

↓

Captured

Replay #3

↓

Captured
```

Deterministic behaviour enables replay to become a reliable architectural capability rather than a probabilistic testing technique.

---

# Interaction with Other ADRs

Replay combines several architectural capabilities introduced earlier.

- **ADR-0004** restores deterministic persisted business state.
- **ADR-0005** reproduces durable asynchronous work.
- **ADR-0006** re-executes deterministic payment lifecycle transitions.
- **ADR-0007** compares newly generated business history against historical events.
- **ADR-0008** exposes replay execution through structured diagnostics.

Replay also establishes the foundation for **ADR-0010**, where virtual time becomes a controllable execution mechanism rather than a passive source of timestamps.

# Alternatives Considered

Several alternative approaches were considered before adopting deterministic scenario replay.

---

## Manual Reproduction

The simplest approach would consist of manually reproducing payment scenarios.

Developers would recreate:

- payment requests;
- provider configuration;
- webhook execution;
- retry behaviour.

While straightforward for simple flows, this approach quickly becomes impractical for complex asynchronous scenarios.

Even minor differences in execution timing may produce different behaviour.

Manual reproduction therefore provides little confidence that the original scenario has actually been reproduced.

---

## Replaying Persisted Events

Another option would consist of replaying the persisted Event Log directly.

```text
PaymentCreated

↓

PaymentAuthorized

↓

PaymentCaptured
```

Although this resembles Event Sourcing, it does not validate the behaviour of the simulator itself.

It merely reconstructs a previously recorded history.

Payment Sandbox instead re-executes the domain model.

The objective is to verify that the current implementation still produces the same observable behaviour.

---

## Snapshot Restoration

Replay could begin from an intermediate snapshot.

For example:

```text
Payment

Status: Authorized

↓

Replay Capture
```

Snapshots reduce execution time.

However, they bypass part of the business execution.

Payment Sandbox prioritises behavioural correctness over replay performance.

Scenarios therefore begin from a well-defined initial business state.

Future optimisations may introduce snapshots without changing replay semantics.

---

## Infrastructure-Level Replay

Another possibility would be reproducing:

- HTTP traffic;
- SQL statements;
- goroutine scheduling;
- operating system timing.

This would significantly increase implementation complexity.

More importantly, infrastructure behaviour is not part of the business contract exposed by the simulator.

Replay intentionally focuses on business semantics rather than runtime implementation details.

---

# Accepted Trade-offs

Deterministic replay introduces several architectural constraints.

These constraints are intentionally accepted.

---

## Greater Architectural Discipline

Deterministic systems require stricter modelling.

Business behaviour must not depend upon:

- execution timing;
- random values;
- hidden mutable state;
- implementation side effects.

This constraint influences the entire architecture.

---

## Additional Scenario Data

Replay scenarios must preserve sufficient information to reproduce execution.

Examples include:

- business commands;
- provider configuration;
- virtual time;
- deterministic identifiers where required.

Scenario descriptions therefore become richer than ordinary test cases.

---

## Slower Than State Restoration

Executing a complete business scenario is generally slower than restoring a snapshot.

The project accepts this overhead because replay validates business behaviour rather than merely restoring state.

Correctness takes precedence over raw execution speed.

---

## Stable Behaviour Becomes a Contract

Once replay scenarios exist, behavioural changes become visible.

Intentional changes require updating the corresponding scenarios.

This increases maintenance effort while significantly improving regression detection.

---

# Implementation Guidelines

The following architectural rules apply.

---

## Replay Must Execute Production Code

Replay should execute the same business logic used during normal operation.

Dedicated replay-specific implementations should be avoided.

The production domain model remains the single source of business behaviour.

---

## Replay Must Start from a Clean Environment

Every replay begins from an isolated execution context.

No mutable runtime state should survive between executions.

Replay must never depend upon previous executions.

---

## Business Inputs Must Be Explicit

Everything capable of influencing business behaviour should be represented explicitly within the scenario.

Hidden configuration should be avoided.

A scenario should completely describe its own execution conditions.

---

## Real Time Must Not Influence Replay

Replay must rely exclusively on virtual time.

The operating system clock should never determine business behaviour.

Temporal progression must remain deterministic.

---

## Randomness Must Be Controlled

Sources of randomness should either be eliminated or made deterministic.

Examples include:

- random identifiers;
- retry jitter;
- generated reference values.

Where randomness cannot be avoided, deterministic seeds should be used.

---

## Observable Behaviour Is the Contract

Replay validates observable business behaviour.

Implementation details such as:

- SQL execution order;
- memory allocation;
- goroutine scheduling;
- internal optimisation;

are intentionally excluded from comparison.

Only externally observable behaviour forms part of the architectural contract.

---

## Behavioural Differences Should Be Explainable

Replay should report meaningful behavioural differences.

For example:

```text
Expected

PaymentCaptured

Observed

PaymentFailed

Reason

Capture limit exceeded
```

Diagnostic output should help developers understand why behaviour changed.

---

# What This Decision Is Not

This ADR does not require:

- Event Sourcing;
- time-travel debugging;
- deterministic operating system scheduling;
- deterministic memory allocation;
- replaying SQL statements;
- replaying HTTP packets;
- reproducing infrastructure failures.

Replay concerns business behaviour.

Infrastructure execution remains an implementation detail.

---

# Revisit Conditions

This ADR should be revisited if:

- replay scenarios become prohibitively expensive to execute;
- provider integrations introduce unavoidable non-determinism;
- distributed execution becomes a primary architectural concern;
- snapshot-based optimisation becomes necessary;
- replay requirements evolve beyond business behaviour into infrastructure simulation.

Any future evolution should preserve the principle that identical business conditions produce identical observable behaviour.

---

# Consequences

## Positive

- Complex payment scenarios become reproducible.
- Regression detection becomes significantly more reliable.
- Behavioural changes become immediately visible.
- Provider implementations can be validated consistently.
- Replay scenarios become executable documentation.
- Developers gain confidence when evolving the simulator.

## Negative

- Greater architectural discipline is required.
- Replay scenarios require ongoing maintenance.
- Deterministic behaviour limits certain implementation techniques.
- Rich scenarios require more metadata than traditional tests.
- Full scenario execution is slower than restoring snapshots.

These costs are accepted because reproducibility is one of the primary objectives of Payment Sandbox.

---

# Decision Summary

Payment Sandbox reproduces payment scenarios by re-executing the domain model under deterministic business conditions.

Replay validates business behaviour rather than infrastructure execution.

Given identical:

- initial state;
- business commands;
- provider configuration;
- virtual time;

the simulator is expected to produce identical observable outcomes.

Deterministic replay therefore becomes a fundamental architectural capability supporting:

- regression detection;
- executable documentation;
- provider validation;
- developer education;
- behavioural verification.

---

# References

- Eric Evans — *Domain-Driven Design*
- Vaughn Vernon — *Implementing Domain-Driven Design*
- Martin Fowler — *Event Sourcing*
- Martin Fowler — *Snapshot*
- Michael Feathers — *Working Effectively with Legacy Code*
- ADR-0004 — SQLite as the Default Persistence Engine
- ADR-0005 — Persist Asynchronous Work
- ADR-0006 — Payment State Machine
- ADR-0007 — Event Log & Audit Trail
- ADR-0008 — Observability & Diagnostics
- ADR-0010 — Virtual Clock
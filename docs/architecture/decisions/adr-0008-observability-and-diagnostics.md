# ADR-0008: Observability & Diagnostics

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox Maintainers
- **Tags:** Observability, Diagnostics, Inspection

---

# Executive Summary

Payment Sandbox is designed to be understandable.

Every significant business operation should be observable through the platform itself without requiring developers to reproduce executions or inspect internal implementation details.

Observability is therefore considered a core architectural capability rather than an operational concern.

The platform exposes the information necessary to explain:

- what happened;
- why it happened;
- when it happened;
- what will happen next.

Diagnostics are built directly upon the business state, the Event Log and the durable execution model established by previous architectural decisions.

---

# Context

Payment systems are inherently asynchronous.

A single payment may involve:

- multiple state transitions;
- delayed webhook deliveries;
- retries;
- scheduled jobs;
- provider-specific behaviour;
- virtual time.

As the number of interacting components increases, understanding system behaviour becomes increasingly difficult.

Traditional debugging techniques often require:

- reading application logs;
- reproducing execution;
- attaching a debugger;
- inspecting database records manually.

These approaches become inefficient as scenarios grow in complexity.

Payment Sandbox is intended to help developers understand provider behaviour.

The architecture should therefore expose its internal behaviour explicitly instead of forcing users to infer it.

---

# Problem Statement

Without explicit observability, developers quickly encounter questions such as:

- Why hasn't this webhook been delivered?
- Why is this payment still pending?
- Which retry is currently scheduled?
- Which transition produced this event?
- Why did the virtual clock advance?
- Which provider behaviour is currently being simulated?

Although the required information already exists inside the system, it is often scattered across:

- current aggregate state;
- asynchronous jobs;
- event history;
- provider adapters.

Reconstructing the complete execution requires manually correlating multiple sources.

This significantly increases debugging complexity.

---

# Decision

Payment Sandbox treats observability as a first-class architectural capability.

Every significant business object should expose sufficient information to understand its current behaviour without requiring direct inspection of implementation details.

Diagnostics are derived from:

- aggregate state;
- immutable business events;
- durable jobs;
- virtual time;
- provider metadata.

The platform intentionally avoids relying on application logs as the primary diagnostic mechanism.

Instead, developers inspect structured business information produced by the domain itself.

---

# Architectural Principles

Payment Sandbox adopts the following principles.

> **A system should explain itself.**

Developers should not need to reconstruct business behaviour from infrastructure artefacts.

Business state, historical events and future work collectively describe the complete behaviour of the simulator.

---

> **Observability follows the domain.**

The platform exposes business concepts rather than implementation details.

Diagnostics therefore focus on concepts such as:

- payments;
- webhooks;
- retries;
- providers;
- scheduled work.

Infrastructure remains a supporting concern.

---

> **Every observable fact must have a business meaning.**

Information exposed through diagnostic interfaces should answer questions that developers naturally ask while understanding payment behaviour.

Metrics or logs without business value should not become part of the architectural model.

---

# Scope

This ADR applies to every capability intended to explain platform behaviour.

Examples include:

- payment inspection;
- webhook inspection;
- retry inspection;
- scheduled job inspection;
- event history;
- provider diagnostics;
- virtual time inspection.

Operational monitoring, infrastructure metrics and deployment health checks are intentionally outside the scope of this decision.

Those concerns belong to the runtime environment rather than to the architectural model of Payment Sandbox.

# Why Observability Is a Domain Concern

Many systems treat observability as an infrastructure feature.

Metrics, logs and traces are collected after the application has already executed.

While this approach is valuable for operating production systems, it often answers technical questions rather than business ones.

For example:

- Was the HTTP request successful?
- How long did the SQL query take?
- Which worker processed the job?

These questions are useful.

However, they rarely explain payment behaviour.

Payment Sandbox instead prioritises business observability.

---

# Business Questions Come First

Developers interacting with a payment provider usually ask questions such as:

- Why is this payment still pending?
- Why wasn't the webhook delivered?
- Which retry is currently scheduled?
- Why was this payment cancelled?
- Which provider behaviour is being simulated?

These are business questions.

The platform should therefore expose business answers.

Infrastructure telemetry alone cannot provide this level of explanation.

---

# The Domain Already Knows the Answers

The simulator already contains nearly all the information required to explain its behaviour.

For example:

- the payment aggregate knows the current lifecycle;
- the Event Log records previous transitions;
- durable jobs describe future work;
- the virtual clock explains simulated time;
- provider adapters expose provider-specific behaviour.

Observability therefore consists primarily of presenting existing business information rather than collecting additional technical data.

---

# Aggregate State Explains the Present

The aggregate represents the current business situation.

For example:

```text
Status: Authorized

Authorized Amount: 100.00

Captured Amount: 40.00

Remaining Capturable Amount: 60.00
```

This immediately answers questions about the current lifecycle.

No event replay is required.

---

# Event History Explains the Past

The Event Log explains how the aggregate reached its current state.

For example:

```text
PaymentCreated

↓

PaymentAuthorized

↓

PartialCaptureCompleted
```

Current state alone cannot explain previous transitions.

Historical events provide that explanation.

---

# Durable Jobs Explain the Future

Asynchronous jobs describe work that has not yet occurred.

For example:

```text
Webhook Delivery

↓

Scheduled At

↓

15:30 UTC
```

or

```text
Retry Attempt

↓

Remaining Attempts: 2

↓

Next Retry: 16:00 UTC
```

The scheduler therefore exposes future business behaviour before it occurs.

This complements current state and historical events.

---

# Virtual Time Explains Temporal Behaviour

Many provider behaviours depend on time.

Examples include:

- delayed captures;
- webhook retries;
- payment expiration;
- timeout processing.

The virtual clock makes these behaviours understandable.

Rather than wondering why nothing happened, developers can immediately observe:

```text
Current Virtual Time

↓

15:28 UTC

↓

Next Scheduled Job

↓

15:30 UTC
```

Time itself becomes observable.

---

# Provider Behaviour Must Be Observable

Different providers expose different business behaviour.

For example:

- immediate capture;
- delayed authorisation;
- asynchronous settlement;
- provider-specific retry policies.

The simulator should expose which behaviour is currently active.

Developers should never have to inspect provider implementation code to understand why a payment behaves differently.

---

# Observability Without Reading Source Code

A primary objective of Payment Sandbox is that developers should rarely need to inspect the implementation.

Instead, the platform itself should explain:

- current state;
- historical decisions;
- future work;
- provider configuration;
- simulated time.

Source code should explain implementation.

Observability should explain behaviour.

---

# Structured Information Instead of Free-Form Logs

Business diagnostics should be structured.

For example:

```text
Payment

Status: Captured

Captured Amount: 100.00

Events: 4

Pending Jobs: 1
```

rather than:

```text
INFO Payment captured

INFO Scheduling webhook

INFO Worker executed
```

Structured information is:

- machine-readable;
- easier to query;
- easier to visualise;
- easier to replay;
- more stable over time.

Free-form logs remain useful for infrastructure diagnostics but are not the primary interface for understanding business behaviour.

---

# A Complete Picture Emerges

Each architectural component contributes a different perspective.

```text
Aggregate
        │
        ▼
Present

Event Log
        │
        ▼
Past

Durable Jobs
        │
        ▼
Future

Virtual Clock
        │
        ▼
Time

Provider Metadata
        │
        ▼
Configuration
```

None of these components is sufficient on its own.

Together, they provide a complete explanation of the simulator's behaviour.

This architectural composition is what makes Payment Sandbox observable by design rather than observable through external tooling.

# Why Observability Is a Domain Concern

Many systems treat observability as an infrastructure feature.

Metrics, logs and traces are collected after the application has already executed.

While this approach is valuable for operating production systems, it often answers technical questions rather than business ones.

For example:

- Was the HTTP request successful?
- How long did the SQL query take?
- Which worker processed the job?

These questions are useful.

However, they rarely explain payment behaviour.

Payment Sandbox instead prioritises business observability.

---

# Business Questions Come First

Developers interacting with a payment provider usually ask questions such as:

- Why is this payment still pending?
- Why wasn't the webhook delivered?
- Which retry is currently scheduled?
- Why was this payment cancelled?
- Which provider behaviour is being simulated?

These are business questions.

The platform should therefore expose business answers.

Infrastructure telemetry alone cannot provide this level of explanation.

---

# The Domain Already Knows the Answers

The simulator already contains nearly all the information required to explain its behaviour.

For example:

- the payment aggregate knows the current lifecycle;
- the Event Log records previous transitions;
- durable jobs describe future work;
- the virtual clock explains simulated time;
- provider adapters expose provider-specific behaviour.

Observability therefore consists primarily of presenting existing business information rather than collecting additional technical data.

---

# Aggregate State Explains the Present

The aggregate represents the current business situation.

For example:

```text
Status: Authorized

Authorized Amount: 100.00

Captured Amount: 40.00

Remaining Capturable Amount: 60.00
```

This immediately answers questions about the current lifecycle.

No event replay is required.

---

# Event History Explains the Past

The Event Log explains how the aggregate reached its current state.

For example:

```text
PaymentCreated

↓

PaymentAuthorized

↓

PartialCaptureCompleted
```

Current state alone cannot explain previous transitions.

Historical events provide that explanation.

---

# Durable Jobs Explain the Future

Asynchronous jobs describe work that has not yet occurred.

For example:

```text
Webhook Delivery

↓

Scheduled At

↓

15:30 UTC
```

or

```text
Retry Attempt

↓

Remaining Attempts: 2

↓

Next Retry: 16:00 UTC
```

The scheduler therefore exposes future business behaviour before it occurs.

This complements current state and historical events.

---

# Virtual Time Explains Temporal Behaviour

Many provider behaviours depend on time.

Examples include:

- delayed captures;
- webhook retries;
- payment expiration;
- timeout processing.

The virtual clock makes these behaviours understandable.

Rather than wondering why nothing happened, developers can immediately observe:

```text
Current Virtual Time

↓

15:28 UTC

↓

Next Scheduled Job

↓

15:30 UTC
```

Time itself becomes observable.

---

# Provider Behaviour Must Be Observable

Different providers expose different business behaviour.

For example:

- immediate capture;
- delayed authorisation;
- asynchronous settlement;
- provider-specific retry policies.

The simulator should expose which behaviour is currently active.

Developers should never have to inspect provider implementation code to understand why a payment behaves differently.

---

# Observability Without Reading Source Code

A primary objective of Payment Sandbox is that developers should rarely need to inspect the implementation.

Instead, the platform itself should explain:

- current state;
- historical decisions;
- future work;
- provider configuration;
- simulated time.

Source code should explain implementation.

Observability should explain behaviour.

---

# Structured Information Instead of Free-Form Logs

Business diagnostics should be structured.

For example:

```text
Payment

Status: Captured

Captured Amount: 100.00

Events: 4

Pending Jobs: 1
```

rather than:

```text
INFO Payment captured

INFO Scheduling webhook

INFO Worker executed
```

Structured information is:

- machine-readable;
- easier to query;
- easier to visualise;
- easier to replay;
- more stable over time.

Free-form logs remain useful for infrastructure diagnostics but are not the primary interface for understanding business behaviour.

---

# A Complete Picture Emerges

Each architectural component contributes a different perspective.

```text
Aggregate
        │
        ▼
Present

Event Log
        │
        ▼
Past

Durable Jobs
        │
        ▼
Future

Virtual Clock
        │
        ▼
Time

Provider Metadata
        │
        ▼
Configuration
```

None of these components is sufficient on its own.

Together, they provide a complete explanation of the simulator's behaviour.

This architectural composition is what makes Payment Sandbox observable by design rather than observable through external tooling.

# Alternatives Considered

Several alternative approaches were evaluated before adopting domain-driven observability.

---

## Application Logs as the Primary Diagnostic Tool

The simplest approach would be to rely exclusively on application logs.

For example:

```text
INFO Payment captured

INFO Scheduling webhook

INFO Delivering webhook
```

This approach is common and useful for diagnosing technical failures.

However, logs present several limitations:

- they are primarily intended for humans;
- their format evolves over time;
- they are difficult to correlate reliably;
- they often mix technical and business concerns.

Most importantly, logs describe what individual components did, not necessarily why the business behaved as it did.

Payment Sandbox therefore treats logs as operational artefacts rather than the primary source of observability.

---

## Infrastructure Telemetry Alone

Another option would consist of relying primarily on infrastructure telemetry such as:

- metrics;
- traces;
- distributed tracing;
- monitoring dashboards.

These tools provide valuable operational insight.

They answer questions such as:

- How many requests were processed?
- How long did an operation take?
- Which service handled a request?

However, they do not naturally explain questions such as:

- Why wasn't the payment captured?
- Which retry is still pending?
- Why is the webhook delayed?

Business observability requires business information.

Infrastructure telemetry complements—but does not replace—it.

---

## Database Inspection

Developers could inspect database tables directly.

For example:

- payments;
- jobs;
- events.

While technically possible, this approach exposes storage implementation rather than business concepts.

Developers become coupled to database schemas and persistence details.

Payment Sandbox instead exposes business-oriented diagnostic models that remain stable even if persistence evolves.

---

## Debugger-Driven Investigation

A debugger provides complete visibility into program execution.

However, it requires reproducing the problem and understanding implementation details.

This makes it unsuitable for routine inspection of payment scenarios.

Observability should make behaviour understandable without stepping through source code.

---

# Accepted Trade-offs

Providing rich diagnostic capabilities introduces additional architectural responsibilities.

These trade-offs are intentionally accepted.

---

## Additional Development Effort

Business information must be modelled explicitly.

Diagnostic models require maintenance as the platform evolves.

This additional effort is considered worthwhile because it significantly improves developer experience.

---

## Additional Storage

Some diagnostic information is derived from persisted business data.

Historical events, scheduled jobs and provider metadata require storage.

The project accepts this overhead in exchange for explainable behaviour.

---

## Stable Diagnostic Contracts

Diagnostic information becomes part of the public developer experience.

Breaking changes should therefore be introduced carefully.

This encourages stable business terminology throughout the project.

---

## Separation of Concerns

Observability intentionally remains separate from execution.

Diagnostic models explain behaviour.

They never control it.

Maintaining this separation requires additional discipline but significantly simplifies reasoning about the system.

---

# Implementation Guidelines

The following architectural rules apply.

---

## Business Concepts Come First

Observability should expose business concepts rather than implementation details.

Examples include:

- payments;
- providers;
- retries;
- webhooks;
- scheduled jobs.

Developers should not need to understand internal implementation to interpret diagnostic information.

---

## Every Observable Value Must Have Meaning

Diagnostic information should answer a business question.

Information that cannot reasonably help explain system behaviour should not become part of the architectural model.

---

## Structured Data Over Free-Form Text

Diagnostic information should be represented through structured models whenever possible.

Examples include:

- payment summaries;
- event timelines;
- pending job lists;
- provider capabilities.

Structured information remains easier to:

- query;
- serialize;
- validate;
- visualise.

---

## Observability Must Be Read-Only

Inspection interfaces must never alter business behaviour.

Observing the platform should not:

- execute jobs;
- advance virtual time;
- trigger provider actions;
- modify payment state.

Diagnostic operations remain side-effect free.

---

## Diagnostic Information Must Be Consistent

The different diagnostic perspectives should describe the same business reality.

For example:

- the aggregate;
- the Event Log;
- pending jobs;
- provider metadata.

Contradictory information should never be presented.

Consistency takes precedence over completeness.

---

## Infrastructure Remains an Implementation Detail

The architecture intentionally does not mandate:

- OpenTelemetry;
- Prometheus;
- Grafana;
- Jaeger;
- Zipkin;
- ELK;
- Loki.

These technologies may expose diagnostic information.

They do not define it.

---

# What This Decision Is Not

This ADR does not require:

- production monitoring;
- distributed tracing;
- infrastructure dashboards;
- log aggregation platforms;
- external telemetry systems;
- alerting systems;
- service-level objectives (SLOs);
- service-level indicators (SLIs).

The objective is to make the simulator understandable.

Operational monitoring remains a deployment concern.

---

# Revisit Conditions

This ADR should be revisited if:

- additional aggregates require specialised diagnostic models;
- provider-specific behaviour cannot be explained through the existing architecture;
- diagnostic information becomes inconsistent across components;
- replay requires richer inspection capabilities;
- observability begins influencing execution behaviour.

Future evolutions should preserve the principle that diagnostics explain the platform without modifying it.

---

# Consequences

## Positive

- Payment behaviour becomes significantly easier to understand.
- Developers spend less time reproducing scenarios.
- Business explanations become independent of implementation details.
- Historical, current and future behaviour can be inspected consistently.
- Provider behaviour becomes transparent.
- Replay and diagnostics naturally complement one another.
- External observability tools can be integrated without changing the domain model.

## Negative

- Additional diagnostic models must be maintained.
- Some information is intentionally duplicated across different views.
- Stable diagnostic contracts require careful evolution.
- Rich observability increases implementation effort.
- Developers must distinguish business observability from operational telemetry.

These costs are accepted because developer understanding is a primary objective of Payment Sandbox.

---

# Decision Summary

Payment Sandbox treats observability as a core architectural capability.

Rather than relying primarily on logs or infrastructure telemetry, the platform exposes structured business information derived from:

- current aggregate state;
- immutable business events;
- durable asynchronous jobs;
- virtual time;
- provider metadata.

Together, these perspectives allow developers to understand the behaviour of the simulator without reproducing executions or inspecting implementation details.

Observability therefore becomes an inherent property of the architecture rather than an operational afterthought.

---

# References

- Google — *Site Reliability Engineering*
- Google — *The Site Reliability Workbook*
- OpenTelemetry Specification
- Cindy Sridharan — *Distributed Systems Observability*
- Charity Majors, Liz Fong-Jones & George Miranda — *Observability Engineering*
- ADR-0004 — SQLite as the Default Persistence Engine
- ADR-0005 — Persist Asynchronous Work
- ADR-0006 — Payment State Machine
- ADR-0007 — Event Log & Audit Trail
- ADR-0009 — Deterministic Scenario Replay
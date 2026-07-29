# ADR-0007: Event Log & Audit Trail

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox Maintainers
- **Tags:** Domain, Events, Audit, Observability

---

# Executive Summary

Payment Sandbox records every meaningful business transition as an immutable event.

These events collectively form the Event Log.

From this Event Log, the platform derives an Audit Trail that explains not only what happened, but also when, why and as the consequence of which business transition.

The Event Log is therefore the historical memory of the simulator.

The Audit Trail is the human-readable explanation of that history.

---

# Context

Payment providers are long-lived systems.

A single payment may evolve over several minutes, hours or even days.

During its lifetime, numerous business events may occur:

- payment created;
- payment authorised;
- payment captured;
- payment refunded;
- webhook scheduled;
- webhook delivered;
- retry scheduled;
- retry exhausted.

Each event contributes to the observable behaviour of the provider.

Simply knowing the current payment state is insufficient to understand how the payment reached that state.

For example, the following two payments may currently appear identical.

```text
Captured
```

However, one may have followed:

```text
Pending

↓

Authorized

↓

Captured
```

while another followed:

```text
Pending

↓

Authorized

↓

Capture Failed

↓

Retry

↓

Captured
```

The final state is identical.

The history is fundamentally different.

Payment Sandbox therefore preserves the complete business history rather than only the latest state.

---

# Problem Statement

Without historical information, numerous questions become impossible to answer.

Examples include:

- Why did this payment fail?
- Which operation changed the current state?
- Which webhook triggered this retry?
- How many capture attempts occurred?
- Which asynchronous jobs were scheduled?
- Which provider behaviour is being simulated?

A mutable payment record only describes the present.

It says nothing about the journey that produced it.

Developers are therefore forced to reconstruct history from:

- application logs;
- debugger sessions;
- provider documentation;
- assumptions.

This significantly complicates debugging, testing and scenario analysis.

---

# Decision

Payment Sandbox records every meaningful business transition as an immutable business event.

Events are append-only.

Once recorded, an event is never modified.

Each event describes something that has already happened.

Examples include:

- PaymentCreated
- PaymentAuthorized
- PaymentCaptured
- PaymentRefunded
- WebhookScheduled
- WebhookDelivered
- RetryScheduled

These events collectively form the Event Log.

The Audit Trail is then derived from those events by enriching them with contextual information useful for inspection, debugging and explanation.

The current payment state remains the authoritative representation of the present.

The Event Log explains how that state came into existence.

---

# Architectural Principles

Payment Sandbox adopts the following principles.

> **State describes where the system is.**

> **Events describe how the system arrived there.**

Neither replaces the other.

Current state enables efficient execution.

Historical events explain the evolution of that state.

Both representations are necessary.

---

# Event Log vs Audit Trail

Although closely related, these concepts serve different purposes.

## Event Log

The Event Log records immutable business facts.

Examples include:

```text
PaymentCreated

↓

PaymentAuthorized

↓

PaymentCaptured
```

The Event Log answers:

> "What happened?"

---

## Audit Trail

The Audit Trail explains those events within their operational context.

For example:

```text
14:02:13

PaymentAuthorized

↓

Triggered by API Request

↓

Correlation ID: ...

↓

Provider: Stripe Sandbox

↓

Result: Success
```

The Audit Trail answers questions such as:

- When did this occur?
- What caused it?
- Which component initiated it?
- Which payment was affected?
- Which provider behaviour was simulated?

The Event Log provides historical truth.

The Audit Trail provides historical understanding.

---

# Scope

This ADR applies to every business event produced by Payment Sandbox.

It includes, but is not limited to:

- payment lifecycle transitions;
- asynchronous job creation;
- webhook execution;
- retry scheduling;
- timeout processing;
- provider-specific business behaviour.

Operational logs, debug logs and infrastructure metrics are intentionally outside the scope of this ADR.

Those concerns are addressed separately by the observability model.

# Why Keep an Event Log?

The current payment state answers one question exceptionally well:

> **What is the current situation?**

It does not answer:

- How did we get here?
- Which decisions were made?
- Which operations failed?
- Which asynchronous work was triggered?
- Which provider behaviour was exercised?

The Event Log exists to answer these questions.

Rather than replacing the current state, it complements it.

---

# Current State Is a Snapshot

A payment aggregate represents the latest known business state.

For example:

```text
Status: Captured

Captured Amount: 100.00

Refunded Amount: 0.00
```

This information is sufficient to continue processing the payment.

However, it contains no historical context.

The aggregate intentionally forgets previous transitions once they have been applied.

This makes current state efficient to query and reason about, but unsuitable for understanding the complete lifecycle.

---

# Events Preserve History

Each business transition produces an immutable event.

For example:

```text
PaymentCreated

↓

PaymentAuthorized

↓

PaymentCaptured
```

Unlike the aggregate, events are never replaced.

Every transition remains part of the payment history.

The complete sequence therefore explains not only where the payment is today, but how it reached its current state.

---

# Business Facts, Not Technical Logs

The Event Log records business facts.

It is not intended to replace application logging.

For example, the following belongs in the Event Log:

```text
PaymentCaptured
```

Whereas the following belongs in operational logs:

```text
HTTP request failed

Connection timeout

Retrying in 5 seconds
```

Business events describe domain behaviour.

Operational logs describe technical execution.

These concerns remain intentionally separated.

---

# Immutable by Design

Events are append-only.

Once recorded, they are never modified or deleted as part of normal operation.

If business state changes, a new event is produced.

For example:

```text
PaymentAuthorized

↓

PaymentCaptured

↓

PaymentRefunded
```

rather than:

```text
PaymentCaptured

↓

(edit existing record)

↓

PaymentRefunded
```

An immutable history provides a trustworthy explanation of system behaviour.

---

# Events Represent Completed Facts

Only successful business transitions become events.

For example:

```text
Authorized

↓

Captured

↓

PaymentCaptured
```

The event represents something that has already occurred.

It never represents an intention.

Likewise, events are not commands.

Commands request work.

Events describe completed work.

This distinction keeps the domain model easy to understand.

---

# Foundation for Inspection

The Event Log enables developers to inspect payment behaviour without reproducing an execution.

Instead of attaching a debugger, one can simply review the event sequence.

For example:

```text
PaymentCreated

↓

PaymentAuthorized

↓

WebhookScheduled

↓

WebhookDelivered

↓

PaymentCaptured
```

The business timeline becomes immediately visible.

---

# Foundation for Replay

The Event Log also serves as the historical input for deterministic scenario replay.

Replay does not require reconstructing aggregate state from events.

Instead, events provide the sequence of business decisions and observable behaviour that occurred during the original execution.

The replay engine described in ADR-0009 builds upon this history to reproduce scenarios consistently.

---

# Foundation for Audit

Developers often need to answer questions such as:

- Why was this webhook sent?
- Why did the retry occur?
- Why was this payment cancelled?

Without historical events, answering these questions requires reconstructing execution from multiple technical sources.

The Event Log provides a single chronological record of business activity.

The Audit Trail enriches this record with contextual information suitable for investigation and explanation.

---

# Interaction with the Payment State Machine

The Event Log does not decide which transitions occur.

The payment aggregate remains responsible for enforcing lifecycle rules.

Conceptually:

```text
Command

↓

Payment Aggregate

↓

State Transition

↓

Business Event

↓

Event Log
```

Events therefore follow state transitions.

They never replace them.

This relationship preserves a clear separation between business decisions and historical recording.

---

# Event Log Is Not Event Sourcing

Although the Event Log resembles an event stream, Payment Sandbox does not adopt Event Sourcing as its persistence model.

Current payment state remains the authoritative source for business execution.

Events provide historical traceability, inspection and replay.

The aggregate is therefore persisted directly, while events complement—not replace—the canonical state.

This decision significantly reduces implementation complexity while preserving most of the practical benefits of event history.

# Architectural Model

Every successful business transition produces an immutable business event.

The Event Log is therefore not written independently.

It is a direct consequence of the domain model.

Conceptually, execution follows the sequence below.

```text
Command

↓

Load Aggregate

↓

Validate Transition

↓

Apply Transition

↓

Produce Business Event

↓

Persist Aggregate

↓

Append Event

↓

Commit
```

The aggregate and its corresponding event become durable together.

The platform therefore guarantees that the current state and its history cannot diverge.

---

# Events Are Persisted Atomically

Business state and business history represent the same decision.

Consequently, they must be committed within the same transaction.

Conceptually:

```text
Transaction

↓

Update Payment

↓

Append Event

↓

Persist Jobs

↓

Commit
```

If the transaction fails:

- the payment is not updated;
- no event is recorded;
- no asynchronous work is scheduled.

This guarantees complete consistency between:

- current state;
- historical events;
- future work.

This complements the persistence guarantees established in ADR-0004 and ADR-0005.

---

# Event Ordering

Events are ordered according to successful business execution.

For a single aggregate, the ordering is deterministic.

For example:

```text
PaymentCreated

↓

PaymentAuthorized

↓

PaymentCaptured

↓

PaymentRefunded
```

Each event follows the previous one.

No event may appear before the transition that produced it.

The Event Log therefore represents a coherent business timeline rather than a collection of unrelated records.

---

# Event Identity

Each event represents one completed business fact.

An event should possess its own identity.

Typical metadata includes:

- Event Identifier;
- Aggregate Identifier;
- Event Type;
- Timestamp;
- Aggregate Version;
- Correlation Identifier;
- Causation Identifier.

These identifiers allow events to be:

- inspected;
- correlated;
- replayed;
- audited.

The precise storage format is intentionally left to the implementation.

---

# Correlation and Causation

Many business operations trigger additional work.

For example:

```text
Capture Request

↓

PaymentCaptured

↓

WebhookScheduled

↓

WebhookDelivered
```

Although these events describe different facts, they belong to the same business flow.

Correlation identifiers allow multiple events to be grouped together.

Causation identifiers explain which event directly caused another.

For example:

```text
PaymentCaptured

↓

WebhookScheduled

↓

WebhookDelivered
```

can be interpreted as:

- `WebhookScheduled` was caused by `PaymentCaptured`;
- `WebhookDelivered` was caused by `WebhookScheduled`.

This relationship greatly simplifies debugging and investigation.

---

# Aggregate Versioning

Every successful transition advances the aggregate version.

Conceptually:

```text
Version 1

PaymentCreated

↓

Version 2

PaymentAuthorized

↓

Version 3

PaymentCaptured
```

The Event Log therefore provides not only chronological ordering but also the evolution of a specific aggregate.

Version numbers may additionally support optimistic concurrency when required by the persistence model.

---

# Interaction with Asynchronous Work

Many business events naturally produce future work.

For example:

```text
PaymentCaptured

↓

WebhookScheduled

↓

Webhook Job
```

The Event Log itself does not execute asynchronous work.

Instead, business events provide the historical explanation for why durable jobs exist.

The scheduler described in ADR-0005 executes jobs independently.

The Event Log merely records that the underlying business decision occurred.

---

# Building the Audit Trail

The Audit Trail is derived from the Event Log.

Additional contextual information may be associated with each event, including:

- execution timestamp;
- simulated provider;
- request identifier;
- correlation identifier;
- execution outcome;
- relevant business metadata.

Conceptually:

```text
Business Event

↓

Context Enrichment

↓

Audit Entry
```

This separation allows the Event Log to remain focused on business facts while enabling richer inspection interfaces.

---

# Event Immutability

Recorded events are never modified.

If new information becomes available, the system records another event.

For example:

```text
WebhookScheduled

↓

WebhookDeliveryFailed

↓

WebhookRetryScheduled

↓

WebhookDelivered
```

The complete sequence explains the behaviour of the simulator without altering historical records.

History therefore remains trustworthy.

---

# Querying History

Current business operations should rely on aggregate state.

Historical inspection should rely on the Event Log.

Conceptually:

```text
Business Operation

↓

Aggregate
```

whereas:

```text
Developer Investigation

↓

Event Log

↓

Audit Trail
```

This separation avoids coupling operational behaviour to historical storage.

The Event Log is optimised for explanation.

The aggregate is optimised for execution.

---

# Interaction with Other ADRs

This ADR builds directly upon previous architectural decisions.

- **ADR-0003** ensures that business events originate from the domain model rather than infrastructure.
- **ADR-0004** guarantees that events are persisted atomically with aggregate state.
- **ADR-0005** ensures that durable jobs are created alongside the events that justify them.
- **ADR-0006** defines the payment transitions that produce business events.

Subsequent ADRs extend these capabilities.

- **ADR-0008** exposes event history through diagnostics and observability.
- **ADR-0009** uses the Event Log as the historical foundation for deterministic scenario replay.
- **ADR-0010** records time-dependent transitions consistently through the virtual clock.
- **ADR-0011** allows provider-specific behaviour to generate provider-specific business events while preserving canonical semantics.

# Alternatives Considered

Several alternative approaches were evaluated before adopting an immutable Event Log.

---

## Current State Only

The simplest approach would consist of storing only the latest payment state.

For example:

```text
Status: Captured

Captured Amount: 100.00
```

This model is sufficient to continue business execution.

However, every previous transition is lost.

Questions such as:

- Why was this payment retried?
- Which webhook was delivered?
- When was the payment authorised?

become impossible to answer without external logs.

This approach was therefore rejected.

---

## Operational Logs Only

Another possibility would be relying exclusively on application logs.

For example:

```text
INFO Payment captured

INFO Scheduling webhook

INFO Delivering webhook
```

Application logs are valuable for diagnosing technical problems.

They are not designed to model business history.

Logs may:

- change format;
- disappear through retention policies;
- omit important business context;
- differ between implementations.

Business history should not depend on operational logging.

---

## Event Sourcing

The project could reconstruct aggregate state entirely from events.

```text
PaymentCreated

↓

PaymentAuthorized

↓

PaymentCaptured

↓

PaymentRefunded
```

This provides:

- complete historical reconstruction;
- temporal queries;
- replay from persisted events.

However, it also introduces substantial complexity:

- aggregate reconstruction;
- snapshots;
- event schema evolution;
- event version migrations;
- stream management.

Payment Sandbox benefits from historical events without requiring full Event Sourcing.

Current state therefore remains the primary execution model.

---

## Audit Records Without Business Events

Another possibility would be writing audit entries directly.

For example:

```text
14:03

Payment captured by API
```

Although suitable for human inspection, audit records lack the semantic precision required for architectural capabilities such as:

- deterministic replay;
- event correlation;
- provider diagnostics;
- lifecycle analysis.

The Event Log therefore becomes the primary historical representation.

Audit information is derived from it.

---

# Accepted Trade-offs

Recording every business event introduces additional storage and implementation complexity.

These costs are intentionally accepted.

---

## Additional Storage

Every meaningful transition produces another persisted record.

Historical information therefore grows over time.

For Payment Sandbox, this growth is considered acceptable.

Storage is inexpensive compared to the value of complete business traceability.

---

## Two Complementary Representations

The platform intentionally stores both:

- current aggregate state;
- immutable business history.

This duplicates certain information.

For example, both representations indicate that a payment is captured.

The duplication is deliberate.

Each representation serves a different purpose:

- aggregate state supports execution;
- event history supports explanation.

---

## Immutable History

Incorrect historical information is never corrected by modifying existing events.

Instead, subsequent events describe the new reality.

This makes historical analysis more reliable.

It also requires developers to think in terms of business evolution rather than mutable records.

---

## Additional Modelling

Meaningful events must be explicitly modelled.

Poorly defined events reduce the usefulness of the Event Log.

The project therefore accepts additional domain modelling work in exchange for a clearer business history.

---

# Implementation Guidelines

The following architectural rules apply.

---

## Every Significant Transition Produces One Event

Business events should represent meaningful domain facts.

Examples include:

- PaymentCreated
- PaymentAuthorized
- PaymentCaptured
- PaymentRefunded
- WebhookScheduled
- WebhookDelivered

Technical implementation details should not appear as business events.

---

## Events Must Be Immutable

After persistence, an event must never be modified.

Corrections are represented by additional events.

Historical integrity always takes precedence over convenience.

---

## Events Must Be Persisted Atomically

Business state, business events and durable jobs represent one business decision.

They must therefore be committed within the same transaction.

Partial persistence is prohibited.

---

## Events Must Remain Business-Oriented

Business events should describe domain behaviour.

For example:

```text
PaymentCaptured
```

rather than:

```text
SQLTransactionCommitted

DatabaseUpdated

WorkerExecuted
```

Infrastructure concerns belong elsewhere.

---

## Event Schemas Must Be Stable

Events become historical records.

Changing their meaning over time makes historical analysis unreliable.

When evolution is necessary, compatibility should be preserved through:

- versioning;
- additive fields;
- explicit migrations.

Historical meaning must remain stable.

---

## Event Names Must Express Facts

Events describe completed business facts.

Names should therefore use the past tense.

Examples:

- PaymentCreated
- PaymentAuthorized
- PaymentCaptured

Avoid names expressing:

- intentions;
- requests;
- commands.

---

## Events Should Be Self-Describing

A business event should contain sufficient information to be understood independently.

Consumers should not need to inspect unrelated records to understand what occurred.

Events should therefore include relevant identifiers and business metadata while avoiding unnecessary duplication.

---

## Audit Entries Must Be Derived

The Audit Trail must be generated from recorded business events.

Audit information may enrich those events with additional contextual metadata.

The reverse relationship is intentionally avoided.

Audit records must never become the primary source of business history.

---

# What This Decision Is Not

This ADR does not require:

- Event Sourcing;
- CQRS event stores;
- distributed event streaming;
- Kafka;
- message brokers;
- immutable operational logs;
- replay from persisted events alone.

The Event Log is a historical record.

It is not the authoritative persistence model of the application.

---

# Revisit Conditions

This ADR should be revisited if:

- aggregate reconstruction from events becomes a project requirement;
- historical storage becomes prohibitively large;
- event schemas evolve faster than compatibility can reasonably support;
- additional aggregates require a shared historical model;
- replay requirements exceed the capabilities of the current Event Log.

Any future evolution should preserve the distinction between:

- current business state;
- immutable historical facts.

---

# Consequences

## Positive

- Complete business history is preserved.
- Investigations become significantly easier.
- Replay scenarios gain a reliable historical foundation.
- Audit information becomes consistent across providers.
- Business behaviour is easier to understand.
- Historical analysis no longer depends on application logs.
- Events remain deterministic and reproducible.

## Negative

- More storage is required.
- Additional event modelling is necessary.
- Event schemas must remain stable over time.
- Developers must think in terms of immutable business history.
- Historical persistence increases implementation effort.

These costs are accepted because traceability is a core objective of Payment Sandbox.

---

# Decision Summary

Payment Sandbox records every meaningful business transition as an immutable business event.

The current aggregate state remains the authoritative representation of the present.

The Event Log preserves the complete business history.

The Audit Trail enriches that history with contextual information suitable for inspection and investigation.

Together, these complementary representations provide:

- reliable business execution;
- deterministic historical traceability;
- reproducible simulations;
- comprehensive diagnostics;
- explainable provider behaviour.

---

# References

- Eric Evans — *Domain-Driven Design*
- Vaughn Vernon — *Implementing Domain-Driven Design*
- Martin Fowler — *Event Sourcing*
- Martin Fowler — *Audit Log*
- Greg Young — *Versioning in an Event Sourced System*
- Enterprise Integration Patterns — *Message History*
- ADR-0004 — SQLite as the Default Persistence Engine
- ADR-0005 — Persist Asynchronous Work
- ADR-0006 — Payment State Machine
- ADR-0008 — Observability & Diagnostics
- ADR-0009 — Deterministic Scenario Replay
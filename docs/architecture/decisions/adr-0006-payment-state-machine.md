# ADR-0006: Payment State Machine

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox Maintainers
- **Tags:** Domain, Payments, State Machine, Consistency

---

# Executive Summary

Payment Sandbox models every payment as an explicit finite state machine.

A payment may only evolve through well-defined state transitions.

Every transition represents a business event that can be validated, audited and reproduced.

This decision provides deterministic behaviour across providers while allowing provider-specific variations to extend the model without compromising the integrity of the core domain.

---

# Context

Payments are long-lived business processes rather than isolated HTTP requests.

Although many payment APIs expose operations such as:

- Create Payment
- Capture Payment
- Refund Payment
- Cancel Payment

these operations do not directly modify arbitrary fields.

Instead, they transition a payment from one business state to another.

For example:

```text
Pending

↓

Authorized

↓

Captured
```

or

```text
Pending

↓

Failed
```

The current state of a payment determines:

- which operations are permitted;
- which operations are forbidden;
- which asynchronous jobs may be scheduled;
- which events may be emitted.

The payment lifecycle therefore represents one of the most fundamental concepts of the entire system.

Although this ADR focuses on payments, the architectural approach is intentionally generic.

The project models long-lived business processes as explicit state machines whose transitions define valid behaviour.

Payments are currently the primary aggregate following this approach.

Future aggregates, such as payouts, disputes or subscriptions, may adopt the same modelling principles while defining their own lifecycle semantics.

---

# Problem Statement

Without an explicit state machine, payment behaviour gradually becomes distributed across the codebase.

Business rules become embedded inside:

- HTTP handlers;
- application services;
- repositories;
- provider implementations;
- asynchronous workers.

The result is a system where valid transitions are difficult to understand and even harder to validate.

Consider a capture request.

Without a defined lifecycle, the application might inadvertently allow:

```text
Failed

↓

Captured
```

or

```text
Refunded

↓

Captured
```

These transitions have no business meaning.

Nevertheless, nothing prevents them unless every component independently performs the same validation.

As the number of providers grows, duplicated validation logic inevitably leads to inconsistent behaviour.

---

# Decision

Payment Sandbox represents every payment as an explicit finite state machine.

Each state defines:

- the current business meaning of the payment;
- the operations that are permitted;
- the operations that are rejected;
- the events that may be emitted;
- the asynchronous work that may be scheduled.

Transitions are explicit.

Invalid transitions are rejected before any business state is modified.

This state machine becomes part of the domain model rather than an implementation detail.

---

# Architectural Principle

Payment Sandbox adopts the following principle:

> **State defines behaviour.**

Business logic should not determine the current state.

The current state determines which business logic may execute.

Rather than asking:

> "Can this operation modify the payment?"

the application asks:

> "Does the current state allow this transition?"

This approach keeps the business model coherent while preventing invalid state combinations.

---

# Canonical Lifecycle

The canonical payment lifecycle is intentionally provider-agnostic.

```text
Pending
    │
    ├──────────────┐
    ▼              ▼
Authorized      Failed
    │
    ├──────────────┐
    ▼              ▼
Partially      Cancelled
Captured
    │
    ▼
Captured
    │
    ├──────────────┐
    ▼              ▼
Partially      Refunded
Refunded
    │
    ▼
Refunded
```

Not every provider supports every state.

Some providers may:

- skip intermediate states;
- introduce additional states;
- merge multiple transitions.

Nevertheless, every provider should remain compatible with the semantic model established by the core domain.

---

# State Ownership

A payment owns its lifecycle.

External components must never mutate payment state directly.

Instead, they request transitions through the domain model.

This guarantees that:

- business invariants remain enforced;
- invalid transitions are impossible;
- provider implementations remain consistent;
- asynchronous execution cannot bypass domain rules.

The payment aggregate therefore becomes the single authority responsible for lifecycle evolution.

---

# Scope

This ADR applies to every payment provider implemented by Payment Sandbox.

Provider-specific behaviour may extend the canonical state machine when required.

However, extensions should preserve the core semantic meaning of the lifecycle and should not weaken the guarantees established by this ADR.

Any provider requiring substantially different lifecycle semantics should document those differences through a dedicated Architectural Decision Record.

# Why an Explicit State Machine?

Representing payment state as a finite state machine provides significantly stronger guarantees than storing an arbitrary status value.

The objective is not merely to know **where** a payment currently is.

The objective is to define **how** it is allowed to evolve.

A payment is therefore viewed as a business process rather than a mutable record.

---

# State Is More Than a Status

Many applications model payment state as a simple enumeration.

For example:

```text
Payment

Status = "AUTHORIZED"
```

While sufficient for displaying information, this representation says very little about the behaviour of the system.

It does not answer questions such as:

- Can this payment be captured?
- Can it be cancelled?
- Can it be refunded?
- Can another authorization occur?
- Should a webhook now be emitted?

The status becomes passive information.

The application must then rediscover the business rules throughout the codebase.

---

# Behaviour Emerges from State

Payment Sandbox adopts the opposite approach.

Each state defines the behaviour that is currently permitted.

For example:

```text
Authorized

↓

Allowed

• Capture
• Partial Capture
• Cancel
```

Whereas:

```text
Captured

↓

Allowed

• Refund
• Partial Refund
```

The state itself becomes the source of truth.

Business rules naturally emerge from the lifecycle instead of being duplicated across multiple services.

---

# Explicit Transition Rules

Every transition is explicitly defined.

```text
Authorized

↓

Captured
```

is valid.

Whereas:

```text
Refunded

↓

Captured
```

is not.

Likewise:

```text
Failed

↓

Authorized
```

has no business meaning.

By modelling transitions explicitly, invalid state changes become impossible by construction rather than prevented through scattered validation logic.

---

# Protecting Business Invariants

One of the primary responsibilities of the state machine is protecting business invariants.

Examples include:

- a failed payment cannot later become captured;
- a fully refunded payment cannot be refunded again;
- a cancelled authorization cannot subsequently be captured;
- captured amounts cannot exceed authorised amounts;
- refunded amounts cannot exceed captured amounts.

These invariants belong to the domain.

They should not depend on:

- controllers;
- repositories;
- SQL constraints;
- provider implementations.

The payment aggregate remains solely responsible for preserving them.

---

# A Shared Semantic Model

Different payment providers expose different APIs.

For example:

Some providers distinguish between:

```text
Authorized

↓

Captured
```

Others immediately perform:

```text
Pending

↓

Captured
```

Still others allow:

- multiple partial captures;
- incremental authorisations;
- delayed settlement;
- asynchronous confirmation.

Despite these differences, the underlying business concepts remain remarkably similar.

Payment Sandbox therefore defines a canonical semantic model that individual providers adapt to rather than replace.

This approach enables a consistent developer experience while remaining flexible enough to model provider-specific behaviour.

---

# Provider Extensions

The canonical lifecycle intentionally represents the common denominator shared by most payment providers.

Providers remain free to extend the lifecycle when necessary.

For example:

```text
Authorized

↓

Pending Review

↓

Captured
```

or

```text
Authorized

↓

Requires Customer Action

↓

Authorized

↓

Captured
```

Such extensions should preserve the semantic meaning of the surrounding states.

The objective is to enrich the model rather than fragment it.

---

# State Drives Asynchronous Behaviour

The payment lifecycle directly influences asynchronous execution.

State transitions may schedule durable work such as:

- webhook deliveries;
- timeout handling;
- delayed settlement;
- retry scheduling.

For example:

```text
Authorized

↓

Schedule Webhook Job

↓

Commit

↓

Scheduler

↓

Worker
```

The state machine therefore defines not only synchronous behaviour but also future execution.

This naturally complements the durable asynchronous execution model defined in ADR-0005.

---

# State Drives Event Generation

Every successful transition represents a business event.

For example:

```text
Pending

↓

Authorized

↓

PaymentAuthorized
```

or

```text
Captured

↓

Refunded

↓

PaymentRefunded
```

Events are therefore consequences of valid transitions rather than independent actions.

This relationship significantly simplifies auditing and replay because every recorded event corresponds to a meaningful business evolution.

The event model described in ADR-0007 naturally builds upon this lifecycle.

---

# A Foundation for the Entire Domain

The payment state machine is one of the central abstractions of Payment Sandbox.

Numerous architectural components rely upon it, including:

- asynchronous execution;
- webhook generation;
- audit history;
- scenario replay;
- provider implementations;
- diagnostics.

By defining payment behaviour through explicit state transitions, the project establishes a single, coherent business model upon which the remainder of the architecture can safely evolve.

# Architectural Model

The payment state machine is the central authority governing the lifecycle of a payment.

Every business operation is expressed as a requested transition rather than a direct mutation of payment state.

The domain therefore follows a simple execution model.

```text
Command

↓

Load Payment

↓

Validate Transition

↓

Apply Transition

↓

Emit Domain Events

↓

Schedule Asynchronous Work

↓

Persist Aggregate

↓

Commit
```

Every successful operation follows this sequence.

The payment aggregate remains responsible for preserving business consistency throughout the lifecycle.

---

# Commands Request Transitions

Commands do not directly modify payment data.

Instead, they request a transition.

Examples include:

- Authorize Payment
- Capture Payment
- Refund Payment
- Cancel Payment

Conceptually, the application performs the following sequence.

```text
Capture Payment Command

↓

Current State = Authorized

↓

Transition Allowed?

↓

Yes

↓

Apply Transition
```

or

```text
Capture Payment Command

↓

Current State = Failed

↓

Transition Allowed?

↓

No

↓

Reject Command
```

The command itself contains no lifecycle rules.

Those rules belong exclusively to the payment aggregate.

---

# The Aggregate Owns the Lifecycle

The payment aggregate is the sole authority responsible for lifecycle evolution.

No other component may modify payment state directly.

Neither:

- repositories;
- application services;
- HTTP handlers;
- workers;
- provider adapters;

may bypass the aggregate.

This guarantees that every transition passes through exactly the same validation logic.

Business consistency therefore becomes an intrinsic property of the model rather than an implementation convention.

---

# State Before Side Effects

A successful transition always modifies business state before producing observable side effects.

Conceptually, execution follows this order.

```text
Validate Transition

↓

Update Payment State

↓

Create Domain Event

↓

Schedule Durable Jobs

↓

Commit

↓

Workers Execute
```

This ordering is intentional.

External systems should only observe transitions that have already become durable.

The simulator therefore avoids inconsistencies between business state and asynchronous execution.

---

# Events Follow State

Domain events are consequences of successful transitions.

For example:

```text
Authorized

↓

Captured

↓

PaymentCaptured
```

The inverse relationship is intentionally avoided.

Events do not decide which transition occurs.

They merely describe what has already happened.

This distinction ensures that the event history remains an accurate representation of the payment lifecycle.

The event model described in ADR-0007 naturally builds upon this principle.

---

# Asynchronous Work Follows State

Likewise, durable asynchronous work originates from successful transitions.

For example:

```text
Pending

↓

Authorized

↓

Webhook Job Created

↓

Commit
```

or

```text
Captured

↓

Refunded

↓

Refund Notification Scheduled
```

The state machine therefore determines not only current behaviour but also future behaviour.

This directly complements the durable execution model defined in ADR-0005.

---

# Invalid Transitions Produce No Side Effects

One of the most important guarantees of the state machine is that rejected transitions are invisible outside the aggregate.

```text
Capture Command

↓

Payment = Failed

↓

Transition Rejected

↓

No Events

↓

No Jobs

↓

No Persistence
```

Nothing changes.

No webhook is emitted.

No audit entry is produced.

No retry is scheduled.

This property significantly simplifies reasoning about system behaviour.

Every externally visible action corresponds to a valid business transition.

---

# Provider Adaptation

The payment state machine defines the canonical business model.

Individual providers adapt their APIs to this model.

For example:

```text
Stripe

PaymentIntent

↓

Authorized

↓

Captured
```

or

```text
Provider X

Immediate Capture

↓

Captured
```

The provider implementation translates external behaviour into the canonical lifecycle.

The domain model itself remains provider-independent.

This separation keeps provider-specific complexity outside the core business model.

---

# Interaction with Persistence

The state machine itself has no knowledge of persistence.

Its responsibility ends once the aggregate has produced:

- the new payment state;
- domain events;
- asynchronous jobs.

Persistence is handled by the application layer in accordance with ADR-0003 and ADR-0004.

Conceptually:

```text
Aggregate

↓

New State

↓

Events

↓

Jobs

↓

Repository

↓

SQLite
```

This separation preserves a clean distinction between business rules and infrastructure concerns.

---

# Deterministic Behaviour

Because every transition follows the same execution model, payment behaviour becomes deterministic.

Given:

- identical initial state;
- identical command;
- identical configuration;

the aggregate will always produce:

- the same transition;
- the same events;
- the same asynchronous jobs.

No behaviour depends on timing, infrastructure or implementation details.

This deterministic execution is essential for reliable testing, scenario replay and reproducible simulations.

---

# Interaction with Other ADRs

This ADR intentionally serves as the business foundation for several architectural decisions.

- **ADR-0003** defines how the state machine is isolated from infrastructure through ports and adapters.
- **ADR-0004** ensures that state transitions, events and jobs are persisted atomically.
- **ADR-0005** guarantees that asynchronous work originates from committed transitions.
- **ADR-0007** records every transition in the event log and audit trail.
- **ADR-0008** uses lifecycle information to expose meaningful diagnostics.
- **ADR-0009** replays transitions deterministically to reproduce complete payment scenarios.
- **ADR-0010** controls time-dependent transitions without modifying the lifecycle model.
- **ADR-0011** allows providers to extend the canonical lifecycle while preserving its semantics.

# Alternatives Considered

Several alternative modelling approaches were considered before adopting an explicit payment state machine.

---

## Mutable Status Field

The simplest approach would consist of storing a status value and allowing application services to update it directly.

```text
payment.status = "captured"
```

This approach requires little initial structure.

However, it provides no guarantee that the transition is valid.

Any component capable of mutating the payment could produce impossible states such as:

```text
Failed

↓

Captured
```

or:

```text
Fully Refunded

↓

Authorized
```

Validation would need to be duplicated throughout the codebase.

The status field would describe the current state without defining the behaviour associated with it.

For these reasons, an unconstrained mutable status field was rejected.

---

## Validation in Application Services

Another option would be to keep the payment model relatively passive and validate transitions inside application services.

For example:

```text
CapturePaymentService

↓

Check current status

↓

Validate captured amount

↓

Update payment
```

This approach can work for small systems.

However, it distributes lifecycle rules across multiple use cases.

Capture, refund and cancellation services would each need to understand the same payment invariants.

As provider-specific behaviour grows, these rules would become increasingly fragmented and difficult to maintain.

Payment lifecycle rules belong to the payment aggregate rather than to orchestration services.

---

## Provider-Owned State Machines

Each provider could define an entirely independent state machine.

This would maximize provider flexibility.

However, it would also fragment the domain model.

The same business concept could acquire different meanings depending on the provider implementation.

For example, one provider might represent a completed payment as `Captured`, while another might expose it as `Paid`, `Settled` or `Completed`.

Provider-specific terminology is valuable at integration boundaries.

It should not replace the canonical semantics of the core domain.

Payment Sandbox therefore adopts a shared semantic lifecycle with controlled provider extensions.

---

## Workflow Engine

A workflow engine could represent payment lifecycles through durable workflows.

Such engines provide advanced capabilities including:

- long-running orchestration;
- retries;
- compensation;
- workflow history;
- distributed execution.

These capabilities significantly exceed the current needs of Payment Sandbox.

Introducing a workflow engine would also blur the distinction between:

- domain state transitions;
- asynchronous job execution;
- infrastructure orchestration.

The payment lifecycle remains a domain concern and should stay independent from the runtime execution mechanism.

---

## Event Sourcing as the Primary State Model

The payment aggregate could be reconstructed exclusively from a stream of events.

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

Event sourcing provides a natural historical model and powerful replay capabilities.

However, it also introduces significant complexity:

- aggregate reconstruction;
- event schema evolution;
- snapshot management;
- event versioning;
- migration of historical streams.

Payment Sandbox requires an event log and deterministic replay, but it does not currently require full event sourcing.

The canonical payment state is therefore persisted directly.

Events describe transitions but are not the sole representation of current state.

---

# Accepted Trade-offs

The explicit state machine introduces additional modelling work.

These costs are intentionally accepted.

---

## More Domain Code

Every supported operation requires an explicit transition rule.

This results in more code than directly updating a status field.

The additional code is considered valuable because it makes business rules visible and testable.

---

## Canonical Semantics Require Mapping

Provider APIs may use terminology that does not exactly match the canonical lifecycle.

Adapters must translate provider-specific states into domain semantics.

This mapping introduces effort but prevents provider-specific vocabulary from leaking into the core model.

---

## Complex Monetary State

Partial captures and partial refunds cannot always be represented accurately through state names alone.

The aggregate must also track monetary values such as:

- authorised amount;
- captured amount;
- refunded amount;
- remaining capturable amount;
- remaining refundable amount.

The lifecycle and monetary invariants must therefore be modelled together.

---

## Controlled Extensibility

Provider extensions cannot mutate the canonical lifecycle arbitrarily.

They must preserve core invariants and semantic compatibility.

This reduces unrestricted flexibility in exchange for consistency across the platform.

---

# State and Monetary Progress

Payment state must not duplicate information that is better represented by amounts.

In particular, partial capture and refund progression should primarily be determined from monetary values.

For example:

```text
Authorized Amount = 100.00

Captured Amount = 40.00

Remaining Capturable Amount = 60.00
```

The aggregate may expose a derived state such as `PartiallyCaptured`, but the captured amount remains the authoritative source of truth.

Likewise:

```text
Captured Amount = 100.00

Refunded Amount = 25.00

Remaining Refundable Amount = 75.00
```

may be represented as `PartiallyRefunded`.

The state machine must therefore combine:

- lifecycle state;
- monetary progression;
- business invariants.

A status value must never become a substitute for accurate monetary modelling.

---

# Terminal States

Some states are terminal.

Once reached, they prohibit all ordinary lifecycle transitions.

Examples include:

- `Failed`;
- `Cancelled`;
- `FullyRefunded`.

A terminal state may still allow non-mutating operations such as:

- retrieval;
- inspection;
- audit access;
- diagnostic export.

Terminal does not mean deleted or inaccessible.

It means the payment lifecycle cannot progress further through normal business operations.

---

# Transition Failure Model

Invalid transition requests must produce explicit domain failures.

A transition failure should identify:

- the requested operation;
- the current state;
- the violated invariant;
- relevant monetary values when applicable.

For example:

```text
Operation: Capture

Current State: Cancelled

Reason: Cancelled payments cannot be captured
```

or:

```text
Operation: Refund

Captured Amount: 100.00

Already Refunded: 80.00

Requested Refund: 30.00

Reason: Refund amount exceeds the remaining refundable amount
```

Failures must remain meaningful at the domain level.

They should not expose persistence or transport concerns.

Application and provider adapters may translate these failures into provider-specific API responses.

---

# Implementation Guidelines

The following rules are part of the architectural decision.

---

## Transitions Must Be Named Operations

State mutation must occur through explicit domain methods.

Examples include:

```text
payment.Authorize(...)

payment.Capture(...)

payment.Refund(...)

payment.Cancel(...)
```

Generic setters such as the following are prohibited:

```text
payment.SetStatus(...)

payment.SetCapturedAmount(...)
```

Named operations preserve business intent.

---

## The Aggregate Must Enforce Invariants

The aggregate must validate:

- current lifecycle state;
- requested monetary amount;
- provider capabilities;
- previous captures and refunds;
- terminal-state restrictions.

Application services may perform preliminary validation for usability.

They must not become the source of truth for lifecycle rules.

---

## Persistence Must Not Reapply Business Logic

Repositories reconstruct and persist aggregate state.

They must not decide whether transitions are valid.

SQL queries and database constraints may reinforce structural integrity, but the domain remains responsible for business validity.

---

## Provider Adapters Must Translate, Not Bypass

Provider adapters may:

- translate provider commands;
- map provider terminology;
- expose provider-specific response models;
- request supported domain transitions.

They must not directly mutate aggregate state or weaken canonical invariants.

---

## Events Must Follow Successful Transitions

Domain events may only be emitted after a transition has been validated and applied.

Rejected operations emit no success event.

For example:

```text
Capture Requested

↓

Transition Validated

↓

State Updated

↓

PaymentCaptured Emitted
```

An invalid capture must not emit `PaymentCaptureFailed` as though a lifecycle transition had occurred.

Operational or request failures may be recorded separately when required by the audit model.

---

## Jobs Must Originate from Valid Transitions

Durable asynchronous work must only be created as a consequence of a successful transition or another explicitly modelled business decision.

Invalid commands must not schedule:

- webhooks;
- retries;
- timeouts;
- delayed callbacks.

This preserves consistency between current state and future work.

---

## Transition Tests Are Mandatory

Every transition must be covered by tests for:

- valid source states;
- invalid source states;
- boundary monetary amounts;
- duplicate operations;
- terminal states;
- provider-specific capabilities.

State-machine tests should be expressed as business scenarios rather than implementation tests.

For example:

```text
Given an authorised payment

When a partial capture is requested

Then the captured amount is increased

And the payment remains capturable

And a capture event is produced
```

---

# What This Decision Is Not

This ADR does not require:

- a dedicated state-machine framework;
- one concrete Go type per state;
- event sourcing;
- a generic workflow engine;
- identical provider-facing statuses;
- encoding every monetary variation as a separate state.

The decision concerns explicit lifecycle semantics and controlled transitions.

The implementation may remain simple as long as these guarantees are preserved.

---

# Revisit Conditions

This ADR should be revisited if:

- provider-specific lifecycles cannot be represented without extensive exceptions;
- the canonical model becomes too broad to remain meaningful;
- additional aggregates require a shared generic lifecycle abstraction;
- full event sourcing becomes a project requirement;
- payment orchestration evolves into long-running workflows with compensation;
- state and monetary progression become impossible to model coherently within the aggregate.

Revisiting this decision does not imply abandoning explicit state transitions.

It may instead lead to:

- lifecycle policies;
- provider-specific transition strategies;
- additional aggregates;
- richer state representations.

---

# Consequences

## Positive

- Business rules remain centralized.
- Invalid payment states are rejected consistently.
- Provider implementations share common semantics.
- Domain behaviour becomes deterministic.
- Transitions are easy to test.
- Events and asynchronous jobs originate from valid business changes.
- Audit and replay gain a stable semantic foundation.
- Monetary invariants remain protected by the aggregate.

## Negative

- The domain model contains more explicit code.
- Provider adapters require lifecycle mapping.
- Partial operations require both state and monetary tracking.
- Extensions must respect canonical semantics.
- Lifecycle changes require careful compatibility analysis.

These costs are accepted because payment lifecycle correctness is a core requirement of the simulator.

---

# Decision Summary

Payment Sandbox models payments as explicit business lifecycles.

Operations request transitions.

The payment aggregate validates those transitions, applies monetary and lifecycle changes, emits corresponding events and produces any required future work.

No external component may mutate payment state directly.

The canonical state machine provides shared semantics across providers while allowing controlled extensions where required.

This decision establishes the domain foundation required for:

- durable asynchronous execution;
- event logging;
- observability;
- deterministic replay;
- virtual time;
- provider integration.

---

# References

- Eric Evans — *Domain-Driven Design*
- Vaughn Vernon — *Implementing Domain-Driven Design*
- Martin Fowler — *State Machine*
- Martin Fowler — *Anemic Domain Model*
- Enterprise Integration Patterns — Process Manager
- ADR-0002 — Domain Model & Bounded Contexts
- ADR-0003 — Targeted Hexagonal Architecture
- ADR-0004 — SQLite as the Default Persistence Engine
- ADR-0005 — Persist Asynchronous Work
- ADR-0007 — Event Log & Audit Trail
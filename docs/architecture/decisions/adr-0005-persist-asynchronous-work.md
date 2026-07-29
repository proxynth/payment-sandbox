# ADR-0005: Persist Asynchronous Work

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox maintainers
- **Tags:** Architecture, Reliability, Scheduling, Webhooks
- **Supersedes:** None
- **Superseded by:** None

---

# Executive Summary

Payment Sandbox persists every asynchronous operation before it becomes executable.

Webhook deliveries, delayed retries, scheduled tasks and future asynchronous capabilities are represented as durable records stored in the persistence layer rather than transient in-memory work.

This decision guarantees deterministic behaviour, crash recovery, reproducible scenarios and transactional consistency.

The scheduler therefore operates on persistent jobs instead of ephemeral queues.

---

# Context

Real payment providers are fundamentally asynchronous systems.

Although an API request may complete within milliseconds, much of the work initiated by that request continues after the HTTP response has been returned.

Examples include:

- webhook deliveries;
- delayed retries;
- exponential backoff;
- fraud analysis;
- reconciliation;
- notification dispatch;
- asynchronous settlement;
- timeout handling.

These activities are not secondary concerns.

They are part of the observable behaviour of the payment provider.

A payment simulator that only reproduces synchronous HTTP interactions cannot accurately model production systems.

Payment Sandbox therefore treats asynchronous execution as a first-class architectural concern.

---

# Problem Statement

The application needs to execute work after the originating HTTP request has already completed.

A naïve implementation might simply enqueue tasks in memory.

For example:

```text
HTTP Request

↓

Create Payment

↓

Push webhook into memory queue

↓

Return 201 Created
```

This approach appears simple.

Unfortunately, it introduces several correctness problems.

If the process terminates immediately after returning the HTTP response, every queued task disappears.

The application has acknowledged work that it is now incapable of completing.

The simulator can no longer reproduce realistic provider behaviour.

This violates one of the primary goals of Payment Sandbox: deterministic execution.

---

# Decision

Every asynchronous operation must be persisted before it becomes eligible for execution.

No asynchronous task exists exclusively in memory.

Instead, asynchronous work is represented explicitly inside the persistence layer.

Typical examples include:

- webhook deliveries;
- retry attempts;
- delayed executions;
- timeout events;
- scheduled maintenance tasks;
- future asynchronous workflows.

The scheduler does not consume transient messages.

It continuously discovers executable jobs by querying durable storage.

The persistence layer therefore becomes the canonical representation of future work.

---

# Why Persistent Jobs?

The architectural question addressed by this ADR is not:

> "How should we execute webhooks?"

Instead, it is:

> "How should the system represent work that has not happened yet?"

Treating future work as durable state provides several important guarantees.

Future execution becomes:

- inspectable;
- recoverable;
- deterministic;
- replayable;
- testable.

More importantly, future work becomes part of the observable state of the simulator.

This significantly improves both correctness and debuggability.

---

# Architectural Principle

Payment Sandbox adopts the following principle:

> **Future work is state.**

This statement may appear counterintuitive.

Many applications consider scheduled execution to be an implementation detail handled by background workers or message brokers.

Payment Sandbox intentionally rejects that view.

Consider a payment whose webhook is scheduled for delivery thirty seconds after creation.

Immediately after the payment is committed, two statements are simultaneously true:

- the webhook has not yet been delivered;
- the provider has already committed to delivering it.

That commitment is part of the provider's state.

Consequently, it must also become part of the simulator's state.

The simulator therefore persists not only what **has happened**, but also what **is expected to happen**.

---

# From Events to Jobs

The creation of asynchronous work follows a simple lifecycle.

```text
HTTP Request

↓

Business Operation

↓

Domain Event

↓

Persistent Job

↓

Commit

↓

Scheduler

↓

Worker

↓

Execution

↓

Persist Result
```

Several important observations can be made.

The worker never invents work.

The scheduler never invents work.

The queue never invents work.

Every executable task originates from durable state created during the original business transaction.

This property guarantees that asynchronous execution remains fully reproducible.

---

# Scope

This ADR applies to every asynchronous capability of Payment Sandbox.

It is intentionally broader than webhook delivery.

Examples include:

- webhook dispatch;
- retry scheduling;
- delayed callbacks;
- timeout processing;
- background maintenance;
- future reconciliation workflows;
- future provider-specific asynchronous behaviours.

Every new asynchronous feature introduced into the project should follow the architectural principles established by this ADR unless explicitly documented otherwise in a future Architectural Decision Record.

# ADR-0005: Persist Asynchronous Work

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox maintainers
- **Tags:** Architecture, Reliability, Scheduling, Webhooks
- **Supersedes:** None
- **Superseded by:** None

---

# Executive Summary

Payment Sandbox persists every asynchronous operation before it becomes executable.

Webhook deliveries, delayed retries, scheduled tasks and future asynchronous capabilities are represented as durable records stored in the persistence layer rather than transient in-memory work.

This decision guarantees deterministic behaviour, crash recovery, reproducible scenarios and transactional consistency.

The scheduler therefore operates on persistent jobs instead of ephemeral queues.

---

# Context

Real payment providers are fundamentally asynchronous systems.

Although an API request may complete within milliseconds, much of the work initiated by that request continues after the HTTP response has been returned.

Examples include:

- webhook deliveries;
- delayed retries;
- exponential backoff;
- fraud analysis;
- reconciliation;
- notification dispatch;
- asynchronous settlement;
- timeout handling.

These activities are not secondary concerns.

They are part of the observable behaviour of the payment provider.

A payment simulator that only reproduces synchronous HTTP interactions cannot accurately model production systems.

Payment Sandbox therefore treats asynchronous execution as a first-class architectural concern.

---

# Problem Statement

The application needs to execute work after the originating HTTP request has already completed.

A naïve implementation might simply enqueue tasks in memory.

For example:

```text
HTTP Request

↓

Create Payment

↓

Push webhook into memory queue

↓

Return 201 Created
```

This approach appears simple.

Unfortunately, it introduces several correctness problems.

If the process terminates immediately after returning the HTTP response, every queued task disappears.

The application has acknowledged work that it is now incapable of completing.

The simulator can no longer reproduce realistic provider behaviour.

This violates one of the primary goals of Payment Sandbox: deterministic execution.

---

# Decision

Every asynchronous operation must be persisted before it becomes eligible for execution.

No asynchronous task exists exclusively in memory.

Instead, asynchronous work is represented explicitly inside the persistence layer.

Typical examples include:

- webhook deliveries;
- retry attempts;
- delayed executions;
- timeout events;
- scheduled maintenance tasks;
- future asynchronous workflows.

The scheduler does not consume transient messages.

It continuously discovers executable jobs by querying durable storage.

The persistence layer therefore becomes the canonical representation of future work.

---

# Why Persistent Jobs?

The architectural question addressed by this ADR is not:

> "How should we execute webhooks?"

Instead, it is:

> "How should the system represent work that has not happened yet?"

Treating future work as durable state provides several important guarantees.

Future execution becomes:

- inspectable;
- recoverable;
- deterministic;
- replayable;
- testable.

More importantly, future work becomes part of the observable state of the simulator.

This significantly improves both correctness and debuggability.

---

# Architectural Principle

Payment Sandbox adopts the following principle:

> **Future work is state.**

This statement may appear counterintuitive.

Many applications consider scheduled execution to be an implementation detail handled by background workers or message brokers.

Payment Sandbox intentionally rejects that view.

Consider a payment whose webhook is scheduled for delivery thirty seconds after creation.

Immediately after the payment is committed, two statements are simultaneously true:

- the webhook has not yet been delivered;
- the provider has already committed to delivering it.

That commitment is part of the provider's state.

Consequently, it must also become part of the simulator's state.

The simulator therefore persists not only what **has happened**, but also what **is expected to happen**.

---

# From Events to Jobs

The creation of asynchronous work follows a simple lifecycle.

```text
HTTP Request

↓

Business Operation

↓

Domain Event

↓

Persistent Job

↓

Commit

↓

Scheduler

↓

Worker

↓

Execution

↓

Persist Result
```

Several important observations can be made.

The worker never invents work.

The scheduler never invents work.

The queue never invents work.

Every executable task originates from durable state created during the original business transaction.

This property guarantees that asynchronous execution remains fully reproducible.

---

# Scope

This ADR applies to every asynchronous capability of Payment Sandbox.

It is intentionally broader than webhook delivery.

Examples include:

- webhook dispatch;
- retry scheduling;
- delayed callbacks;
- timeout processing;
- background maintenance;
- future reconciliation workflows;
- future provider-specific asynchronous behaviours.

Every new asynchronous feature introduced into the project should follow the architectural principles established by this ADR unless explicitly documented otherwise in a future Architectural Decision Record.

# Architectural Model

Persisting asynchronous work fundamentally changes the execution model of the application.

Instead of relying on transient queues or background goroutines, Payment Sandbox executes work through a deterministic pipeline driven entirely by durable state.

The lifecycle of an asynchronous operation is therefore explicit.

```text
                    Business Transaction

                           │
                           ▼
                ┌─────────────────────┐
                │ Create Payment      │
                │ Persist Events      │
                │ Persist Job         │
                └──────────┬──────────┘
                           │
                        COMMIT
                           │
                           ▼
                  Durable Database State
                           │
                Scheduler discovers work
                           │
                           ▼
                    Worker acquires job
                           │
                           ▼
                  Execute asynchronous work
                           │
                           ▼
                   Persist execution result
```

Every transition is durable.

Nothing exists exclusively in memory.

---

# Transactional Job Creation

A persistent job is not created by the scheduler.

It is created by the business transaction that generates the need for future execution.

For example, creating a payment may produce the following transaction:

```text
BEGIN

↓

Insert Payment

↓

Insert Domain Event

↓

Insert Webhook Job

↓

Insert Idempotency Record

↓

COMMIT
```

The scheduler is not involved.

If the transaction rolls back, **the job never existed**.

If the transaction commits, **the job is guaranteed to exist**.

This property is essential because it eliminates synchronization problems between business state and asynchronous execution.

---

# Transactional Outbox Without a Message Broker

This architecture intentionally adopts the core principle of the **Transactional Outbox** pattern.

Instead of publishing messages to an external broker during the transaction, the application persists future work inside the same database transaction.

```text
┌─────────────────────────────┐
│ Payment                     │
│ Domain Events               │
│ Webhook Job                 │
│ Retry Metadata              │
└──────────────┬──────────────┘
               │
            COMMIT
               │
               ▼
Scheduler discovers committed work
```

The important observation is that **commit becomes the publication mechanism**.

There is no second phase.

There is no "publish after commit" race condition.

The scheduler only observes committed jobs.

---

# Scheduler Responsibilities

The scheduler has deliberately limited responsibilities.

It is **not** responsible for executing business logic.

It does **not** understand payments.

It does **not** understand webhooks.

Its sole responsibility is identifying executable work.

Conceptually, the scheduler performs the following loop:

```text
Loop

↓

Find executable jobs

↓

Attempt acquisition

↓

Dispatch to worker

↓

Repeat
```

This simplicity is intentional.

A scheduler that understands business concepts quickly becomes difficult to evolve.

Instead, the scheduler only understands:

- execution time;
- execution state;
- leasing;
- retries;
- completion.

Everything else belongs elsewhere.

---

# Worker Responsibilities

Workers execute jobs.

Nothing more.

Workers do not decide **whether** a job should exist.

They only decide **how** to execute work that already exists.

```text
Acquire Job

↓

Load Context

↓

Execute

↓

Persist Outcome

↓

Release Job
```

Every worker should therefore satisfy the following properties:

- stateless;
- restartable;
- deterministic;
- replaceable.

A worker should be able to disappear at any moment without compromising system consistency.

---

# Job Lifecycle

Every asynchronous operation follows the same lifecycle.

```text
Pending

↓

Leased

↓

Running

↓

Completed
```

or

```text
Pending

↓

Leased

↓

Running

↓

Failed

↓

Retry Scheduled

↓

Pending
```

The lifecycle itself is durable.

It is never inferred from application memory.

This makes debugging significantly easier because the current execution state is always observable.

---

# Leasing

Multiple workers may exist simultaneously.

Without coordination, two workers could execute the same job.

To prevent this, workers temporarily lease work before execution.

```text
Pending Job

↓

Lease acquired

↓

Worker owns execution

↓

Lease released

↓

Completed
```

A lease is intentionally temporary.

If a worker crashes before completing execution, the lease eventually expires.

The scheduler may then safely reassign the job to another worker.

This behaviour naturally supports crash recovery without introducing distributed locking systems.

---

# Retry Model

Failures are expected.

Consequently, retries are first-class concepts rather than exceptional situations.

Each failed execution updates the persistent job state.

```text
Attempt #1

↓

Failure

↓

Attempts = 1

↓

Next execution = +30s

↓

Pending
```

Subsequent executions continue from persisted state.

Nothing depends on worker memory.

---

# Idempotent Execution

Workers should assume that every job may execute more than once.

Reasons include:

- process crashes;
- lease expiration;
- network failures;
- timeout ambiguity;
- manual replay.

Therefore, job handlers should be idempotent whenever possible.

This aligns naturally with payment systems, where duplicate webhook deliveries are common.

Rather than hiding duplicate execution, Payment Sandbox intentionally models it.

---

# Crash Recovery

One of the strongest arguments for persistent jobs is crash recovery.

Consider the following sequence.

```text
Webhook Job

Status = Running

↓

Worker crashes

↓

Lease expires

↓

Scheduler discovers job

↓

Worker #2 resumes execution
```

No manual intervention is required.

The system eventually converges toward completion.

This property significantly improves resilience while remaining conceptually simple.

---

# Observable Execution Engine

Because every execution transition is persisted, the scheduler becomes completely observable.

At any point in time, developers can inspect:

- pending jobs;
- leased jobs;
- completed jobs;
- retry history;
- execution timestamps;
- worker ownership;
- failure reasons.

This visibility transforms asynchronous execution from an opaque runtime behaviour into inspectable application state.

Debugging therefore becomes substantially easier.

---

# Interaction with Other ADRs

This decision intentionally complements several other architectural decisions.

- **ADR-0003** defines the persistence boundary through repositories and ports.
- **ADR-0004** guarantees that durable work is stored transactionally in SQLite.
- **ADR-0006** distinguishes durable business truth from what clients observe.
- **ADR-0007** relies on persisted jobs to replay scenarios deterministically.
- **ADR-0008** allows the scheduler to advance virtual time without modifying its execution model.

Together, these ADRs define a coherent execution engine in which asynchronous work is durable, deterministic and fully observable.

# Alternatives Considered

Several alternative execution models were evaluated before adopting durable asynchronous work.

Each has strengths.

None aligns as closely with the goals of Payment Sandbox.

---

## Background Goroutines

The simplest approach consists of launching goroutines directly after the business transaction completes.

```text
HTTP Request

↓

Create Payment

↓

go DeliverWebhook()

↓

Return Response
```

This design is attractive because it requires almost no infrastructure.

However, it immediately introduces several problems.

If the process terminates:

- running goroutines disappear;
- pending work is lost;
- retries cannot resume;
- execution history is unavailable.

The architecture becomes dependent on process lifetime.

Payment Sandbox intentionally rejects this model.

---

## In-Memory Queues

Maintaining an internal queue slightly improves execution control.

```text
Payment

↓

Memory Queue

↓

Worker
```

Nevertheless, the queue remains transient.

It cannot survive:

- crashes;
- upgrades;
- process restarts.

It also prevents developers from inspecting pending work.

The queue becomes hidden runtime state.

---

## External Message Brokers

Message brokers such as:

- RabbitMQ;
- Kafka;
- NATS;
- Redis Streams;

provide excellent solutions for distributed systems.

Payment Sandbox deliberately targets a different problem space.

Introducing a broker would require:

- additional infrastructure;
- network communication;
- operational monitoring;
- message acknowledgements;
- broker lifecycle management.

The complexity would significantly exceed the needs of a local-first simulator.

---

## Workflow Engines

Platforms such as Temporal or Cadence provide durable workflow execution.

These systems solve substantially broader problems including:

- long-running workflows;
- distributed orchestration;
- compensation;
- workflow versioning;
- activity replay.

Payment Sandbox intentionally implements a much smaller execution model focused exclusively on deterministic simulation.

The concepts remain inspirational.

The operational footprint does not.

---

# Accepted Trade-offs

This architecture intentionally accepts several trade-offs.

---

## Polling Instead of Push

The scheduler periodically discovers executable jobs.

This introduces a small amount of polling.

The additional database queries are considered acceptable because:

- workloads remain relatively small;
- SQLite performs indexed lookups efficiently;
- simplicity outweighs the benefits of introducing a broker.

Future persistence adapters may optimize discovery differently while preserving identical behaviour.

---

## At-Least-Once Execution

Workers are expected to occasionally execute jobs more than once.

This is intentional.

Payment providers frequently deliver:

- duplicate webhooks;
- repeated callbacks;
- retried notifications.

Payment Sandbox models these behaviours faithfully.

Idempotency therefore becomes an explicit architectural concern rather than an implementation accident.

---

## Additional Persistent State

Persisting jobs increases the number of database records.

This additional storage is considered beneficial.

Historical execution data improves:

- debugging;
- reproducibility;
- observability;
- replay.

Storage is intentionally exchanged for visibility.

---

# Implementation Guidelines

Every persistence implementation must preserve the following behavioural guarantees.

---

## Job Creation

A job must always be created inside the same transaction as the business state that requires it.

Creating jobs after commit is prohibited.

Doing so would reintroduce race conditions between business state and asynchronous execution.

---

## Scheduler Independence

The scheduler must remain completely generic.

It must never contain provider-specific logic.

It only understands:

- execution time;
- execution state;
- leases;
- retries.

Business decisions remain inside application services.

---

## Worker Independence

Workers must remain stateless.

A worker should be able to terminate unexpectedly at any point without compromising system consistency.

Every execution must derive entirely from durable state.

---

## Observable State

Execution state must remain queryable.

Developers should always be able to inspect:

- pending work;
- completed work;
- retries;
- failures;
- execution timestamps.

Hidden execution state should be avoided.

---

# What This Decision Is Not

This ADR does not claim that:

- every application should persist asynchronous work;
- every system should avoid message brokers;
- polling is universally preferable;
- workflow engines are unnecessary;
- asynchronous execution should always occur inside a database.

These statements would be incorrect.

This ADR only describes the architecture that best satisfies the goals of Payment Sandbox.

The decision is contextual rather than universal.

---

# Revisit Conditions

This ADR should be reconsidered if:

- asynchronous workloads become distributed across multiple nodes;
- polling becomes a measurable bottleneck;
- hosted deployments become the primary distribution model;
- long-running workflows exceed the capabilities of the current scheduler;
- persistent jobs introduce unacceptable operational costs.

Future evolution may include alternative execution engines provided they preserve the same observable behaviour.

---

# Consequences

## Positive

- Durable asynchronous execution.
- Crash recovery.
- Deterministic replay.
- Inspectable execution state.
- Simplified debugging.
- Generic scheduler.
- Stateless workers.
- Transactional consistency.

---

## Negative

- Additional persistence overhead.
- Scheduler polling.
- More database records.
- At-least-once execution semantics.
- Slightly higher implementation complexity.

These trade-offs are accepted because they reinforce the architectural goals of the project.

---

# Decision Summary

Payment Sandbox models asynchronous execution as durable application state rather than transient runtime activity.

Future work is committed together with the business state that created it.

The scheduler merely discovers committed work.

Workers execute that work.

Execution results become durable state again.

This architecture produces an execution engine that is:

- deterministic;
- observable;
- crash-resistant;
- reproducible;
- operationally simple.

These properties are fundamental to building a payment simulator whose behaviour faithfully reflects that of real payment providers.

---

# References

- Martin Fowler — *Patterns of Enterprise Application Architecture*
- Chris Richardson — *Microservices Patterns* (Transactional Outbox)
- Gregor Hohpe & Bobby Woolf — *Enterprise Integration Patterns*
- Pat Helland — *Life Beyond Distributed Transactions*
- Temporal Documentation — Durable Execution Concepts
- SQLite Documentation
- ADR-0003 — Targeted Hexagonal Architecture
- ADR-0004 — Use SQLite as the Default Persistence Engine
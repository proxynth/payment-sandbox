# ADR-0004: Use SQLite as the Default Persistence Engine

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox maintainers
- **Tags:** Persistence, Architecture, Storage
- **Supersedes:** None
- **Superseded by:** None

> ## Executive Summary
>
> Payment Sandbox adopts SQLite as its canonical persistence engine.
>
> The project intentionally optimizes for deterministic execution,
> developer experience and operational simplicity rather than horizontal scalability.
>
> SQLite provides ACID transactions, durable asynchronous work and
> zero-infrastructure deployment while remaining fully compatible with
> the architectural goals of the project.

---

# Context

Payment Sandbox is designed as a **local-first payment systems simulator**.

Its primary purpose is **not** to process real financial transactions nor to compete with production-grade payment service providers. Instead, it allows developers to build, test and debug payment integrations under realistic conditions while remaining entirely self-contained.

Unlike production payment infrastructures, which are optimized for horizontal scalability and high availability, Payment Sandbox optimizes for different qualities:

- deterministic execution;
- reproducible scenarios;
- minimal installation friction;
- fast feedback loops;
- inspectable state;
- predictable behaviour;
- operational simplicity.

The persistence layer therefore serves a different purpose than it would in a production payment platform.

Its role is not merely to store data.

It is responsible for preserving the complete execution state of a simulation, including asynchronous work, failures, retries, idempotency records and domain history.

Choosing the persistence engine therefore has architectural consequences that extend far beyond data storage.

---

# Problem Statement

Several persistence strategies were considered.

Examples include:

- SQLite
- PostgreSQL
- MySQL
- embedded key-value databases (BoltDB/Bbolt, BadgerDB)
- purely in-memory persistence
- pluggable persistence selected during startup

Each option optimizes different characteristics.

For example:

| Characteristic | SQLite | PostgreSQL |
|---------------|---------|------------|
| Zero installation | ✅ | ❌ |
| ACID transactions | ✅ | ✅ |
| Horizontal scalability | ❌ | ✅ |
| Operational complexity | Very low | Medium |
| Local developer experience | Excellent | Good |
| Single binary distribution | ✅ | ❌ |

The objective of this ADR is **not** to determine which database is objectively "better".

Instead, the objective is to identify which persistence engine best supports the philosophy and long-term goals of Payment Sandbox.

---

# Decision

SQLite is adopted as the **reference persistence engine** of Payment Sandbox.

Every official distribution of the project is expected to work out of the box using SQLite.

SQLite is considered the canonical implementation against which the behaviour of any future persistence adapter must remain compatible.

All durable state is stored inside SQLite, including:

- Payments
- Captures
- Refunds
- Domain Events
- Idempotency Records
- Webhook Delivery Jobs
- Delivery Attempts
- Scheduled Tasks
- Scenario Metadata
- Audit Information

The application architecture must not expose SQLite-specific concepts to the domain model.

Instead, SQLite remains an infrastructure concern hidden behind explicit persistence boundaries defined by ADR-0003.

---

# Architectural Principles

This decision is guided by five architectural principles.

## 1. Local-first

Payment Sandbox should be runnable within seconds.

A developer should be able to clone the repository and immediately execute:

```bash
docker run payment-sandbox
```

or

```bash
payment-sandbox serve
```

without first provisioning external infrastructure.

A simulator that requires Docker Compose, PostgreSQL, Redis and multiple configuration files before handling its first HTTP request creates unnecessary friction.

Removing that friction is considered a feature.

---

## 2. Determinism over Scale

Payment Sandbox is fundamentally a deterministic simulator.

Reproducing exactly the same execution twice is considerably more valuable than supporting thousands of concurrent requests.

The simulator is intended to answer questions such as:

- What happens if the connection is lost after the transaction commits?
- What happens if a webhook is delivered three times?
- What happens if the provider responds after thirty seconds?
- What happens if the client retries using the same idempotency key?

None of these scenarios require horizontal scaling.

They require correctness.

SQLite provides the transactional guarantees necessary to preserve deterministic execution.

---

## 3. Operational Simplicity

Every additional runtime dependency increases the cost of using the project.

Requiring PostgreSQL would mean:

- starting another process;
- configuring credentials;
- managing ports;
- initializing databases;
- handling upgrades;
- documenting additional installation steps.

Those tasks are perfectly acceptable for production systems.

They are unnecessary complexity for a development simulator.

Operational simplicity is therefore considered an architectural requirement rather than a convenience.

---

## 4. Durable Simulations

Payment Sandbox intentionally models asynchronous behaviour.

Examples include:

- delayed webhooks;
- scheduled retries;
- retry backoff;
- ambiguous failures;
- duplicate deliveries;
- provider outages.

These behaviours cannot be modelled correctly if application state disappears when the process exits.

Consequently, all asynchronous work must be persisted.

SQLite provides durable transactional storage without introducing additional infrastructure.

---

## 5. Inspectability

Understanding payment systems often requires inspecting their internal state.

Developers should be able to stop the simulator, open the database using any SQLite browser and immediately observe:

- the current payment state;
- the complete event history;
- pending webhook deliveries;
- retry counters;
- scheduled execution timestamps;
- idempotency records.

The persistence layer therefore doubles as an educational tool.

Being able to inspect the system is considered a first-class capability rather than an implementation detail.

# Why SQLite?

SQLite is often perceived as a lightweight database intended for small applications.

That perception is misleading.

SQLite is a fully ACID-compliant relational database with mature transaction semantics, indexing capabilities, foreign key constraints and a battle-tested storage engine. It is deployed in billions of devices and powers systems where reliability is significantly more important than horizontal scalability.

The question addressed by this ADR is therefore not:

> "Can SQLite scale as far as PostgreSQL?"

Instead, the relevant question is:

> "Does SQLite provide every capability required by Payment Sandbox while minimizing operational complexity?"

The answer is yes.

---

## SQLite Matches the Product Philosophy

Every architectural decision should reinforce the purpose of the product.

Payment Sandbox is intentionally designed around the following priorities:

1. Developer Experience
2. Deterministic Behaviour
3. Operational Simplicity
4. Fast Feedback Loops
5. Reproducible Simulations

SQLite aligns naturally with every one of these priorities.

Conversely, databases optimized for large-scale distributed deployments solve problems that Payment Sandbox deliberately does not have.

Choosing SQLite therefore is not a compromise.

It is an architectural alignment.

---

## Zero Infrastructure

One of the primary objectives of Payment Sandbox is reducing the time between discovering the project and executing the first successful simulation.

The ideal onboarding experience looks like:

```text
git clone

↓

go build

↓

./payment-sandbox serve

↓

Ready.
```

No infrastructure should be required beyond the executable itself.

This objective would immediately be compromised if the simulator required:

- PostgreSQL
- Docker Compose
- Redis
- external configuration
- network connectivity
- database provisioning

Each additional dependency increases the cognitive load imposed on new users.

For an educational and testing tool, that cost is unjustified.

SQLite allows the simulator to remain completely self-contained.

---

## Single Binary Distribution

One of the long-term goals of the project is to distribute Payment Sandbox as a single executable.

The ideal distribution model is:

```text
payment-sandbox

↓

Creates database if missing

↓

Runs migrations

↓

Starts HTTP server

↓

Ready.
```

No installation guide.

No infrastructure checklist.

No prerequisite services.

The executable owns its persistence lifecycle.

This deployment model significantly simplifies:

- local experimentation;
- workshops;
- conference demonstrations;
- CI pipelines;
- temporary environments;
- customer support.

---

## Excellent Docker Experience

Containerized execution follows the same philosophy.

Running the simulator should require only:

```bash
docker run payment-sandbox
```

rather than:

```text
Docker Compose

├── PostgreSQL
├── Redis
├── Payment Sandbox
└── Network configuration
```

The objective is not to eliminate Docker Compose entirely.

Rather, Docker Compose should remain optional instead of mandatory.

This distinction dramatically lowers the barrier to entry.

---

## Fast Continuous Integration

Continuous Integration environments benefit enormously from embedded databases.

Every pipeline can:

- create a fresh database;
- execute migrations;
- run deterministic scenarios;
- destroy everything afterwards.

No database service must be provisioned beforehand.

No cleanup is necessary.

No shared state survives between executions.

This naturally improves reproducibility.

---

## ACID Guarantees

Payment Sandbox intentionally models transactional payment systems.

Examples include:

- payment authorization;
- capture;
- refund;
- idempotency;
- webhook scheduling.

Many of these operations require atomic updates.

For example:

```text
Create Payment

↓

Persist Domain Event

↓

Persist Webhook Job

↓

Persist Idempotency Record

↓

COMMIT
```

Either every operation succeeds or none of them does.

Partial persistence would create impossible payment states.

SQLite provides full transactional guarantees that satisfy these requirements without additional infrastructure.

---

## Referential Integrity

Payments contain relationships.

For example:

```text
Payment

├── Capture
├── Refund
├── Domain Events
├── Webhook Jobs
└── Idempotency Records
```

These relationships should be enforced by the database itself whenever possible.

SQLite supports:

- foreign keys;
- cascading deletes;
- unique constraints;
- indexes;
- check constraints.

These capabilities reduce the amount of defensive application code required.

Business invariants remain protected even if implementation mistakes occur.

---

## Durability

Payment Sandbox deliberately persists asynchronous work.

A scheduled webhook should survive:

- process termination;
- operating system restart;
- Docker restart;
- unexpected crashes.

For example:

```text
Payment Created

↓

Webhook scheduled for T+30s

↓

Application crashes

↓

Restart

↓

Webhook still pending
```

This behaviour would not be possible with an in-memory persistence model.

Durability is therefore considered a functional requirement rather than a technical preference.

---

# Why Not PostgreSQL?

PostgreSQL is an outstanding database.

Many production payment platforms—including real payment service providers—should absolutely prefer PostgreSQL over SQLite.

However, architectural decisions must always be evaluated within the context of the product being built.

The question is not whether PostgreSQL is technically superior.

The question is whether PostgreSQL provides sufficient additional value to justify its operational cost.

For Payment Sandbox, the answer is currently no.

---

## Features We Intentionally Do Not Need

PostgreSQL provides capabilities such as:

- streaming replication;
- logical replication;
- clustering;
- partitioning;
- high write concurrency;
- advanced query planning;
- LISTEN / NOTIFY;
- SKIP LOCKED;
- sophisticated permission management.

These are excellent features.

None of them is currently essential to the mission of Payment Sandbox.

Supporting capabilities that the product neither requires nor exposes would unnecessarily complicate both development and maintenance.

Architectural simplicity is preferred over premature scalability.

---

## PostgreSQL Remains a Future Option

Choosing SQLite today does not reject PostgreSQL forever.

Instead, this ADR defines SQLite as the reference implementation.

Future persistence adapters may support PostgreSQL when requirements evolve, provided they preserve the observable behaviour defined by the canonical implementation.

In other words:

SQLite defines correctness.

Alternative databases may define scalability.

# Architectural Consequences

Selecting SQLite is not an isolated infrastructure decision.

It influences the design of transaction boundaries, asynchronous processing, concurrency management and the overall execution model of Payment Sandbox.

The architecture is therefore intentionally designed around SQLite's strengths while avoiding patterns that would amplify its limitations.

---

# Transaction Boundaries

One of the primary responsibilities of the persistence layer is preserving system consistency.

A payment operation is never reduced to a single database row.

Creating a payment may involve persisting:

- the payment itself;
- one or more domain events;
- an idempotency record;
- one or more webhook delivery jobs;
- audit metadata.

These changes represent a single business operation.

They must therefore either all succeed or all fail.

The persistence layer must never expose intermediate states.

The transaction boundary therefore surrounds the complete business operation.

```text
HTTP Request
      │
      ▼
Application Service
      │
      ▼
Begin Transaction
      │
      ▼
Create Payment
      │
      ▼
Persist Domain Events
      │
      ▼
Persist Webhook Jobs
      │
      ▼
Persist Idempotency Record
      │
      ▼
Commit Transaction
      │
      ▼
Return HTTP Response
```

The HTTP response is produced only after the transaction commits successfully.

This guarantees that the observable behaviour of the simulator always reflects durable state.

---

# Keeping Transactions Short

SQLite allows multiple concurrent readers but only one writer at a time.

This characteristic is frequently misunderstood as a severe limitation.

For Payment Sandbox it is largely irrelevant.

The application deliberately keeps write transactions extremely short.

A transaction should contain only:

- domain validation;
- state persistence;
- event persistence;
- scheduling metadata.

A transaction must **never** include:

- outbound HTTP requests;
- webhook delivery;
- artificial delays;
- retries;
- waiting for external services;
- expensive computations.

For example, this is considered correct:

```text
Begin

↓

Persist Payment

↓

Persist Event

↓

Persist Job

↓

Commit
```

Whereas this is explicitly forbidden:

```text
Begin

↓

Persist Payment

↓

Send Webhook

↓

Wait 10 seconds

↓

Retry

↓

Commit
```

Holding a database write lock while waiting for external systems would dramatically reduce concurrency and increase contention.

The architecture therefore isolates persistence from side effects.

---

# Durable Asynchronous Work

Payment providers are fundamentally asynchronous systems.

A successful payment frequently produces work that must happen later.

Examples include:

- webhook delivery;
- retry scheduling;
- delayed notifications;
- reconciliation tasks.

These operations cannot depend on in-memory queues.

Instead, every asynchronous action becomes durable work.

```text
Payment Created

↓

Create Webhook Job

↓

Commit

↓

Worker picks Job

↓

Send Webhook

↓

Persist Result
```

The scheduler therefore consumes persistent jobs rather than transient messages.

This allows Payment Sandbox to recover cleanly after crashes or restarts.

---

# Why Not Execute Webhooks Inside Transactions?

It may appear simpler to immediately send webhooks before committing the transaction.

That approach creates several problems.

Consider the following sequence:

```text
Begin Transaction

↓

Create Payment

↓

Send Webhook

↓

Webhook succeeds

↓

Commit fails
```

The external system has now observed a payment that never existed.

This violates one of the core consistency guarantees of the simulator.

The inverse situation is equally problematic:

```text
Begin

↓

Create Payment

↓

Commit succeeds

↓

Application crashes

↓

Webhook never sent
```

Persisting webhook jobs before commit solves both problems.

The worker always resumes from durable state.

---

# Scheduler Architecture

The scheduler is intentionally simple.

Its responsibility is not to execute business logic.

Its only responsibility is discovering executable jobs.

```text
SQLite

↓

Pending Jobs

↓

Scheduler

↓

Worker Pool

↓

Execution
```

This separation produces several advantages:

- scheduler remains deterministic;
- workers remain stateless;
- failures are isolated;
- retries become straightforward.

The scheduler therefore coordinates execution rather than performing it.

---

# Worker Design

Workers should be completely replaceable.

A worker starts by acquiring work from the persistence layer.

```text
Worker

↓

Acquire Job

↓

Execute

↓

Persist Outcome

↓

Release
```

Workers never depend on in-memory application state.

Every piece of information required to execute a job already exists inside SQLite.

This greatly simplifies:

- crash recovery;
- testing;
- replay;
- deterministic execution.

---

# WAL Mode

SQLite supports multiple journaling strategies.

Payment Sandbox uses **Write-Ahead Logging (WAL)**.

Compared to the traditional rollback journal, WAL provides significant advantages for this workload.

Most importantly:

- readers do not block writers;
- writers do not interrupt readers;
- read concurrency improves substantially.

This is particularly valuable because Payment Sandbox performs many more reads than writes.

For example:

```text
API

───────────────► Reads

Scheduler

───────────────► Reads

Dashboard

───────────────► Reads

Workers

───────────────► Short Writes
```

WAL therefore aligns naturally with the expected access pattern.

---

# Concurrency Model

The application embraces SQLite's concurrency model instead of attempting to circumvent it.

The expected workload consists of:

- many concurrent reads;
- relatively infrequent writes;
- very short write transactions.

Consequently:

```text
Reader
Reader
Reader
Reader
Reader

↓

Writer

↓

Reader
Reader
Reader
```

This access pattern fits SQLite extremely well.

Attempting to optimize for hundreds of concurrent writers would introduce unnecessary architectural complexity while providing little practical value.

---

# Busy Timeout

Although write contention is expected to remain low, temporary lock contention can still occur.

Instead of immediately failing, SQLite should be configured with an appropriate busy timeout.

This allows short-lived write transactions to complete naturally before another writer retries.

The timeout is considered a resilience mechanism rather than a concurrency strategy.

Persistent lock contention is treated as an architectural smell that should be investigated rather than hidden.

---

# Repository Responsibilities

Repositories expose business-oriented persistence operations.

They do **not** expose SQL semantics.

Examples include:

- SavePayment
- FindPayment
- SaveWebhookJob
- AcquirePendingJobs
- SaveIdempotencyRecord

The domain remains completely unaware of:

- SQL dialects;
- SQLite pragmas;
- indexes;
- transactions;
- WAL configuration.

This separation ensures that future persistence adapters can be introduced without affecting domain behaviour.

---

# Interaction with Future Persistence Adapters

SQLite defines the canonical behaviour of the simulator.

Future adapters, including PostgreSQL, are expected to reproduce the same observable behaviour.

Behavioural compatibility is considered more important than implementation similarity.

An adapter may internally exploit PostgreSQL-specific features such as `SKIP LOCKED` or `LISTEN / NOTIFY`, but these optimizations must remain invisible from the perspective of the domain model and public APIs.

In other words:

- the architecture is portable;
- the behaviour is canonical;
- the infrastructure is replaceable.

# Alternatives Considered

Architectural decisions are rarely made by selecting the objectively "best" technology.

Instead, they consist of selecting the technology that best satisfies the constraints of the system being built.

The following alternatives were evaluated before selecting SQLite.

---

## PostgreSQL

### Advantages

- Excellent write concurrency.
- Mature query planner.
- Rich indexing capabilities.
- Native replication.
- High availability.
- Advanced locking primitives.
- `LISTEN / NOTIFY`.
- `SKIP LOCKED`.
- Extensive operational tooling.

### Why It Was Not Selected

PostgreSQL solves problems that Payment Sandbox intentionally does not have.

The simulator is not expected to:

- process millions of payments;
- run as a distributed cluster;
- provide high availability;
- support multiple application nodes by default;
- survive datacenter failures.

Introducing PostgreSQL as the default persistence layer would increase:

- installation complexity;
- operational overhead;
- onboarding friction;
- maintenance burden.

without providing proportional value to the primary use cases.

PostgreSQL therefore remains a valid future adapter rather than the canonical implementation.

---

## MySQL

MySQL was considered for the same reasons as PostgreSQL.

Although mature and widely deployed, it offers no compelling advantage for the current objectives of Payment Sandbox.

Supporting multiple SQL dialects would increase maintenance effort while providing little benefit for a local-first simulator.

---

## Embedded Key-Value Databases

Examples include:

- BoltDB
- Bbolt
- BadgerDB

### Advantages

- Extremely small footprint.
- Excellent raw performance.
- Embedded deployment.
- Simple distribution.

### Why They Were Not Selected

Payment Sandbox manipulates highly relational data.

Examples include:

```text
Payment

├── Captures
├── Refunds
├── Events
├── Webhook Jobs
├── Delivery Attempts
└── Idempotency Records
```

Representing these relationships in a key-value database would move responsibility for:

- referential integrity;
- joins;
- constraints;
- indexing;
- transactional consistency

from the database into application code.

That additional complexity provides little practical benefit.

SQLite already offers an embedded deployment model while preserving the expressive power of a relational database.

---

## DuckDB

DuckDB is an exceptional analytical database.

Its strengths include:

- column-oriented execution;
- analytical queries;
- OLAP workloads.

Payment Sandbox performs transactional workloads rather than analytical ones.

The project therefore aligns much more closely with SQLite than DuckDB.

---

## Redis

Redis is frequently used for queues, caching and ephemeral state.

However, it is not intended to serve as the primary durable storage engine for Payment Sandbox.

Persisting the complete payment state inside Redis would introduce unnecessary operational complexity while weakening relational guarantees.

Redis may eventually become an optional optimization for specific workloads, but never the canonical source of truth.

---

## In-Memory Persistence

An in-memory implementation offers an excellent developer experience for unit testing.

However, it fundamentally conflicts with several architectural goals.

The following information would disappear whenever the process exits:

- pending webhook deliveries;
- retry schedules;
- execution history;
- idempotency records;
- audit trail;
- scenario state.

A simulator that forgets its own execution history cannot faithfully reproduce real payment providers.

In-memory persistence is therefore appropriate for isolated tests, but not for the reference implementation.

---

# Implementation Guidelines

This section documents implementation rules that every persistence implementation must follow.

These rules are considered part of the architecture rather than implementation details.

---

## SQLite Configuration

Unless explicitly documented otherwise, every SQLite database should be configured using:

- `PRAGMA foreign_keys = ON`
- `PRAGMA journal_mode = WAL`
- `PRAGMA synchronous = NORMAL`
- `PRAGMA busy_timeout = <configured value>`

These settings provide the best balance between durability, performance and concurrency for the expected workload.

Future changes to these settings should be documented through a dedicated ADR.

---

## Migration Strategy

Database schema changes must be versioned.

The application is responsible for:

- creating the database when missing;
- executing pending migrations;
- refusing to start if migrations fail.

Schema evolution must never require manual SQL execution.

The executable owns the lifecycle of its database.

---

## Transaction Rules

Transactions should satisfy the following constraints.

### Transactions MUST

- remain as short as possible;
- modify only durable business state;
- persist asynchronous work before commit;
- preserve referential integrity.

### Transactions MUST NOT

- perform HTTP requests;
- call webhook endpoints;
- sleep or wait;
- retry external operations;
- execute expensive computations.

Transactions exist solely to preserve consistency.

They are not workflow engines.

---

## Repository Responsibilities

Repositories encapsulate persistence.

Repositories are responsible for:

- mapping domain objects;
- transaction participation;
- optimistic or pessimistic persistence strategies;
- SQL execution.

Repositories are not responsible for:

- business rules;
- retry logic;
- webhook orchestration;
- scheduling;
- observability.

---

## Scheduler Responsibilities

The scheduler must never execute business logic.

Its sole responsibility is identifying executable work.

```text
SQLite

↓

Pending Jobs

↓

Scheduler

↓

Workers
```

This separation makes the execution engine deterministic and testable.

---

## Worker Responsibilities

Workers execute durable jobs.

Workers are expected to be stateless.

Every execution must derive exclusively from persisted state.

No worker should require in-memory application context to resume execution.

This guarantees crash recovery.

---

## Failure Handling

Persistence failures are treated differently from business failures.

Examples:

```text
Business Failure

↓

Persist Failure

↓

Commit

↓

Return Error
```

versus

```text
Persistence Failure

↓

Rollback

↓

Nothing happened
```

The simulator intentionally distinguishes between domain failures and infrastructure failures.

This distinction is fundamental to modelling payment systems accurately.

---

# Trade-offs

Choosing SQLite implies accepting several limitations.

These limitations are considered acceptable because they align with the intended workload.

## Single Writer

SQLite allows only one concurrent writer.

Rather than attempting to eliminate this limitation, the architecture embraces it by ensuring that write transactions remain:

- short;
- deterministic;
- free of external I/O.

The resulting contention is expected to remain negligible.

---

## Vertical Rather Than Horizontal Scaling

SQLite scales primarily by making a single node efficient.

Payment Sandbox deliberately prioritizes:

- reproducibility;
- correctness;
- simplicity;

over horizontal scalability.

Should distributed deployments become a product requirement, PostgreSQL can be introduced as an alternative persistence adapter.

---

## Behaviour Over Throughput

The simulator values behavioural accuracy over raw performance.

Correctly modelling:

- retries;
- ambiguous failures;
- webhook delivery;
- idempotency;

is significantly more valuable than maximizing transactions per second.

---

# What This Decision Is Not

This ADR should not be interpreted as claiming that:

- SQLite is superior to PostgreSQL.
- SQLite is appropriate for every payment system.
- SQLite is suitable for high-volume payment providers.
- distributed databases are unnecessary.
- operational scalability lacks value.

None of these statements is true.

Instead, this ADR asserts that SQLite is the persistence engine that best satisfies the architectural goals of Payment Sandbox.

The decision is contextual rather than universal.

---

# Revisit Conditions

This ADR should be re-evaluated if one or more of the following conditions become true:

- Payment Sandbox primarily targets hosted deployments rather than local execution.
- Multiple application instances must coordinate through a shared database.
- Sustained write contention becomes a measurable bottleneck.
- SQLite-specific limitations significantly complicate new features.
- Operational requirements evolve beyond the capabilities of an embedded database.

Revisiting this ADR does not necessarily imply replacing SQLite.

A more likely evolution would be introducing PostgreSQL as an additional supported persistence adapter while preserving SQLite as the default local-first experience.

---

# Consequences

## Positive

- Zero external infrastructure.
- Extremely simple installation.
- Fast onboarding.
- Excellent Docker experience.
- Reproducible local environments.
- Deterministic persistence.
- Durable asynchronous execution.
- Transactional consistency.
- Easy inspection using standard SQLite tooling.
- Low operational maintenance.

## Negative

- Single writer architecture.
- Limited horizontal scalability.
- Careful transaction boundaries required.
- Certain PostgreSQL optimizations cannot be used.
- Future multi-node deployments may require an additional persistence implementation.

These consequences are accepted because they reinforce the product philosophy rather than contradict it.

---

# Decision Summary

Payment Sandbox is intentionally designed as a local-first, deterministic payment systems simulator.

SQLite enables the project to remain:

- easy to install;
- easy to understand;
- easy to inspect;
- easy to test;
- easy to distribute.

Rather than selecting the database capable of solving the largest possible problem, this ADR selects the database that most closely matches the problem Payment Sandbox is actually trying to solve.

For this reason, SQLite is adopted as the canonical persistence engine of the project.

---

# References

- SQLite Documentation — https://sqlite.org/docs.html
- SQLite Write-Ahead Logging — https://sqlite.org/wal.html
- SQLite Locking and Concurrency — https://sqlite.org/lockingv3.html
- Martin Fowler — *Patterns of Enterprise Application Architecture*
- Gregor Hohpe & Bobby Woolf — *Enterprise Integration Patterns*
- Pat Helland — *Life Beyond Distributed Transactions*
- ADR-0001 — Modular Monolith
- ADR-0003 — Targeted Hexagonal Architecture
- ADR-0005 — Persist Asynchronous Work
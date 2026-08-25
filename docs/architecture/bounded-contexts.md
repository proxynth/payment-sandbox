# Bounded Contexts

Payment Sandbox is organised around four bounded contexts. Together they simulate the behaviour of a payment service provider while keeping payment rules, simulation logic, asynchronous execution and diagnostic capabilities clearly separated.

## Purpose

Payment Sandbox is implemented as a modular monolith composed of multiple bounded contexts.

Each bounded context owns a specific business responsibility, defines its own ubiquitous language and protects its own consistency boundaries.

Bounded contexts are introduced to organise the business domain, **not** the source code.

The repository structure follows these boundaries as the project evolves, but packages are only created once they own concrete behaviour.

---

## Context Overview

The rendered canonical version of the context and dependency diagram is
available in [Architecture Diagrams](diagrams.md#contexts-and-dependency-direction).

The following diagram illustrates the responsibilities of each bounded context and the flow of information between them.

                                        ┌──────────────────────┐
                    │      Simulation      │
                    │----------------------│
                    │ Scenarios            │
                    │ Rules                │
                    │ Fault Injection      │
                    └──────────┬───────────┘
                               │
                     Behaviour Plan
                               │
                               ▼
                    ┌──────────────────────┐
                    │       Payment        │
                    │----------------------│
                    │ Lifecycle            │
                    │ Money                │
                    │ Invariants           │
                    └──────────┬───────────┘
                               │
                        Business Events
                               │
                               ▼
                    ┌──────────────────────┐
                    │      Runtime         │
                    │----------------------│
                    │ Jobs                 │
                    │ Scheduler            │
                    │ Retries              │
                    │ Callbacks            │
                    └──────────┬───────────┘
                               │
                         Read Models
                               │
                               ▼
                    ┌──────────────────────┐
                    │    Administration    │
                    │----------------------│
                    │ Timeline             │
                    │ Diagnostics          │
                    │ Inspection           │
                    └──────────────────────┘

---

## Initial Bounded Contexts

### Payment

#### Responsibility

The Payment context represents the canonical payment domain.

It owns the complete payment lifecycle independently of any payment service provider.

#### Owns

- Payment
- Authorization
- Capture
- Cancellation
- Refund
- Money
- Payment state transitions
- Business invariants

#### Does not own

- Provider-specific terminology
- HTTP
- Persistence
- Scheduling
- Webhooks

---

### Simulation

#### Responsibility

The Simulation context determines how the sandbox behaves.

It evaluates scenarios, applies rules and decides which behaviour should be simulated.

#### Owns

- Scenario
- Scenario rules
- Fault injection
- Random behaviour
- Deterministic execution
- Behaviour planning

#### Does not own

- Payment state
- Persistence
- Scheduling

---

### Runtime

#### Responsibility

The Runtime context performs durable asynchronous work.

It is responsible for executing work over time, independently of the business domain.

#### Owns

- Durable jobs
- Scheduling
- Retries
- Delayed execution
- Outbound callbacks

#### Does not own

- Payment rules
- Simulation rules

---

### Administration

#### Responsibility

The Administration context exposes diagnostic and inspection capabilities.

It provides visibility into the behaviour of the sandbox without owning business state.

#### Owns

- Timeline views
- Diagnostics
- Inspection APIs
- Administrative read models

#### Does not own

- Payments
- Scenarios
- Jobs

---

## Context Relationships

The bounded contexts collaborate through explicit contracts.

### Simulation → Payment

Simulation determines how the Payment context should behave for a given scenario.

It never modifies payment state directly.

### Payment → Runtime

Payment produces business events and requests asynchronous work.

Runtime decides when and how this work is performed.

### Administration → *

Administration consumes information exposed by other bounded contexts.

It never owns or mutates business state.

---

## Ubiquitous Language

| Term | Definition |
|------|------------|
| Payment | Canonical representation of a payment transaction. |
| Authorization | Approval allowing funds to be captured later. |
| Capture | Collection of previously authorised funds. |
| Refund | Return of previously captured funds. |
| Cancellation | Cancellation of an uncaptured payment. |
| Scenario | Deterministic description of simulated behaviour. |
| Job | Durable unit of asynchronous work. |
| Callback | Outbound notification generated by the simulated provider. |

---

## Implementation Rules

The following rules guide the implementation of bounded contexts throughout the project.

- Each bounded context owns a single business responsibility.
- Business concepts have a single owner.
- Mutable business state must not be shared across contexts.
- Provider-specific terminology must not leak into the Payment context.
- Cross-context communication should occur through explicit contracts.
- Cross-cutting technical concerns belong to the Platform layer.
- Bounded contexts are materialised in the source tree only when they own concrete behaviour.
- Empty context directories must not be created.

These rules favour simplicity, evolvability and explicit ownership while keeping the architecture aligned with the project's architectural principles and ADRs.

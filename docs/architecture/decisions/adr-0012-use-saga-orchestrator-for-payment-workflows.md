# ADR-0012: Use a Saga orchestrator for multi-step payment workflows

## Status

Accepted

## Context

A payment workflow can span several provider operations. Each operation can
complete later, be delivered more than once, or fail after a previous step has
already changed the payment. A single database transaction cannot cover the
provider boundary. The existing scheduler and worker already provide durable,
virtual-time-controlled work inside the modular monolith, but no component
currently owns the workflow state or compensation decisions.

## Decision

Payment workflows use a Saga coordinated by one explicit orchestrator.

- The Saga instance stores its current step, completed steps, compensation
  steps, deterministic seed, status, and version.
- The orchestrator is the only component that chooses the next step or starts
  compensation. It does not mutate the payment directly; step handlers use
  the payment domain and provider contracts.
- A Saga message is an envelope containing a stable message ID, Saga ID,
  payment ID, step, payload, seed, scheduled virtual time, and attempt.
- Messages are persisted as scheduler jobs. The scheduler and worker are the
  internal at-least-once transport; Redis, Kafka, and a separate broker are
  deliberately out of scope.
- Message handling is idempotent. A duplicate for a step that is no longer
  current is acknowledged without executing the step again.
- A failed step compensates completed work in reverse business order. A
  captured payment is refunded then cancelled; an authorised payment is
  cancelled. Compensation itself is durable and replayable.
- State is persisted before the next message is published. Production
  adapters must perform the state/message write in one database transaction
  or use an equivalent durable outbox guarantee.

## Consequences

The project demonstrates Saga and orchestrator patterns without pretending to
be a distributed system. Crash recovery, retries, duplicate delivery,
compensation, virtual time, and seeded provider decisions are testable. The
trade-off is additional durable state and explicit workflow diagnostics.

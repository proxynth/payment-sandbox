# Runtime guarantees and limitations

This document summarizes runtime behavior covered by the project's automated tests. It is not a formal proof, and guarantees apply only to the paths and scenarios tested.

## Behavior covered by tests

- **Payment changes and related records:** Covered payment transitions persist the payment state, event, and associated scheduler jobs atomically in SQLite.
- **Idempotent requests:** Tests cover duplicate and concurrent requests on supported payment endpoints, including conflicts when a key is reused with different input.
- **Durable jobs:** Scheduler jobs and lifecycle snapshots are persisted. Configured retry policies bound total attempts; allowed retries return to pending at their scheduled time, and a terminal decision persists the job as `exhausted`. On recovery, a previously failed row is checked against the configured attempt budget before its handler runs. Exhausted jobs remain inspectable and are not executed again.
- **Scheduler lease fencing:** Each SQLite acquisition advances a per-job lease generation. Worker state writes are accepted only for the generation currently stored; an expired worker cannot overwrite a later acquisition, even when the owner name is reused. The scheduler audit records the generation with each accepted snapshot.
- **Deterministic scenarios:** Scenario runs use configured seeds and virtual time to produce reproducible provider outcomes within the simulation. This does not make the entire live runtime deterministic.
- **Operational history:** Payment events, job lifecycle records, and webhook delivery attempts can be correlated. Causal metadata and runtime sequence values are available for records created after the corresponding database migrations.

## Known limitations

- **Webhook delivery is at-least-once.** If a receiver accepts a request but the process crashes before recording completion, a retry can deliver it again. Receivers should handle duplicates idempotently; exactly-once delivery over HTTP is not guaranteed.
- **Lease fencing does not cancel an in-flight handler.** If a lease expires and the job is reacquired, an earlier handler may already have performed an external effect; fencing rejects its later scheduler-state write but cannot undo that effect. Handlers must tolerate at-least-once execution.
- **Scenario replay is not full runtime replay.** The project can reproduce simulations within their defined scope, but it does not automatically reconstruct and re-execute an entire historical runtime, including external effects.
- **Older records have less metadata.** Records created before the relevant migrations may not have causal links or runtime sequence values. Missing historical data is not inferred.
- **SQLite is the persistence boundary.** The tested transactional guarantees apply to the local SQLite-backed runtime. The project does not claim multi-node coordination or distributed high availability.
- **Webhook audit records omit request and response bodies.** They retain delivery metadata and outcomes, but not callback payloads.

## Verification

The repository's automated checks include `make check` and `make test-race`. These checks provide evidence for the tested behavior; they do not prove correctness for every possible execution or failure interleaving.

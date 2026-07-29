ADR-0001: Use a modular monolith

* Status: Accepted
* Date: 2026-07-28
* Deciders: Project maintainers
* Technical area: Architecture
* Related decisions:
    * ADR-0002: Define a canonical payment model
    * ADR-0003: Use targeted hexagonal architecture
    * ADR-0005: Persist asynchronous work

Context

Payment Sandbox is an open-source tool designed to simulate the behaviour of a payment service provider.

It must expose synchronous payment APIs while also modelling asynchronous and distributed-system behaviours such as:

* delayed webhook delivery;
* duplicate webhook delivery;
* out-of-order events;
* retries;
* ambiguous operation results;
* temporary inconsistencies;
* concurrent capture or refund requests;
* failures occurring before or after a business operation is persisted.

Because the project deals with distributed-system concerns, a distributed runtime architecture could appear to be a natural choice.

For example, the system could be decomposed into separate services for:

* the payment API;
* scenario execution;
* event scheduling;
* webhook delivery;
* administration;
* observability.

However, the primary purpose of Payment Sandbox is to simulate distributed-system behaviours for its users. The project does not need to be physically distributed in order to model those behaviours correctly.

The initial product must also remain:

* easy to run locally;
* easy to embed in automated tests;
* easy to package as a single Docker container;
* understandable by contributors;
* operationally lightweight;
* inexpensive to evolve while the domain and public contracts are still changing.

Introducing multiple deployable services from the beginning would create substantial operational and architectural complexity before that complexity provides product value.

At the same time, implementing the project as an unstructured monolith would make it difficult to maintain clear domain boundaries and could lead to coupling between:

* HTTP handlers;
* payment rules;
* scenario decisions;
* database code;
* schedulers;
* webhook workers;
* administrative features.

The architecture therefore needs to preserve strong internal boundaries without requiring multiple independently deployed services.

Decision

Payment Sandbox will be implemented as a modular monolith.

The default distribution will consist of:

* one Go binary;
* one application process;
* one primary persistence system;
* one Docker image;
* one deployable unit.

The application will be divided into explicit functional modules.

The initial module boundaries are expected to include:

* Payment;
* Simulation;
* Webhook;
* Idempotency;
* Scheduler;
* Administration;
* Observability;
* Platform.

These names are indicative rather than permanent. Modules may be renamed, merged or split as the domain becomes clearer.

Each module will:

* own a clearly defined responsibility;
* expose a limited internal API;
* avoid direct access to another module’s internal implementation;
* keep domain rules separate from transport and persistence concerns;
* depend on other modules only through explicit application-level contracts where necessary.

The architecture will not require every module to reproduce the same directory structure mechanically.

For example, a module with meaningful domain behaviour may contain:

payment/
├── domain/
├── application/
└── adapters/

A simpler technical module may use a flatter structure if additional layers do not improve clarity.

The application composition root will remain close to the executable entry point. Dependency construction and runtime wiring will not be hidden behind a dependency injection framework.

Asynchronous work, such as webhook delivery, may execute in background workers inside the same process. However, durable work must be persisted before execution and must not rely solely on:

* goroutines;
* in-memory channels;
* timers;
* process-local queues.

A module may later become a separate service only when there is a demonstrated operational or product requirement.

The initial module boundaries must therefore be clear enough to permit future extraction, but the project will not introduce abstractions solely to make hypothetical extraction easier.

Rationale

A single deployable unit improves developer experience

The primary users of Payment Sandbox are developers running it:

* on their workstation;
* through Docker Compose;
* in an integration test;
* in a CI pipeline;
* in an ephemeral test environment.

A single process reduces the amount of required setup.

Users should not need to deploy and coordinate:

* an API service;
* a worker service;
* a broker;
* multiple databases;
* a service discovery mechanism.

A modular monolith supports the intended local-first experience.

Distributed behaviours do not require microservices

The important distributed-system properties exist at the interaction boundary between the sandbox and the application under test.

Payment Sandbox can model:

* at-least-once webhook delivery;
* delayed visibility;
* duplicate messages;
* request timeouts;
* connection resets;
* ambiguous commits;
* retries;
* race conditions;
* eventual consistency;

without deploying each concern as a separate network service.

The correctness of these simulations depends on explicit state transitions, transaction boundaries, scheduling and fault injection—not on the number of runtime processes.

The domain and contracts will evolve

The initial payment model, scenario DSL and administrative API are expected to evolve significantly.

Keeping the modules within one codebase and one process makes refactoring less expensive while the product is still discovering its stable boundaries.

Premature service extraction would make routine changes require:

* network contract changes;
* compatibility management;
* cross-service migrations;
* deployment coordination;
* distributed tracing;
* additional failure handling.

Those costs are not justified during the initial phases.

Internal boundaries are still required

Choosing a monolith does not imply accepting unrestricted coupling.

The project must remain modular because the major areas have different responsibilities and rates of change.

For example:

* Payment protects monetary and state-transition invariants.
* Simulation decides which behaviour should be applied.
* Webhook manages payloads, signatures and delivery attempts.
* Scheduler handles eligible durable work.
* Administration exposes test control and inspection capabilities.
* Platform contains technical adapters and runtime concerns.

Keeping these responsibilities explicit supports maintainability, testing and future evolution.

Operational simplicity is a product feature

Payment Sandbox is an infrastructure tool.

Its own installation and operation form part of its user experience.

A single executable and a single container are therefore not merely implementation conveniences. They are product requirements.

Consequences

Positive consequences

* The project can be distributed as one Go binary.
* Local execution requires minimal configuration.
* Docker and CI usage remain straightforward.
* Cross-module refactoring remains inexpensive during early development.
* Transactions can cover payment mutations and event creation without distributed transaction coordination.
* Contributors can understand and run the complete system more easily.
* The project avoids introducing a broker or orchestration platform before either is necessary.
* Internal boundaries can still reflect the domain and support isolated testing.
* Background workers can share the same deployment while durable work remains persisted.
* Observability can correlate API, domain and worker activity within one runtime.

Negative consequences

* All modules share the same process and therefore the same failure domain.
* A crash in one component may interrupt the entire sandbox instance.
* CPU-intensive or blocking work can affect other modules if resource usage is not controlled.
* Modules cannot initially be scaled independently.
* The application may require disciplined ownership to prevent internal boundaries from eroding.
* Database access patterns from different modules may create contention.
* Future extraction into separate services may still require architectural changes.
* A single binary may eventually become larger than necessary for some specialised deployments.

Neutral or accepted trade-offs

* High availability is not an initial objective.
* Independent scaling is not an initial objective.
* Process-level isolation between modules is not an initial objective.
* The sandbox is expected to run as an isolated instance for a developer, test suite or test environment.
* Restarting the process is acceptable as long as persisted state and scheduled work can recover correctly when persistence is enabled.

Architectural constraints

The modular monolith decision introduces the following constraints.

Modules must not bypass explicit boundaries

A module must not directly manipulate another module’s internal domain objects or persistence implementation.

For example:

* an HTTP handler must not update payment database rows directly;
* the scenario module must not mutate payment state directly;
* the webhook module must not infer payment transitions;
* the payment module must not perform outbound HTTP delivery.

Cross-module interactions must happen through explicit use cases, commands, queries or narrowly scoped contracts.

The domain must not depend on runtime infrastructure

Domain code must not directly depend on:

* HTTP handlers;
* SQL drivers;
* logging frameworks;
* OpenTelemetry;
* environment variables;
* filesystem configuration;
* goroutine scheduling.

Durable asynchronous work must survive process interruption

Background work may execute in-process, but its source of truth must be persisted.

A process restart must not silently lose:

* generated payment events;
* scheduled webhook deliveries;
* retry state;
* delivery attempts.

The composition root must remain explicit

Runtime construction should remain visible and understandable.

The project will not introduce a dependency injection framework merely to automate object construction.

Manual wiring is preferred unless the composition becomes demonstrably unmanageable.

Shared packages must remain limited

A generic shared, common or utils package must not become a dumping ground.

Code should remain in the module that owns its meaning unless it represents a genuinely stable, cross-cutting primitive.

Examples of potentially shared primitives include:

* clock abstractions;
* identifier generation;
* transaction boundaries;
* observability setup.

Even these abstractions should be introduced only when justified.

Alternatives considered

Alternative 1: Microservices from the beginning

The system could be divided into separate services such as:

* payment API;
* scenario service;
* webhook service;
* scheduler;
* administration API.

This option was rejected because it would introduce:

* network communication between internal components;
* multiple deployment units;
* service discovery or configuration;
* more complex local development;
* more complex CI setup;
* distributed transaction problems;
* increased observability requirements;
* contract versioning between services.

These costs would not improve the initial product capabilities.

Microservices would also risk turning the project into a demonstration of infrastructure complexity rather than a focused payment simulation tool.

Alternative 2: Unstructured monolith

All functionality could be implemented in a flat application organised primarily by technical category:

handlers/
services/
repositories/
models/
workers/

This option was rejected because it tends to obscure domain ownership and encourages coupling across unrelated concerns.

As the product grows, payment behaviour, simulation rules and webhook delivery would become difficult to evolve independently.

Alternative 3: Plugin-based architecture

The system could be built around a plugin runtime, with payment models, faults and provider profiles implemented as dynamically loaded extensions.

This option was rejected for the initial architecture because it would require early decisions about:

* plugin lifecycle;
* binary compatibility;
* extension contracts;
* isolation;
* versioning;
* security;
* error handling.

The product does not yet have stable enough boundaries to define a durable plugin API.

Static extension through Go packages is sufficient initially.

Alternative 4: Serverless functions

Each operation or asynchronous component could be deployed as a function.

This option was rejected because it would conflict with:

* local-first execution;
* deterministic test control;
* simple packaging;
* persistent scheduler behaviour;
* portability across environments.

It would also tie the project to an execution model that is not necessary for its core purpose.

Alternative 5: Separate API and worker binaries

The repository could provide at least two processes:

* an API process;
* a worker process.

This option is plausible and may become useful later.

It was not selected as the default because the initial workload does not justify separate deployment, scaling or lifecycle management.

The code should not prevent adding a dedicated worker executable later, but no architectural abstraction will be introduced solely for that hypothetical requirement.

Rejected interpretations

This decision does not mean that:

* all code must live in one package;
* modules may access each other’s database tables freely;
* every operation must execute synchronously;
* durable queues are unnecessary;
* concurrency concerns can be ignored;
* process-local memory can be the source of truth;
* future service extraction is prohibited.

It means only that separate runtime deployment is not the default architecture.

Implementation guidance

The initial repository may evolve toward a structure similar to:

cmd/
└── payment-sandbox/
└── main.go
internal/
├── payment/
│   ├── domain/
│   ├── application/
│   └── adapters/
│
├── simulation/
│   ├── domain/
│   ├── application/
│   └── adapters/
│
├── webhook/
│   ├── application/
│   ├── delivery/
│   └── signing/
│
├── scheduler/
├── idempotency/
├── adminapi/
├── providerapi/
├── observability/
└── platform/
├── clock/
├── database/
├── httpserver/
└── config/

This structure is illustrative.

Directories should only be added when they contain real responsibilities. Empty architectural placeholders should not be committed merely to match the diagram.

Module interaction example

A payment creation flow may follow this sequence:

1. The Provider API receives a create-payment request.
2. The API adapter maps the HTTP request to an application command.
3. The Simulation module determines the applicable behaviour plan.
4. The Payment application service executes the requested domain transition.
5. The payment and its generated event are persisted in one transaction.
6. A webhook delivery job is persisted.
7. The API adapter applies the scenario-defined response behaviour.
8. The in-process scheduler later claims the persisted job.
9. The Webhook module attempts delivery and records the result.
10. The Administration API exposes the resulting timeline.

All of these steps may occur in one process while still representing asynchronous and distributed behaviour from the perspective of the application under test.

Operational model

The default runtime will contain:

* an HTTP server;
* one or more background workers;
* a persistence connection;
* graceful shutdown coordination;
* health and readiness endpoints;
* structured logging;
* metrics and tracing instrumentation.

The process must shut down cleanly by:

* stopping intake of new work;
* cancelling or completing in-flight operations according to documented rules;
* releasing reserved jobs;
* closing the HTTP server;
* closing persistence resources.

Revisit when

This decision should be reconsidered when one or more of the following conditions are demonstrated:

* API traffic and webhook workloads require independent scaling.
* A hosted version requires strong process isolation between modules.
* Multiple teams need independent ownership and release cycles.
* A component must be deployed in a different trust or network boundary.
* Long-running webhook work materially affects API availability.
* A provider profile requires an isolated runtime or dependency set.
* The single process becomes a measurable performance bottleneck.
* Fault containment requirements exceed what a shared process can provide.
* Independent worker deployment materially improves reliability or operations.

Reconsidering this ADR does not imply that the whole system must become microservices.

A likely intermediate evolution would be:

* retain the modular monolith codebase;
* provide separate API and worker binaries;
* keep a shared persistence model;
* extract only the components with demonstrated operational needs.

Compliance

Code reviews should verify that new functionality:

* belongs to an identified module;
* does not bypass module responsibilities;
* does not introduce infrastructure dependencies into the domain;
* does not add a new deployable service without revisiting this ADR;
* does not add speculative abstractions solely for future service extraction.

Outcome

Payment Sandbox will begin as a modular monolith because this architecture provides the best balance between:

* product simplicity;
* local developer experience;
* architectural clarity;
* transactional correctness;
* evolvability;
* operational cost.

The project will model distributed-system behaviour explicitly while avoiding a physically distributed architecture until concrete requirements justify it.
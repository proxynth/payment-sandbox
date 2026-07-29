# ADR-0003: Use targeted hexagonal architecture

- Status: Accepted
- Date: 2026-07-29
- Deciders: Project maintainers
- Technical area: Application architecture
- Related decisions:
    - ADR-0001: Use a modular monolith
    - ADR-0002: Define a canonical payment model
    - ADR-0004: Use SQLite as the default persistence engine
    - ADR-0005: Persist asynchronous work

## Context

Payment Sandbox must remain easy to understand, test and extend while modelling concerns such as:

- payment lifecycle rules;
- idempotency;
- concurrency;
- persisted asynchronous work;
- webhook delivery;
- deterministic time;
- fault injection;
- HTTP APIs;
- storage;
- observability.

Several of these concerns depend on technical infrastructure or external effects.

Examples include:

- reading the current time;
- generating identifiers;
- generating deterministic random values;
- persisting payments and events;
- controlling transaction boundaries;
- sending outbound webhook requests;
- scheduling durable work;
- exposing HTTP endpoints;
- exporting logs, metrics and traces.

The project also needs to remain testable without making every test dependent on:

- a real clock;
- a live HTTP destination;
- global random generation;
- a specific database implementation;
- process-level timers;
- runtime configuration.

Hexagonal architecture provides a useful model for isolating the application and domain from external infrastructure through ports and adapters.

However, a mechanical or maximalist interpretation of hexagonal architecture can produce unnecessary complexity.

Common symptoms include:

- an interface for every concrete type;
- one port for every method;
- repositories for entities that do not require independent persistence;
- application services wrapping trivial functions;
- adapters delegating to other adapters without adding semantics;
- generated mocks for every dependency;
- package hierarchies created before responsibilities exist;
- domain objects converted through many nearly identical representations;
- abstractions introduced only because they may theoretically be useful later.

Payment Sandbox must demonstrate architectural discipline without turning architecture into ceremony.

The project therefore needs a clear rule for determining where hexagonal boundaries provide actual value and where direct code is preferable.

## Decision

Payment Sandbox will use a **targeted hexagonal architecture**.

Hexagonal architecture will be applied around meaningful architectural boundaries, especially where the application depends on external effects, volatile infrastructure or behaviour that must be controlled during tests.

The project will not require every type or component to be represented by an interface.

An interface or port should be introduced only when at least one of the following applies:

- the dependency represents an external system or side effect;
- the dependency must be replaced to make behaviour deterministic;
- multiple production implementations are expected or already exist;
- the dependency forms a meaningful boundary between modules;
- the abstraction protects application or domain code from infrastructure details;
- the abstraction defines a capability rather than mirroring a concrete implementation.

Concrete types are preferred when:

- there is only one implementation;
- the implementation is stable and internal;
- no substitution is required;
- no architectural boundary is crossed;
- an interface would merely duplicate the public methods of a struct;
- introducing the abstraction would not improve testability or clarity.

## Architectural model

The project will distinguish broadly between:

- domain;
- application;
- inbound adapters;
- outbound ports;
- outbound adapters;
- composition root.

This distinction is conceptual and does not require every module to contain every layer.

### Domain

The domain contains business concepts, invariants and transitions.

Examples include:

- Payment;
- Money;
- Capture;
- Refund;
- payment lifecycle rules;
- monetary invariants;
- domain errors.

Domain code must not depend directly on:

- HTTP;
- SQL;
- configuration files;
- observability SDKs;
- environment variables;
- outbound network clients;
- process-level concurrency primitives;
- wall-clock access.

The domain may depend on small domain-level abstractions only when the behaviour is genuinely part of the domain model.

### Application

The application layer orchestrates use cases.

Examples include:

- create a payment;
- capture a payment;
- cancel a payment;
- refund a payment;
- load and apply a scenario;
- schedule a webhook delivery;
- inspect a payment timeline.

Application code may:

- load domain state;
- invoke domain behaviour;
- coordinate repositories;
- define transaction boundaries;
- invoke external capabilities through ports;
- translate domain outcomes into application results.

Application code should not contain transport-specific response formatting or SQL-specific logic.

### Inbound adapters

Inbound adapters invoke application use cases.

Examples include:

- Provider REST API;
- Administration REST API;
- CLI commands;
- test harnesses;
- future provider profiles.

Inbound adapters are responsible for:

- parsing input;
- protocol-level validation;
- authentication or access control when applicable;
- mapping requests to application commands or queries;
- mapping application results to protocol responses;
- propagating request context and correlation information.

Inbound adapters must not bypass application use cases to modify persistence directly.

### Outbound ports

Outbound ports represent capabilities required by the application or domain but implemented by infrastructure.

Likely ports include:

- PaymentRepository;
- RefundRepository;
- EventRepository;
- DeliveryJobRepository;
- TransactionManager;
- Clock;
- IDGenerator;
- RandomSource;
- WebhookSender;
- ScenarioStore, if scenario persistence is introduced;
- DecisionLog, if it is not implemented as a direct application component.

Ports should be small and capability-oriented.

They must not expose infrastructure-specific details unless those details are intentional application concerns.

### Outbound adapters

Outbound adapters implement ports using concrete infrastructure.

Examples include:

- SQLite repositories;
- PostgreSQL repositories;
- system clock;
- virtual clock;
- seeded pseudo-random generator;
- outbound HTTP webhook sender;
- in-memory implementation for narrow tests;
- OpenTelemetry exporters;
- filesystem scenario loader.

### Composition root

The composition root creates and connects concrete implementations.

It should remain explicit and close to the executable entry point.

The project will prefer manual dependency wiring.

A dependency injection framework will not be introduced unless manual composition becomes demonstrably difficult to maintain.

## Initial ports

The following ports are considered justified in the initial architecture.

### Clock

The project must control time for:

- timestamps;
- delayed events;
- retry scheduling;
- timeout simulation;
- deterministic tests;
- future virtual-clock support.

The application must not call `time.Now()` directly in business-relevant code.

Conceptually:

```text
Clock
- Now() time.Time
```

Additional methods should be introduced only when required.

Sleeping and scheduling should not automatically be added to the same port. Reading time and coordinating durable work are separate responsibilities.

### IDGenerator

Sandbox identifiers should be:

- opaque;
- unique;
- predictable in format;
- controllable in tests when useful.

An identifier generator is a meaningful boundary because identifiers are created outside the payment domain's business rules.

The interface should represent identifier creation, not expose a third-party UUID or ULID library.

### RandomSource

Random fault injection must remain reproducible.

The simulation engine should depend on a controlled random source rather than global package-level randomness.

The port should expose only the operations required by scenario evaluation.

It must not expose a generic random API broader than the product needs.

### Repositories

Repositories are justified for aggregate or persisted resource access where application use cases need a stable storage abstraction.

Likely examples:

- PaymentRepository;
- PaymentEventRepository;
- WebhookDeliveryRepository;
- IdempotencyRepository.

Repositories should follow application needs rather than mirror database tables.

A repository should not be introduced for every value object or child entity.

### TransactionManager

Some use cases require atomic persistence of:

- a payment mutation;
- a generated event;
- a delivery job;
- an idempotency result.

The application therefore requires an explicit transaction boundary.

The abstraction must remain narrow.

It should not become a large generic Unit of Work exposing every repository in the system unless this proves necessary.

### WebhookSender

Outbound webhook delivery crosses a network boundary and must be replaceable in tests.

The port may represent:

- request creation;
- delivery;
- transport result;
- timeout or network failure.

Retry policy, delivery state transitions and scheduling remain application responsibilities rather than being hidden inside the HTTP adapter.

## Components that should remain concrete initially

The following components should normally remain concrete unless a later requirement justifies an abstraction.

### Application services

Use cases such as `CreatePayment` or `GetPayment` do not need interfaces solely for mocking.

Inbound adapters may depend directly on concrete application services.

### Domain services

A domain service should remain a concrete type unless several meaningful strategies exist.

### HTTP handlers

Handlers are adapters, not ports.

They do not require interfaces by default.

### Scenario parser

A YAML parser may remain concrete while there is one supported format.

The application should depend on a typed scenario representation rather than directly on YAML nodes.

If several scenario sources or formats later exist, an appropriate boundary may then be introduced.

### Validation helpers

Simple stateless validation functions should remain functions or concrete types.

### Mappers

Mapping code should remain explicit and concrete.

A generic mapper abstraction will not be introduced.

### Logging

Domain code will not depend on a logger interface.

Application and adapter layers may use structured logging directly through a project-level logging convention where appropriate.

Logging must not be required for business correctness.

## Interface ownership

Interfaces should generally be owned by the package that consumes the capability, not by the package that implements it.

For example, if the payment application requires payment persistence, the repository contract should be defined near the payment application rather than inside the SQLite adapter.

This follows the principle that the consumer defines what it needs.

Exceptions may exist for intentionally shared cross-module contracts, but they must be explicit.

## Interface size

Ports should be narrow.

A port should represent a cohesive capability and avoid exposing unrelated operations.

For example, avoid:

```text
Storage
- SavePayment
- GetPayment
- SaveRefund
- ListEvents
- DeleteAll
- ReserveDelivery
- UpdateDelivery
- HealthCheck
```

Prefer boundaries aligned with use cases or ownership.

However, this rule must not be interpreted as requiring one-method interfaces everywhere.

A repository may expose several closely related operations when they form one cohesive capability.

## Test strategy

Targeted hexagonal architecture supports multiple test levels.

### Domain tests

Domain tests use domain objects directly.

They should not require mocks.

They verify:

- invariants;
- transitions;
- monetary rules;
- domain errors.

### Application tests

Application tests may use:

- lightweight in-memory fakes;
- test-specific adapters;
- real SQLite adapters;
- deterministic clocks;
- deterministic identifier generators;
- deterministic random sources.

Mocks should be used when interaction verification is important, not as a default substitute for all dependencies.

### Adapter tests

Adapters should be tested against their actual protocol or technology.

Examples:

- SQLite repositories tested against SQLite;
- HTTP handlers tested with `httptest`;
- webhook sender tested against a local HTTP server;
- migrations tested by applying them to a real database instance.

### End-to-end tests

End-to-end tests should exercise the composed application with real adapters whenever practical.

The architecture must not optimise unit-test isolation at the expense of realistic integration tests.

## Rationale

### External effects require control

Clock access, randomness, network calls and persistence directly affect determinism and reliability.

Placing them behind explicit ports allows the product to:

- reproduce scenarios;
- simulate failures;
- test without sleeping;
- validate concurrency;
- replace infrastructure where useful;
- keep domain rules independent from runtime choices.

### Not every dependency is an architectural boundary

A project can become harder to understand when every collaborator is abstracted.

Unnecessary interfaces:

- hide concrete behaviour;
- increase navigation cost;
- create more names and files;
- encourage interaction-heavy tests;
- make refactoring more expensive;
- provide little actual decoupling.

Targeted use of ports preserves the benefits of hexagonal architecture without imposing uniform layering.

### Explicit dependencies improve maintainability

Application services should receive the capabilities they need.

This makes:

- side effects visible;
- tests controllable;
- module responsibilities clearer;
- runtime composition understandable.

### Manual wiring is appropriate for a Go modular monolith

Go constructors and explicit composition are generally sufficient for the expected application size.

Manual wiring provides:

- compile-time safety;
- transparent startup behaviour;
- simple debugging;
- no runtime container;
- no hidden lifecycle.

### Infrastructure should remain replaceable where replacement matters

SQLite may later coexist with PostgreSQL.

A system clock may coexist with a virtual clock.

A real HTTP sender may coexist with a fake or recording sender.

These are meaningful substitution points.

By contrast, replacing a simple application service implementation is not an initial requirement.

## Consequences

### Positive consequences

- Domain rules remain independent from transport and persistence.
- Time and randomness can be controlled deterministically.
- Outbound network behaviour can be simulated and tested.
- SQLite can later be complemented by another persistence adapter.
- Application use cases expose their real dependencies.
- Infrastructure changes are less likely to leak into the domain.
- The project avoids interface proliferation.
- Manual dependency wiring remains understandable.
- Tests can use the appropriate level of realism.
- Module boundaries are easier to enforce.

### Negative consequences

- Contributors must exercise judgement about when to introduce an interface.
- Architectural consistency cannot be enforced by a purely mechanical folder template.
- Some tests may use real adapters and therefore be slower than isolated mock-based tests.
- Manual wiring may become verbose as the application grows.
- Poorly designed ports may still leak infrastructure assumptions.
- Refactoring a concrete dependency into a port later may require changes.
- Different modules may have slightly different internal structures.

### Accepted trade-offs

- Some concrete dependencies may initially be harder to substitute.
- The project prefers introducing abstractions when needed over predicting every future extension.
- Not all application tests will be pure unit tests.
- Infrastructure adapters may expose technology-specific configuration internally.
- The architecture values conceptual boundaries more than visual symmetry in the repository.

## Architectural constraints

### No infrastructure dependencies in the domain

Domain packages must not import:

- SQL drivers;
- HTTP server packages;
- OpenTelemetry packages;
- configuration libraries;
- YAML parsers;
- concrete persistence adapters.

Use of standard-library primitives such as `time.Time` is acceptable.

Direct retrieval of the current wall-clock time is not.

### No speculative interfaces

An interface must have a stated purpose.

Pull requests introducing a new interface should be able to explain:

- which boundary it represents;
- why substitution matters;
- who consumes it;
- why a concrete dependency is insufficient.

“Testing” alone is not always sufficient justification when a real adapter can be tested cheaply or a concrete collaborator can be used directly.

### No generic repository abstraction

The project will not define a universal repository such as:

```text
Repository[T]
- Save(T)
- Find(ID)
- Delete(ID)
```

Payment persistence, webhook delivery reservation and idempotency have different semantics.

Their contracts should reflect those semantics.

### No infrastructure-shaped ports

Ports should not expose raw SQL concepts such as:

- rows;
- statements;
- SQL transactions;
- database-specific error codes.

Likewise, the webhook sending port should not force application code to depend directly on a specific HTTP client implementation.

### No hidden transactions

Repository methods must not silently create transaction boundaries that prevent an application use case from committing several related changes atomically.

Transaction ownership must be explicit for operations that require atomicity.

### No domain event bus by default

Domain events may be returned explicitly by domain operations or collected during application orchestration.

The project will not introduce a generic in-process event bus solely to decouple code that can interact explicitly.

## Alternatives considered

### Alternative 1: Strict hexagonal architecture everywhere

Every component could be represented by:

- a port;
- an adapter;
- an interface;
- an implementation;
- a constructor;
- a mock.

This option was rejected because it would create significant ceremony and obscure simple code.

It would also encourage developers to optimise for architectural uniformity rather than clear responsibilities.

### Alternative 2: Traditional layered architecture

The project could use layers such as:

```text
handler
service
repository
model
```

This option is easy to recognise but often leads to:

- technical rather than domain-oriented packaging;
- services with mixed responsibilities;
- persistence models leaking upward;
- weak module ownership;
- coupling through shared models.

Some layered concepts remain useful, but they will be applied within functional modules rather than as global horizontal layers.

### Alternative 3: Direct infrastructure dependencies everywhere

Application services could directly use:

- `sql.DB`;
- `http.Client`;
- `time.Now`;
- package-level randomness.

This option would reduce initial file count and abstraction.

It was rejected because it would make:

- deterministic testing harder;
- virtual time harder to introduce;
- fault injection less controlled;
- persistence replacement more invasive;
- domain and infrastructure concerns easier to mix.

### Alternative 4: Dependency injection framework

A framework could automatically resolve constructors and lifecycles.

This option was rejected initially because:

- runtime wiring is expected to remain manageable;
- hidden construction complicates debugging;
- Go code benefits from explicit composition;
- the project does not require dynamic dependency resolution.

This decision may be revisited if composition becomes materially difficult.

### Alternative 5: Mock-driven architecture

Interfaces could be introduced primarily to generate mocks for all collaborators.

This option was rejected because it tends to produce tests coupled to implementation interactions rather than observable behaviour.

The project prefers:

- domain tests without mocks;
- real adapter tests;
- lightweight fakes where useful;
- mocks only for meaningful interaction contracts.

### Alternative 6: Framework-defined architecture

A web or application framework could define handlers, services, repositories and dependency management.

This option was rejected because Payment Sandbox should keep its architecture independent from a framework and use Go's standard capabilities wherever practical.

## Rejected interpretations

This decision does not mean that:

- interfaces are discouraged;
- every port must have multiple production implementations;
- in-memory repositories should replace integration tests;
- all modules must use the same package structure;
- application services may contain infrastructure logic;
- adapters may bypass domain invariants;
- direct use of concrete types is always preferable;
- hexagonal boundaries must map one-to-one to directories.

It means that boundaries should be introduced intentionally and justified by actual responsibilities.

## Implementation guidance

A payment creation use case may conceptually depend on:

```text
CreatePayment
- transaction boundary
- payment repository
- event repository
- clock
- identifier generator
```

The HTTP handler may depend directly on the concrete `CreatePayment` application service.

The service itself may depend on narrow ports defined by the payment application package.

A conceptual flow is:

```text
HTTP request
    ↓
Provider API adapter
    ↓
CreatePayment application service
    ↓
Payment domain
    ↓
PaymentRepository port
PaymentEventRepository port
TransactionManager port
    ↓
SQLite adapters
```

### Example package ownership

```text
internal/payment/
├── domain/
│   ├── payment.go
│   ├── money.go
│   └── errors.go
│
├── application/
│   ├── create_payment.go
│   ├── get_payment.go
│   ├── payment_repository.go
│   └── transaction.go
│
└── adapters/
    └── sqlite/
        └── payment_repository.go
```

The consuming application package owns the repository contract.

The SQLite adapter implements it.

This structure is illustrative rather than mandatory.

### Example of an unjustified interface

Avoid introducing:

```text
type PaymentService interface {
    Create(...)
}
```

when:

- there is one concrete application service;
- handlers can depend on it directly;
- no meaningful alternative implementation exists;
- the interface exists only so a handler test can mock it.

The handler can instead be tested through the real application service with controlled dependencies.

### Example of a justified interface

A clock is justified because:

- production uses real time;
- tests use fixed time;
- future scenarios may use virtual time;
- deterministic scheduling depends on it.

## Evolution rules

A concrete dependency may become a port later when a real need appears.

The change should be considered when:

- a second meaningful implementation is required;
- deterministic testing cannot otherwise be achieved cleanly;
- infrastructure details begin leaking into application code;
- a module boundary becomes unclear;
- an external capability must be isolated.

Conversely, an interface should be removed when:

- it has one trivial implementation;
- it mirrors the concrete type exactly;
- it creates no meaningful boundary;
- it exists only for unused generated mocks;
- it makes the code harder to follow without improving substitution.

## Revisit when

This decision should be reconsidered when one or more of the following conditions are demonstrated:

- manual composition becomes difficult to understand or maintain;
- provider profiles require a stronger extension model;
- modules need separate release or deployment boundaries;
- several implementations of major application capabilities coexist;
- plugin support becomes a validated product requirement;
- the number of narrow ports materially harms comprehension;
- repository boundaries no longer align with persistence and transaction needs;
- tests become excessively dependent on infrastructure despite the current ports;
- architectural enforcement requires additional tooling or package rules.

Possible future changes may include:

- separate composition packages;
- generated compile-time wiring;
- stronger module-level contracts;
- dedicated test harness adapters;
- extraction of shared ports;
- selective use of a dependency injection tool.

Reconsideration does not imply adopting interfaces for all components.

## Compliance

Code reviews should verify that:

- domain code remains independent from infrastructure;
- new interfaces represent meaningful capabilities;
- interfaces are owned by their consumers where practical;
- ports remain cohesive and reasonably narrow;
- application services do not format HTTP responses;
- handlers do not access repositories directly;
- repositories do not hide incompatible transaction boundaries;
- global time and random sources are not used in deterministic logic;
- mocks are not introduced without a clear testing need;
- new package layers correspond to real responsibilities;
- dependency construction remains explicit.

Static dependency checks may later be added to CI if package boundaries become difficult to enforce through review alone.

## Outcome

Payment Sandbox will use hexagonal architecture selectively around boundaries where it improves:

- deterministic behaviour;
- testability;
- infrastructure independence;
- module ownership;
- long-term evolvability.

The project will not treat interfaces, adapters or layers as goals in themselves.

The preferred architecture is the simplest one that keeps business rules independent, external effects explicit and important dependencies controllable.
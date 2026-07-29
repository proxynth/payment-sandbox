# ADR-0011: Provider Plugin Model

- **Status:** Accepted
- **Date:** 2026-07-29
- **Deciders:** Payment Sandbox Maintainers
- **Tags:** Providers, Extensibility, Plugins

---

# Executive Summary

Payment Sandbox isolates payment-provider-specific behaviour behind a provider plugin model.

The simulator core defines the business contracts required to execute payment scenarios.

Individual payment providers implement these contracts without introducing provider-specific logic into the core domain.

This separation allows new providers to be added independently while preserving a stable and deterministic simulation platform.

The architecture therefore remains open for extension while remaining closed for modification.

---

# Context

Every payment provider exposes different capabilities.

Examples include:

- payment authorisation;
- capture semantics;
- refund behaviour;
- webhook payloads;
- retry strategies;
- asynchronous operations;
- provider-specific constraints.

Although these behaviours differ, they all represent variations of the same payment domain.

Embedding provider-specific logic directly inside the simulator core would progressively couple the domain model to individual providers.

Over time, such coupling would reduce maintainability and make the platform increasingly difficult to evolve.

---

# Problem Statement

Supporting multiple providers introduces a common architectural challenge.

Each provider exposes:

- different APIs;
- different timing rules;
- different webhook formats;
- different error models;
- different business capabilities.

Without clear architectural boundaries, these differences naturally spread throughout the codebase.

Business services gradually become filled with provider-specific conditionals.

For example:

```text
if Stripe

...

else if Adyen

...

else if Mollie

...
```

Such designs tightly couple the payment domain to implementation details.

Adding a new provider eventually requires modifications throughout the simulator.

This violates the modular architecture established by previous ADRs.

---

# Decision

Payment Sandbox adopts a provider plugin model.

The simulator core defines stable business contracts representing payment-provider capabilities.

Individual providers implement these contracts within isolated modules.

The core interacts exclusively with provider abstractions.

It never depends upon provider-specific implementations.

Provider behaviour therefore becomes an extension of the architecture rather than a modification of the core.

---

# Architectural Principles

Payment Sandbox adopts the following principles.

> **The payment domain owns the contracts.**

Provider implementations adapt themselves to the business model.

The business model does not adapt itself to providers.

---

> **Providers are replaceable.**

Adding, removing or updating a provider should not require changes to the simulator core.

---

> **Provider behaviour must remain deterministic.**

Different providers may implement different business rules.

However, identical provider configuration and identical business inputs should always produce identical observable behaviour.

---

> **Extensions must preserve architectural integrity.**

Provider modules extend the simulator.

They must not weaken the boundaries established by previous architectural decisions.

---

# Scope

This ADR applies to every payment-provider implementation integrated into Payment Sandbox.

Examples include:

- provider capabilities;
- payment operations;
- webhook generation;
- asynchronous provider behaviour;
- provider configuration;
- provider-specific validation.

Infrastructure concerns such as:

- plugin packaging;
- dynamic loading;
- dependency injection mechanisms;
- build systems;

are intentionally outside the scope of this decision.

This ADR defines architectural responsibilities rather than implementation techniques.

# Why Providers Are Extensions

The payment domain exists independently of any particular payment provider.

Business concepts such as:

- payments;
- authorisations;
- captures;
- refunds;
- cancellations;

remain meaningful regardless of whether the underlying provider is Stripe, Adyen, Mollie or another implementation.

Providers therefore implement the domain.

They do not define it.

---

# The Domain Owns the Language

The simulator defines a common business language shared by every provider.

Examples include:

- Authorize Payment
- Capture Payment
- Refund Payment
- Cancel Payment

These operations express business intent.

Each provider translates that intent into its own implementation.

The core domain therefore remains independent from provider-specific terminology.

---

# Providers Represent Variations

Although providers expose different APIs, many differences are variations of the same business concepts.

For example:

```text
Business Intent

↓

Capture Payment

↓

Stripe Implementation

↓

Stripe Capture API
```

or

```text
Business Intent

↓

Capture Payment

↓

Adyen Implementation

↓

Adyen Capture Request
```

The simulator models the common business capability.

Providers implement the variation.

---

# Provider Differences Remain Local

Different providers may implement:

- different validation rules;
- different webhook payloads;
- different retry policies;
- different asynchronous behaviour;
- different timing constraints.

These differences should remain confined to the provider module.

The simulator core should not accumulate provider-specific conditional logic.

---

# The Core Never Chooses Behaviour

The payment domain should never contain decisions such as:

```text
if Stripe

...

else if Adyen

...

else if Mollie
```

Instead:

```text
Payment Command

↓

Provider Contract

↓

Selected Provider

↓

Provider Behaviour
```

The core expresses business intent.

The provider determines how that intent is realised.

---

# Providers Participate in the Architecture

Provider implementations are not isolated libraries.

They participate fully in the architectural model established by previous ADRs.

For example, providers should:

- respect payment lifecycle rules;
- schedule durable work;
- generate business events;
- use the virtual clock;
- expose observable behaviour.

The plugin model therefore extends the architecture rather than bypassing it.

---

# Determinism Across Providers

Different providers naturally produce different business outcomes.

For example:

```text
Stripe

↓

Capture Immediately
```

versus

```text
Another Provider

↓

Capture Pending
```

These behavioural differences are expected.

However, each provider must remain internally deterministic.

Given:

- identical business commands;
- identical provider configuration;
- identical virtual time;
- identical initial state;

the provider should always produce the same observable behaviour.

---

# A Stable Core

As additional providers are introduced, the simulator core should remain largely unchanged.

Conceptually:

```text
              Simulator Core
                     │
    ┌────────────────┼────────────────┐
    ▼                ▼                ▼
 Stripe Plugin   Adyen Plugin   Mollie Plugin
```

The architecture evolves by adding new provider modules rather than modifying existing business components.

This significantly reduces the risk of regressions within the core platform.

---

# Consistent User Experience

Although providers expose different capabilities, the simulator should present a consistent experience to its users.

For example:

- scenarios are executed identically;
- replay behaves identically;
- observability follows the same principles;
- diagnostics use the same concepts.

Provider-specific behaviour exists where required.

Everything else remains consistent across the platform.

---

# A Foundation Built by Previous ADRs

The provider plugin model relies upon the architectural foundations established by earlier decisions.

- **ADR-0002** defines a provider-independent payment domain.
- **ADR-0003** isolates provider implementations through architectural boundaries.
- **ADR-0004** provides a common persistence model.
- **ADR-0005** enables providers to schedule durable asynchronous work.
- **ADR-0006** defines the canonical payment lifecycle.
- **ADR-0007** records provider behaviour through business events.
- **ADR-0008** exposes provider execution through structured diagnostics.
- **ADR-0009** reproduces provider behaviour deterministically.
- **ADR-0010** provides a shared business timeline for every provider.

Provider implementations therefore extend an existing architecture rather than introducing their own.

# Architectural Model

The provider plugin model extends the simulator through well-defined business contracts.

The simulator core remains responsible for orchestrating business execution.

Providers implement provider-specific behaviour without modifying the surrounding architecture.

Conceptually:

```text
Business Command

↓

Payment Domain

↓

Provider Contract

↓

Provider Plugin

↓

Provider Result

↓

Domain Processing

↓

Business Events

↓

Durable Jobs
```

Provider implementations therefore participate in the business workflow rather than replacing it.

---

# Stable Business Contracts

The simulator exposes stable business contracts representing provider capabilities.

Conceptually:

```text
Payment Domain

↓

Provider Contract

↓

Provider Implementation
```

Providers depend upon these contracts.

The contracts never depend upon individual providers.

This preserves a clear dependency direction throughout the architecture.

---

# Provider Selection

Before business execution begins, the simulator selects the provider associated with the current scenario.

Conceptually:

```text
Scenario

↓

Configured Provider

↓

Provider Plugin

↓

Business Execution
```

Once selected, the provider remains responsible for implementing provider-specific behaviour throughout the scenario.

Business services remain independent from provider identity.

---

# Business Workflow Remains Central

Provider plugins execute within the existing business workflow.

Conceptually:

```text
Command

↓

Aggregate

↓

Provider Plugin

↓

State Transition

↓

Business Events

↓

Durable Jobs
```

The provider contributes business behaviour.

The aggregate continues to own business state and lifecycle.

The overall execution pipeline remains unchanged.

---

# Providers Cannot Bypass the Domain

Provider implementations must never modify persisted business state directly.

Instead, providers return business results that are interpreted by the domain.

Conceptually:

```text
Provider Plugin

↓

Provider Result

↓

Payment Aggregate

↓

Validated State Transition
```

Business invariants therefore remain enforced by the aggregate rather than by individual providers.

---

# Interaction with Durable Jobs

Providers may request future work.

For example:

```text
Capture Completed

↓

Schedule Webhook

↓

Durable Job
```

However, providers do not execute asynchronous work directly.

The durable job infrastructure introduced in **ADR-0005** remains responsible for scheduling and execution.

---

# Interaction with the Virtual Clock

Provider behaviour may depend on time.

Examples include:

- retry delays;
- payment expiration;
- settlement windows;
- asynchronous callbacks.

Every provider evaluates these rules using the shared virtual clock.

Conceptually:

```text
Provider Rule

↓

Virtual Clock

↓

Business Decision
```

Temporal behaviour therefore remains deterministic across every provider.

---

# Interaction with Observability

Provider execution contributes to the simulator's observable behaviour.

Examples include:

- executed provider operations;
- generated business events;
- scheduled work;
- provider responses.

These elements become available through the common diagnostic model introduced in **ADR-0008**.

Provider implementations should expose meaningful business information without leaking internal implementation details.

---

# Replay Executes Providers Unchanged

Replay does not simulate providers differently.

The same provider implementation executes during both:

- normal execution;
- deterministic replay.

Conceptually:

```text
Recorded Scenario

↓

Provider Plugin

↓

Business Behaviour

↓

Replay

↓

Provider Plugin

↓

Business Behaviour
```

Replay therefore validates the actual provider implementation rather than a simplified replay model.

---

# Extending the Platform

Adding a new provider should primarily consist of implementing the required business contracts.

Conceptually:

```text
Existing Architecture

↓

New Provider Plugin

↓

No Core Changes
```

The architecture evolves through extension rather than modification.

Previously implemented providers remain unaffected.

---

# Interaction with Other ADRs

The provider plugin model completes the architectural layering established throughout the project.

- **ADR-0001** provides the modular structure into which providers integrate.
- **ADR-0002** defines a provider-independent payment domain.
- **ADR-0003** isolates provider implementations behind architectural boundaries.
- **ADR-0004** offers a shared persistence model.
- **ADR-0005** executes provider-requested work through durable jobs.
- **ADR-0006** preserves payment lifecycle invariants regardless of provider behavior.
- **ADR-0007** records provider outcomes as immutable business events.
- **ADR-0008** exposes provider execution through structured diagnostics.
- **ADR-0009** verifies provider behaviour through deterministic replay.
- **ADR-0010** provides the shared temporal model governing every provider.

Provider plugins therefore become interchangeable architectural extensions while preserving the deterministic behavior and integrity of the simulator.

# Alternatives Considered

Several architectural approaches were evaluated before adopting a provider plugin model.

---

## Provider Logic Embedded in the Core

The most straightforward approach would consist of implementing every provider directly within the simulator core.

For example:

```text
Payment Service

↓

if Stripe

...

else if Adyen

...

else if Mollie
```

While initially simple, this approach causes the core domain to accumulate provider-specific knowledge.

Adding a new provider eventually requires modifying existing business components.

This reduces maintainability and violates the architectural boundaries established throughout the project.

---

## Provider-Specific Domains

Another possibility would be modelling each provider as an independent domain.

For example:

```text
Stripe Domain

Adyen Domain

Mollie Domain
```

Although this isolates implementations, it duplicates common payment concepts.

The simulator would lose its unified business model.

Replay, observability and diagnostics would also become inconsistent across providers.

Payment Sandbox instead defines one payment domain shared by every provider.

---

## Generic Provider Abstraction

A single generic provider interface could expose every possible operation.

Conceptually:

```text
Payment Provider

↓

Everything
```

As additional providers are introduced, such abstractions tend to grow continuously.

Providers supporting only a subset of capabilities become increasingly difficult to model cleanly.

The architecture therefore favours business-oriented contracts that can evolve while preserving clear boundaries.

---

## Provider-Owned Business Logic

Providers could be responsible for managing payment state directly.

For example:

```text
Provider

↓

Update Payment

↓

Persist Changes
```

This would duplicate business rules across provider implementations.

The simulator core would gradually lose ownership of payment lifecycle invariants.

Payment Sandbox instead keeps business decisions inside the domain model.

Providers contribute provider-specific behaviour.

The domain remains responsible for business consistency.

---

# Accepted Trade-offs

The provider plugin model intentionally introduces several architectural constraints.

---

## Additional Architectural Boundaries

Provider implementations must interact through stable contracts.

This introduces an additional abstraction layer.

However, the resulting separation significantly improves maintainability and long-term evolution.

---

## More Components

Each provider becomes an independent architectural module.

This increases the overall number of components within the system.

The resulting modularity makes provider evolution considerably safer.

---

## Contract Evolution Requires Care

Changes to provider contracts may affect multiple implementations.

Contract evolution therefore requires careful versioning and compatibility management.

Stable contracts reduce unnecessary changes across providers.

---

## Shared Architectural Rules

Providers are not completely autonomous.

They must respect:

- payment lifecycle rules;
- durable job scheduling;
- deterministic execution;
- virtual time;
- structured observability.

This slightly limits implementation freedom while preserving architectural consistency.

---

# Implementation Guidelines

The following architectural rules apply.

---

## The Core Owns Business Decisions

Provider implementations should never enforce payment lifecycle rules independently.

Business decisions remain the responsibility of the payment domain.

---

## Providers Translate Business Intent

Providers receive business intent.

They translate that intent into provider-specific behaviour.

They should avoid exposing provider-specific concepts throughout the simulator.

---

## Providers Should Be Replaceable

Adding or removing a provider should not require modifications to existing provider implementations.

Provider modules should remain independent from one another.

---

## Provider Behaviour Must Be Deterministic

Given identical:

- business commands;
- provider configuration;
- virtual time;
- initial business state;

provider execution should always produce identical observable behaviour.

---

## Providers Must Respect Platform Services

Providers should integrate with existing architectural capabilities.

Examples include:

- durable jobs;
- event logging;
- replay;
- diagnostics;
- virtual time.

Provider implementations should extend these capabilities rather than replacing them.

---

## Business Results Should Be Explicit

Provider interactions should produce explicit business results.

The payment domain remains responsible for interpreting these results and applying business state transitions.

---

# What This Decision Is Not

This ADR does not require:

- dynamic plugin loading;
- Go runtime plugins;
- WebAssembly modules;
- dependency injection frameworks;
- runtime module discovery;
- microservices per provider;
- remote provider execution.

The provider plugin model defines architectural boundaries.

It does not prescribe a specific implementation mechanism.

---

# Revisit Conditions

This ADR should be revisited if:

- provider contracts become excessively broad;
- multiple independent payment domains emerge;
- providers require capabilities incompatible with the existing business model;
- runtime extensibility becomes a primary architectural requirement;
- future architectural evolution requires finer-grained capability contracts.

Future changes should preserve the principle that providers extend the simulator without modifying its core architecture.

---

# Consequences

## Positive

- New providers can be introduced with minimal impact on the simulator core.
- Provider implementations remain isolated.
- Business rules stay centralised within the payment domain.
- Replay behaves consistently across providers.
- Observability remains uniform.
- Long-term maintainability improves as the number of providers grows.

## Negative

- Provider implementations must respect common architectural contracts.
- Additional abstractions slightly increase implementation complexity.
- Contract evolution requires careful coordination.
- Providers cannot freely bypass architectural services provided by the simulator.

These trade-offs are accepted because architectural consistency is considered more valuable than unrestricted implementation flexibility.

---

# Decision Summary

Payment Sandbox extends its capabilities through provider plugins implementing stable business contracts.

The simulator core remains responsible for payment lifecycle management, persistence, replay, observability and deterministic execution.

Provider implementations contribute provider-specific behaviour while preserving the architectural guarantees established throughout the platform.

The provider plugin model therefore enables long-term extensibility without compromising the integrity, determinism or maintainability of the simulator.

---

# References

- Eric Evans — *Domain-Driven Design*
- Robert C. Martin — *Clean Architecture*
- Martin Fowler — *Patterns of Enterprise Application Architecture*
- Martin Fowler — *Inversion of Control Containers and the Dependency Injection Pattern*
- ADR-0001 — Modular Monolith
- ADR-0002 — Domain Model & Bounded Contexts
- ADR-0003 — Targeted Hexagonal Architecture
- ADR-0005 — Persist Asynchronous Work
- ADR-0006 — Payment State Machine
- ADR-0008 — Observability & Diagnostics
- ADR-0009 — Deterministic Scenario Replay
- ADR-0010 — Virtual Clock
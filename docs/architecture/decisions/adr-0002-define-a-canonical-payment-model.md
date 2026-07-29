# ADR-0002: Define a canonical payment model

- Status: Accepted
- Date: 2026-07-28
- Deciders: Project maintainers
- Technical area: Domain architecture
- Related decisions:
  - ADR-0001: Use a modular monolith
  - ADR-0003: Use targeted hexagonal architecture
  - ADR-0006: Separate domain outcome from client observation

## Context

Payment Sandbox is designed to simulate payment service providers for backend development, integration testing and resilience testing.

Payment service providers expose different concepts, terminology, resource models and state machines.

For example, providers may differ on:

- whether authorization and payment are represented by the same resource;
- whether capture is an action, a separate resource or an implicit operation;
- whether multiple partial captures are supported;
- how cancellation and authorization reversal are distinguished;
- whether refunds belong directly to a payment or to a captured transaction;
- how asynchronous payment methods are represented;
- how provider-side processing states are exposed;
- how events and webhook payloads are structured;
- how idempotency is scoped;
- how business failures are represented;
- how provider references and merchant references are named.

Attempting to reproduce one existing provider exactly would make the project immediately understandable to users of that provider, but would strongly couple the product to that provider's public API and domain model.

Attempting to support several providers directly in the core would instead risk producing a model composed of:

- optional fields;
- provider-specific flags;
- ambiguous status names;
- weak invariants;
- conditional behaviour;
- abstractions representing the lowest common denominator.

Payment Sandbox also needs to simulate behaviours that may not be directly expressible in an official provider sandbox, such as:

- committing a payment operation while losing the HTTP response;
- exposing a temporarily stale resource state;
- producing deliberately inconsistent observations;
- duplicating or reordering webhook deliveries;
- applying deterministic fault scenarios across multiple operations.

The project therefore needs an internal payment model that is:

- independent from a specific provider;
- precise enough to enforce meaningful payment invariants;
- limited enough to remain understandable;
- extensible without pretending to cover every payment system;
- suitable for deterministic simulation;
- translatable into provider-specific APIs later.

## Decision

Payment Sandbox will define and own a **canonical payment model**.

The canonical model will represent the payment concepts required by the product independently from any specific payment service provider.

The core domain will not directly reuse:

- Stripe resource models;
- Adyen resource models;
- PayPal resource models;
- provider-specific status names;
- provider-specific request or response payloads;
- provider-specific webhook schemas.

Provider compatibility may later be introduced through adapters or provider profiles that translate between an external provider-shaped contract and the canonical model.

The canonical model is not intended to be a universal payment standard.

It is a deliberately constrained domain model designed to support the use cases of Payment Sandbox.

## Initial domain concepts

The initial canonical model will be built around the following concepts:

- Payment;
- Capture;
- Refund;
- PaymentEvent;
- WebhookDelivery;
- IdempotencyRecord.

Additional concepts may be introduced only when required by supported use cases.

### Payment

A `Payment` represents the merchant's intent to collect a specific amount in a specific currency.

A payment owns the main financial and lifecycle invariants of the initial domain.

A payment is expected to contain at least:

- a unique sandbox identifier;
- an optional merchant reference;
- an amount;
- a currency;
- its current lifecycle state;
- the total authorized amount;
- the total captured amount;
- the total refunded amount;
- creation and update timestamps;
- optional metadata;
- a concurrency or version value if required by persistence.

A payment must not depend on an external provider identifier in order to exist.

Provider-specific identifiers may be attached by an adapter or profile but are not the identity of the canonical payment.

### Capture

A `Capture` represents a request to collect all or part of an authorized amount.

A capture should be modelled explicitly rather than only by changing a status on the payment because it may need its own:

- identifier;
- amount;
- creation time;
- status;
- idempotency semantics;
- failure details;
- audit history.

The initial implementation may support only a single full capture if that is sufficient for the first milestone.

The canonical model must nevertheless avoid making future partial or multiple captures impossible.

### Refund

A `Refund` represents the return of all or part of a captured amount.

A refund has its own identity and lifecycle.

It should contain at least:

- a refund identifier;
- the related payment identifier;
- the refunded amount;
- its state;
- creation and update timestamps;
- an optional merchant reference;
- failure information when applicable.

A refund must not reduce the total captured amount.

Instead, the payment should track captured and refunded amounts separately.

### PaymentEvent

A `PaymentEvent` represents a domain-relevant fact that occurred in the sandbox.

Examples include:

- `payment.created`;
- `payment.authorized`;
- `payment.capture_succeeded`;
- `payment.capture_failed`;
- `payment.cancelled`;
- `payment.refund_succeeded`.

The final event vocabulary will be defined alongside the domain lifecycle.

A payment event is not the same thing as a webhook delivery.

One event may result in:

- no webhook delivery;
- one delivery;
- multiple duplicate deliveries;
- several retry attempts;
- deliveries to several destinations.

### WebhookDelivery

A `WebhookDelivery` represents one scheduled or attempted delivery of an event to a destination.

It belongs to the webhook delivery domain, not to the payment aggregate.

A delivery may contain:

- its own identifier;
- the related event identifier;
- the destination;
- the scheduled delivery time;
- its delivery state;
- the attempt count;
- the last HTTP response;
- the last transport error;
- the signature mode;
- timestamps.

This distinction is required to model at-least-once delivery and repeated delivery attempts accurately.

### IdempotencyRecord

An `IdempotencyRecord` represents the association between:

- an idempotency scope;
- an idempotency key;
- a request fingerprint;
- an operation;
- a stored result or operation reference.

The idempotency mechanism is related to application processing and concurrency control.

It should not be embedded as an incidental field inside the payment model.

## Canonical money representation

Monetary amounts will be represented as integer minor units together with an explicit currency.

Examples:

```text
1999 EUR
```

represents 19,99 € when the currency uses two decimal places.

The core model must not use binary floating-point numbers for monetary values.

A money value must contain:

- an integer amount;
- an ISO 4217 currency code.

The initial model will not automatically perform currency conversion.

Operations combining amounts must require matching currencies.

The model may later need explicit currency metadata for currencies with:

- zero decimal places;
- two decimal places;
- three decimal places.

Currency exponent handling belongs to validation and presentation concerns. The persisted financial amount remains an integer.

## Initial payment lifecycle

The initial canonical lifecycle is expected to include a limited set of states such as:

```text
created
authorized
partially_captured
captured
cancelled
failed
partially_refunded
refunded
```

This list is indicative and must be refined in a dedicated domain decision or lifecycle specification before implementation.

The lifecycle must distinguish between:

- the payment's business state;
- individual capture states;
- individual refund states;
- external observations exposed by a scenario.

A payment lifecycle state must not encode transport-level outcomes.

For example, an HTTP timeout must not become a payment status.

## Domain invariants

The canonical model will enforce explicit invariants.

Initial invariants are expected to include:

- a payment amount must be strictly positive;
- a payment currency must be present and valid;
- captured amount must never exceed authorized amount;
- refunded amount must never exceed captured amount;
- amounts involved in one payment must use the same currency;
- a cancelled payment cannot be captured;
- a fully refunded payment cannot be refunded again;
- a failed payment cannot be captured unless a specific future workflow explicitly permits recovery;
- the same business transition must not be applied twice unintentionally;
- concurrent operations must not violate monetary totals.

The exact transition rules will be documented and tested as the lifecycle is implemented.

## Separation from API representations

The canonical domain model will not be used directly as the HTTP request or response model.

Provider API adapters will map between:

- external request DTOs;
- application commands and queries;
- domain entities or value objects;
- external response DTOs.

This separation allows:

- the canonical model to preserve domain semantics;
- APIs to evolve independently;
- provider profiles to expose different representations;
- administrative APIs to expose richer diagnostic data;
- transport-specific fields to remain outside the domain.

The project must avoid automatic serialization of internal domain entities as public API responses.

## Separation from provider profiles

Provider profiles may later expose contracts resembling existing payment providers.

A provider profile may translate:

```text
provider-shaped request
        ↓
canonical application command
        ↓
canonical payment model
        ↓
provider-shaped response or event
```

A profile may adapt:

- route names;
- field names;
- status names;
- error formats;
- event names;
- signature formats;
- idempotency headers;
- resource nesting.

A profile must not require provider-specific conditions to spread throughout the core domain.

When a provider behaviour cannot be represented faithfully by the canonical model, the project must choose explicitly between:

- extending the canonical model because the concept is broadly useful;
- implementing the behaviour in the provider profile;
- documenting that the behaviour is unsupported;
- creating a specialised domain module if the workflow is fundamentally different.

The core must not be weakened solely to claim compatibility.

## Rationale

### A provider-independent core preserves product identity

Payment Sandbox aims to become a general-purpose payment integration testing tool.

Its long-term value depends on its ability to model payment-system behaviour rather than one provider's API design.

Owning a canonical model allows the project to define consistent semantics for:

- lifecycle transitions;
- monetary invariants;
- ambiguity;
- events;
- retries;
- concurrency;
- inspection.

### A canonical model enables deterministic scenarios

Scenario effects need stable domain concepts.

A scenario should be able to request operations such as:

```text
authorize payment
persist capture
schedule refund event
expose stale payment view
```

These operations should not depend on whether one provider calls the resource a payment intent, a transaction, an order or a payment session.

### It avoids accidental vendor lock-in

Starting from a cloned API would make later generalisation costly.

Concepts specific to the first provider would likely leak into:

- persistence;
- event names;
- test fixtures;
- scenario rules;
- application services;
- public documentation.

A canonical model creates a deliberate boundary before those assumptions become structural.

### It supports multiple API surfaces

The same canonical model may support:

- the default Payment Sandbox API;
- an administration API;
- a Stripe-inspired profile;
- an Adyen-inspired profile;
- test helper libraries;
- a future user interface.

These surfaces may expose different representations while sharing the same business rules where appropriate.

### It creates a clear place for payment expertise

A simple HTTP mock server does not need a rich domain model.

Payment Sandbox does because part of its purpose is to test whether integrations respect payment invariants.

The canonical model provides a focused area where domain-driven design brings concrete value.

## Consequences

### Positive consequences

- Core payment rules remain independent from specific providers.
- Domain terminology is consistent throughout the core.
- Provider profiles can be added without replacing the payment engine.
- Scenarios can target stable canonical operations.
- Monetary and lifecycle invariants can be tested centrally.
- The project can expose more than one API representation.
- The domain can evolve based on product needs rather than vendor changes.
- Event and webhook delivery semantics can remain consistent across profiles.
- Contributors can reason about one internal model before learning provider adapters.

### Negative consequences

- Users familiar with a specific PSP must learn the canonical API.
- Provider-specific features may not map perfectly to the core model.
- Translation layers add implementation work.
- Some concepts may be duplicated between canonical and provider-specific representations.
- The project must maintain its own terminology and documentation.
- Poorly chosen canonical abstractions could become difficult to evolve.
- The model may be criticised for not matching a particular real-world provider exactly.
- Adding support for fundamentally different payment methods may require new models rather than small extensions.

### Accepted trade-offs

- Complete compatibility with every PSP is not an objective.
- The canonical model will intentionally omit many payment-industry concepts initially.
- Provider profiles may support only a documented subset of a provider API.
- Some simulated inconsistencies may exist only at the observation layer and not in canonical persisted state.
- The model may evolve through breaking changes before the first stable release.

## Architectural constraints

### The canonical model must remain explicit

The core must not become a collection of loosely typed maps such as:

```text
map[string]any
```

for domain-relevant data.

Extensible metadata may use a flexible representation, but financial values, lifecycle states and identifiers must remain typed.

### Provider terminology must not leak into the core

Core packages must not introduce provider-specific names unless the concept has become an intentional canonical term.

Examples of terminology that must not be adopted accidentally include:

- provider product names;
- provider-specific resource names;
- provider-specific event names;
- provider-specific error codes.

### The canonical model must not become a union of all providers

Fields must not be added merely because one provider exposes them.

A new concept belongs in the canonical model only when at least one of the following is true:

- it is required by the default sandbox use cases;
- it represents a meaningful payment-domain concept;
- it is needed by several provider profiles;
- it is necessary to preserve an important invariant;
- it enables a relevant resilience scenario.

### Domain entities must protect their invariants

State must not be mutated through unrestricted setters.

Business operations should be represented through explicit methods or domain services such as:

- authorize;
- capture;
- cancel;
- refund.

The exact implementation style may remain pragmatic, but invalid states must not be easy to construct.

### Observation faults must not corrupt canonical truth by default

A scenario may expose:

- a stale status;
- an invalid response;
- a misleading webhook;
- a temporary inconsistency.

These observations should normally be produced by the simulation or adapter layer without corrupting the canonical domain state.

The scenario engine may deliberately mutate domain state only through valid or explicitly modelled domain operations.

## Alternatives considered

### Alternative 1: Clone Stripe's model

The project could use concepts such as PaymentIntent and reproduce Stripe's endpoints and lifecycle.

This option would provide:

- immediate familiarity;
- many existing integration examples;
- a clear reference implementation;
- a large potential audience.

It was rejected as the core model because:

- it would couple the project to Stripe terminology;
- it would make other provider profiles harder to support;
- Stripe's product model is not a universal payment model;
- the project would risk being perceived as an incomplete Stripe emulator;
- provider changes could indirectly force core changes;
- legal and maintenance expectations around compatibility would become higher.

A Stripe-inspired provider profile may still be added later.

### Alternative 2: Clone Adyen's model

This option presents the same structural problems as cloning Stripe.

It would also introduce concepts and workflows strongly shaped by Adyen's platform and API history.

It was rejected for the same reasons.

### Alternative 3: Use the lowest common denominator

The model could include only:

- an identifier;
- an amount;
- a status;
- a refund flag.

This option would be simple but would not support the project's intended value.

It would make it difficult to model:

- authorization versus capture;
- partial operations;
- monetary invariants;
- idempotent retries;
- concurrent updates;
- detailed events.

It was rejected because it would reduce Payment Sandbox to a stateful HTTP stub.

### Alternative 4: Build a fully generic workflow engine

The system could represent every payment flow as arbitrary states and transitions configured by users.

This option could support many provider models without changing the core.

It was rejected because:

- users would need to define payment semantics themselves;
- the system could not enforce meaningful payment invariants;
- scenario files would become complex workflow programs;
- documentation and interoperability would suffer;
- the product would lose its payment-native identity.

A controlled scenario engine may configure outcomes, but it will operate on known canonical concepts.

### Alternative 5: Model every payment method from the beginning

The canonical model could immediately include:

- cards;
- direct debit;
- bank transfer;
- wallets;
- asynchronous vouchers;
- mandates;
- recurring payments;
- disputes;
- payouts.

This option was rejected because these workflows have materially different state machines and guarantees.

Trying to unify them prematurely would produce a vague and highly conditional model.

The initial model will focus on a limited authorization, capture, cancellation and refund workflow.

### Alternative 6: No domain model

The sandbox could store configured request-response mappings and generate webhooks from templates.

This option would minimise implementation effort.

It was rejected because it would not allow the project to:

- protect financial invariants;
- test concurrent payment operations;
- reason about valid transitions;
- simulate ambiguous results meaningfully;
- differentiate itself from general-purpose mock servers.

## Rejected interpretations

This decision does not mean that:

- every payment provider must map perfectly to the canonical model;
- the canonical model is an industry standard;
- every domain concept requires an aggregate;
- every resource requires its own repository;
- all payment methods must share one lifecycle;
- provider profiles are forbidden from maintaining additional state;
- the public default API must expose internal entities directly;
- the model must be designed completely before implementation begins.

It means that Payment Sandbox owns a stable internal vocabulary and uses it as the foundation for its default behaviour.

## Implementation guidance

The payment module may initially contain concepts such as:

```text
internal/payment/
├── domain/
│   ├── payment.go
│   ├── payment_state.go
│   ├── money.go
│   ├── capture.go
│   ├── refund.go
│   └── errors.go
│
├── application/
│   ├── create_payment.go
│   ├── get_payment.go
│   ├── capture_payment.go
│   ├── cancel_payment.go
│   └── refund_payment.go
│
└── adapters/
    └── persistence/
```

This structure is illustrative.

Files and packages should be introduced based on actual responsibilities rather than to reproduce a theoretical architecture.

### Example canonical payment

A canonical payment may be represented conceptually as:

```text
Payment
- ID: pay_01...
- MerchantReference: order_123
- Amount: 4999 EUR
- State: authorized
- AuthorizedAmount: 4999 EUR
- CapturedAmount: 0 EUR
- RefundedAmount: 0 EUR
- CreatedAt: ...
- UpdatedAt: ...
```

The exact data structure remains an implementation concern.

### Example provider mapping

A provider profile could map an external request:

```json
{
  "amount": {
    "value": 4999,
    "currency": "EUR"
  },
  "reference": "order_123"
}
```

to a canonical command:

```text
CreatePayment
- amount: 4999 EUR
- merchant reference: order_123
```

The canonical result could then be mapped back to the provider-shaped response.

## Model evolution

Changes to the canonical model must be evaluated according to:

- impact on domain invariants;
- impact on persisted data;
- impact on the default API;
- impact on the scenario DSL;
- impact on provider profiles;
- migration requirements;
- backward compatibility expectations.

Before the first stable release, breaking model changes are acceptable when they improve conceptual integrity.

After the first stable release, changes to public APIs and scenario formats must follow the project's compatibility policy.

Internal canonical model changes may still occur, provided adapters preserve supported public contracts.

## Revisit when

This decision should be reconsidered when one or more of the following conditions are demonstrated:

- the canonical model cannot represent several important provider workflows without pervasive exceptions;
- provider profiles require frequent core changes for provider-specific concepts;
- asynchronous payment methods become a primary project use case;
- subscription billing becomes a core rather than peripheral feature;
- the default API and provider profiles no longer share meaningful domain behaviour;
- the canonical model becomes dominated by optional or mutually exclusive fields;
- several distinct payment lifecycles require separate bounded contexts;
- a recognised external payment standard becomes appropriate for the project's needs.

Reconsidering this ADR may lead to:

- splitting card-like and asynchronous payment models;
- defining separate bounded contexts;
- introducing specialised canonical models;
- narrowing the scope of the default payment model;
- adopting an external standard for a specific integration surface.

It should not automatically lead to replacing the core with a provider clone.

## Compliance

Code reviews should verify that new payment functionality:

- uses canonical terminology in the core;
- does not expose domain entities directly as public DTOs;
- protects monetary and lifecycle invariants;
- does not introduce provider-specific fields without explicit justification;
- keeps events separate from webhook deliveries;
- keeps transport failures separate from payment states;
- documents any new canonical concept;
- includes transition and invariant tests;
- updates this ADR or supersedes it when its assumptions no longer hold.

## Outcome

Payment Sandbox will use a canonical payment model because it provides the best balance between:

- payment-domain correctness;
- provider independence;
- scenario expressiveness;
- testability;
- future provider compatibility;
- long-term product identity.

The model will be intentionally limited and will evolve from concrete use cases.

Payment Sandbox will not attempt to define a universal payment ontology, nor will it make a single provider's API the foundation of its core.
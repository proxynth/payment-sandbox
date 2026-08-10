# Payment Sandbox

Payment Sandbox is an open-source payment systems simulator written in Go.

It helps backend developers test payment integrations against realistic business workflows, asynchronous events and distributed-system failures without depending on a real payment service provider.

> Payment Sandbox is not a payment service provider and does not process real money.

## Why Payment Sandbox?

Traditional HTTP mock servers can return predefined responses, inject latency or simulate network failures. Payment integrations require more than HTTP mocking.

A payment may succeed even when the client receives a timeout. A webhook may arrive multiple times, late, out of order or before a resource becomes visible through the API. Concurrent capture or refund requests may race against each other.

Payment Sandbox is designed to make these situations reproducible and testable.

## Goals

Payment Sandbox aims to provide:

* a canonical payment API for integration testing;
* realistic payment lifecycle simulation;
* deterministic and reproducible scenarios;
* configurable API and network failures;
* asynchronous webhook delivery;
* idempotency and concurrency testing;
* observable execution timelines;
* a local-first and CI-friendly developer experience.

## Architecture

Payment Sandbox is designed as a deterministic payment simulation platform.

Its architecture is documented through a series of Architecture Decision Records (ADRs) covering:

- modular monolith architecture;
- domain modelling;
- targeted hexagonal architecture;
- durable asynchronous work;
- deterministic payment lifecycle;
- immutable business event history;
- observability;
- deterministic replay;
- virtual time;
- provider extensibility.

See [docs/architecture](docs/architecture).

## Example use cases

Payment Sandbox can be used to test situations such as:

* a payment is authorized successfully;
* a payment is declined for a business reason;
* a payment is created but the HTTP response is lost;
* the same request is retried with an idempotency key;
* two capture requests are sent concurrently;
* a refund succeeds asynchronously;
* a webhook is delayed;
* a webhook is delivered more than once;
* webhooks are delivered out of order;
* a webhook contains an invalid signature;
* the API temporarily returns stale or inconsistent state.

## Development

Common development commands are exposed through the Makefile.

```bash
make help
```

Typical workflow:

```bash
make fmt
make test
make lint
make check
```

Build and run the application:

```
make build
make run
```

### Logging

Payment Sandbox uses structured logging through Go's `log/slog`.

Default configuration:

```dotenv
PAYMENT_SANDBOX_LOG_LEVEL=info
PAYMENT_SANDBOX_LOG_FORMAT=text
```

JSON output can be enabled with:

```bash
PAYMENT_SANDBOX_LOG_FORMAT=json make run
```

### Testing

Run the complete test suite:

```bash
go test ./...
```

Run tests with the Go race detector:

```bash
go test -race ./...
```

Generate a coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

#### Testing conventions

- Tests should be deterministic.
- Unit tests should live next to the code they exercise.
- Prefer table-driven tests when several inputs exercise the same behaviour.
- Tests should not depend on external services.
- Avoid real sleeps for time-dependent behaviour.
- Prefer observable behaviour over implementation-detail assertions.

## Lint

```bash
golangci-lint run
```

Check formatting without modifying files:

```bash
golangci-lint fmt --diff
```

## Core design principles

### Deterministic by default

Failures must be reproducible.

Randomized fault injection may be supported, but every randomized execution must be associated with a known seed and an inspectable decision log.

### Payment-native

Payment Sandbox models payment concepts and invariants instead of serving only static HTTP fixtures.

### Local-first

The sandbox should be easy to run locally, in Docker and in automated test pipelines.

### Observable by design

Every request, domain transition, generated event and webhook delivery attempt should be inspectable.

### Pragmatic architecture

The project uses domain-driven design, hexagonal architecture and distributed-systems patterns only where they provide concrete value.

It deliberately avoids unnecessary microservices, framework-heavy abstractions and interfaces without meaningful substitution points.

## Initial domain scope

The initial scope is intentionally limited to:

* payments;
* authorizations;
* captures;
* cancellations;
* refunds;
* payment events;
* webhook delivery attempts;
* idempotency.

Future versions may introduce provider-specific profiles, additional payment methods, disputes, subscriptions and record/replay capabilities.

## Planned architecture

Payment Sandbox is designed as a modular monolith.

The main modules are expected to include:

* Payment;
* Simulation;
* Webhook;
* Idempotency;
* Scheduler;
* Administration;
* Observability.

External concerns such as persistence, clocks, random generation and outbound HTTP delivery are isolated behind explicit architectural boundaries when substitution provides real value.

## Scenario model

Scenarios will describe independently:

* what happens in the payment domain;
* what the API client observes;
* what asynchronous events are generated;
* how and when webhooks are delivered;
* whether temporary inconsistencies are exposed.

A scenario may eventually look like this:

```yaml
version: "1"

seed: 42817

rules:
  - name: payment-created-but-response-lost
    
    match:
      operation: payment.create
      attempt: 1
  
    effects:
      domain:
        result: authorized
  
      response:
        fault: connection_reset
  
      webhooks:
        - event: payment.authorized
          delay: 5s
          duplicate: 1
```

The scenario format is not stable yet.

## Schema

```
API

↓

Payment Domain

↓

Provider Plugin

↓

Business Events

↓

Durable Jobs

↓

Virtual Clock

↓

Replay / Diagnostics
```

## Project status

Payment Sandbox is currently in its design and early implementation phase.

The first milestone is a vertical slice covering:

1. payment creation;
2. persistence;
3. event creation;
4. webhook scheduling;
5. webhook delivery;
6. administrative inspection;
7. structured logs and health endpoints.

No stable API or scenario compatibility is guaranteed before the first tagged release.

## Non-goals

Payment Sandbox is not intended to:

* process real payments;
* store real cardholder data;
* replace official provider sandboxes;
* guarantee complete compatibility with any payment provider;
* simulate an entire banking or acquiring infrastructure;
* require a distributed deployment to model distributed-system behaviour.

## Security

Payment Sandbox must never be exposed publicly with unrestricted webhook destinations or administrative endpoints.

Security-sensitive behaviours, including outbound webhook requests and access to private network ranges, will be documented and restricted as the project evolves.

Do not use real payment credentials, cardholder data or production secrets.

## Further reading

- Architecture overview
- ADR Index
- Design Principles

## Contributing

The contribution model will be documented once the initial architecture and public contracts have stabilized.

Early discussions should focus on:

* payment-domain invariants;
* scenario reproducibility;
* failure semantics;
* webhook delivery guarantees;
* developer experience;
* explicit architectural trade-offs.

## License

The project license has not yet been selected.
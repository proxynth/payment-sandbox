# Payment Sandbox

Payment Sandbox is a payment systems simulator written in Go.

[![Latest release](https://img.shields.io/github/v/release/proxynth/payment-sandbox?sort=semver)](https://github.com/proxynth/payment-sandbox/releases/latest)

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

## Architecture overview

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

The provider model is described in the [provider guide](docs/providers.md).
The replay workflow is described in the [scenario guide](docs/scenarios.md).

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

Choose the path that matches your goal:

* **Use Payment Sandbox:** follow the [user installation guide](docs/installation.md)
  to download the latest release without cloning the repository or installing
  Go. This is the recommended path for trying the product.
* **Explore or contribute to the code:** follow the [Getting Started
  guide](docs/getting-started.md) to clone, configure and validate a source
  checkout.

Both paths lead to the same HTTP API and first-payment walkthrough.

See the [Contributing guide](CONTRIBUTING.md) for module boundaries, testing,
commits and pull requests.

Maintainers can find the release procedure in the [Releases guide](docs/releases.md).

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

The application loads configuration from environment variables during startup.
When running from a source checkout, `make run` automatically loads `.env` if
it exists. Published binaries require the variables to be exported by the
shell that starts them.

### Try it from source

If you have cloned the repository, start the sandbox from its root:

```bash
make run
```

In another terminal, check that it is ready, create a payment and authorize it:

```bash
curl http://127.0.0.1:8080/health/ready

curl -X POST http://127.0.0.1:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"id":"demo-payment","amount":1000,"currency":"EUR"}'

curl -X POST http://127.0.0.1:8080/payments/demo-payment/authorize
curl http://127.0.0.1:8080/payments/demo-payment
```

This demonstrates the basic payment lifecycle without requiring a provider
account or a separate application. Continue with the
[complete installation and first-payment walkthrough](docs/installation.md#make-a-first-payment-request)
to explore webhooks and the local database.

If you are using a published binary, follow the same scenario from the
[release installation guide](docs/installation.md#make-a-first-payment-request)
after downloading and starting the latest release.

### Configuration

Payment Sandbox is configured through environment variables.

Available variables:

| Variable | Default | Values |
|---|---|---|
| `PAYMENT_SANDBOX_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `PAYMENT_SANDBOX_LOG_FORMAT` | `text` | `text`, `json` |
| `PAYMENT_SANDBOX_HTTP_ADDRESS` | `:8080` | HTTP listen address |
| `PAYMENT_SANDBOX_ADMIN_TOKEN` | none | Bearer token required for `/admin/*` routes |

Example:

```bash
PAYMENT_SANDBOX_LOG_LEVEL=debug \
PAYMENT_SANDBOX_LOG_FORMAT=json \
make run
```

See .env.example for the currently supported configuration.

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

Future versions may introduce additional payment methods, disputes and subscriptions.

## Architecture

Payment Sandbox is implemented as a modular monolith. The current runtime is
organized around these modules:

* Payment and its state machine;
* Provider profiles and deterministic execution;
* Durable scheduling and asynchronous work;
* Saga orchestration;
* Webhook registration and delivery;
* Scenario replay and diagnostics;
* Administration and platform concerns.

External concerns such as persistence, clocks, random generation and outbound HTTP delivery are isolated behind explicit architectural boundaries when substitution provides real value.

## Scenario model

The current deterministic scenario and replay model is documented in the
[scenario guide](docs/scenarios.md). The scenario format is not yet a stable,
versioned standalone file format.

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

Payment Sandbox is under active development. Published versions are listed on
the [GitHub Releases page](https://github.com/proxynth/payment-sandbox/releases).
The latest release can always be found at
[/releases/latest](https://github.com/proxynth/payment-sandbox/releases/latest).

### Implemented

The current release provides the following capabilities for local and
integration testing:

- canonical payment lifecycle API;
- durable SQLite-backed scheduler and asynchronous jobs;
- virtual business time with wall-clock lease recovery;
- deterministic fake, Stripe and Adyen provider profiles;
- immutable payment event history and diagnostics;
- webhook endpoint registration and outbound delivery;
- deterministic scenario replay;
- Saga orchestration for provider-backed payment workflows.

### Currently evolving

- the HTTP API and its public compatibility guarantees;
- the provider profile and failure-injection model;
- the scenario model and replay ergonomics;
- operational documentation and observability details.

Compatibility guarantees are defined by the documentation for each published
release rather than by the major version alone.

### Planned

The following areas are intentionally outside the current stable contract:

- a versioned, standalone scenario file format;
- broader provider behaviour profiles and integration examples;
- additional public API capabilities built on the existing domain model.

Planned items are design directions, not promises for a specific release.

For a local build, run `go run ./cmd/payment-sandbox --version`; published
release binaries report the version injected from their release tag.

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

Webhook registration rejects local and private IP literals. Runtime outbound
delivery resolves DNS at connection time, rejects private, loopback, link-local
and unspecified addresses, does not use an HTTP proxy, and does not follow
redirects. Administrative endpoints require the configured
`PAYMENT_SANDBOX_ADMIN_TOKEN` as a bearer token, but should still be kept on a
trusted local or private network and protected by an upstream network policy.

Do not use real payment credentials, cardholder data or production secrets.

## Further reading

- [Architecture overview](docs/architecture/README.md)
- [Architecture diagrams](docs/architecture/diagrams.md)
- [Architecture principles](docs/architecture/principles.md)
- [Architectural Decision Records](docs/architecture/decisions/)
- [Getting Started guide](docs/getting-started.md)
- [Installation and first-payment walkthrough](docs/installation.md#make-a-first-payment-request)
- [Provider guide](docs/providers.md)
- [Scenario guide](docs/scenarios.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md) for the
current contribution and validation workflow.

Early discussions should focus on:

* payment-domain invariants;
* scenario reproducibility;
* failure semantics;
* webhook delivery guarantees;
* developer experience;
* explicit architectural trade-offs.

## License

Payment Sandbox is licensed under the MIT License. See [LICENSE](LICENSE) for
the complete license text.

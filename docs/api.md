# HTTP API

Payment Sandbox exposes a local HTTP API for creating payments, driving their
canonical lifecycle and inspecting deterministic simulator behaviour. The
[OpenAPI contract](openapi.json) is the machine-readable reference and can be
used to generate a client for a Node.js or NestJS application.

## Start the server

Follow the [installation guide](installation.md) for a published binary or the
[getting started guide](getting-started.md) for a source checkout. The default
base URL is `http://127.0.0.1:8080`.

Health, payment and webhook routes are public. Routes under `/admin/` require
the `PAYMENT_SANDBOX_ADMIN_TOKEN` value as a bearer token:

```bash
export ADMIN_TOKEN='replace-with-a-long-random-value'
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://127.0.0.1:8080/admin/providers
```

All JSON requests must use `Content-Type: application/json`. Errors use this
shape:

```json
{"error":{"code":"payment_not_found","message":"payment not found"}}
```

## Health

| Method | Route | Purpose |
| --- | --- | --- |
| `GET` | `/health/live` | Returns `{"status":"alive"}` when the process is running. |
| `GET` | `/health/ready` | Returns `{"status":"ready"}` when the API is ready, otherwise `503`. |

## Payments

Payment amounts are integer minor units: `1000` EUR represents €10.00. The
caller supplies the payment identifier; the API does not invent an identifier.

```bash
curl -X POST http://127.0.0.1:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"id":"demo-payment","amount":1000,"currency":"EUR"}'

curl http://127.0.0.1:8080/payments/demo-payment
```

Creation returns `201 Created` and a `Location` header. A payment response
contains its current `status`, monetary totals and aggregate `version`.

| Method | Route | Body |
| --- | --- | --- |
| `POST` | `/payments/{id}/authorize` | none |
| `POST` | `/payments/{id}/capture` | `{"amount":1000,"currency":"EUR"}` |
| `POST` | `/payments/{id}/cancel` | none |
| `POST` | `/payments/{id}/refund` | `{"amount":1000,"currency":"EUR"}` |

Commands return the updated payment with `200 OK`. Invalid transitions,
amounts and concurrency conflicts return `409`; an unknown payment returns
`404`.

```bash
curl -X POST http://127.0.0.1:8080/payments/demo-payment/authorize
curl -X POST http://127.0.0.1:8080/payments/demo-payment/capture \
  -H 'Content-Type: application/json' \
  -d '{"amount":1000,"currency":"EUR"}'
```

Provider-driven delayed work and callback delivery are represented as durable
jobs and executed by the runtime worker. They do not introduce a second HTTP
state model.

## Webhook endpoints

| Method | Route | Purpose |
| --- | --- | --- |
| `POST` | `/webhook-endpoints` | Register an HTTP callback destination. |
| `GET` | `/webhook-endpoints` | List registered destinations. |
| `GET` | `/webhook-endpoints/{id}` | Retrieve one destination. |

```bash
curl -X POST http://127.0.0.1:8080/webhook-endpoints \
  -H 'Content-Type: application/json' \
  -d '{"id":"demo-client","url":"http://127.0.0.1:9000/webhooks"}'
```

Registration accepts `http` and `https` URLs and does not immediately send a
callback. Delivery follows the durable-job lifecycle.

## Administration

Administration routes require `Authorization: Bearer <token>`.

### Virtual time

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://127.0.0.1:8080/admin/time

curl -X POST http://127.0.0.1:8080/admin/time/advance \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"by":"1h"}'
```

`by` is a positive Go duration such as `30m` or `1h`. Advancing business time
is explicit and does not by itself execute every due job; the scheduler and
worker process durable work afterwards.

### Inspection routes

| Method | Route | Purpose |
| --- | --- | --- |
| `GET` | `/admin/providers` | List registered provider identifiers. |
| `GET` | `/admin/scenarios/{id}` | Inspect provider, seed, initial state and commands. |
| `GET` | `/admin/payments/{id}/timeline` | Read payment state and immutable business events. |
| `GET` | `/admin/diagnostics/payments/{id}` | Read state, events, virtual time and providers. |

All inspection views are read-only. Scenario inspection does not execute the
scenario. Timeline and diagnostics explain simulator behaviour without access
to the database or source code.

## API clients

Import [openapi.json](openapi.json) into Swagger Editor or use it with an
OpenAPI client generator. For the planned NestJS demonstration application,
it can be the contract from which a TypeScript client is generated.

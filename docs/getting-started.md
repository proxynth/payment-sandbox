# Getting Started

This guide takes a new contributor from a clean checkout to a validated local
build of Payment Sandbox.

## Prerequisites

Install the following tools:

- Go `1.25.7` or a compatible Go `1.25` release, as declared in `go.mod`;
- GNU Make;
- Git;
- `golangci-lint`, required by `make fmt` and `make check`.

Check the Go version before starting:

```bash
go version
```

## Get the repository

Clone the repository and enter its directory:

```bash
git clone https://github.com/proxynth/payment-sandbox.git
cd payment-sandbox
```

## Configure the local environment

Copy the tracked example configuration:

```bash
cp .env.example .env
```

The application reads configuration from exported environment variables. The
`make run` target loads `.env` automatically when it exists. To load the same
configuration in the current shell for other commands, use:

```bash
set -a
source .env
set +a
```

The available settings are:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PAYMENT_SANDBOX_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn` or `error` |
| `PAYMENT_SANDBOX_LOG_FORMAT` | `text` | Log format: `text` or `json` |
| `PAYMENT_SANDBOX_HTTP_ADDRESS` | `:8080` | Configured HTTP address |
| `PAYMENT_SANDBOX_ADMIN_TOKEN` | none | Required bearer token for `/admin/*` routes; use a long random value |
| `PAYMENT_SANDBOX_DATABASE_PATH` | `payment-sandbox.db` | SQLite database path |
| `PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT` | `5s` | SQLite busy timeout |

Keep the local `.env` file uncommitted. The repository ignores it; only
`.env.example` is tracked.

The admin token is intentionally not given a default. Set a unique, high
entropy value before starting the application. Health and payment/webhook
routes do not require this token; administrative and diagnostic routes do.

## Validate the checkout

Run the repository workflow from its root:

```bash
make fmt
make check
make test-race
```

These commands respectively format the Go source, run the build/tests/lint
checks, and execute the complete test suite with the race detector.

`make check` also runs `go mod tidy` and verifies that it does not modify
`go.mod` or `go.sum`.

## Build and run

Build the application binary:

```bash
make build
```

The binary is written to `bin/payment-sandbox`.

Run the application from the repository:

```bash
make run
```

This starts the configured HTTP server and keeps the process running until you
stop it with `Ctrl+C`. For installing a published release without a repository
checkout or Go, use the [user installation guide](installation.md) instead.

Once the server is running, use the
[first-payment walkthrough](installation.md#make-a-first-payment-request) to
exercise the same API flow as a published release.

## Useful commands

```bash
make help       # list Make targets
make test       # run the complete test suite
make coverage   # generate coverage.out
make lint       # run golangci-lint
```

## Troubleshooting

### The configuration is ignored

Environment files are not loaded by the Go process. Run `source .env` with
auto-export enabled as shown above, or export the variables directly in the
shell that starts the application.

### `make check` reports modified module files

Run `go mod tidy`, inspect the resulting change and confirm that dependencies
are intentional. A clean checkout should leave `go.mod` and `go.sum` unchanged
after `make check`.

### The database is locked

Stop another local Payment Sandbox process using the same database, or point
the process at a separate file:

```bash
PAYMENT_SANDBOX_DATABASE_PATH=/tmp/payment-sandbox-local.db make run
```

### Formatting or linting fails

Run `make fmt`, review the changes, then rerun `make check`. Do not skip the
race test before opening a pull request.

## Where to go next

- [Architecture index](architecture/README.md)
- [Architecture diagrams](architecture/diagrams.md)
- [Architecture principles](architecture/principles.md)
- [Architectural Decision Records](architecture/decisions/)
- [Contributing guide](../CONTRIBUTING.md)
- [Agent and contribution workflow](../AGENTS.md)

# Contributing to Payment Sandbox

Thank you for contributing to Payment Sandbox. This guide covers the normal
workflow for changing the repository, validating a change and opening a pull
request.

## Before you start

Read the [Getting Started guide](docs/getting-started.md) to install the local
tooling and configure a checkout. The supported development prerequisites
are:

- Go `1.25.7` or another compatible Go `1.25` release;
- Git;
- GNU Make;
- `golangci-lint`.

Payment Sandbox is designed to run locally without external infrastructure.
The [installation guide](docs/installation.md) is for using a published
binary and is not required for repository development.

## Understand the change first

Before changing code, identify the business boundary that owns the behaviour
and read the relevant architecture documentation:

- `internal/payment` owns the canonical payment model, state transitions and
  payment application services;
- `internal/provider` contains provider contracts and provider-specific
  implementations;
- `internal/replay` defines deterministic scenario inputs and replay;
- `internal/scheduler` owns durable job scheduling and worker execution;
- `internal/webhook` owns webhook endpoint and delivery behaviour;
- `internal/administration` exposes inspection and diagnostic use cases;
- `internal/api` and `internal/app` compose the HTTP application;
- `internal/platform` contains shared technical concerns such as clocks,
  configuration, logging and SQLite persistence.

Modules should expose the smallest useful contract. Do not make one module
reach into another module's persistence implementation or bypass its
application boundary. Start with the [architecture index](docs/architecture/README.md)
and read the applicable ADRs before making a cross-cutting change.

For replay or provider work, pay particular attention to these invariants:

- the same inputs and virtual time must produce the same business behaviour;
- future asynchronous work is durable state, not an in-memory-only task;
- providers return outcomes and do not mutate payment state directly;
- the payment state machine remains the authority for valid transitions;
- provider-specific behaviour stays behind provider contracts.

The [scenario guide](docs/scenarios.md) and [provider guide](docs/providers.md)
explain these rules with examples.

## Create a change

Start from an up-to-date `main` branch and create a descriptive feature
branch:

```bash
git fetch origin main
git switch main
git pull --ff-only origin main
git switch -c feature/<descriptive-name>
```

Keep a branch focused on one coherent change. Do not include generated local
files, `.env`, databases or coverage output in a commit.

Search for existing implementations and tests before adding new abstractions.
When changing a domain or application contract, update every implementation,
fake and caller, then run focused tests before the complete validation suite.

## Test while developing

Run the smallest relevant package test during iteration. For example:

```bash
go test ./internal/payment/domain
go test ./internal/replay/...
go test ./internal/provider/...
```

Use the repository Make targets for the final validation. `make check` runs
format verification, module tidiness, build, tests and linting. It does not
run the race detector.

```bash
make fmt
make check
make test-race
```

`make fmt` may modify Go files. Review those changes before committing.
`make check` must leave `go.mod` and `go.sum` unchanged. `make test-race`
executes all packages with Go's race detector and is required even when the
change appears to be documentation-only if the repository workflow requires
the complete suite.

The same validation concerns are represented by separate CI jobs for build and
tests, linting and the race detector. Check the [CI workflow](.github/workflows/ci.yml)
when changing validation commands.

## Commit changes

Review the complete diff before committing:

```bash
git diff --check
git status --short
git diff --cached
```

Use a Conventional Commit subject with a czg-style emoji. Examples:

```text
feat: :sparkles: add payment capture flow
docs: :memo: update scenario guide
fix: :bug: handle expired scheduler lease
```

Keep the subject concise and explain additional context in the commit body
when needed. Do not commit secrets or add an unsolicited co-author trailer.

## Open a pull request

Push the branch and open a pull request against `main`:

```bash
git push -u origin feature/<descriptive-name>
```

The pull request description should state:

- what changed and why;
- the relevant architecture or domain boundary;
- how the change was validated;
- any known limitation or follow-up.

Keep the pull request reviewable. Update documentation when public behaviour,
commands, configuration or architectural decisions change. If a change
alters an accepted architectural decision, update the relevant ADR or add a
new one rather than leaving the rationale implicit.

Reviewers should be able to run the documented validation commands from a
clean checkout. Resolve review comments with follow-up commits or an amended
history agreed with the reviewer; do not rewrite a shared branch without
coordination.

## Documentation map

- [Getting Started](docs/getting-started.md): repository setup and local run;
- [Installation](docs/installation.md): published binary installation;
- [Scenario Guide](docs/scenarios.md): deterministic replay workflows;
- [Provider Guide](docs/providers.md): provider contracts and extension;
- [Release Guide](docs/releases.md): tagged release procedure;
- [Architecture index](docs/architecture/README.md): principles, modules and ADRs;
- [`AGENTS.md`](AGENTS.md): instructions for the repository agent workflow.

Keep `AGENTS.md` specific to agent collaboration. General contributor policy
belongs in this guide.

# Scenario Guide

Payment Sandbox scenarios describe a deterministic payment execution. They
are useful when a test, a bug report or a documentation example needs to
reproduce the same business behaviour more than once.

A scenario is not an HTTP script and it is not a list of sleeps. It is a
complete set of business inputs for one isolated execution:

```text
scenario inputs
      |
      v
validate -> select provider -> execute ordered commands -> inspect result
                                  |
                                  v
                         durable async work
                                  |
                    advance virtual time explicitly
```

Given the same scenario inputs, provider implementation and simulator version,
replay produces the same observable business result. Operating-system timing,
goroutine scheduling and network access do not participate in the result.

## When to use a scenario

Use a scenario to:

- reproduce a payment lifecycle or provider outcome;
- verify a regression with a stable set of inputs;
- compare provider behaviour under the same business commands;
- document a pending operation and its later completion;
- inspect the payment state and future work produced by an execution.

The current replay implementation is an application boundary used by the
simulator and its tests. It is not yet a versioned, standalone scenario file
format or a public HTTP endpoint. Repository setup and application startup are
covered by the [Getting Started guide](getting-started.md).

## Scenario inputs

The scenario domain object contains these inputs:

| Input | Meaning |
| --- | --- |
| `ID` | Stable identifier for the execution and its result. |
| `InitialPayments` | Optional payment states restored before commands start. |
| `Commands` | Ordered business operations executed from first to last. |
| `Provider` | Provider identity and provider-owned profile. |
| `InitialVirtualTime` | UTC-normalized starting point for business time. |
| `Seed` | Explicit deterministic input passed to configurable providers. |

The seed is not a random number generator and does not identify a payment. It
is an input interpreted by the selected provider. For example, the built-in
`seeded` profile maps `seed % 3` to succeeded, failed or pending
authorization. The mapping is provider-owned; the replay core only carries the
seed into provider configuration.

The provider identity and profile are also execution-scoped. The replay core
resolves the provider from the registry, then asks configurable providers to
create an execution-specific instance with the selected profile and seed.

## Commands

Commands are validated before execution and run in the order in which they
appear. A payment command refers to its payment through `PaymentID`; it does
not implicitly select the first payment in the scenario.

| Command | Required fields | Effect |
| --- | --- | --- |
| `create_payment` | payment ID, positive amount | Creates a pending payment. |
| `start_payment_saga` | payment ID, positive amount | Starts the durable authorize/capture Saga for the payment. |
| `authorize` | payment ID | Asks the provider to authorize it. |
| `capture` | payment ID, positive amount | Asks the provider to capture it. |
| `refund` | payment ID, positive amount | Asks the provider to refund it. |
| `cancel` | payment ID | Asks the provider to cancel it. |
| `advance_time` | positive duration | Advances virtual time without waiting. |
| `execute_async` | operation ID | Runs one due asynchronous operation. |

Amounts include their currency and are required for create, capture, refund
and `start_payment_saga`. `advance_time` and `execute_async` are execution
controls, so they do not carry a payment ID.

The command list must respect the payment state machine. For example, a
capture normally follows authorization, and a refund follows a captured
payment. Invalid domain transitions are rejected by the payment application
services; the replay runner does not bypass those rules.

## A complete synchronous scenario

The scenario factory creates a stable payment identifier from the scenario
identifier. This keeps the example deterministic and avoids hard-coded IDs
that are unrelated to the scenario being replayed:

```go
amount, err := paymentdomain.NewMoney(10000, "EUR")
if err != nil {
    return err
}

scenario, err := scenarios.NewWithProfile(
    scenarios.PaymentLifecycle,
    "fake",
    "success",
    "checkout-42",
    time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
    amount,
    42,
)
if err != nil {
    return err
}

result, err := engine.Replay(context.Background(), *scenario)
if err != nil {
    return err
}
```

`PaymentLifecycle` expands to create, authorize, capture and refund commands.
The provider returns successful outcomes, and the normal payment application
services apply the corresponding transitions. The result contains the final
payment states, the provider configuration, the seed and the current virtual
time.

The runner uses an execution-scoped repository. Replaying the scenario does
not mutate the production database or the state of another replay.

## Pending operations and virtual time

A provider may return `pending` instead of a final outcome. In that case the
runtime persists each returned `AsyncOperation` as a durable scheduler job.
The payment remains in its current state until the provider returns a final
outcome and that outcome is accepted by the payment state machine.

The replay must then make the two phases explicit:

```text
authorize -> pending operation is persisted
advance_time(1 minute) -> virtual time reaches the scheduled instant
execute_async(operation ID) -> scheduler and worker resume the operation
```

For the built-in `pending_authorize` profile, the operation is scheduled one
virtual minute after authorization. The operation ID should be read from the
first replay result rather than invented or hard-coded. A caller that needs to
execute a pending operation can replay the same scenario with follow-up
commands built from the returned `result.AsyncOperations[0].ID`:

```go
start := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

pending, err := scenarios.NewWithProfile(
    scenarios.PaymentLifecycle,
    "fake",
    "pending_authorize",
    "checkout-pending",
    start,
    amount,
    42,
)
if err != nil {
    return err
}

pending.Commands = pending.Commands[:2]
firstResult, err := engine.Replay(context.Background(), *pending)
if err != nil {
    return err
}
if len(firstResult.AsyncOperations) != 1 {
    return errors.New("expected one pending operation")
}

operationID := firstResult.AsyncOperations[0].ID
second := *pending
second.Commands = []replaydomain.Command{
    pending.Commands[0],
    pending.Commands[1],
    {Type: replaydomain.CommandAdvanceTime, Duration: time.Minute},
    {Type: replaydomain.CommandExecuteAsync, OperationID: operationID},
}

secondResult, err := engine.Replay(context.Background(), second)
if err != nil {
    return err
}
```

`execute_async` does not sleep and cannot force work to run early. The
scheduler checks the operation's virtual due time, and the worker executes the
provider-owned payload only when the job is eligible. This is the same durable
work model used by the runtime; replay makes the execution-scoped repository
visible and deterministic.

## Results and errors

The replay result exposes:

- the scenario ID and provider configuration;
- the deterministic seed used for provider configuration;
- the current virtual time after the last command;
- the final payment states;
- the asynchronous operations created during execution.

The runner stops at the first command error and reports its command index and
type. Validation errors include missing scenario data, an invalid provider,
duplicate initial payments, invalid amounts and malformed command controls.
Provider business outcomes are distinct from Go errors:

- `succeeded` is applied through the normal payment transition;
- `failed` moves authorization to `Failed`, while unsupported failed
  transitions are reported as execution errors;
- `pending` persists future work and leaves the payment unchanged until the
  asynchronous operation completes.

Providers return outcomes and never mutate payment state directly. The payment
state machine remains the authority for valid transitions.

## Deterministic replay checklist

When recording or reproducing a scenario, keep all of these values:

1. the scenario ID and ordered commands;
2. initial payment states, when the execution starts from existing payments;
3. provider ID and profile;
4. the deterministic seed;
5. the initial virtual time;
6. the simulator revision.

Changing any of these may intentionally describe a different simulation.
Avoid real-time waits, generated identifiers and external provider calls in a
scenario. If the scenario creates a payment, derive related identifiers from
the scenario data and carry returned operation IDs into later commands.

## Architecture decisions

- [ADR-0005: Persist Asynchronous Work](architecture/decisions/adr-0005-persist-asynchronous-work.md)
- [ADR-0006: Payment State Machine](architecture/decisions/adr-0006-payment-state-machine.md)
- [ADR-0009: Deterministic Scenario Replay](architecture/decisions/adr-0009-deterministric-scenario-replay.md)
- [ADR-0010: Virtual Clock](architecture/decisions/adr-0010-virtual-clock.md)
- [ADR-0011: Provider Plugin Model](architecture/decisions/adr-0011-provider-plugin-model.md)

For provider profiles, outcomes and extension points, see the [provider guide](providers.md).

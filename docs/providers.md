# Provider model

Payment Sandbox simulates provider behaviour behind a stable domain contract.
The payment domain owns the language of business operations; provider modules
implement the variations of those operations.

This guide describes the provider model for contributors and scenario authors.
For installing and running a published binary, see the
[user installation guide](installation.md).

## What a provider is

A provider is a local implementation of payment-provider behaviour. It is not
a network client and it does not process real money.

The core provider contract exposes four business capabilities:

- authorize a payment;
- capture an amount;
- refund an amount;
- cancel a payment.

Each operation receives a value snapshot of the payment, the requested amount
when relevant, and the current business time. It returns an
`OperationResult` or an implementation error.

```go
type Provider interface {
    Identity() ProviderIdentity
    Authorize(context.Context, AuthorizeRequest) (OperationResult, error)
    Capture(context.Context, CaptureRequest) (OperationResult, error)
    Refund(context.Context, RefundRequest) (OperationResult, error)
    Cancel(context.Context, CancelRequest) (OperationResult, error)
}
```

The capability interfaces are kept separate internally so a future provider
can advertise a narrower surface. The complete `Provider` contract currently
requires all four capabilities.

Providers receive a `PaymentSnapshot`, not a mutable payment aggregate. They
cannot change payment state directly. Their result is interpreted by the
application layer and applied through the payment state machine.

## Identity and registry

Every provider has a stable `ProviderID`. The registry uses that identity to
register and resolve implementations:

```go
registry := providerdomain.NewRegistry()

if err := registry.Register(fake.New()); err != nil {
    return err
}

provider, err := registry.Resolve("fake")
if err != nil {
    return err
}
```

The registry:

- rejects nil providers and invalid identities;
- rejects duplicate identities rather than silently replacing a provider;
- resolves providers by identity;
- returns registered identities in stable order.

The registry owns selection, not provider lifecycle or provider-specific
configuration. The core does not contain branches such as `if Stripe` or
`if Adyen`.

## Built-in providers

The application currently registers these provider identities:

| Provider ID | Implementation | Current behaviour |
| --- | --- | --- |
| `fake` | `internal/provider/fake` | Deterministic local provider for simulations and contract tests |
| `stripe` | `internal/provider/stripe` | Deterministic Stripe-shaped provider identity |
| `adyen` | `internal/provider/adyen` | Deterministic Adyen-shaped provider identity |

The `stripe` and `adyen` modules currently embed the shared deterministic
provider. Their observable distinction is their provider identity and the
provider references they produce. They do not call Stripe or Adyen APIs.
Provider-specific behaviour can be added inside those modules without
changing the payment domain or replay orchestration.

## Outcomes

Provider operations return one of three business outcomes:

| Outcome | Meaning |
| --- | --- |
| `succeeded` | The provider accepted the operation immediately |
| `failed` | The provider returned a deterministic business failure |
| `pending` | The provider scheduled future work and has not completed the operation |

`failed` is a business result, not a Go error. A Go error represents an
invalid request or an implementation/infrastructure failure. The replay layer
validates both the returned outcome and every asynchronous operation before
continuing.

Provider references are deterministic identifiers such as:

```text
fake:authorize:checkout-payment
stripe:capture:checkout-payment
adyen:refund:checkout-payment
```

Failed results append `:failed`. A pending result has no completed provider
reference because the operation is not finished yet.

## Deterministic profiles

The shared deterministic provider supports these current profiles:

| Profile | Authorize behaviour | Other operations |
| --- | --- | --- |
| `success` or empty | Succeeds immediately | Succeed immediately |
| `fail_authorize` | Returns `failed` | Succeed immediately |
| `pending_authorize` | Returns `pending` and schedules work one virtual minute later | Succeed immediately |
| `seeded` | Uses the scenario seed to choose success, failure or pending | Succeed immediately |

Profiles are provider-owned configuration. The replay core passes the profile
through without interpreting its meaning.

For the built-in `seeded` profile, the current mapping is:

| `seed % 3` | Authorize result |
| --- | --- |
| `0` | `succeeded` |
| `1` | `failed` |
| `2` | `pending` |

When the result is pending, the delay is `seed % 5 + 1` virtual minutes. The
seed is not random state and it does not identify a provider. It is an explicit
scenario input used to select reproducible provider behaviour.

## Scenarios select providers

Provider selection belongs to the scenario configuration. A scenario carries
the provider identity, profile, initial virtual time and deterministic seed:

```go
amount, err := paymentdomain.NewMoney(1000, "EUR")
if err != nil {
    return err
}

scenario, err := scenarios.NewWithProfile(
    scenarios.PaymentLifecycle,
    "fake",
    "seeded",
    "checkout-42",
    time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
    amount,
    5,
)
```

This configuration means:

1. resolve the `fake` provider;
2. configure it with the `seeded` profile and seed `5`;
3. execute the scenario from the supplied virtual time;
4. apply the returned outcomes through the normal payment application services.

With seed `5`, `5 % 3 == 2`, so authorization is pending and its asynchronous
work is scheduled for `5 % 5 + 1`, or one virtual minute later.

The same scenario inputs produce the same provider outcome. Changing the
provider ID, profile, seed, initial virtual time or business commands changes
the scenario intentionally and should be treated as a different simulation.

## Asynchronous provider work

A provider may return an `AsyncOperation` when an outcome is pending. The
operation contains its own ID, payment ID, type and scheduled virtual time.
The provider requests the work; the runtime is responsible for persisting,
scheduling and executing it.

The lifecycle is:

```text
Provider returns pending result
        |
        v
Runtime persists asynchronous operation
        |
        v
Virtual time advances explicitly
        |
        v
Worker resumes provider-owned work
        |
        v
Result passes through the payment state machine
```

The provider does not start goroutines, sleep for real time or mutate the
payment aggregate. Durable work and explicit virtual time make the result
replayable.

## Adding a provider

Add a provider as an isolated package implementing the domain contract:

```go
package example

type Provider struct {
    // provider-specific configuration stays here
}

func New() *Provider {
    return &Provider{}
}

func (p *Provider) Identity() providerdomain.ProviderIdentity {
    return providerdomain.ProviderIdentity{ID: "example"}
}
```

The concrete type must implement `Authorize`, `Capture`, `Refund` and
`Cancel`, validate the received snapshot and return valid domain outcomes.
Provider-specific validation and configuration stay in the provider package.
The payment domain, application services and replay orchestration should not
gain provider-specific branches.

Register the implementation at the composition boundary, then add focused
provider contract tests and scenario coverage. Follow the existing
`internal/provider/fake`, `internal/provider/stripe` and
`internal/provider/adyen` package boundaries as examples.

## Architectural references

- [ADR-0011: Provider Plugin Model](architecture/decisions/adr-0011-provider-plugin-model.md)
- [ADR-0009: Deterministic Scenario Replay](architecture/decisions/adr-0009-deterministric-scenario-replay.md)
- [ADR-0010: Virtual Clock](architecture/decisions/adr-0010-virtual-clock.md)
- [ADR-0005: Persist Asynchronous Work](architecture/decisions/adr-0005-persist-asynchronous-work.md)
- [ADR-0006: Payment State Machine](architecture/decisions/adr-0006-payment-state-machine.md)

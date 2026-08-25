# Architecture Diagrams

This page contains the canonical diagrams for Payment Sandbox. They describe
responsibilities and flows at the architecture level; they are not class or
implementation diagrams.

## Contexts and dependency direction

The system is a modular monolith with four bounded contexts. Simulation plans
behaviour, Payment owns business invariants, Runtime executes durable work and
Administration observes the system.

```mermaid
flowchart LR
    client[API client]

    subgraph simulation[Simulation context]
        scenario[Scenario and replay]
        rules[Behaviour rules and deterministic seed]
    end

    subgraph payment[Payment context]
        aggregate[Payment aggregate]
        transitions[Validated state transitions]
    end

    subgraph runtime[Runtime context]
        jobs[Durable jobs]
        scheduler[Scheduler]
        worker[Worker]
        clock[Virtual clock]
    end

    subgraph providers[Provider plugins]
        contracts[Provider contracts]
        implementations[Fake / Stripe / Adyen]
    end

    subgraph administration[Administration context]
        inspection[Inspection and diagnostics]
    end

    client --> scenario
    scenario --> rules
    rules -->|business commands| aggregate
    aggregate --> transitions
    transitions -->|events and future work| jobs
    jobs --> scheduler --> worker
    worker --> contracts
    contracts --> implementations
    clock -. controls .-> scheduler
    aggregate -. observable state .-> inspection
    jobs -. execution history .-> inspection
```

The arrows represent collaboration through contracts. Simulation and Runtime
do not mutate Payment state directly, and providers do not own payment
transitions.

## Deterministic replay

Replay receives every input that may affect business behaviour. The provider
configuration is opaque to the replay core and is interpreted by the selected
provider implementation.

```mermaid
flowchart LR
    inputs[Scenario inputs<br/>commands<br/>provider configuration<br/>seed<br/>initial virtual time]
    validate[Validate scenario]
    restore[Restore initial payment state]
    select[Select and configure provider]
    execute[Execute commands through application services]
    outcome{Provider outcome}
    transition[Apply valid payment transition]
    pending[Create durable provider job]
    advance[Advance virtual time]
    tick[Scheduler tick]
    resume[Worker resumes provider operation]
    result[Observable replay result<br/>payment states<br/>events<br/>jobs<br/>provider references]

    inputs --> validate --> restore --> select --> execute --> outcome
    outcome -->|succeeded| transition --> result
    outcome -->|failed| transition
    outcome -->|pending| pending --> advance --> tick --> resume --> outcome
```

A pending outcome does not transition the payment by itself; the eventual
provider result must pass through the payment state machine.

## Durable asynchronous execution

Asynchronous work follows the Runtime lifecycle. The scheduler decides when a
job is eligible, while the worker executes the registered handler. Neither
component interprets provider-specific payloads.

```mermaid
sequenceDiagram
    participant P as Payment / Provider
    participant DB as Durable job repository
    participant S as Scheduler
    participant W as Worker
    participant C as Virtual clock
    participant H as Provider job handler

    P->>DB: Persist job with scheduled time and payload
    C->>S: Tick at virtual time
    S->>DB: Find executable jobs
    DB-->>S: Pending job
    S->>DB: Lease job
    S->>W: Dispatch leased job
    W->>DB: Mark running
    W->>H: Execute payload
    H->>P: Return provider outcome
    W->>DB: Mark completed or failed
    P->>P: Apply outcome through domain transition
```

The production Runtime uses durable persistence. Replay uses the same job
lifecycle with an execution-scoped repository so one replay cannot affect
another replay.

## Provider plugin boundary

The core expresses business intent and validates business results. Providers
translate that intent and keep their profiles, references, timing and failure
decisions local to their own implementation.

```mermaid
flowchart LR
    intent[Core business intent<br/>Authorize / Capture / Refund / Cancel]
    contract[Stable provider contract]
    provider[Selected provider implementation]
    config[Opaque provider configuration<br/>profile and deterministic seed]
    response[Provider result<br/>succeeded / failed / pending]
    domain[Payment aggregate<br/>validated transition]

    intent --> contract --> provider --> response --> domain
    config --> provider
```

Provider implementations are replaceable. The replay core must not branch on
provider identity or duplicate provider-specific rules.

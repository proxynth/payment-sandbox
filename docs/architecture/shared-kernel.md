# Shared Kernel

## Purpose

Payment Sandbox does not currently define a concrete Shared Kernel.

Bounded contexts should remain autonomous by default.

A Shared Kernel may be introduced only when multiple bounded contexts genuinely depend on the same stable semantic 
concept.

The objective is to avoid accidental coupling through generic shared packages.

---

# What Is a Shared Kernel?

In Domain-Driven Design, a Shared Kernel represents a deliberately shared subset of the domain model used by multiple 
bounded contexts.

Sharing a concept creates coupling.

That coupling must therefore be intentional, explicit and justified.

A concept qualifies for the Shared Kernel only when:

- it has identical semantics in every consuming context;
- multiple contexts genuinely need to manipulate it;
- duplicated representations would introduce unnecessary translation or inconsistency;
- the concept is sufficiently stable;
- all consuming contexts agree on its evolution.

--- 

# What Does Not Belong in the Shared Kernel?

The Shared Kernel must not become a generic location for reusable code.

The following do not qualify solely because there are used in several places:

- helper functions;
- logging;
- configuration;
- persistence utilities;
- HTTP utilities;
- serialization helpers;
- generic errors;
- constants;
- DTOs;
- framework abstractions.

Cross-cutting technical concerns belong to the Platform layer.

Reusable code is not automatically shared domain knowledge.

---

# Business Ownership Comes First

A concept should remain inside the bounded context that owns it meaning unless another context genuinely shares the same
 semantics.

For example, `Money` currently gelongs to the Payment context.

Although `Simulation` or `Administration` may need to represent monetary values, this does not automatically make `Money` 
part of the Shared Kernel.

Those contexts may consume Payment contracts or map values into their own representations.

Shared usage alone is insufficient justification for shared ownership.

---

# Translation Is Acceptable

Duplicating a small representation across bounded contexts is sometimes preferable to introducing permanent semantic 
coupling.

For example:

```text
Payment Context

Money
  │
  ▼
Application Contract
  │
  ▼
Administration Read Model

Amount
Currency
```

Administration does not necessarily need to import the Payment domain model.

Translation between contexts is acceptable when it preserves autonomy and keeps dependencies explicit.

---

# Candidate Concepts

The following concepts may eventually be considered for the Shared Kernel:

- strongly typed identifiers;
- currency representation;
- deterministic execution identifiers;
- correlation identifiers.

None of these concepts are included automatically.

Each candidate must be evaluated when a concrete cross-context requirement appears.

---

# Explicit Non-Candidates

The following concepts should remain owned by their bounded context:

## Payment

- Payment
- Authorization
- Capture
- Refund
- Cancellation
- Payment lifecycle

## Simulation 

- Scenario
- Rule
- Fault
- Behavior Plan

## Execution

- Job
- Retry
- Execution Attempt

The concept have context-specific semantics and should not be moved into a shared package merely to simplify imports.

---

# Dependency Rules

If a Shared Kernel is introduced later, it must:

- remain small;
- contain only stable semantic concepts;
- contain no infrastructure dependencies;
- contain no application orchestration;
- contain no provider-specific behavior;
- avoid depending on bounded contexts.

The dependency direction would be:

```text
Payment ───────┐
               │
Simulation ────┼──► Shared Kernel
               │
Execution ─────┘
```

The Shared Kernel must never depend back on those contexts.

---

# Repository Structure

No Shared Kernel package exists today.

The repository must therefore not contain placeholder packages such as:

```text
internal/shared
internal/common
internal/kernel
internal/utils
```

If a Shared Kernel becomes justified later, its location reflect its semantic role explicitly.

For example:

```text
internal/kernel/
```

should only be introduced after concrete share concepts exist.

---

# Decision Rule

Before moving a concept into the Shared Kernel, maintainers should answer:

1. Which bounded contexts need this concepts?
2. Does it have exactly the same meaning in each context?
3. Is translation more expensive or dangerous than coupling?
4. Is the concept stable?
5. Who owns changes to the shared definition?

If these questions cannot be answered clearly, the concept should remain context-local.

---

# Current Decision

Payment Sandbox currently has no Shared Kernel implementation.

This is intentional*.

The architecture favors explicit context ownership and translation over premature shared abstractions.

A Shared Kernel will be introduced only when concrete domain requirements justify the additional coupling.
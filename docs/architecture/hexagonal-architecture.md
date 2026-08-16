# Targeted Hexagonal Architecture

## Purpose

Payment Sandbox applies hexagonal architecture selectively.

The objective is to protect domain and application behavior from infrastructure concerns where doing so improves 
testability, determinism or evolvability.

Hexagonal architecture is not applied mechanically to every package.

---

# Conceptual Layers

A module may contain the following conceptual responsibilities:

## Domain

Owns: 

- business concepts;
- invariants;
- state transitions;
- domain behavior.

This domain must not depend on:

- HTTP;
- SQL;
- filesystem;
- external providers;
- operating system time;
- logging infrastructure.

## Application

Coordinates domain behavior and use cases.

Application code may depends on ports when it requires capabilities provided by infrastructure or another module.

## Outbound Ports

Define capabilities required by the application.

Examples may eventually include:

- payment persistence;
- business clock;
- outbound callback delivery;
- deterministic random generation.

Ports belong to the code that consumes the capability.

## Outbound Adapters

Implement outbound ports using concrete infrastructure.

Examples may eventually include:

- SQLite repositories;
- HTTP callback clients;
- operating system clocks.

---

# Dependency Direction

Dependencies always point toward business behavior.

```text
Infrastructure
      │
      ▼
Application
      │
      ▼
Domain
```

The domain must never import infrastructure  packages.

Infrastructure may depend on application or domain contracts.

---

# When to Introduce an Interface

An interface should be introduced only when a least one of the following is true:

- the dependency represents an external side effect;
- deterministic testing requires substitution;
- multiple implementations are expected;
- the dependency crosses a meaningful module boundary;
- the abstraction protects business code from infrastructure details.

An interface should not be introduced merely because a concrete type exists.

---

# Port Ownership

Ports are defined by the consumer.

For example, if the Payment application requires persistence:

```text
Payment Application
        │
        ▼
PaymentRepository
        ▲
        │
SQLite Adapter
```

`PaymentRepository` belongs to the Payment module.

It does not belong to the SQLite infrastructure package.

The consumer defines the capability it needs.

The adapter satisfies that capability.

---

# Module Structure

Module structure follows responsibility.

A simple module may remain flat:

```text
payment/
├── money.go
└── payment.go
```

A module that develops meaningful architectural boundaries may evolve toward:

```text
payment/
├── domain/
├── application/
└── adapters/
```

Every directory must represent a real responsibility.

Empty architectural layers are prohibited.

---

# Repository-Level Rules

Do not create generic package such as:

```text
internal/ports
internal/adapters
internal/interfaces
internal/repositories
```

Ports and adapters belong to the module whose responsibility they support.

Cross-cutting technical capabilities belong under:

```text
internal/platform
```

Application composition belongs under:

```text
internal/app
```
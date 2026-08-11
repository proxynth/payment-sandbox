# Module Conventions

Payment Sandbox is implemented as a modular monolith.

Functional capabilities are organised under:

`internal/<module>`

Examples may eventually include:

- payment
- simulation
- scheduler
- administration

Modules are introduced only when they own a concrete responsibility.

## Dependency rules

A module:

- owns its business responsibility;
- exposes the smallest useful public contract;
- does not expose infrastructure details;
- does not directly access another module's persistence implementation;
- does not bypass another module's application boundary.

Cross-cutting technical concerns belong under:

`internal/platform`

Application composition belongs under:

`internal/app`

Package structures are responsibility-driven and do not need to be identical across modules.
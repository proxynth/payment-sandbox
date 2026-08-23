# Payment Sandbox - Agent Workflow

## Before starting a ticket

1. Identify the current Linear ticket and read its description, related ADRs, and relevant existing implementation.
2. Check the working tree and stop if unrelated local changes would be affected.
3. Synchronize with the latest `main`:

   ```bash
   git fetch origin main
   git switch main
   git pull --ff-only origin main
   ```

4. Create a feature branch from the updated `main`:

   ```bash
   git switch -c feature/<descriptive-name>
   ```

   Feature branch names must be descriptive and must not contain Linear ticket numbers.
5. Move the Linear ticket to **In Progress**.
6. Add the ticket description to its body immediately after switching to it, using the established template: Objective, Context, Scope, and Acceptance criteria. Verify in Linear that the body was actually saved before starting implementation.

## Implementation guidelines

- Follow the architecture documented in `docs/architecture/` and the applicable ADRs.
- Preserve the existing module boundaries and deterministic execution model.
- Keep provider-specific behaviour behind provider contracts; do not add provider conditionals to the core domain.
- Search for existing implementations and tests before adding new ones.
- Keep each ticket focused and avoid implementing work assigned to later tickets.

## Required validation

Before committing, run all three commands from the repository root:

```bash
make fmt
make check
make test-race
```

Resolve every formatting, linting, build, test, and race-detector failure before continuing.

## Commit conventions

- Use a signed commit through the configured SSH signing agent.
- Follow Conventional Commits with a czg-style emoji, for example:

  ```text
  feat: :sparkles: implement provider contracts
  ```

- Do not add a `Co-authored-by` trailer.
- Review the complete staged diff before committing.

## Pull request workflow

1. Push the feature branch and create a pull request linked to the Linear ticket.
2. Wait for all GitHub checks to pass.
3. Merge only after the checks are green, using squash merge when appropriate.
4. Confirm that `main` contains the merge commit.
5. Link the pull request in Linear and move the ticket to **Done** only after the merge is confirmed.

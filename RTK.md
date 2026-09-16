# RTK - Rust Token Killer (Codex CLI)

**Usage**: Token-optimized CLI proxy for shell commands.

## Rule

When working in this repository through Codex CLI, always prefix shell
commands with `rtk`. Use the most specific RTK command available. For
commands without a dedicated RTK integration, use `rtk proxy`.

Examples:

```bash
rtk git status
rtk go test ./...
rtk golangci-lint run
rtk proxy make check
```

The GitHub Actions workflows intentionally keep their native commands. They
run on clean hosted runners where RTK is not a repository dependency.

## Meta Commands

```bash
rtk gain            # Token savings analytics
rtk gain --history  # Recent command savings history
rtk proxy <cmd>     # Run raw command without filtering
```

## Verification

```bash
rtk --version
rtk gain
rtk proxy which rtk
```

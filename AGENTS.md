# jira-cli

Standalone agent-first Jira CLI.

## Purpose

Build a local-first Jira tool for coding agents.

The binary is `jira`.

## Contract

- `jira --help` is the primary documentation surface
- `jira <subcommand> --help` must stay explicit and example-driven
- commands that return structured data must support `--json`
- no interactive prompts or TUI flows
- results on stdout, diagnostics on stderr

## Repo Workflow

1. Keep the command surface agent-oriented and explicit.
2. Prefer extending help text and tests together when changing CLI behavior.
3. Avoid overspecifying usage in repo docs when `jira --help` can carry the contract directly.
4. Keep installation simple: local binary at `/Users/vabole/.local/bin/jira`.
5. Keep live Jira validation first-class: offline tests are required, but real API golden recording should cover the released command shapes.
6. Prefer agent-runnable workflows such as `make test`, `make test-cover`, and `make record-live-goldens` over ad hoc local scripts.
7. Live golden workflow:
   - auth comes from the normal Jira env vars (`JIRA_SITE`, `JIRA_EMAIL`, `JIRA_TOKEN` or `JIRA_API_TOKEN`)
   - choose one safe issue with `JIRA_GOLDEN_ISSUE_KEY`
   - record with `make record-live-goldens`
   - review `testdata/goldens/live/`
   - verify with `go test ./...`

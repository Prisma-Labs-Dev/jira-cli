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
5. Keep this public repo thin: public tests should use unit coverage plus synthetic fixtures only.
6. Real Jira-derived fixtures, recorder tooling, and replay validation belong in the private companion repo `repos/jira-cli-private`.
7. Prefer agent-runnable workflows such as `make test`, `make test-cover`, and `make verify-local` over ad hoc local scripts.
8. Keep `testdata/goldens/synthetic/` focused on edge cases that are hard to guarantee from one live tenant or one reference issue.

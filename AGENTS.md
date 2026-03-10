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

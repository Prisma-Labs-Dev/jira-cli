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
5. Keep `Formula/jira.rb` and the README install/update instructions aligned when changing packaging.

## Historical Reference

- `AGENT_PROMPT.md` is committed as an archival copy of the original design brief for this overhaul.
- Treat it as background context, not the live contract.
- The authoritative current contract remains the CLI help surfaces plus committed tests and goldens.

## Live Jira Validation

- Prefer env-driven live validation over checked-in local config.
- In this workspace, `make test-live-bw` is the preferred path for future agents.
- In this tenant, Jira and Confluence share the same API token, but the Jira site is `https://jira-eu-aholddelhaize.atlassian.net`.
- Run live validation with `JIRA_SITE="https://jira-eu-aholddelhaize.atlassian.net" make test-live-bw`.
- That flow reads the Bitwarden item `Confluence CLI`, derives `JIRA_EMAIL` and `JIRA_TOKEN`, applies the explicit Jira site override, and runs the live E2E suite without printing secrets.
- If the Bitwarden item name changes, set `BW_JIRA_ITEM_NAME` before invoking the target.

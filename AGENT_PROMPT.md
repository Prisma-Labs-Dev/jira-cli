# Agent Prompt: Make `jira` Useful for Agent-First Jira Research

You are working in the `Prisma-Labs-Dev/jira-cli` repository.

Your job is **not** to turn this into a human-oriented interactive Jira client. Your job is to make it a reliable, explicit, low-noise CLI that coding agents can use to explore Jira data safely and efficiently, much like the workspace's Confluence CLI.

## Mission

Build a practical, agent-first Jira CLI that exposes the most useful **read-oriented** Jira Cloud API surfaces for investigation and research workflows.

The binary is `jira`.

## Source-of-truth principles

Follow these repo-local and workspace-level rules:

- `jira --help` and subcommand `--help` are the main documentation surface.
- No interactive prompts, no setup wizard, no TUI.
- Every command that returns structured data must support `--json`.
- Results go to stdout; diagnostics and errors go to stderr.
- Prefer explicit commands and flags over hidden behavior.
- Keep output sized for agent use; do not dump huge payloads by default.
- Make it easy for agents to understand the output shape without jq spelunking.
- Use current Jira Cloud APIs only. Do not build on deprecated endpoints.

## Clean-slate rule

You do **not** need to preserve backwards compatibility.

You are free to:

- remove broken or misleading commands
- rename commands that do not fit the agent-first model
- replace shaky scaffolds with cleaner implementations
- simplify or redesign the command tree if it improves clarity
- delete dead compatibility code that only preserves a bad interface

The goal is not to minimally patch the existing CLI. The goal is to end up with the cleanest, most maintainable, most agent-usable Jira CLI.

## Product framing

Treat this tool as:

- a **Jira API access layer for agents**
- a **research/discovery CLI**
- a **bounded-output summarizer** over verbose Jira API responses

Do **not** optimize first for:

- human convenience shortcuts
- interactive browsing flows
- mutation commands
- full raw response dumps by default
- backwards compatibility with a weak or partially implemented command surface

## What "useful" means here

An agent using this CLI should be able to:

- discover what projects, boards, issue types, statuses, and fields exist
- fetch specific issues with a predictable, compact structure
- search issues without relying on deprecated endpoints
- inspect saved filters and reusable discovery surfaces
- understand the shape of the output from built-in help alone
- request a compact default view or structured JSON without massive payloads

## Live Jira findings from this environment

These findings came from probing the tenant and should inform implementation:

- Jira Cloud is reachable.
- There are many accessible projects (`1063` in one probe), so pagination and limits matter.
- Boards are exposed through Agile API. Example: project `SCWI` has board `Scrum board Warehouse Inbound`.
- Saved filters are exposed and useful for discovery.
- Field inventory is large (`1076` fields, `1032` custom in one probe), so field browsing needs filtering and limits.
- Issue detail retrieval is useful and already surfaces parent links, issue links, comments, labels, components, fix versions, and subtasks.
- Project workflow/status data is exposed and useful for research.
- `/rest/api/3/search` returns `410 Gone` in this tenant.
- `/rest/api/3/search/jql` works and should be the basis for search support.

## Recommended command roadmap

Prioritize these commands first.

### 1. Identity and environment

- `jira me`
- `jira serverinfo`

These are already good sanity-check commands and should stay explicit.

### 2. Project discovery

- `jira project list`
- `jira project get <KEY>`
- `jira project statuses <KEY>`

Why:
- agents need to discover the right project before they search it
- project statuses are valuable for understanding workflow semantics

### 3. Board discovery

- `jira board list --project <KEY>`
- `jira board get <ID>`

Why:
- boards tell agents how work is organized
- project-to-board lookup is useful for research and backlog discovery

### 4. Issue reads

- `jira issue get <KEY>`

This should support:
- bounded default field set
- `--fields` for expansion
- `--json`
- a compact text rendering

Strongly consider subviews such as:

- `jira issue links <KEY>`
- `jira issue comments <KEY>`

if that keeps outputs smaller and easier to reason about than one giant issue payload.

### 5. Issue search

- `jira issue search --jql <QUERY>`
- `jira issue search --project <KEY> --status <NAME> --assignee <VALUE>`

Requirements:
- use `/rest/api/3/search/jql`, not the deprecated `/rest/api/3/search`
- support `--limit`
- support `--fields`
- default to a compact result shape
- return pagination/navigation hints
- do not leave deprecated-search compatibility fallbacks in place

### 6. Filter discovery

- `jira filter list`
- `jira filter get <ID>`

Why:
- saved filters are high-signal discovery artifacts for agents
- they may reveal team workflows and canonical queries

### 7. Field discovery

- `jira field list`
- maybe `jira field get <ID-or-name>`

Requirements:
- support `--custom-only`
- support `--search`
- support `--limit`
- make the output compact and predictable

This is important because Jira custom fields are numerous and otherwise painful to discover.

## Output design requirements

Design output for context-window efficiency.

### Default text output

Text output should be:

- compact
- readable
- intentionally summarized
- consistent across commands

For list commands, default to a concise row-per-item summary.

For object commands, default to a curated field subset, not the entire API payload.

### JSON output

Every structured command must support `--json`.

JSON should be:

- stable
- shaped for agents
- pre-parsed
- compact by default

Do **not** dump raw Jira API responses unless explicitly requested.

Prefer a normalized envelope like:

```json
{
  "items": [...],
  "page": {
    "limit": 50,
    "nextCursor": null,
    "nextHint": "use --cursor <value>"
  },
  "schema": {
    "itemType": "issue-summary",
    "fields": ["key", "summary", "status", "assignee"]
  }
}
```

For single-object reads, prefer:

```json
{
  "item": {...},
  "schema": {
    "itemType": "issue-detail",
    "fields": [...]
  }
}
```

The key idea is that agents should not have to infer the shape by trial and error.

## Help surface requirements

The help text should teach usage without external docs.

Each command help should include:

- purpose
- required and optional flags
- output behavior
- examples
- notes about defaults and pagination

Where useful, include:

- default field set
- supported filters
- JSON shape notes

## Size and pagination strategy

Make large Jira surfaces safe by default.

For list/search commands:

- default `--limit` should be modest
- include pagination support where the API provides it
- expose the next-page mechanism explicitly
- never return hundreds of large objects by default

If the API uses start-at pagination, expose that cleanly, e.g.:

- `--limit`
- `--start-at`

If a cursor-like mechanism exists, expose that explicitly.

## Schema / response-map strategy

One of the main goals is to help agents avoid jq spelunking.

Pick at least one of these approaches, ideally both:

1. Shape JSON responses with a small, stable CLI-owned schema.
2. Add command help that documents the emitted JSON fields and default field sets.

If you add a schema helper command, keep it simple:

- `jira schema issue-summary`
- `jira schema issue-detail`

But do not overbuild this if help text and stable envelopes are enough.

## Error-handling requirements

- errors must be explicit and actionable
- no silent fallbacks
- if a Jira endpoint is deprecated or unsupported, say so directly
- if a user requests too much output, encourage narrower flags
- preserve stable exit codes

The known `410 Gone` behavior on `/rest/api/3/search` should be treated as a signal to remove old assumptions, not to preserve legacy search codepaths. Route search through the working API surface and keep deprecated APIs out of the design.

## Authentication and config

Preserve the existing agent-oriented config contract:

- flags override env vars
- env vars override config file

Relevant env vars include:

- `JIRA_SITE`
- `JIRA_BASE_URL`
- `JIRA_EMAIL`
- `JIRA_TOKEN`
- `JIRA_API_TOKEN`
- config path override if supported

No interactive auth flow should be required.

## Tests and validation

When implementing, extend tests with behavior.

Do not rely only on unit tests and mocked assumptions. Add end-to-end validation that proves the CLI matches the real Jira API behavior.

At minimum, cover:

- root help and subcommand help
- JSON output shape
- limits and pagination flags
- error behavior for invalid inputs
- search path using the working Jira search endpoint
- field filtering and compact output rules
- behavior when deprecated endpoints are unavailable
- command output size discipline for list/search flows

Add a golden-driven validation layer for real CLI behavior.

This should include:

- end-to-end golden tests for representative commands
- stable fixtures or golden outputs for both text and `--json`
- golden coverage for help text, compact list output, and structured envelopes
- live validation against current Jira APIs where this repo already supports that workflow

The point is to verify the contract that agents consume, not just the internal implementation.

If the repository needs a clearer split, use:

- unit tests for parsers, config, and renderers
- golden tests for CLI output contracts
- end-to-end validation for live Jira API integration

Run:

```bash
make test
make install-local
make verify-local
```

## Suggested implementation order

1. Fix and harden `issue get` behavior and help.
2. Implement `issue search` using the working `/search/jql` API.
3. Add `project list`, `project get`, and `project statuses`.
4. Add `board list`.
5. Add `filter list`.
6. Add `field list` with bounded output.
7. Add or refine golden/end-to-end validation for the resulting command contracts.
8. Revisit whether schema helper commands are needed.

## Deliverable expectation

Do not stop at exposing raw API transport.

The final result should feel like:

- a small, sharp research CLI
- predictable enough for autonomous agents
- explicit enough to be self-discoverable from `--help`
- compact enough to avoid wasting context windows

If you need to choose between "more API coverage" and "clean agent-oriented contracts", prefer the contract.

If you need to choose between "preserving an older interface" and "shipping a cleaner agent-first interface", prefer the cleaner interface.

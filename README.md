# jira-cli

Agent-first Jira CLI with explicit commands, compact default output, stable `--json` envelopes, and no interactive mode.

The binary name is `jira`.

## Historical Reference

The original implementation brief that drove this repo overhaul is archived in `AGENT_PROMPT.md`.

Use it as design history and context for future agent work, but treat `jira --help`, subcommand help, and the committed tests/goldens as the current contract.

## Principles

- explicit commands over shortcut-heavy UX
- machine-readable output with `--json`
- primary results on stdout
- diagnostics and errors on stderr
- no prompts, no TUI, no setup wizard
- bounded default output and explicit pagination
- help text as the primary contract surface

## Install Local

```bash
cd /Users/vabole/repos/jira-cli
make install-local
```

This builds the local binary to `/Users/vabole/.local/bin/jira`.

## Start With Help

```bash
jira --help
jira issue search --help
jira project list --help
jira field list --help
```

## Command Surface

Current read-oriented scope:

- `jira me`
- `jira serverinfo`
- `jira issue get`
- `jira issue search`
- `jira issue comments`
- `jira project list`
- `jira project get`
- `jira project statuses`
- `jira board list`
- `jira board get`
- `jira filter list`
- `jira filter get`
- `jira field list`
- `jira field get`

## JSON Contract

Structured commands return stable envelopes instead of raw Jira API payloads.

List commands use:

```json
{
  "items": [],
  "page": {
    "limit": 50,
    "startAt": 0,
    "returned": 0,
    "nextStartAt": 50,
    "nextHint": "use --start-at 50"
  },
  "schema": {
    "itemType": "issue-summary",
    "fields": ["key", "id", "fields.summary"]
  }
}
```

Single-object commands use:

```json
{
  "item": {},
  "schema": {
    "itemType": "issue-detail",
    "fields": ["key", "id", "fields.summary"]
  }
}
```

## Examples

```bash
jira issue get SCWI-282 --json
jira issue search --project SCWI --status "In Progress"
jira issue comments SCWI-282 --limit 20 --json
jira project list --limit 25 --json
jira board list --project SCWI
jira filter get 10001 --json
jira field list --custom-only --search warehouse
```

## Validation

```bash
make lint
make test
make install-local
make verify-local
```

## Live Validation

Live E2E validation is env-driven and intentionally separate from the default test suite.

```bash
export JIRA_SITE="https://jira-eu-aholddelhaize.atlassian.net"
export JIRA_EMAIL="agent@example.com"
export JIRA_TOKEN="..."
make test-live
```

The live tests call the real Jira APIs, normalize the results into safe contract summaries, and compare those summaries against committed goldens.

### Credential Sources

The CLI always resolves credentials in this order:

1. flags
2. environment variables
3. `~/.config/jira/config.json`

For live testing, future agents should prefer short-lived env vars over committing config files.

### Future Agent Workflow

In this workspace, the reusable Bitwarden item is `Confluence CLI`. It stores:

- `CONFLUENCE_URL`
- `CONFLUENCE_EMAIL`
- `CONFLUENCE_API_TOKEN`

The Jira and Confluence API tokens are the same in this tenant, but the Jira base URL is different from the Confluence URL.

For this workspace, use:

```bash
export JIRA_SITE="https://jira-eu-aholddelhaize.atlassian.net"
```

For future agents, the simplest path is:

```bash
JIRA_SITE="https://jira-eu-aholddelhaize.atlassian.net" make test-live-bw
```

That script:

- reads the `Confluence CLI` Bitwarden item
- derives `JIRA_EMAIL` and `JIRA_TOKEN`
- uses `JIRA_SITE` or `BW_JIRA_SITE` as an explicit Jira host override
- exports those vars only for the test process
- runs the live E2E suite without printing the secret values

If the item name ever changes, set:

```bash
BW_JIRA_ITEM_NAME="New Item Name" JIRA_SITE="https://jira-eu-aholddelhaize.atlassian.net" make test-live-bw
```

# jira-cli

Agent-first Jira CLI with explicit commands, `--help`, `--json`, and no interactive mode.

The binary name is `jira`.

## Principles

- explicit commands over shortcut-heavy UX
- machine-readable output with `--json`
- primary results on stdout
- diagnostics and errors on stderr
- no prompts, no TUI, no setup wizard

## Install With Homebrew

```bash
brew tap Prisma-Labs-Dev/tap
brew install jira-cli
```

Then start with:

```bash
jira --help
jira me --help
jira issue get --help
```

Example:

```bash
jira issue get SCWI-282 --json
```

Search example:

```bash
jira issue search --project SCWI --status "To Do" --json
```

## Install Local

```bash
cd /Users/vabole/ah/ah-workspace/repos/jira-cli
make install-local
```

This builds the local binary to `/Users/vabole/.local/bin/jira`.

## Commands

Use the CLI help as the main documentation surface:

```bash
jira --help
jira me --help
jira serverinfo --help
jira issue get --help
jira issue search --help
```

## Auth Inputs

Config precedence is:

1. flags
2. env vars
3. config file

Supported env vars:

- `JIRA_SITE` or `JIRA_BASE_URL`
- `JIRA_EMAIL`
- `JIRA_TOKEN` or `JIRA_API_TOKEN`
- `JIRA_CONFIG`

Default config path:

- `~/.config/jira/config.json`

## Current Scope

- `jira me`
- `jira serverinfo`
- `jira issue get`
- `jira issue search`

## Agent Use

Keep this repo documentation short and let `jira --help` carry the interface contract.

Typical agent flow:

```bash
jira --help
jira me --json
jira issue get SCWI-282 --fields summary,status,assignee --json
jira issue search --jql 'project = SCWI ORDER BY updated DESC' --json
```

Results go to stdout. Errors and diagnostics go to stderr. The CLI does not prompt or open interactive flows.

## Testing

Offline tests:

```bash
make test
make test-cover
```

Recorded contract checks:

```bash
go test ./...
```

This public repo covers:

- unit and command tests
- synthetic search edge-case fixtures in `testdata/goldens/synthetic/`

## Private Live Validation

Real Jira-derived fixtures, recorder tooling, and live replay verification live in the private companion repository `Prisma-Labs-Dev/jira-cli-private`.

That keeps tenant-derived artifacts out of public git history while still allowing private contributors to validate the CLI against real Jira traffic.

## CI

GitHub Actions runs `go test ./...` on pushes and pull requests. Public CI enforces the code plus synthetic fixture contract. Private live-fixture validation runs outside this repo.

## Tasks

```bash
make test
make test-cover
make install-local
make verify-local
```

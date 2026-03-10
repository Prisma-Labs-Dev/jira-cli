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

## Current Scope

- `jira me`
- `jira serverinfo`
- `jira issue get`
- `jira issue search` scaffold

## Tasks

```bash
make test
make install-local
make verify-local
```

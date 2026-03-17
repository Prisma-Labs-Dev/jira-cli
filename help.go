package main

const rootHelp = `jira - agent-first Jira CLI for deterministic research and discovery

Usage:
  jira <command> [flags]
  jira <command> --help

Commands:
  me           Print the authenticated Jira user
  serverinfo   Print Jira server metadata
  issue        Read issues, search with JQL, and inspect comments
  project      Discover projects and project workflow statuses
  board        Discover Jira boards through the Agile API
  filter       Discover saved filters
  field        Discover Jira fields with bounded output
  version      Print the CLI version

Contract:
  - non-interactive only
  - primary results on stdout
  - diagnostics and errors on stderr
  - explicit flags over hidden defaults
  - per-command --json for structured output
  - config resolution is flags > env > config file
  - compact default output with explicit pagination flags

Examples:
  jira me --json
  jira issue get SCWI-282 --json
  jira issue search --project SCWI --status "In Progress"
  jira issue search --jql 'project = SCWI ORDER BY updated DESC' --limit 25 --json
  jira project list --limit 25
  jira board list --project SCWI --json
  jira filter list --limit 20
  jira field list --custom-only --search warehouse --json`

const meHelp = `jira me

Purpose:
  Print the authenticated Jira user for sanity checks and automation setup.

Usage:
  jira me [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: compact key/value summary.
  JSON: { "item": { ... }, "schema": { "itemType": "jira-user", ... } }

Resolution:
  Flags override env vars.
  Env vars override the config file.
  Supported env vars: JIRA_SITE, JIRA_BASE_URL, JIRA_EMAIL, JIRA_TOKEN, JIRA_API_TOKEN, JIRA_CONFIG.
  Default config path: ~/.config/jira/config.json

Examples:
  jira me
  jira me --json`

const serverInfoHelp = `jira serverinfo

Purpose:
  Print Jira Cloud server metadata for environment validation.

Usage:
  jira serverinfo [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: compact key/value summary.
  JSON: { "item": { ... }, "schema": { "itemType": "server-info", ... } }

Examples:
  jira serverinfo
  jira serverinfo --json`

const issueHelp = `jira issue - explicit issue read commands

Usage:
  jira issue <command> [flags]
  jira issue <command> --help

Commands:
  get         Fetch one issue by key with a bounded field set
  search      Search issues using /rest/api/3/search/jql
  comments    List comments for one issue with bounded output

Examples:
  jira issue get SCWI-282 --json
  jira issue search --project SCWI --status "To Do"
  jira issue search --jql 'project = SCWI ORDER BY updated DESC' --json
  jira issue comments SCWI-282 --limit 20 --json`

const issueGetHelp = `jira issue get

Purpose:
  Fetch one issue by key with a compact, agent-owned field map.

Usage:
  jira issue get <ISSUE-KEY> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --fields <names>       Comma-separated field list
                         default: summary,status,assignee,issuetype,priority,parent,labels,components,updated
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: compact key/value summary.
  JSON: { "item": { "key": "...", "fields": { ... } }, "schema": { "itemType": "issue-detail", ... } }

Notes:
  The JSON shape is CLI-owned and compact. It does not dump the raw Jira issue payload.

Examples:
  jira issue get SCWI-282
  jira issue get SCWI-282 --fields summary,status,labels --json`

const issueSearchHelp = `jira issue search

Purpose:
  Search issues through the working Jira Cloud endpoint: /rest/api/3/search/jql.

Usage:
  jira issue search [flags]

Flags:
  --assignee <value>     Filter by assignee value or function, such as currentUser()
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --fields <names>       Comma-separated field list
                         default: summary,status,assignee,priority,updated
  --jql <query>          Raw JQL to execute
  --json                 Print a stable JSON envelope
  --limit <n>            Max issues to return (default: 50)
  --project <key>        Filter by project key
  --site <url>           Jira base URL override
  --start-at <n>         Offset for paginated results (default: 0)
  --status <name>        Filter by status (repeatable)
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: one compact row per issue plus pagination summary.
  JSON: {
    "items": [...],
    "page": { "limit": 50, "startAt": 0, "returned": 50, "nextStartAt": 50, ... },
    "schema": { "itemType": "issue-summary", "fields": ["key", "id", "fields.summary", ...] }
  }

Notes:
  Use either --jql or explicit filters in one call.
  If you omit --jql, the CLI builds a simple JQL expression from --project, --status, and --assignee.

Examples:
  jira issue search --project SCWI --status "To Do"
  jira issue search --assignee currentUser() --json
  jira issue search --jql 'project = SCWI ORDER BY updated DESC' --limit 25 --start-at 50 --json`

const issueCommentsHelp = `jira issue comments

Purpose:
  List comments for one issue with compact text excerpts and stable JSON.

Usage:
  jira issue comments <ISSUE-KEY> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --limit <n>            Max comments to return (default: 50)
  --site <url>           Jira base URL override
  --start-at <n>         Offset for paginated results (default: 0)
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: one compact row per comment plus pagination summary.
  JSON: { "items": [...], "page": { ... }, "schema": { "itemType": "issue-comment", ... } }

Examples:
  jira issue comments SCWI-282
  jira issue comments SCWI-282 --limit 20 --json`

const projectHelp = `jira project - explicit project discovery commands

Usage:
  jira project <command> [flags]
  jira project <command> --help

Commands:
  list        List projects with bounded output
  get         Fetch one project by key
  statuses    Fetch workflow statuses for one project

Examples:
  jira project list --limit 25
  jira project get SCWI --json
  jira project statuses SCWI`

const projectListHelp = `jira project list

Purpose:
  List accessible Jira projects with compact summaries and explicit pagination.

Usage:
  jira project list [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --limit <n>            Max projects to return (default: 50)
  --search <value>       Optional project search string
  --site <url>           Jira base URL override
  --start-at <n>         Offset for paginated results (default: 0)
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: one compact row per project plus pagination summary.
  JSON: { "items": [...], "page": { ... }, "schema": { "itemType": "project-summary", ... } }

Examples:
  jira project list
  jira project list --search warehouse --limit 25 --json`

const projectGetHelp = `jira project get

Purpose:
  Fetch one project by key with a compact, agent-oriented summary.

Usage:
  jira project get <PROJECT-KEY> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: compact key/value summary.
  JSON: { "item": { ... }, "schema": { "itemType": "project-detail", ... } }

Examples:
  jira project get SCWI
  jira project get SCWI --json`

const projectStatusesHelp = `jira project statuses

Purpose:
  Fetch project workflow statuses grouped by issue type.

Usage:
  jira project statuses <PROJECT-KEY> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: one row per issue type with status names.
  JSON: { "item": { "projectKey": "...", "issueTypes": [...] }, "schema": { "itemType": "project-statuses", ... } }

Examples:
  jira project statuses SCWI
  jira project statuses SCWI --json`

const boardHelp = `jira board - explicit Agile board discovery commands

Usage:
  jira board <command> [flags]
  jira board <command> --help

Commands:
  list        List boards for a project
  get         Fetch one board by id

Examples:
  jira board list --project SCWI
  jira board get 123 --json`

const boardListHelp = `jira board list

Purpose:
  List boards through the Jira Agile API for one project.

Usage:
  jira board list --project <PROJECT-KEY> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --limit <n>            Max boards to return (default: 50)
  --project <key>        Project key or id to scope the board search
  --site <url>           Jira base URL override
  --start-at <n>         Offset for paginated results (default: 0)
  --token <value>        Jira API token override
  --type <value>         Optional board type, such as scrum or kanban
  -h, --help             Show this help

Output:
  Text: one compact row per board plus pagination summary.
  JSON: { "items": [...], "page": { ... }, "schema": { "itemType": "board-summary", ... } }

Examples:
  jira board list --project SCWI
  jira board list --project SCWI --type scrum --json`

const boardGetHelp = `jira board get

Purpose:
  Fetch one board by id.

Usage:
  jira board get <BOARD-ID> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: compact key/value summary.
  JSON: { "item": { ... }, "schema": { "itemType": "board-detail", ... } }

Examples:
  jira board get 123
  jira board get 123 --json`

const filterHelp = `jira filter - explicit saved-filter discovery commands

Usage:
  jira filter <command> [flags]
  jira filter <command> --help

Commands:
  list        List saved filters with bounded output
  get         Fetch one saved filter by id

Examples:
  jira filter list --limit 25
  jira filter get 10001 --json`

const filterListHelp = `jira filter list

Purpose:
  List saved filters that can guide agent research.

Usage:
  jira filter list [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --limit <n>            Max filters to return (default: 50)
  --search <value>       Optional filter name search string
  --site <url>           Jira base URL override
  --start-at <n>         Offset for paginated results (default: 0)
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: one compact row per filter plus pagination summary.
  JSON: { "items": [...], "page": { ... }, "schema": { "itemType": "filter-summary", ... } }

Examples:
  jira filter list
  jira filter list --search warehouse --json`

const filterGetHelp = `jira filter get

Purpose:
  Fetch one saved filter by id.

Usage:
  jira filter get <FILTER-ID> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: compact key/value summary including JQL.
  JSON: { "item": { ... }, "schema": { "itemType": "filter-detail", ... } }

Examples:
  jira filter get 10001
  jira filter get 10001 --json`

const fieldHelp = `jira field - explicit field discovery commands

Usage:
  jira field <command> [flags]
  jira field <command> --help

Commands:
  list        List fields with search and custom-field filtering
  get         Fetch one field by id or exact name

Examples:
  jira field list --custom-only --search warehouse
  jira field get customfield_10010 --json`

const fieldListHelp = `jira field list

Purpose:
  List Jira fields with bounded output so agents can discover custom fields safely.

Usage:
  jira field list [flags]

Flags:
  --config <path>        Optional config file path
  --custom-only          Restrict results to custom fields
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --limit <n>            Max fields to return (default: 50)
  --search <value>       Optional field search string
  --site <url>           Jira base URL override
  --start-at <n>         Offset for paginated results (default: 0)
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: one compact row per field plus pagination summary.
  JSON: { "items": [...], "page": { ... }, "schema": { "itemType": "field-summary", ... } }

Examples:
  jira field list --custom-only
  jira field list --search warehouse --limit 25 --json`

const fieldGetHelp = `jira field get

Purpose:
  Fetch one field by id or exact field name.

Usage:
  jira field get <FIELD-ID-OR-NAME> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print a stable JSON envelope
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Output:
  Text: compact key/value summary.
  JSON: { "item": { ... }, "schema": { "itemType": "field-detail", ... } }

Examples:
  jira field get customfield_10010
  jira field get "Warehouse Slot" --json`

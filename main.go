package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const packageVersion = "0.1.1"

const rootHelp = `jira - agent-first Jira CLI for deterministic automation

Usage:
  jira <command> [flags]
  jira <command> --help

Commands:
  issue        Read Jira issues through explicit subcommands
  me           Print the authenticated Jira user
  serverinfo   Print Jira server metadata
  version      Print the CLI version

Contract:
  - non-interactive only
  - primary results on stdout
  - diagnostics and errors on stderr
  - explicit flags over hidden defaults
  - per-command --json for structured output
  - config resolution is flags > env > config file

Examples:
  jira me --json
  jira serverinfo --json
  jira issue get SCWI-282 --fields summary,status --json
  jira issue search --project SCWI --status "To Do" --json
  jira issue search --jql 'project = SCWI ORDER BY updated DESC' --json`

const resolutionHelp = `Resolution:
  Flags override env vars.
  Env vars override the config file.
  Supported env vars: JIRA_SITE, JIRA_BASE_URL, JIRA_EMAIL, JIRA_TOKEN, JIRA_API_TOKEN, JIRA_CONFIG.
  Default config path: ~/.config/jira/config.json`

const issueHelp = `jira issue - explicit issue read commands

Usage:
  jira issue <command> [flags]
  jira issue <command> --help

Commands:
  get         Fetch one issue by key
  search      Search issues with explicit filters or raw JQL

Examples:
  jira issue get SCWI-282 --json
  jira issue search --project SCWI --assignee currentUser() --json
  jira issue search --jql 'project = SCWI ORDER BY updated DESC' --json`

const meHelp = `jira me

Usage:
  jira me [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print machine-readable JSON
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Resolution:
` + resolutionHelp + `

Examples:
  jira me --json
  jira me --site https://example.atlassian.net --email agent@example.com --token "$JIRA_API_TOKEN" --json`

const serverInfoHelp = `jira serverinfo

Usage:
  jira serverinfo [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --json                 Print machine-readable JSON
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Resolution:
` + resolutionHelp + `

Examples:
  jira serverinfo --json
  jira serverinfo --site https://example.atlassian.net --json`

const issueGetHelp = `jira issue get

Usage:
  jira issue get <ISSUE-KEY> [flags]

Flags:
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --fields <names>       Comma-separated field list to request (default: summary,status,assignee)
  --json                 Print machine-readable JSON
  --site <url>           Jira base URL override
  --token <value>        Jira API token override
  -h, --help             Show this help

Resolution:
` + resolutionHelp + `

Examples:
  jira issue get SCWI-282
  jira issue get SCWI-282 --fields summary,status,assignee --json`

const issueSearchHelp = `jira issue search

Usage:
  jira issue search [flags]

Flags:
  --assignee <value>     Filter by assignee value
  --config <path>        Optional config file path
  --email <value>        Jira user email override
  --fields <names>       Comma-separated field list to request (default: summary,status,assignee)
  --jql <query>          Raw JQL to execute
  --json                 Print machine-readable JSON
  --limit <n>            Max issues to return (default: 50)
  --project <key>        Filter by project key
  --site <url>           Jira base URL override
  --status <name>        Filter by status (repeatable)
  --token <value>        Jira API token override
  -h, --help             Show this help

Resolution:
` + resolutionHelp + `

Notes:
  Use either explicit filters or --jql in one call.
  Explicit filter mode builds a literal JQL query and appends ORDER BY updated DESC.

Examples:
  jira issue search --project SCWI --status "To Do" --json
  jira issue search --assignee currentUser() --fields key,summary,status --json
  jira issue search --jql 'project = SCWI ORDER BY updated DESC' --json`

const versionHelp = `jira version

Usage:
  jira version

Examples:
  jira version`

type cliEnvironment struct {
	stderr io.Writer
	stdout io.Writer
}

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type commandOptions struct {
	configPath string
	email      string
	json       bool
	site       string
	token      string
}

type issueGetOptions struct {
	commandOptions
	fields string
	key    string
}

type issueSearchOptions struct {
	commandOptions
	assignee string
	fields   string
	jql      string
	limit    int
	project  string
	statuses []string
}

var configEnvironmentFactory = defaultConfigEnvironment
var jiraAPIFactory = func(config resolvedRuntimeConfig) (jiraAPI, error) {
	return newJiraClient(config), nil
}

func main() {
	os.Exit(run(os.Args[1:], cliEnvironment{
		stderr: os.Stderr,
		stdout: os.Stdout,
	}))
}

func run(argv []string, env cliEnvironment) int {
	if len(argv) == 0 || isHelpFlag(argv[0]) {
		_, _ = fmt.Fprintln(env.stdout, rootHelp)
		return 0
	}

	switch argv[0] {
	case "version":
		if len(argv) > 1 && isHelpFlag(argv[1]) {
			_, _ = fmt.Fprintln(env.stdout, versionHelp)
			return 0
		}
		_, _ = fmt.Fprintln(env.stdout, packageVersion)
		return 0
	case "--version":
		if len(argv) > 1 && isHelpFlag(argv[1]) {
			_, _ = fmt.Fprintln(env.stdout, versionHelp)
			return 0
		}
		_, _ = fmt.Fprintln(env.stdout, packageVersion)
		return 0
	case "issue":
		if err := runIssue(argv[1:], env); err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: %s\n", err)
			return 1
		}
		return 0
	case "me":
		if err := runMe(argv[1:], env); err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: %s\n", err)
			return 1
		}
		return 0
	case "serverinfo":
		if err := runServerInfo(argv[1:], env); err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: %s\n", err)
			return 1
		}
		return 0
	default:
		_, _ = fmt.Fprintf(env.stderr, "Error: unknown command: %s\n\n%s\n", argv[0], rootHelp)
		return 1
	}
}

func runIssue(argv []string, env cliEnvironment) error {
	if len(argv) == 0 || isHelpFlag(argv[0]) {
		_, _ = fmt.Fprintln(env.stdout, issueHelp)
		return nil
	}

	switch argv[0] {
	case "get":
		return runIssueGet(argv[1:], env)
	case "search":
		return runIssueSearch(argv[1:], env)
	default:
		return fmt.Errorf("unknown issue command: %s", argv[0])
	}
}

func runMe(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseCommandOptions("me", argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, meHelp)
		return nil
	}

	resolved, err := resolveRuntimeConfig(options, configEnvironmentFactory())
	if err != nil {
		return err
	}
	if err := resolved.Validate(configRequirements{requireSite: true, requireEmail: true, requireToken: true}); err != nil {
		return err
	}
	client, err := jiraAPIFactory(resolved)
	if err != nil {
		return err
	}
	response, err := client.GetMyself(context.Background())
	if err != nil {
		return err
	}
	return writeOutput(response, options.json, renderMeText(response), env)
}

func runServerInfo(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseCommandOptions("serverinfo", argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, serverInfoHelp)
		return nil
	}

	resolved, err := resolveRuntimeConfig(options, configEnvironmentFactory())
	if err != nil {
		return err
	}
	if err := resolved.Validate(configRequirements{requireSite: true}); err != nil {
		return err
	}
	client, err := jiraAPIFactory(resolved)
	if err != nil {
		return err
	}
	response, err := client.GetServerInfo(context.Background())
	if err != nil {
		return err
	}
	return writeOutput(response, options.json, renderServerInfoText(response), env)
}

func runIssueGet(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseIssueGetOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, issueGetHelp)
		return nil
	}

	resolved, err := resolveRuntimeConfig(options.commandOptions, configEnvironmentFactory())
	if err != nil {
		return err
	}
	if err := resolved.Validate(configRequirements{requireSite: true, requireEmail: true, requireToken: true}); err != nil {
		return err
	}
	client, err := jiraAPIFactory(resolved)
	if err != nil {
		return err
	}
	fields := splitCommaList(options.fields)
	response, err := client.GetIssue(context.Background(), options.key, fields)
	if err != nil {
		return err
	}
	return writeOutput(response, options.json, renderIssueText(response, fields), env)
}

func runIssueSearch(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseIssueSearchOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, issueSearchHelp)
		return nil
	}

	resolved, err := resolveRuntimeConfig(options.commandOptions, configEnvironmentFactory())
	if err != nil {
		return err
	}
	if err := resolved.Validate(configRequirements{requireSite: true, requireEmail: true, requireToken: true}); err != nil {
		return err
	}
	client, err := jiraAPIFactory(resolved)
	if err != nil {
		return err
	}
	fields := splitCommaList(options.fields)
	jql := options.jql
	if jql == "" {
		jql, err = buildSearchJQL(options)
		if err != nil {
			return err
		}
	}
	response, err := client.SearchIssues(context.Background(), jiraSearchRequest{
		Fields:     fields,
		JQL:        jql,
		MaxResults: options.limit,
	})
	if err != nil {
		return err
	}
	return writeOutput(response, options.json, renderSearchText(response, fields), env)
}

func parseCommandOptions(name string, argv []string) (commandOptions, bool, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var options commandOptions
	var help bool

	flags.StringVar(&options.configPath, "config", "", "Optional config file path")
	flags.StringVar(&options.email, "email", "", "Jira user email override")
	flags.BoolVar(&options.json, "json", false, "Print machine-readable JSON")
	flags.StringVar(&options.site, "site", "", "Jira base URL override")
	flags.StringVar(&options.token, "token", "", "Jira API token override")
	flags.BoolVar(&help, "help", false, "Show help")
	flags.BoolVar(&help, "h", false, "Show help")

	if err := flags.Parse(argv); err != nil {
		return commandOptions{}, false, normalizeFlagError(name, err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return commandOptions{}, false, fmt.Errorf("%s does not accept positional arguments", name)
	}

	return options, false, nil
}

func parseIssueGetOptions(argv []string) (issueGetOptions, bool, error) {
	flags := flag.NewFlagSet("issue get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	options := issueGetOptions{fields: "summary,status,assignee"}
	var help bool

	flags.StringVar(&options.configPath, "config", "", "Optional config file path")
	flags.StringVar(&options.email, "email", "", "Jira user email override")
	flags.StringVar(&options.fields, "fields", options.fields, "Comma-separated field list")
	flags.BoolVar(&options.json, "json", false, "Print machine-readable JSON")
	flags.StringVar(&options.site, "site", "", "Jira base URL override")
	flags.StringVar(&options.token, "token", "", "Jira API token override")
	flags.BoolVar(&help, "help", false, "Show help")
	flags.BoolVar(&help, "h", false, "Show help")

	normalized, err := normalizeInterspersedArgs(argv, map[string]bool{
		"--config": true,
		"--email":  true,
		"--fields": true,
		"--site":   true,
		"--token":  true,
	})
	if err != nil {
		return issueGetOptions{}, false, err
	}

	if err := flags.Parse(normalized); err != nil {
		return issueGetOptions{}, false, normalizeFlagError("issue get", err)
	}
	if help {
		return options, true, nil
	}
	args := flags.Args()
	if len(args) != 1 {
		return issueGetOptions{}, false, errors.New("issue get requires exactly one issue key")
	}
	options.key = args[0]
	return options, false, nil
}

func parseIssueSearchOptions(argv []string) (issueSearchOptions, bool, error) {
	flags := flag.NewFlagSet("issue search", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	options := issueSearchOptions{
		fields: "summary,status,assignee",
		limit:  50,
	}
	var statuses stringList
	var help bool

	flags.StringVar(&options.assignee, "assignee", "", "Filter by assignee")
	flags.StringVar(&options.configPath, "config", "", "Optional config file path")
	flags.StringVar(&options.email, "email", "", "Jira user email override")
	flags.StringVar(&options.fields, "fields", options.fields, "Comma-separated field list")
	flags.StringVar(&options.jql, "jql", "", "Raw JQL")
	flags.BoolVar(&options.json, "json", false, "Print machine-readable JSON")
	flags.IntVar(&options.limit, "limit", options.limit, "Max issues to return")
	flags.StringVar(&options.project, "project", "", "Project key")
	flags.StringVar(&options.site, "site", "", "Jira base URL override")
	flags.Var(&statuses, "status", "Filter by status")
	flags.StringVar(&options.token, "token", "", "Jira API token override")
	flags.BoolVar(&help, "help", false, "Show help")
	flags.BoolVar(&help, "h", false, "Show help")

	if err := flags.Parse(argv); err != nil {
		return issueSearchOptions{}, false, normalizeFlagError("issue search", err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return issueSearchOptions{}, false, errors.New("issue search does not accept positional arguments")
	}
	if options.limit <= 0 {
		return issueSearchOptions{}, false, errors.New("issue search requires --limit to be greater than 0")
	}
	if options.jql != "" && (options.project != "" || options.assignee != "" || len(statuses) > 0) {
		return issueSearchOptions{}, false, errors.New("issue search accepts either --jql or explicit filters, not both")
	}
	if options.jql == "" && options.project == "" && options.assignee == "" && len(statuses) == 0 {
		return issueSearchOptions{}, false, errors.New("issue search requires --jql or at least one explicit filter")
	}
	options.statuses = []string(statuses)
	return options, false, nil
}

func normalizeFlagError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", name, err)
}

func normalizeInterspersedArgs(argv []string, valueFlags map[string]bool) ([]string, error) {
	var flags []string
	var positionals []string

	for index := 0; index < len(argv); index++ {
		current := argv[index]
		if current == "--" {
			positionals = append(positionals, argv[index+1:]...)
			break
		}
		if strings.HasPrefix(current, "-") {
			flags = append(flags, current)
			if valueFlags[current] {
				if index+1 >= len(argv) {
					return nil, fmt.Errorf("missing value for %s", current)
				}
				index++
				flags = append(flags, argv[index])
			}
			continue
		}
		positionals = append(positionals, current)
	}

	return append(flags, positionals...), nil
}

func scaffoldOnlyError(command string) error {
	return fmt.Errorf("%s is not implemented yet; config resolution is ready but Jira transport is still pending", command)
}

func isHelpFlag(value string) bool {
	return value == "--help" || value == "-h"
}

func writeOutput(value any, jsonOutput bool, plainText string, env cliEnvironment) error {
	if jsonOutput {
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(env.stdout, string(encoded))
		return nil
	}

	_, _ = fmt.Fprintln(env.stdout, plainText)
	return nil
}

func splitCommaList(value string) []string {
	raw := strings.Split(value, ",")
	parts := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts = append(parts, item)
	}
	return parts
}

func buildSearchJQL(options issueSearchOptions) (string, error) {
	var clauses []string
	if options.project != "" {
		clauses = append(clauses, "project = "+quoteJQLValue(options.project))
	}
	if options.assignee != "" {
		clauses = append(clauses, "assignee = "+renderJQLOperand(options.assignee))
	}
	if len(options.statuses) == 1 {
		clauses = append(clauses, "status = "+quoteJQLValue(options.statuses[0]))
	}
	if len(options.statuses) > 1 {
		values := make([]string, 0, len(options.statuses))
		for _, status := range options.statuses {
			values = append(values, quoteJQLValue(status))
		}
		clauses = append(clauses, "status in ("+strings.Join(values, ", ")+")")
	}
	if len(clauses) == 0 {
		return "", errors.New("issue search could not build JQL from empty explicit filters")
	}
	return strings.Join(clauses, " AND ") + " ORDER BY updated DESC", nil
}

func renderJQLOperand(value string) string {
	if isLikelyJQLFunction(value) {
		return value
	}
	return quoteJQLValue(value)
}

func quoteJQLValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}

func isLikelyJQLFunction(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasSuffix(value, "()") || strings.Contains(value, " ") {
		return false
	}
	if len(value) < 3 {
		return false
	}
	for index, r := range value[:len(value)-2] {
		if index == 0 {
			if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') {
				return false
			}
			continue
		}
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
	}
	return true
}

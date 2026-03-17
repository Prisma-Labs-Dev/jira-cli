package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strings"
)

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

type paginationOptions struct {
	limit   int
	startAt int
}

type issueGetOptions struct {
	commandOptions
	fields string
	key    string
}

type issueSearchOptions struct {
	commandOptions
	paginationOptions
	assignee string
	fields   string
	jql      string
	project  string
	statuses []string
}

type issueCommentsOptions struct {
	commandOptions
	paginationOptions
	key string
}

type projectListOptions struct {
	commandOptions
	paginationOptions
	search string
}

type projectKeyOptions struct {
	commandOptions
	key string
}

type boardListOptions struct {
	commandOptions
	paginationOptions
	project   string
	boardType string
}

type boardGetOptions struct {
	commandOptions
	id string
}

type filterListOptions struct {
	commandOptions
	paginationOptions
	search string
}

type filterGetOptions struct {
	commandOptions
	id string
}

type fieldListOptions struct {
	commandOptions
	paginationOptions
	customOnly bool
	search     string
}

type fieldGetOptions struct {
	commandOptions
	identifier string
}

var functionTokenPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\(\)$`)

func runMe(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseCommandOptions("me", argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, meHelp)
		return nil
	}
	client, err := configuredJiraAPI(options, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.GetMyself(context.Background())
	if err != nil {
		return err
	}
	envelope := singleEnvelope[jiraMyselfResponse]{Item: response, Schema: meSchema()}
	return writeOutput(envelope, options.json, renderMeText(response), env)
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
	client, err := configuredJiraAPI(options, configRequirements{requireSite: true})
	if err != nil {
		return err
	}
	response, err := client.GetServerInfo(context.Background())
	if err != nil {
		return err
	}
	envelope := singleEnvelope[jiraServerInfoResponse]{Item: response, Schema: serverInfoSchema()}
	return writeOutput(envelope, options.json, renderServerInfoText(response), env)
}

func configuredJiraAPI(options commandOptions, requirements configRequirements) (jiraAPI, error) {
	resolved, err := resolveRuntimeConfig(options, configEnvironmentFactory())
	if err != nil {
		return nil, err
	}
	if err := resolved.Validate(requirements); err != nil {
		return nil, err
	}
	return jiraAPIFactory(resolved)
}

func parseCommandOptions(name string, argv []string) (commandOptions, bool, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options commandOptions
	var help bool
	addCommonFlags(flags, &options, &help)
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

func addCommonFlags(flags *flag.FlagSet, options *commandOptions, help *bool) {
	flags.StringVar(&options.configPath, "config", "", "Optional config file path")
	flags.StringVar(&options.email, "email", "", "Jira user email override")
	flags.BoolVar(&options.json, "json", false, "Print machine-readable JSON")
	flags.StringVar(&options.site, "site", "", "Jira base URL override")
	flags.StringVar(&options.token, "token", "", "Jira API token override")
	flags.BoolVar(help, "help", false, "Show help")
	flags.BoolVar(help, "h", false, "Show help")
}

func addPaginationFlags(flags *flag.FlagSet, options *paginationOptions) {
	flags.IntVar(&options.limit, "limit", options.limit, "Max results to return")
	flags.IntVar(&options.startAt, "start-at", options.startAt, "Result offset")
}

func validatePagination(name string, options paginationOptions) error {
	if options.limit <= 0 {
		return fmt.Errorf("%s requires --limit to be greater than 0", name)
	}
	if options.startAt < 0 {
		return fmt.Errorf("%s requires --start-at to be greater than or equal to 0", name)
	}
	return nil
}

func normalizeFlagError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", name, err)
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

func reorderSinglePositionalArgs(argv []string, valueFlags map[string]bool) []string {
	flagArgs := make([]string, 0, len(argv))
	positionals := make([]string, 0, 1)
	expectValue := false

	for _, arg := range argv {
		if expectValue {
			flagArgs = append(flagArgs, arg)
			expectValue = false
			continue
		}
		if arg == "--" {
			positionals = append(positionals, argv[len(flagArgs)+len(positionals)+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flagArgs = append(flagArgs, arg)
			flagName := arg
			if separator := strings.Index(flagName, "="); separator >= 0 {
				continue
			}
			if valueFlags[flagName] {
				expectValue = true
			}
			continue
		}
		positionals = append(positionals, arg)
	}

	return append(flagArgs, positionals...)
}

func quoteJQLValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return `""`
	}
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value
	}
	if functionTokenPattern.MatchString(value) {
		return value
	}
	if isSimpleJQLToken(value) {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func isSimpleJQLToken(value string) bool {
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-' || char == '.' {
			continue
		}
		return false
	}
	return true
}

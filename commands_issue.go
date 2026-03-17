package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

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
	case "comments":
		return runIssueComments(argv[1:], env)
	default:
		return fmt.Errorf("unknown issue command: %s", argv[0])
	}
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
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	fields := splitCommaList(options.fields)
	response, err := client.GetIssue(context.Background(), options.key, fields)
	if err != nil {
		return err
	}
	item := issueToItem(response, fields)
	envelope := singleEnvelope[issueItem]{Item: item, Schema: issueSchema("issue-detail", fields)}
	return writeOutput(envelope, options.json, renderIssueText(item, fields), env)
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
	fields := splitCommaList(options.fields)
	jql, err := resolveIssueSearchJQL(options)
	if err != nil {
		return err
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.SearchIssues(context.Background(), jiraIssueSearchRequest{
		JQL:     jql,
		Fields:  fields,
		Limit:   options.limit,
		StartAt: options.startAt,
	})
	if err != nil {
		return err
	}
	items := make([]issueItem, 0, len(response.Issues))
	for _, issue := range response.Issues {
		items = append(items, issueToItem(issue, fields))
	}
	page := buildPageFromTotal(response.StartAt, options.limit, len(items), response.Total)
	envelope := listEnvelope[issueItem]{Items: items, Page: page, Schema: issueSchema("issue-summary", fields)}
	return writeOutput(envelope, options.json, renderIssueListText(items, fields, page), env)
}

func runIssueComments(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseIssueCommentsOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, issueCommentsHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.GetIssueComments(context.Background(), options.key, options.startAt, options.limit)
	if err != nil {
		return err
	}
	items := make([]issueCommentItem, 0, len(response.Comments))
	for _, comment := range response.Comments {
		items = append(items, commentToItem(comment))
	}
	page := buildPageFromTotal(response.StartAt, options.limit, len(items), response.Total)
	envelope := listEnvelope[issueCommentItem]{Items: items, Page: page, Schema: issueCommentsSchema()}
	return writeOutput(envelope, options.json, renderIssueCommentsText(items, page), env)
}

func parseIssueGetOptions(argv []string) (issueGetOptions, bool, error) {
	flags := flag.NewFlagSet("issue get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := issueGetOptions{fields: defaultIssueGetFields}
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	flags.StringVar(&options.fields, "fields", options.fields, "Comma-separated field list")
	argv = reorderSinglePositionalArgs(argv, map[string]bool{
		"--config": true,
		"--email":  true,
		"--fields": true,
		"--site":   true,
		"--token":  true,
	})
	if err := flags.Parse(argv); err != nil {
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
	options := issueSearchOptions{fields: defaultIssueSearchFields, paginationOptions: paginationOptions{limit: defaultListLimit}}
	var statuses stringList
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	addPaginationFlags(flags, &options.paginationOptions)
	flags.StringVar(&options.assignee, "assignee", "", "Filter by assignee")
	flags.StringVar(&options.fields, "fields", options.fields, "Comma-separated field list")
	flags.StringVar(&options.jql, "jql", "", "Raw JQL")
	flags.StringVar(&options.project, "project", "", "Project key")
	flags.Var(&statuses, "status", "Filter by status")
	if err := flags.Parse(argv); err != nil {
		return issueSearchOptions{}, false, normalizeFlagError("issue search", err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return issueSearchOptions{}, false, errors.New("issue search does not accept positional arguments")
	}
	if err := validatePagination("issue search", options.paginationOptions); err != nil {
		return issueSearchOptions{}, false, err
	}
	if options.jql != "" && (options.project != "" || options.assignee != "" || len(statuses) > 0) {
		return issueSearchOptions{}, false, errors.New("issue search accepts either --jql or explicit filters, not both")
	}
	options.statuses = []string(statuses)
	return options, false, nil
}

func parseIssueCommentsOptions(argv []string) (issueCommentsOptions, bool, error) {
	flags := flag.NewFlagSet("issue comments", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := issueCommentsOptions{paginationOptions: paginationOptions{limit: defaultListLimit}}
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	addPaginationFlags(flags, &options.paginationOptions)
	argv = reorderSinglePositionalArgs(argv, map[string]bool{
		"--config":   true,
		"--email":    true,
		"--limit":    true,
		"--site":     true,
		"--start-at": true,
		"--token":    true,
	})
	if err := flags.Parse(argv); err != nil {
		return issueCommentsOptions{}, false, normalizeFlagError("issue comments", err)
	}
	if help {
		return options, true, nil
	}
	args := flags.Args()
	if len(args) != 1 {
		return issueCommentsOptions{}, false, errors.New("issue comments requires exactly one issue key")
	}
	if err := validatePagination("issue comments", options.paginationOptions); err != nil {
		return issueCommentsOptions{}, false, err
	}
	options.key = args[0]
	return options, false, nil
}

func resolveIssueSearchJQL(options issueSearchOptions) (string, error) {
	if strings.TrimSpace(options.jql) != "" {
		return strings.TrimSpace(options.jql), nil
	}
	var clauses []string
	if strings.TrimSpace(options.project) != "" {
		clauses = append(clauses, fmt.Sprintf("project = %s", quoteJQLValue(options.project)))
	}
	if len(options.statuses) == 1 {
		clauses = append(clauses, fmt.Sprintf("status = %s", quoteJQLValue(options.statuses[0])))
	}
	if len(options.statuses) > 1 {
		parts := make([]string, 0, len(options.statuses))
		for _, status := range options.statuses {
			parts = append(parts, quoteJQLValue(status))
		}
		clauses = append(clauses, fmt.Sprintf("status in (%s)", strings.Join(parts, ", ")))
	}
	if strings.TrimSpace(options.assignee) != "" {
		clauses = append(clauses, fmt.Sprintf("assignee = %s", quoteJQLValue(options.assignee)))
	}
	if len(clauses) == 0 {
		return "", errors.New("issue search requires --jql or at least one explicit filter")
	}
	return strings.Join(clauses, " AND "), nil
}

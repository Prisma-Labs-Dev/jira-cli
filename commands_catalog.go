package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

func runProject(argv []string, env cliEnvironment) error {
	if len(argv) == 0 || isHelpFlag(argv[0]) {
		_, _ = fmt.Fprintln(env.stdout, projectHelp)
		return nil
	}
	switch argv[0] {
	case "list":
		return runProjectList(argv[1:], env)
	case "get":
		return runProjectGet(argv[1:], env)
	case "statuses":
		return runProjectStatuses(argv[1:], env)
	default:
		return fmt.Errorf("unknown project command: %s", argv[0])
	}
}

func runProjectList(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseProjectListOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, projectListHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.ListProjects(context.Background(), jiraProjectListRequest{Search: options.search, Limit: options.limit, StartAt: options.startAt})
	if err != nil {
		return err
	}
	items := make([]projectItem, 0, len(response.Values))
	for _, project := range response.Values {
		items = append(items, projectToItem(project))
	}
	page := buildPageFromTotal(response.StartAt, options.limit, len(items), response.Total)
	envelope := listEnvelope[projectItem]{Items: items, Page: page, Schema: projectSchema("project-summary")}
	return writeOutput(envelope, options.json, renderProjectListText(items, page), env)
}

func runProjectGet(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseProjectKeyOptions("project get", argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, projectGetHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.GetProject(context.Background(), options.key)
	if err != nil {
		return err
	}
	item := projectToItem(response)
	envelope := singleEnvelope[projectItem]{Item: item, Schema: projectSchema("project-detail")}
	return writeOutput(envelope, options.json, renderProjectText(item), env)
}

func runProjectStatuses(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseProjectKeyOptions("project statuses", argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, projectStatusesHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.GetProjectStatuses(context.Background(), options.key)
	if err != nil {
		return err
	}
	item := projectStatusesToItem(options.key, response)
	envelope := singleEnvelope[projectStatusesItem]{Item: item, Schema: projectStatusesSchema()}
	return writeOutput(envelope, options.json, renderProjectStatusesText(item), env)
}

func runBoard(argv []string, env cliEnvironment) error {
	if len(argv) == 0 || isHelpFlag(argv[0]) {
		_, _ = fmt.Fprintln(env.stdout, boardHelp)
		return nil
	}
	switch argv[0] {
	case "list":
		return runBoardList(argv[1:], env)
	case "get":
		return runBoardGet(argv[1:], env)
	default:
		return fmt.Errorf("unknown board command: %s", argv[0])
	}
}

func runBoardList(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseBoardListOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, boardListHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.ListBoards(context.Background(), jiraBoardListRequest{Project: options.project, Type: options.boardType, Limit: options.limit, StartAt: options.startAt})
	if err != nil {
		return err
	}
	items := make([]boardItem, 0, len(response.Values))
	for _, board := range response.Values {
		items = append(items, boardToItem(board))
	}
	page := buildBoardPage(response, options.limit)
	envelope := listEnvelope[boardItem]{Items: items, Page: page, Schema: boardSchema("board-summary")}
	return writeOutput(envelope, options.json, renderBoardListText(items, page), env)
}

func runBoardGet(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseBoardGetOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, boardGetHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.GetBoard(context.Background(), options.id)
	if err != nil {
		return err
	}
	item := boardToItem(response)
	envelope := singleEnvelope[boardItem]{Item: item, Schema: boardSchema("board-detail")}
	return writeOutput(envelope, options.json, renderBoardText(item), env)
}

func runFilter(argv []string, env cliEnvironment) error {
	if len(argv) == 0 || isHelpFlag(argv[0]) {
		_, _ = fmt.Fprintln(env.stdout, filterHelp)
		return nil
	}
	switch argv[0] {
	case "list":
		return runFilterList(argv[1:], env)
	case "get":
		return runFilterGet(argv[1:], env)
	default:
		return fmt.Errorf("unknown filter command: %s", argv[0])
	}
}

func runFilterList(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseFilterListOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, filterListHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.ListFilters(context.Background(), jiraFilterListRequest{Search: options.search, Limit: options.limit, StartAt: options.startAt})
	if err != nil {
		return err
	}
	items := make([]filterItem, 0, len(response.Values))
	for _, filter := range response.Values {
		items = append(items, filterToItem(filter))
	}
	page := buildPageFromTotal(response.StartAt, options.limit, len(items), response.Total)
	envelope := listEnvelope[filterItem]{Items: items, Page: page, Schema: filterSchema("filter-summary")}
	return writeOutput(envelope, options.json, renderFilterListText(items, page), env)
}

func runFilterGet(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseFilterGetOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, filterGetHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.GetFilter(context.Background(), options.id)
	if err != nil {
		return err
	}
	item := filterToItem(response)
	envelope := singleEnvelope[filterItem]{Item: item, Schema: filterSchema("filter-detail")}
	return writeOutput(envelope, options.json, renderFilterText(item), env)
}

func runField(argv []string, env cliEnvironment) error {
	if len(argv) == 0 || isHelpFlag(argv[0]) {
		_, _ = fmt.Fprintln(env.stdout, fieldHelp)
		return nil
	}
	switch argv[0] {
	case "list":
		return runFieldList(argv[1:], env)
	case "get":
		return runFieldGet(argv[1:], env)
	default:
		return fmt.Errorf("unknown field command: %s", argv[0])
	}
}

func runFieldList(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseFieldListOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, fieldListHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.ListFields(context.Background(), jiraFieldListRequest{Search: options.search, CustomOnly: options.customOnly, Limit: options.limit, StartAt: options.startAt})
	if err != nil {
		return err
	}
	items := make([]fieldItem, 0, len(response.Values))
	for _, field := range response.Values {
		items = append(items, fieldToItem(field))
	}
	page := buildPageFromTotal(response.StartAt, options.limit, len(items), response.Total)
	envelope := listEnvelope[fieldItem]{Items: items, Page: page, Schema: fieldSchema("field-summary")}
	return writeOutput(envelope, options.json, renderFieldListText(items, page), env)
}

func runFieldGet(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseFieldGetOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, fieldGetHelp)
		return nil
	}
	client, err := configuredJiraAPI(options.commandOptions, configRequirements{requireSite: true, requireEmail: true, requireToken: true})
	if err != nil {
		return err
	}
	response, err := client.GetField(context.Background(), options.identifier)
	if err != nil {
		return err
	}
	item := fieldToItem(response)
	envelope := singleEnvelope[fieldItem]{Item: item, Schema: fieldSchema("field-detail")}
	return writeOutput(envelope, options.json, renderFieldText(item), env)
}

func parseProjectListOptions(argv []string) (projectListOptions, bool, error) {
	flags := flag.NewFlagSet("project list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := projectListOptions{paginationOptions: paginationOptions{limit: defaultListLimit}}
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	addPaginationFlags(flags, &options.paginationOptions)
	flags.StringVar(&options.search, "search", "", "Project search string")
	if err := flags.Parse(argv); err != nil {
		return projectListOptions{}, false, normalizeFlagError("project list", err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return projectListOptions{}, false, errors.New("project list does not accept positional arguments")
	}
	if err := validatePagination("project list", options.paginationOptions); err != nil {
		return projectListOptions{}, false, err
	}
	return options, false, nil
}

func parseProjectKeyOptions(name string, argv []string) (projectKeyOptions, bool, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options projectKeyOptions
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	argv = reorderSinglePositionalArgs(argv, map[string]bool{
		"--config": true,
		"--email":  true,
		"--site":   true,
		"--token":  true,
	})
	if err := flags.Parse(argv); err != nil {
		return projectKeyOptions{}, false, normalizeFlagError(name, err)
	}
	if help {
		return options, true, nil
	}
	args := flags.Args()
	if len(args) != 1 {
		return projectKeyOptions{}, false, fmt.Errorf("%s requires exactly one project key", name)
	}
	options.key = args[0]
	return options, false, nil
}

func parseBoardListOptions(argv []string) (boardListOptions, bool, error) {
	flags := flag.NewFlagSet("board list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := boardListOptions{paginationOptions: paginationOptions{limit: defaultListLimit}}
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	addPaginationFlags(flags, &options.paginationOptions)
	flags.StringVar(&options.project, "project", "", "Project key or id")
	flags.StringVar(&options.boardType, "type", "", "Board type")
	if err := flags.Parse(argv); err != nil {
		return boardListOptions{}, false, normalizeFlagError("board list", err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return boardListOptions{}, false, errors.New("board list does not accept positional arguments")
	}
	if strings.TrimSpace(options.project) == "" {
		return boardListOptions{}, false, errors.New("board list requires --project")
	}
	if err := validatePagination("board list", options.paginationOptions); err != nil {
		return boardListOptions{}, false, err
	}
	return options, false, nil
}

func parseBoardGetOptions(argv []string) (boardGetOptions, bool, error) {
	flags := flag.NewFlagSet("board get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options boardGetOptions
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	argv = reorderSinglePositionalArgs(argv, map[string]bool{
		"--config": true,
		"--email":  true,
		"--site":   true,
		"--token":  true,
	})
	if err := flags.Parse(argv); err != nil {
		return boardGetOptions{}, false, normalizeFlagError("board get", err)
	}
	if help {
		return options, true, nil
	}
	args := flags.Args()
	if len(args) != 1 {
		return boardGetOptions{}, false, errors.New("board get requires exactly one board id")
	}
	options.id = args[0]
	return options, false, nil
}

func parseFilterListOptions(argv []string) (filterListOptions, bool, error) {
	flags := flag.NewFlagSet("filter list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := filterListOptions{paginationOptions: paginationOptions{limit: defaultListLimit}}
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	addPaginationFlags(flags, &options.paginationOptions)
	flags.StringVar(&options.search, "search", "", "Filter name search string")
	if err := flags.Parse(argv); err != nil {
		return filterListOptions{}, false, normalizeFlagError("filter list", err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return filterListOptions{}, false, errors.New("filter list does not accept positional arguments")
	}
	if err := validatePagination("filter list", options.paginationOptions); err != nil {
		return filterListOptions{}, false, err
	}
	return options, false, nil
}

func parseFilterGetOptions(argv []string) (filterGetOptions, bool, error) {
	flags := flag.NewFlagSet("filter get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options filterGetOptions
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	argv = reorderSinglePositionalArgs(argv, map[string]bool{
		"--config": true,
		"--email":  true,
		"--site":   true,
		"--token":  true,
	})
	if err := flags.Parse(argv); err != nil {
		return filterGetOptions{}, false, normalizeFlagError("filter get", err)
	}
	if help {
		return options, true, nil
	}
	args := flags.Args()
	if len(args) != 1 {
		return filterGetOptions{}, false, errors.New("filter get requires exactly one filter id")
	}
	options.id = args[0]
	return options, false, nil
}

func parseFieldListOptions(argv []string) (fieldListOptions, bool, error) {
	flags := flag.NewFlagSet("field list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := fieldListOptions{paginationOptions: paginationOptions{limit: defaultListLimit}}
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	addPaginationFlags(flags, &options.paginationOptions)
	flags.BoolVar(&options.customOnly, "custom-only", false, "Only custom fields")
	flags.StringVar(&options.search, "search", "", "Field search string")
	if err := flags.Parse(argv); err != nil {
		return fieldListOptions{}, false, normalizeFlagError("field list", err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return fieldListOptions{}, false, errors.New("field list does not accept positional arguments")
	}
	if err := validatePagination("field list", options.paginationOptions); err != nil {
		return fieldListOptions{}, false, err
	}
	return options, false, nil
}

func parseFieldGetOptions(argv []string) (fieldGetOptions, bool, error) {
	flags := flag.NewFlagSet("field get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options fieldGetOptions
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	argv = reorderSinglePositionalArgs(argv, map[string]bool{
		"--config": true,
		"--email":  true,
		"--site":   true,
		"--token":  true,
	})
	if err := flags.Parse(argv); err != nil {
		return fieldGetOptions{}, false, normalizeFlagError("field get", err)
	}
	if help {
		return options, true, nil
	}
	args := flags.Args()
	if len(args) != 1 {
		return fieldGetOptions{}, false, errors.New("field get requires exactly one field id or exact field name")
	}
	options.identifier = args[0]
	return options, false, nil
}

func buildBoardPage(response jiraBoardPageResponse, limit int) pageDescriptor {
	var total *int
	if response.Total > 0 {
		totalCopy := response.Total
		total = &totalCopy
	}
	return buildPage(response.StartAt, limit, len(response.Values), total, !response.IsLast)
}

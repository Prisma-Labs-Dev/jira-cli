package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const (
	boardSnapshotIssueFieldsCSV = "summary,status,assignee,priority,issuetype,updated"
	boardSnapshotFetchPageSize  = 100
)

func runBoardSnapshot(argv []string, env cliEnvironment) error {
	options, helpRequested, err := parseBoardSnapshotOptions(argv)
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(env.stdout, boardSnapshotHelp)
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

	ctx := context.Background()
	boardID, err := resolveBoardSnapshotBoardID(ctx, client, resolved, options)
	if err != nil {
		return err
	}
	board, err := client.GetBoard(ctx, boardID)
	if err != nil {
		return err
	}

	snapshot, err := buildBoardSnapshot(ctx, client, board, options)
	if err != nil {
		return err
	}
	envelope := singleEnvelope[boardSnapshotItem]{Item: snapshot, Schema: boardSnapshotSchema()}
	return writeOutput(envelope, options.json, renderBoardSnapshotText(snapshot), env)
}

func parseBoardSnapshotOptions(argv []string) (boardSnapshotOptions, bool, error) {
	flags := flag.NewFlagSet("board snapshot", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := boardSnapshotOptions{paginationOptions: paginationOptions{limit: defaultListLimit}}
	var help bool
	addCommonFlags(flags, &options.commandOptions, &help)
	addPaginationFlags(flags, &options.paginationOptions)
	flags.StringVar(&options.boardID, "board", "", "Board id")
	flags.BoolVar(&options.defaultBoard, "default", false, "Use the configured default board")
	flags.BoolVar(&options.me, "me", false, "Include current-user issues for the board scope")
	flags.StringVar(&options.project, "project", "", "Project key or id used to resolve a single board")
	flags.StringVar(&options.boardType, "type", "", "Optional board type when resolving by project")
	if err := flags.Parse(argv); err != nil {
		return boardSnapshotOptions{}, false, normalizeFlagError("board snapshot", err)
	}
	if help {
		return options, true, nil
	}
	if len(flags.Args()) > 0 {
		return boardSnapshotOptions{}, false, errors.New("board snapshot does not accept positional arguments")
	}
	if err := validatePagination("board snapshot", options.paginationOptions); err != nil {
		return boardSnapshotOptions{}, false, err
	}

	selectorCount := 0
	if strings.TrimSpace(options.boardID) != "" {
		selectorCount++
	}
	if options.defaultBoard {
		selectorCount++
	}
	if strings.TrimSpace(options.project) != "" {
		selectorCount++
	}
	if selectorCount != 1 {
		return boardSnapshotOptions{}, false, errors.New("board snapshot requires exactly one of --board, --project, or --default")
	}
	if strings.TrimSpace(options.boardType) != "" && strings.TrimSpace(options.project) == "" {
		return boardSnapshotOptions{}, false, errors.New("board snapshot accepts --type only together with --project")
	}
	return options, false, nil
}

func resolveBoardSnapshotBoardID(ctx context.Context, client jiraAPI, resolved resolvedRuntimeConfig, options boardSnapshotOptions) (string, error) {
	if value := strings.TrimSpace(options.boardID); value != "" {
		return value, nil
	}
	if options.defaultBoard {
		if strings.TrimSpace(resolved.defaultBoardID) == "" {
			return "", errors.New("board snapshot --default requires JIRA_DEFAULT_BOARD or defaultBoardId in ~/.config/jira/config.json")
		}
		return resolved.defaultBoardID, nil
	}

	response, err := client.ListBoards(ctx, jiraBoardListRequest{Project: options.project, Type: options.boardType, Limit: defaultListLimit, StartAt: 0})
	if err != nil {
		return "", err
	}
	if len(response.Values) == 0 {
		return "", fmt.Errorf("board snapshot found no boards for project %s", options.project)
	}
	if len(response.Values) > 1 || !response.IsLast {
		return "", fmt.Errorf("board snapshot found multiple boards for project %s; use --board <id> or --type. candidates: %s", options.project, summarizeBoardCandidates(response.Values))
	}
	return strconv.Itoa(response.Values[0].ID), nil
}

func summarizeBoardCandidates(values []jiraBoardResponse) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, board := range values {
		label := fmt.Sprintf("%d:%s", board.ID, strings.TrimSpace(board.Name))
		if strings.TrimSpace(board.Type) != "" {
			label += fmt.Sprintf("(%s)", board.Type)
		}
		parts = append(parts, label)
		if len(parts) == 5 {
			break
		}
	}
	if len(values) > len(parts) {
		parts = append(parts, fmt.Sprintf("+%d more", len(values)-len(parts)))
	}
	return strings.Join(parts, ", ")
}

func buildBoardSnapshot(ctx context.Context, client jiraAPI, board jiraBoardResponse, options boardSnapshotOptions) (boardSnapshotItem, error) {
	requestedFields := boardSnapshotIssueFields()
	item := boardSnapshotItem{
		Board:        boardToItem(board),
		Source:       boardSnapshotSourceItem{Type: "board"},
		StatusCounts: []boardStatusCountItem{},
		Issues:       []issueItem{},
		Page:         buildPageFromTotal(options.startAt, options.limit, 0, 0),
		Totals:       boardSnapshotTotals{TotalIssues: 0},
	}

	fetchIssues := func(startAt, limit int) (jiraIssueSearchResponse, error) {
		return client.ListBoardIssues(ctx, strconv.Itoa(board.ID), jiraAgileIssueListRequest{Fields: requestedFields, Limit: limit, StartAt: startAt})
	}

	if strings.EqualFold(strings.TrimSpace(board.Type), "scrum") {
		sprints, err := client.ListBoardSprints(ctx, strconv.Itoa(board.ID), jiraSprintListRequest{States: []string{"active"}, Limit: 2, StartAt: 0})
		if err != nil {
			return boardSnapshotItem{}, err
		}
		if len(sprints.Values) > 0 {
			activeSprint := sprints.Values[0]
			item.Source = boardSnapshotSourceItem{Type: "active-sprint", Sprint: sprintToItem(activeSprint)}
			fetchIssues = func(startAt, limit int) (jiraIssueSearchResponse, error) {
				return client.ListSprintIssues(ctx, strconv.Itoa(activeSprint.ID), jiraAgileIssueListRequest{Fields: requestedFields, Limit: limit, StartAt: startAt})
			}
		}
	}

	allIssues, err := collectAgileIssues(fetchIssues)
	if err != nil {
		return boardSnapshotItem{}, err
	}
	item.StatusCounts = buildBoardStatusCounts(allIssues)
	item.Issues = paginateIssueItems(allIssues, options.startAt, options.limit, requestedFields)
	item.Page = buildPageFromTotal(options.startAt, options.limit, len(item.Issues), len(allIssues))
	item.Totals = boardSnapshotTotals{TotalIssues: len(allIssues)}

	if options.me {
		me, err := client.GetMyself(ctx)
		if err != nil {
			return boardSnapshotItem{}, err
		}
		item.Me = &boardSnapshotUserItem{AccountID: me.AccountID, DisplayName: me.DisplayName, EmailAddress: me.EmailAddress}
		item.MyIssues = filterMyIssueItems(allIssues, me, requestedFields)
		item.Totals.MyIssues = len(item.MyIssues)
	}

	return item, nil
}

func boardSnapshotIssueFields() []string {
	return splitCommaList(boardSnapshotIssueFieldsCSV)
}

func collectAgileIssues(fetchPage func(startAt, limit int) (jiraIssueSearchResponse, error)) ([]jiraIssueResponse, error) {
	all := make([]jiraIssueResponse, 0, boardSnapshotFetchPageSize)
	for startAt := 0; ; {
		response, err := fetchPage(startAt, boardSnapshotFetchPageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, response.Issues...)
		returned := len(response.Issues)
		if returned == 0 {
			break
		}
		if response.Total > 0 && response.StartAt+returned >= response.Total {
			break
		}
		if returned < boardSnapshotFetchPageSize {
			break
		}
		startAt = response.StartAt + returned
	}
	return all, nil
}

func buildBoardStatusCounts(issues []jiraIssueResponse) []boardStatusCountItem {
	counts := map[string]int{}
	for _, issue := range issues {
		status := issueStatusName(issue.Fields)
		if status == "" {
			status = "Unknown"
		}
		counts[status]++
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]boardStatusCountItem, 0, len(names))
	for _, name := range names {
		items = append(items, boardStatusCountItem{Name: name, Count: counts[name]})
	}
	return items
}

func issueStatusName(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}
	value, ok := fields["status"]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if name, ok := typed["name"].(string); ok {
			return strings.TrimSpace(name)
		}
	}
	return strings.TrimSpace(formatValue(normalizeCompactValue(value)))
}

func paginateIssueItems(all []jiraIssueResponse, startAt, limit int, requestedFields []string) []issueItem {
	if startAt >= len(all) {
		return []issueItem{}
	}
	end := startAt + limit
	if end > len(all) {
		end = len(all)
	}
	items := make([]issueItem, 0, end-startAt)
	for _, issue := range all[startAt:end] {
		items = append(items, issueToItem(issue, requestedFields))
	}
	return items
}

func filterMyIssueItems(all []jiraIssueResponse, me jiraMyselfResponse, requestedFields []string) []issueItem {
	items := make([]issueItem, 0)
	for _, issue := range all {
		if !issueAssignedToUser(issue.Fields, me) {
			continue
		}
		items = append(items, issueToItem(issue, requestedFields))
	}
	return items
}

func issueAssignedToUser(fields map[string]any, me jiraMyselfResponse) bool {
	if len(fields) == 0 {
		return false
	}
	value, ok := fields["assignee"]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case string:
		candidate := strings.TrimSpace(typed)
		return candidate != "" && (strings.EqualFold(candidate, me.DisplayName) || strings.EqualFold(candidate, me.EmailAddress))
	case map[string]any:
		for _, pair := range []struct {
			key      string
			expected string
		}{
			{key: "accountId", expected: me.AccountID},
			{key: "emailAddress", expected: me.EmailAddress},
			{key: "displayName", expected: me.DisplayName},
			{key: "name", expected: me.DisplayName},
		} {
			candidate, _ := typed[pair.key].(string)
			candidate = strings.TrimSpace(candidate)
			if candidate != "" && pair.expected != "" && strings.EqualFold(candidate, pair.expected) {
				return true
			}
		}
	}
	return false
}

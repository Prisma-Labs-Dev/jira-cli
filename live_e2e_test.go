package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestLiveCLIContracts(t *testing.T) {
	requireLiveConfig(t)

	restore := withLiveFactoriesForTest(t)
	defer restore()

	me := mustRunLiveJSON[singleEnvelope[jiraMyselfResponse]](t, []string{"me", "--json"})
	assertCanonicalGolden(t, "testdata/live-me-json.golden", map[string]any{
		"schema": me.Schema,
		"presence": map[string]bool{
			"accountId":   strings.TrimSpace(me.Item.AccountID) != "",
			"displayName": strings.TrimSpace(me.Item.DisplayName) != "",
			"active":      true,
		},
	})

	serverInfo := mustRunLiveJSON[singleEnvelope[jiraServerInfoResponse]](t, []string{"serverinfo", "--json"})
	assertCanonicalGolden(t, "testdata/live-serverinfo-json.golden", map[string]any{
		"schema": serverInfo.Schema,
		"presence": map[string]bool{
			"baseUrl":        strings.TrimSpace(serverInfo.Item.BaseURL) != "",
			"deploymentType": strings.TrimSpace(serverInfo.Item.DeploymentType) != "",
			"version":        strings.TrimSpace(serverInfo.Item.Version) != "",
		},
		"deploymentType": serverInfo.Item.DeploymentType,
	})

	projects := mustRunLiveJSON[listEnvelope[projectItem]](t, []string{"project", "list", "--limit", "10", "--json"})
	if len(projects.Items) == 0 {
		t.Fatal("expected at least one live project")
	}
	assertCanonicalGolden(t, "testdata/live-project-list-json.golden", map[string]any{
		"schema":          projects.Schema,
		"itemsReturned":   len(projects.Items),
		"pageLimit":       projects.Page.Limit,
		"nextHintPresent": strings.TrimSpace(projects.Page.NextHint) != "",
		"firstItem": map[string]bool{
			"key":  strings.TrimSpace(projects.Items[0].Key) != "",
			"name": strings.TrimSpace(projects.Items[0].Name) != "",
		},
	})

	projectKey := projects.Items[0].Key
	projectStatuses := mustRunLiveJSON[singleEnvelope[projectStatusesItem]](t, []string{"project", "statuses", projectKey, "--json"})
	assertCanonicalGolden(t, "testdata/live-project-statuses-json.golden", map[string]any{
		"schema":                  projectStatuses.Schema,
		"issueTypesCountPositive": len(projectStatuses.Item.IssueTypes) > 0,
		"firstIssueTypeHasStatuses": len(projectStatuses.Item.IssueTypes) > 0 &&
			len(projectStatuses.Item.IssueTypes[0].Statuses) > 0,
	})

	fields := mustRunLiveJSON[listEnvelope[fieldItem]](t, []string{"field", "list", "--limit", "1", "--json"})
	if len(fields.Items) == 0 {
		t.Fatal("expected at least one live field")
	}
	assertCanonicalGolden(t, "testdata/live-field-list-json.golden", map[string]any{
		"schema":          fields.Schema,
		"itemsReturned":   len(fields.Items),
		"pageLimit":       fields.Page.Limit,
		"nextHintPresent": strings.TrimSpace(fields.Page.NextHint) != "",
		"firstItem": map[string]bool{
			"id":   strings.TrimSpace(fields.Items[0].ID) != "",
			"name": strings.TrimSpace(fields.Items[0].Name) != "",
		},
	})

	filters := mustRunLiveJSON[listEnvelope[filterItem]](t, []string{"filter", "list", "--limit", "1", "--json"})
	if len(filters.Items) == 0 {
		t.Fatal("expected at least one live filter")
	}
	assertCanonicalGolden(t, "testdata/live-filter-list-json.golden", map[string]any{
		"schema":        filters.Schema,
		"itemsReturned": len(filters.Items),
		"pageLimit":     filters.Page.Limit,
		"firstItem": map[string]bool{
			"id":   strings.TrimSpace(filters.Items[0].ID) != "",
			"name": strings.TrimSpace(filters.Items[0].Name) != "",
		},
	})

	searchDiscovery := mustRunLiveJSON[listEnvelope[issueItem]](t, []string{"issue", "search", "--jql", "updated is not EMPTY ORDER BY updated DESC", "--limit", "10", "--fields", "summary,status,updated", "--json"})
	if len(searchDiscovery.Items) == 0 {
		t.Fatal("expected at least one live issue")
	}

	search := mustRunLiveJSON[listEnvelope[issueItem]](t, []string{"issue", "search", "--jql", "updated is not EMPTY ORDER BY updated DESC", "--limit", "1", "--fields", "summary,status,updated", "--json"})
	assertCanonicalGolden(t, "testdata/live-issue-search-json.golden", map[string]any{
		"schema":        search.Schema,
		"itemsReturned": len(search.Items),
		"pageLimit":     search.Page.Limit,
		"firstItem": map[string]bool{
			"key":            len(search.Items) > 0 && strings.TrimSpace(search.Items[0].Key) != "",
			"fields.summary": len(search.Items) > 0 && hasField(search.Items[0].Fields, "summary"),
			"fields.status":  len(search.Items) > 0 && hasField(search.Items[0].Fields, "status"),
			"fields.updated": len(search.Items) > 0 && hasField(search.Items[0].Fields, "updated"),
		},
	})

	issueKey := searchDiscovery.Items[0].Key
	issue := mustRunLiveJSON[singleEnvelope[issueItem]](t, []string{"issue", "get", issueKey, "--fields", "summary,status,updated", "--json"})
	assertCanonicalGolden(t, "testdata/live-issue-get-json.golden", map[string]any{
		"schema": issue.Schema,
		"presence": map[string]bool{
			"key":            strings.TrimSpace(issue.Item.Key) != "",
			"fields.summary": hasField(issue.Item.Fields, "summary"),
			"fields.status":  hasField(issue.Item.Fields, "status"),
			"fields.updated": hasField(issue.Item.Fields, "updated"),
		},
	})

	if boardProject, ok := findProjectWithBoard(t, projects.Items); ok {
		boards := mustRunLiveJSON[listEnvelope[boardItem]](t, []string{"board", "list", "--project", boardProject, "--limit", "1", "--json"})
		if len(boards.Items) > 0 {
			assertCanonicalGolden(t, "testdata/live-board-list-json.golden", map[string]any{
				"schema":        boards.Schema,
				"itemsReturned": len(boards.Items),
				"pageLimit":     boards.Page.Limit,
				"firstItem": map[string]bool{
					"id":   boards.Items[0].ID != 0,
					"name": strings.TrimSpace(boards.Items[0].Name) != "",
					"type": strings.TrimSpace(boards.Items[0].Type) != "",
				},
			})

			snapshot := mustRunLiveJSON[singleEnvelope[boardSnapshotItem]](t, []string{"board", "snapshot", "--board", fmt.Sprintf("%d", boards.Items[0].ID), "--limit", "10", "--me", "--json"})
			assertCanonicalGolden(t, "testdata/live-board-snapshot-json.golden", map[string]any{
				"schema": snapshot.Schema,
				"board": map[string]bool{
					"id":   snapshot.Item.Board.ID != 0,
					"name": strings.TrimSpace(snapshot.Item.Board.Name) != "",
					"type": strings.TrimSpace(snapshot.Item.Board.Type) != "",
				},
				"sourceTypeKnown": snapshot.Item.Source.Type == "active-sprint" || snapshot.Item.Source.Type == "board",
				"totals": map[string]bool{
					"totalIssuesNonNegative": snapshot.Item.Totals.TotalIssues >= 0,
					"myIssuesNonNegative":    snapshot.Item.Totals.MyIssues >= 0,
				},
				"statusCountsPresent": snapshot.Item.StatusCounts != nil,
				"pageLimit":           snapshot.Item.Page.Limit,
				"meIncluded":          snapshot.Item.Me != nil,
			})
		}
	}

	if issueWithComments, ok := findIssueWithComments(t, searchDiscovery.Items); ok {
		comments := mustRunLiveJSON[listEnvelope[issueCommentItem]](t, []string{"issue", "comments", issueWithComments, "--limit", "1", "--json"})
		if len(comments.Items) > 0 {
			assertCanonicalGolden(t, "testdata/live-issue-comments-json.golden", map[string]any{
				"schema":        comments.Schema,
				"itemsReturned": len(comments.Items),
				"pageLimit":     comments.Page.Limit,
				"firstItem": map[string]bool{
					"id":       strings.TrimSpace(comments.Items[0].ID) != "",
					"bodyText": strings.TrimSpace(comments.Items[0].BodyText) != "",
				},
			})
		}
	}
}

func requireLiveConfig(t *testing.T) {
	t.Helper()
	if os.Getenv("JIRA_LIVE_E2E") != "1" {
		t.Skip("set JIRA_LIVE_E2E=1 to run live Jira contract tests")
	}
	var missing []string
	for _, key := range []string{"JIRA_SITE", "JIRA_EMAIL", "JIRA_TOKEN"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		t.Skipf("missing live Jira env: %s", strings.Join(missing, ", "))
	}
}

func withLiveFactoriesForTest(t *testing.T) func() {
	t.Helper()
	originalConfigFactory := configEnvironmentFactory
	originalAPIFactory := jiraAPIFactory
	configEnvironmentFactory = defaultConfigEnvironment
	jiraAPIFactory = func(config resolvedRuntimeConfig) (jiraAPI, error) {
		return newJiraClient(config), nil
	}
	return func() {
		configEnvironmentFactory = originalConfigFactory
		jiraAPIFactory = originalAPIFactory
	}
}

func mustRunLiveJSON[T any](t *testing.T, argv []string) T {
	t.Helper()
	stdoutBuffer := &strings.Builder{}
	stderrBuffer := &strings.Builder{}
	exitCode := run(argv, cliEnvironment{stderr: stderrBuffer, stdout: stdoutBuffer})
	if exitCode != 0 {
		t.Fatalf("live command failed: %s\nstderr=%s", strings.Join(argv, " "), strings.TrimSpace(stderrBuffer.String()))
	}
	var decoded T
	if err := json.Unmarshal([]byte(stdoutBuffer.String()), &decoded); err != nil {
		t.Fatalf("decode live JSON for %s: %v\nstdout=%s", strings.Join(argv, " "), err, stdoutBuffer.String())
	}
	return decoded
}

func assertCanonicalGolden(t *testing.T, path string, value any) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal canonical value: %v", err)
	}
	var canonical any
	if err := json.Unmarshal(encoded, &canonical); err != nil {
		t.Fatalf("decode canonical value: %v", err)
	}
	encoded, err = json.MarshalIndent(canonical, "", "  ")
	if err != nil {
		t.Fatalf("marshal canonical value: %v", err)
	}
	assertGolden(t, path, string(encoded))
}

func hasField(fields map[string]any, key string) bool {
	if len(fields) == 0 {
		return false
	}
	value, ok := fields[key]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case []string:
		return len(typed) > 0
	default:
		return value != nil
	}
}

func findProjectWithBoard(t *testing.T, projects []projectItem) (string, bool) {
	t.Helper()
	preferredKeys := make([]string, 0, len(projects))
	for _, project := range projects {
		if project.Key == "SCWI" {
			preferredKeys = append([]string{"SCWI"}, preferredKeys...)
			continue
		}
		preferredKeys = append(preferredKeys, project.Key)
	}
	for _, key := range preferredKeys {
		boards := mustRunLiveJSON[listEnvelope[boardItem]](t, []string{"board", "list", "--project", key, "--limit", "1", "--json"})
		if len(boards.Items) > 0 {
			return key, true
		}
	}
	return "", false
}

func findIssueWithComments(t *testing.T, issues []issueItem) (string, bool) {
	t.Helper()
	for _, issue := range issues {
		comments := mustRunLiveJSON[listEnvelope[issueCommentItem]](t, []string{"issue", "comments", issue.Key, "--limit", "1", "--json"})
		if len(comments.Items) > 0 {
			return issue.Key, true
		}
	}
	return "", false
}

func TestLiveCLIContractsDocumentation(t *testing.T) {
	if os.Getenv("JIRA_LIVE_E2E") != "1" {
		t.Skip("documentation smoke test only when live mode is requested")
	}
	if strings.TrimSpace(os.Getenv("JIRA_SITE")) == "" {
		t.Skip("missing JIRA_SITE")
	}
	if !strings.Contains(rootHelp, "compact default output") {
		t.Fatalf("expected root help to mention compact output")
	}
	if !strings.Contains(fmt.Sprint(serverInfoSchema().Fields), "deploymentType") {
		t.Fatalf("expected server info schema to include deploymentType")
	}
}

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelpSurfacesMatchGolden(t *testing.T) {
	cases := []struct {
		name       string
		argv       []string
		goldenPath string
	}{
		{name: "root", argv: nil, goldenPath: "testdata/root-help.golden"},
		{name: "issue-search-help", argv: []string{"issue", "search", "--help"}, goldenPath: "testdata/issue-search-help.golden"},
		{name: "board-snapshot-help", argv: []string{"board", "snapshot", "--help"}, goldenPath: "testdata/board-snapshot-help.golden"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exitCode := runForTest(tc.argv)
			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			assertGolden(t, tc.goldenPath, stdout)
		})
	}
}

func TestRunIssueGetRequiresKey(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"issue", "get"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "issue get requires exactly one issue key") {
		t.Fatalf("expected missing key error, got %q", stderr)
	}
}

func TestRunIssueSearchRejectsMixedQueryModes(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"issue", "search", "--jql", "project = SCWI", "--project", "SCWI"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "issue search accepts either --jql or explicit filters, not both") {
		t.Fatalf("expected mixed mode error, got %q", stderr)
	}
}

func TestRunIssueSearchRequiresQueryMode(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"issue", "search"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "issue search requires --jql or at least one explicit filter") {
		t.Fatalf("expected query mode error, got %q", stderr)
	}
}

func TestRunIssueSearchPrintsJSONEnvelope(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		searchIssues: jiraIssueSearchResponse{
			StartAt:    0,
			MaxResults: 50,
			Total:      1,
			Issues: []jiraIssueResponse{{
				ID:  "10001",
				Key: "SCWI-282",
				Fields: map[string]any{
					"summary":  "Implement thing",
					"status":   map[string]any{"name": "In Progress"},
					"assignee": map[string]any{"displayName": "Agent User"},
					"priority": map[string]any{"name": "High"},
					"updated":  "2026-03-17T09:00:00.000Z",
				},
			}},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "search", "--project", "SCWI", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertGolden(t, "testdata/issue-search-json.golden", stdout)
}

func TestRunIssueSearchTextIncludesPaginationHint(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		searchIssues: jiraIssueSearchResponse{
			StartAt:    1,
			MaxResults: 2,
			Total:      4,
			Issues: []jiraIssueResponse{
				{Key: "SCWI-282", Fields: map[string]any{"summary": "Implement thing", "status": map[string]any{"name": "In Progress"}, "assignee": map[string]any{"displayName": "Agent User"}, "priority": map[string]any{"name": "High"}, "updated": "2026-03-17T09:00:00.000Z"}},
				{Key: "SCWI-283", Fields: map[string]any{"summary": "Ship thing", "status": map[string]any{"name": "Done"}, "assignee": map[string]any{"displayName": "Agent User"}, "priority": map[string]any{"name": "Medium"}, "updated": "2026-03-17T10:00:00.000Z"}},
			},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "search", "--project", "SCWI", "--limit", "2", "--start-at", "1"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "nextStartAt=3") {
		t.Fatalf("expected pagination hint, got %q", stdout)
	}
}

func TestRunIssueCommentsPrintsJSONEnvelope(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		comments: jiraCommentPageResponse{
			StartAt:    0,
			MaxResults: 1,
			Total:      2,
			Comments: []jiraCommentResponse{{
				ID:      "20001",
				Author:  &jiraUserRef{DisplayName: "Agent Reviewer"},
				Created: "2026-03-17T09:00:00.000Z",
				Updated: "2026-03-17T09:30:00.000Z",
				Body:    map[string]any{"text": "First comment"},
			}},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "comments", "SCWI-282", "--limit", "1", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "issue-comment"`) || !strings.Contains(stdout, `"nextHint": "use --start-at 1"`) {
		t.Fatalf("expected issue comments envelope, got %q", stdout)
	}
}

func TestRunIssueCommentsAcceptsFlagsAfterIssueKey(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		comments: jiraCommentPageResponse{
			StartAt:    0,
			MaxResults: 1,
			Total:      1,
			Comments: []jiraCommentResponse{{
				ID:      "20001",
				Created: "2026-03-17T09:00:00.000Z",
				Body:    map[string]any{"text": "First comment"},
			}},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "comments", "SCWI-282", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "issue-comment"`) {
		t.Fatalf("expected JSON envelope, got %q", stdout)
	}
}

func TestRunIssueGetAcceptsFlagsAfterIssueKey(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		issue: jiraIssueResponse{
			Key: "SCWI-282",
			Fields: map[string]any{
				"summary": "Implement thing",
				"status":  map[string]any{"name": "In Progress"},
			},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "get", "SCWI-282", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "issue-detail"`) {
		t.Fatalf("expected JSON envelope, got %q", stdout)
	}
}

func TestRunProjectListPrintsJSONEnvelope(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		projects: jiraProjectPageResponse{
			StartAt:    0,
			MaxResults: 50,
			Total:      1,
			Values:     []jiraProjectResponse{{Key: "SCWI", Name: "Warehouse Inbound", ProjectTypeKey: "software", Style: "classic", Lead: &jiraUserRef{DisplayName: "Agent Lead"}}},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"project", "list", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertGolden(t, "testdata/project-list-json.golden", stdout)
}

func TestRunProjectGetAcceptsFlagsAfterProjectKey(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		project: jiraProjectResponse{ID: "10000", Key: "SCWI", Name: "Warehouse Inbound", ProjectTypeKey: "software", Style: "classic"},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"project", "get", "SCWI", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "project-detail"`) {
		t.Fatalf("expected JSON envelope, got %q", stdout)
	}
}

func TestRunProjectStatusesPrintsText(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		statuses: []jiraProjectIssueTypeStatusesResponse{{
			Name: "Bug",
			Statuses: []jiraStatusRefResponse{
				{Name: "To Do"},
				{Name: "Done"},
			},
		}},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"project", "statuses", "SCWI"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Bug: To Do, Done") {
		t.Fatalf("expected statuses text, got %q", stdout)
	}
}

func TestRunProjectStatusesAcceptsFlagsAfterProjectKey(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		statuses: []jiraProjectIssueTypeStatusesResponse{{
			Name:     "Bug",
			Statuses: []jiraStatusRefResponse{{Name: "To Do"}},
		}},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"project", "statuses", "SCWI", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "project-statuses"`) {
		t.Fatalf("expected JSON envelope, got %q", stdout)
	}
}

func TestRunBoardListRequiresProject(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"board", "list"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "board list requires --project") {
		t.Fatalf("expected missing project error, got %q", stderr)
	}
}

func TestRunBoardGetRequiresID(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"board", "get"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "board get requires exactly one board id") {
		t.Fatalf("expected missing board id error, got %q", stderr)
	}
}

func TestRunBoardGetAcceptsFlagsAfterBoardID(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		board: jiraBoardResponse{ID: 7, Name: "Warehouse Board", Type: "scrum", Location: &jiraBoardLocationResponse{ProjectKey: "SCWI", ProjectName: "Warehouse"}},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"board", "get", "7", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "board-detail"`) {
		t.Fatalf("expected JSON envelope, got %q", stdout)
	}
}

func TestRunFilterGetPrintsJSONEnvelope(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		filter: jiraFilterResponse{ID: "10001", Name: "Warehouse Work", JQL: "project = SCWI", Owner: &jiraUserRef{DisplayName: "Agent User"}},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"filter", "get", "10001", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "filter-detail"`) || !strings.Contains(stdout, `"jql": "project = SCWI"`) {
		t.Fatalf("expected filter envelope, got %q", stdout)
	}
}

func TestRunFilterGetRequiresID(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"filter", "get"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "filter get requires exactly one filter id") {
		t.Fatalf("expected missing filter id error, got %q", stderr)
	}
}

func TestRunFieldGetAcceptsFlagsAfterIdentifier(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		field: jiraFieldResponse{ID: "customfield_10010", Name: "Warehouse Slot", Custom: true, Searchable: true, Orderable: true, Schema: &jiraFieldSchema{Type: "string"}},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"field", "get", "customfield_10010", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "field-detail"`) {
		t.Fatalf("expected JSON envelope, got %q", stdout)
	}
}

func TestRunFieldListTextIsCompact(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		fields: jiraFieldPageResponse{
			StartAt:    0,
			MaxResults: 50,
			Total:      1,
			Values:     []jiraFieldResponse{{ID: "customfield_10010", Name: "Warehouse Slot", Custom: true, Searchable: true, Orderable: true, Schema: &jiraFieldSchema{Type: "string"}}},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"field", "list", "--custom-only"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertGolden(t, "testdata/field-list-text.golden", stdout)
}

func TestRunFieldGetRequiresIdentifier(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"field", "get"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "field get requires exactly one field id or exact field name") {
		t.Fatalf("expected missing field identifier error, got %q", stderr)
	}
}

func TestResolveIssueSearchJQLBuildsExpectedQuery(t *testing.T) {
	jql, err := resolveIssueSearchJQL(issueSearchOptions{
		project:  "SCWI",
		statuses: []string{"To Do", "In Progress"},
		assignee: "currentUser()",
	})
	if err != nil {
		t.Fatalf("resolveIssueSearchJQL: %v", err)
	}
	expected := `project = SCWI AND status in ("To Do", "In Progress") AND assignee = currentUser()`
	if jql != expected {
		t.Fatalf("expected %q, got %q", expected, jql)
	}
}

func TestValidatePaginationRejectsNegativeStartAt(t *testing.T) {
	err := validatePagination("field list", paginationOptions{limit: 10, startAt: -1})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--start-at") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMePrintsJSON(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		myself: jiraMyselfResponse{AccountID: "abc-123", DisplayName: "Agent User", EmailAddress: "agent@example.com", Active: true},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"me", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"itemType": "jira-user"`) {
		t.Fatalf("expected JSON envelope, got %q", stdout)
	}
}

func TestRunMeSurfacesClientErrors(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{myselfErr: errors.New("jira /rest/api/3/myself failed with status 401: Unauthorized")})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"me"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Unauthorized") {
		t.Fatalf("expected client error, got %q", stderr)
	}
}

func runForTest(argv []string) (string, string, int) {
	stdoutBuffer := &strings.Builder{}
	stderrBuffer := &strings.Builder{}
	exitCode := run(argv, cliEnvironment{stderr: stderrBuffer, stdout: stdoutBuffer})
	return strings.TrimSpace(stdoutBuffer.String()), strings.TrimSpace(stderrBuffer.String()), exitCode
}

type stubJiraAPI struct {
	issue           jiraIssueResponse
	issueErr        error
	searchIssues    jiraIssueSearchResponse
	searchErr       error
	comments        jiraCommentPageResponse
	commentsErr     error
	myself          jiraMyselfResponse
	myselfErr       error
	serverInfo      jiraServerInfoResponse
	serverErr       error
	projects        jiraProjectPageResponse
	projectsErr     error
	project         jiraProjectResponse
	projectErr      error
	statuses        []jiraProjectIssueTypeStatusesResponse
	statusesErr     error
	boards          jiraBoardPageResponse
	boardsErr       error
	board           jiraBoardResponse
	boardErr        error
	sprints         jiraSprintPageResponse
	sprintsErr      error
	boardIssues     jiraIssueSearchResponse
	boardIssuesErr  error
	sprintIssues    jiraIssueSearchResponse
	sprintIssuesErr error
	filters         jiraFilterPageResponse
	filtersErr      error
	filter          jiraFilterResponse
	filterErr       error
	fields          jiraFieldPageResponse
	fieldsErr       error
	field           jiraFieldResponse
	fieldErr        error
}

func (stub stubJiraAPI) GetIssue(_ context.Context, _ string, _ []string) (jiraIssueResponse, error) {
	if stub.issueErr != nil {
		return jiraIssueResponse{}, stub.issueErr
	}
	return stub.issue, nil
}

func (stub stubJiraAPI) SearchIssues(_ context.Context, _ jiraIssueSearchRequest) (jiraIssueSearchResponse, error) {
	if stub.searchErr != nil {
		return jiraIssueSearchResponse{}, stub.searchErr
	}
	return stub.searchIssues, nil
}

func (stub stubJiraAPI) GetIssueComments(_ context.Context, _ string, _ int, _ int) (jiraCommentPageResponse, error) {
	if stub.commentsErr != nil {
		return jiraCommentPageResponse{}, stub.commentsErr
	}
	return stub.comments, nil
}

func (stub stubJiraAPI) GetMyself(_ context.Context) (jiraMyselfResponse, error) {
	if stub.myselfErr != nil {
		return jiraMyselfResponse{}, stub.myselfErr
	}
	return stub.myself, nil
}

func (stub stubJiraAPI) GetServerInfo(_ context.Context) (jiraServerInfoResponse, error) {
	if stub.serverErr != nil {
		return jiraServerInfoResponse{}, stub.serverErr
	}
	return stub.serverInfo, nil
}

func (stub stubJiraAPI) ListProjects(_ context.Context, _ jiraProjectListRequest) (jiraProjectPageResponse, error) {
	if stub.projectsErr != nil {
		return jiraProjectPageResponse{}, stub.projectsErr
	}
	return stub.projects, nil
}

func (stub stubJiraAPI) GetProject(_ context.Context, _ string) (jiraProjectResponse, error) {
	if stub.projectErr != nil {
		return jiraProjectResponse{}, stub.projectErr
	}
	return stub.project, nil
}

func (stub stubJiraAPI) GetProjectStatuses(_ context.Context, _ string) ([]jiraProjectIssueTypeStatusesResponse, error) {
	if stub.statusesErr != nil {
		return nil, stub.statusesErr
	}
	return stub.statuses, nil
}

func (stub stubJiraAPI) ListBoards(_ context.Context, _ jiraBoardListRequest) (jiraBoardPageResponse, error) {
	if stub.boardsErr != nil {
		return jiraBoardPageResponse{}, stub.boardsErr
	}
	return stub.boards, nil
}

func (stub stubJiraAPI) GetBoard(_ context.Context, _ string) (jiraBoardResponse, error) {
	if stub.boardErr != nil {
		return jiraBoardResponse{}, stub.boardErr
	}
	return stub.board, nil
}

func (stub stubJiraAPI) ListBoardSprints(_ context.Context, _ string, _ jiraSprintListRequest) (jiraSprintPageResponse, error) {
	if stub.sprintsErr != nil {
		return jiraSprintPageResponse{}, stub.sprintsErr
	}
	return stub.sprints, nil
}

func (stub stubJiraAPI) ListBoardIssues(_ context.Context, _ string, _ jiraAgileIssueListRequest) (jiraIssueSearchResponse, error) {
	if stub.boardIssuesErr != nil {
		return jiraIssueSearchResponse{}, stub.boardIssuesErr
	}
	return stub.boardIssues, nil
}

func (stub stubJiraAPI) ListSprintIssues(_ context.Context, _ string, _ jiraAgileIssueListRequest) (jiraIssueSearchResponse, error) {
	if stub.sprintIssuesErr != nil {
		return jiraIssueSearchResponse{}, stub.sprintIssuesErr
	}
	return stub.sprintIssues, nil
}

func (stub stubJiraAPI) ListFilters(_ context.Context, _ jiraFilterListRequest) (jiraFilterPageResponse, error) {
	if stub.filtersErr != nil {
		return jiraFilterPageResponse{}, stub.filtersErr
	}
	return stub.filters, nil
}

func (stub stubJiraAPI) GetFilter(_ context.Context, _ string) (jiraFilterResponse, error) {
	if stub.filterErr != nil {
		return jiraFilterResponse{}, stub.filterErr
	}
	return stub.filter, nil
}

func (stub stubJiraAPI) ListFields(_ context.Context, _ jiraFieldListRequest) (jiraFieldPageResponse, error) {
	if stub.fieldsErr != nil {
		return jiraFieldPageResponse{}, stub.fieldsErr
	}
	return stub.fields, nil
}

func (stub stubJiraAPI) GetField(_ context.Context, _ string) (jiraFieldResponse, error) {
	if stub.fieldErr != nil {
		return jiraFieldResponse{}, stub.fieldErr
	}
	return stub.field, nil
}

func withFactoriesForTest(t *testing.T, env configEnvironment, api jiraAPI) func() {
	t.Helper()
	originalConfigFactory := configEnvironmentFactory
	originalAPIFactory := jiraAPIFactory
	configEnvironmentFactory = func() configEnvironment { return env }
	jiraAPIFactory = func(resolvedRuntimeConfig) (jiraAPI, error) { return api, nil }
	return func() {
		configEnvironmentFactory = originalConfigFactory
		jiraAPIFactory = originalAPIFactory
	}
}

func testRuntimeEnv() configEnvironment {
	return configEnvironment{
		homeDir: func() (string, error) { return os.TempDir(), nil },
		lookupEnv: func(key string) (string, bool) {
			values := map[string]string{
				"JIRA_SITE":  "https://example.atlassian.net",
				"JIRA_EMAIL": "agent@example.com",
				"JIRA_TOKEN": "test-token",
			}
			value, ok := values[key]
			return value, ok
		},
		readFile: nil,
	}
}

func assertGolden(t *testing.T, path string, actual string) {
	t.Helper()
	expectedBytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	expected := strings.TrimSpace(string(expectedBytes))
	if actual != expected {
		t.Fatalf("golden mismatch for %s\n--- expected ---\n%s\n--- actual ---\n%s", path, expected, actual)
	}
}

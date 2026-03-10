package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunPrintsHelpSurfaces(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		expected string
	}{
		{
			name:     "root",
			argv:     nil,
			expected: "jira - agent-first Jira CLI",
		},
		{
			name:     "issue",
			argv:     []string{"issue"},
			expected: "jira issue - explicit issue read commands",
		},
		{
			name:     "issue get",
			argv:     []string{"issue", "get", "--help"},
			expected: "jira issue get",
		},
		{
			name:     "me",
			argv:     []string{"me", "--help"},
			expected: "--json",
		},
		{
			name:     "serverinfo",
			argv:     []string{"serverinfo", "--help"},
			expected: "--json",
		},
		{
			name:     "issue search",
			argv:     []string{"issue", "search", "--help"},
			expected: "--json",
		},
		{
			name:     "version",
			argv:     []string{"version", "--help"},
			expected: "jira version",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exitCode := runForTest(tc.argv)
			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
			}
			if !strings.Contains(stdout, tc.expected) {
				t.Fatalf("expected help text %q in stdout, got %q", tc.expected, stdout)
			}
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

func TestRunIssueSearchRequiresQueryInput(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"issue", "search"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "issue search requires --jql or at least one explicit filter") {
		t.Fatalf("expected missing query error, got %q", stderr)
	}
}

func TestRunMePrintsJSON(t *testing.T) {
	restore := withFactoriesForTest(t, configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
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
	}, stubJiraAPI{
		myself: jiraMyselfResponse{
			AccountID:    "abc-123",
			DisplayName:  "Agent User",
			EmailAddress: "agent@example.com",
			Active:       true,
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"me", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"accountId": "abc-123"`) {
		t.Fatalf("expected JSON output, got %q", stdout)
	}
}

func TestRunServerInfoPrintsText(t *testing.T) {
	restore := withFactoriesForTest(t, configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
		lookupEnv: func(key string) (string, bool) {
			values := map[string]string{
				"JIRA_SITE": "https://example.atlassian.net",
			}
			value, ok := values[key]
			return value, ok
		},
		readFile: nil,
	}, stubJiraAPI{
		serverInfo: jiraServerInfoResponse{
			BaseURL:        "https://example.atlassian.net",
			DeploymentType: "Cloud",
			Version:        "1000.0.0",
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"serverinfo"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "deployment_type: Cloud") {
		t.Fatalf("expected text output, got %q", stdout)
	}
}

func TestRunMeSurfacesClientErrors(t *testing.T) {
	restore := withFactoriesForTest(t, configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
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
	}, stubJiraAPI{
		myselfErr: errors.New("jira /rest/api/3/myself failed with status 401: Unauthorized"),
	})
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

func TestRunIssueGetPrintsText(t *testing.T) {
	restore := withFactoriesForTest(t, configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
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
	}, stubJiraAPI{
		issue: jiraIssueResponse{
			Key: "SCWI-282",
			Fields: map[string]any{
				"summary": "Implement thing",
				"status":  map[string]any{"name": "In Progress"},
			},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "get", "SCWI-282"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "summary: Implement thing") {
		t.Fatalf("expected issue text, got %q", stdout)
	}
}

func TestRunIssueGetAcceptsFlagsAfterKey(t *testing.T) {
	restore := withFactoriesForTest(t, configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
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
	}, stubJiraAPI{
		issue: jiraIssueResponse{
			Key: "SCWI-282",
			Fields: map[string]any{
				"summary": "Implement thing",
				"status":  map[string]any{"name": "In Progress"},
			},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "get", "SCWI-282", "--fields", "summary,status"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "status: In Progress") {
		t.Fatalf("expected issue text, got %q", stdout)
	}
}

func TestRunIssueSearchPrintsJSON(t *testing.T) {
	restore := withFactoriesForTest(t, configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
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
	}, stubJiraAPI{
		search: jiraSearchResponse{
			JQL:        `project = "SCWI" ORDER BY updated DESC`,
			MaxResults: 2,
			Total:      1,
			Issues: []jiraIssueResponse{
				{
					Key: "SCWI-282",
					Fields: map[string]any{
						"summary": "Implement thing",
						"status":  map[string]any{"name": "In Progress"},
					},
				},
			},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "search", "--project", "SCWI", "--limit", "2", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"key": "SCWI-282"`) {
		t.Fatalf("expected JSON issue output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"jql": "project = \"SCWI\" ORDER BY updated DESC"`) {
		t.Fatalf("expected JSON jql output, got %q", stdout)
	}
}

func TestRunIssueSearchPrintsText(t *testing.T) {
	restore := withFactoriesForTest(t, configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
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
	}, stubJiraAPI{
		search: jiraSearchResponse{
			JQL:        `project = "SCWI" ORDER BY updated DESC`,
			MaxResults: 1,
			Total:      1,
			Issues: []jiraIssueResponse{
				{
					Key: "SCWI-282",
					Fields: map[string]any{
						"summary": "Implement thing",
						"status":  map[string]any{"name": "In Progress"},
					},
				},
			},
		},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"issue", "search", "--project", "SCWI"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `jql: project = "SCWI" ORDER BY updated DESC`) {
		t.Fatalf("expected text jql output, got %q", stdout)
	}
	if !strings.Contains(stdout, "- SCWI-282 | summary=Implement thing | status=In Progress") {
		t.Fatalf("expected search result line, got %q", stdout)
	}
}

func TestParseIssueSearchOptionsPreservesRepeatedStatuses(t *testing.T) {
	options, helpRequested, err := parseIssueSearchOptions([]string{"--project", "SCWI", "--status", "To Do", "--status", "In Progress"})
	if err != nil {
		t.Fatalf("parseIssueSearchOptions: %v", err)
	}
	if helpRequested {
		t.Fatal("expected helpRequested to be false")
	}
	if len(options.statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %v", options.statuses)
	}
	if options.statuses[0] != "To Do" || options.statuses[1] != "In Progress" {
		t.Fatalf("unexpected statuses: %v", options.statuses)
	}
}

func TestParseIssueSearchOptionsRejectsNonPositiveLimit(t *testing.T) {
	_, _, err := parseIssueSearchOptions([]string{"--project", "SCWI", "--limit", "0"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "issue search requires --limit to be greater than 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseIssueSearchOptionsRejectsPositionalArguments(t *testing.T) {
	_, _, err := parseIssueSearchOptions([]string{"--project", "SCWI", "extra"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "issue search does not accept positional arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildSearchJQLFromExplicitFilters(t *testing.T) {
	jql, err := buildSearchJQL(issueSearchOptions{
		project:  "SCWI",
		assignee: "currentUser()",
		statuses: []string{"To Do", "In Progress"},
	})
	if err != nil {
		t.Fatalf("buildSearchJQL: %v", err)
	}
	expected := `project = "SCWI" AND assignee = currentUser() AND status in ("To Do", "In Progress") ORDER BY updated DESC`
	if jql != expected {
		t.Fatalf("unexpected JQL\nexpected: %s\ngot:      %s", expected, jql)
	}
}

func TestBuildSearchJQLEscapesLiteralOperands(t *testing.T) {
	jql, err := buildSearchJQL(issueSearchOptions{
		project:  `SC"WI`,
		assignee: `agent\name`,
		statuses: []string{`Needs "Review"`},
	})
	if err != nil {
		t.Fatalf("buildSearchJQL: %v", err)
	}
	expected := `project = "SC\"WI" AND assignee = "agent\\name" AND status = "Needs \"Review\"" ORDER BY updated DESC`
	if jql != expected {
		t.Fatalf("unexpected JQL\nexpected: %s\ngot:      %s", expected, jql)
	}
}

func TestBuildSearchJQLRequiresFilters(t *testing.T) {
	_, err := buildSearchJQL(issueSearchOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "empty explicit filters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsLikelyJQLFunction(t *testing.T) {
	if !isLikelyJQLFunction("currentUser()") {
		t.Fatal("expected currentUser() to be treated as a JQL function")
	}
	for _, value := range []string{"display name", "1currentUser()", "membersOf(team-a)", "agent@example.com"} {
		if isLikelyJQLFunction(value) {
			t.Fatalf("expected %q not to be treated as a JQL function", value)
		}
	}
}

func runForTest(argv []string) (string, string, int) {
	stdoutBuffer := &strings.Builder{}
	stderrBuffer := &strings.Builder{}
	exitCode := run(argv, cliEnvironment{
		stderr: stderrBuffer,
		stdout: stdoutBuffer,
	})
	return strings.TrimSpace(stdoutBuffer.String()), strings.TrimSpace(stderrBuffer.String()), exitCode
}

type stubJiraAPI struct {
	issue      jiraIssueResponse
	issueErr   error
	myself     jiraMyselfResponse
	myselfErr  error
	search     jiraSearchResponse
	searchErr  error
	serverInfo jiraServerInfoResponse
	serverErr  error
}

func (stub stubJiraAPI) GetIssue(_ context.Context, _ string, _ []string) (jiraIssueResponse, error) {
	if stub.issueErr != nil {
		return jiraIssueResponse{}, stub.issueErr
	}
	return stub.issue, nil
}

func (stub stubJiraAPI) GetMyself(_ context.Context) (jiraMyselfResponse, error) {
	if stub.myselfErr != nil {
		return jiraMyselfResponse{}, stub.myselfErr
	}
	return stub.myself, nil
}

func (stub stubJiraAPI) SearchIssues(_ context.Context, _ jiraSearchRequest) (jiraSearchResponse, error) {
	if stub.searchErr != nil {
		return jiraSearchResponse{}, stub.searchErr
	}
	return stub.search, nil
}

func (stub stubJiraAPI) GetServerInfo(_ context.Context) (jiraServerInfoResponse, error) {
	if stub.serverErr != nil {
		return jiraServerInfoResponse{}, stub.serverErr
	}
	return stub.serverInfo, nil
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

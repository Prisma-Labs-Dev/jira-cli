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

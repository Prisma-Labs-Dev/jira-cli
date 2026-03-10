package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type goldenCaseDefinition struct {
	name string
	args []string
	load func(*testing.T, []byte) stubJiraAPI
}

func TestSyntheticGoldens(t *testing.T) {
	cases := []goldenCaseDefinition{
		{
			name: "issue_search_empty",
			args: []string{"issue", "search", "--project", "DEMO", "--fields", "summary,status,assignee"},
			load: func(t *testing.T, raw []byte) stubJiraAPI {
				t.Helper()
				var response jiraSearchResponse
				if err := json.Unmarshal(raw, &response); err != nil {
					t.Fatalf("unmarshal issue search empty response: %v", err)
				}
				return stubJiraAPI{search: response}
			},
		},
		{
			name: "issue_search_multi",
			args: []string{"issue", "search", "--project", "DEMO", "--limit", "2", "--fields", "summary,status,assignee"},
			load: func(t *testing.T, raw []byte) stubJiraAPI {
				t.Helper()
				var response jiraSearchResponse
				if err := json.Unmarshal(raw, &response); err != nil {
					t.Fatalf("unmarshal issue search multi response: %v", err)
				}
				return stubJiraAPI{search: response}
			},
		},
	}

	runGoldenCases(t, "synthetic", cases)
}

func runGoldenCases(t *testing.T, suite string, cases []goldenCaseDefinition) {
	t.Helper()

	for _, current := range cases {
		t.Run(current.name, func(t *testing.T) {
			jsonPath := filepath.Join("testdata", "goldens", suite, current.name+".stdout.json")
			textPath := filepath.Join("testdata", "goldens", suite, current.name+".stdout.txt")

			jsonFixture, err := os.ReadFile(jsonPath)
			if err != nil {
				t.Fatalf("read json fixture: %v", err)
			}
			textFixture, err := os.ReadFile(textPath)
			if err != nil {
				t.Fatalf("read text fixture: %v", err)
			}

			restore := withFactoriesForTest(t, configEnvironment{
				homeDir: func() (string, error) { return t.TempDir(), nil },
				lookupEnv: func(key string) (string, bool) {
					values := map[string]string{
						"JIRA_SITE":  "https://jira.example.test",
						"JIRA_EMAIL": "agent@example.com",
						"JIRA_TOKEN": "test-token",
					}
					value, ok := values[key]
					return value, ok
				},
				readFile: nil,
			}, current.load(t, jsonFixture))
			defer restore()

			stdout, stderr, exitCode := runForTest(current.args)
			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			if strings.TrimSpace(stdout) != strings.TrimSpace(string(textFixture)) {
				t.Fatalf("text golden mismatch\nexpected:\n%s\n\ngot:\n%s", string(textFixture), stdout)
			}

			stdout, stderr, exitCode = runForTest(append(append([]string{}, current.args...), "--json"))
			if exitCode != 0 {
				t.Fatalf("expected json exit code 0, got %d, stderr=%q", exitCode, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr for json case, got %q", stderr)
			}
			if strings.TrimSpace(stdout) != strings.TrimSpace(string(jsonFixture)) {
				t.Fatalf("json golden mismatch\nexpected:\n%s\n\ngot:\n%s", string(jsonFixture), stdout)
			}
		})
	}
}

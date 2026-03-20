package main

import (
	"strings"
	"testing"
)

func TestRunBoardSnapshotRequiresSelector(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"board", "snapshot"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "board snapshot requires exactly one of --board, --project, or --default") {
		t.Fatalf("unexpected error: %q", stderr)
	}
}

func TestRunBoardSnapshotRejectsTypeWithoutProject(t *testing.T) {
	stdout, stderr, exitCode := runForTest([]string{"board", "snapshot", "--board", "7", "--type", "scrum"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "board snapshot accepts --type only together with --project") {
		t.Fatalf("unexpected error: %q", stderr)
	}
}

func TestRunBoardSnapshotPrintsJSONEnvelope(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		board:   jiraBoardResponse{ID: 7, Name: "Warehouse Board", Type: "scrum", Location: &jiraBoardLocationResponse{ProjectKey: "SCWI", ProjectName: "Warehouse Inbound", DisplayName: "Warehouse"}},
		sprints: jiraSprintPageResponse{Values: []jiraSprintResponse{{ID: 42, Name: "Sprint 1", State: "active", Goal: "Ship slotting fixes", OriginBoardID: 7}}},
		sprintIssues: jiraIssueSearchResponse{
			StartAt:    0,
			MaxResults: 100,
			Total:      2,
			Issues: []jiraIssueResponse{
				{
					ID:  "10001",
					Key: "SCWI-336",
					Fields: map[string]any{
						"summary":   "Move product dialog",
						"status":    map[string]any{"name": "Progress"},
						"assignee":  map[string]any{"accountId": "abc-123", "displayName": "Agent User"},
						"priority":  map[string]any{"name": "Normal"},
						"issuetype": map[string]any{"name": "Story"},
						"updated":   "2026-03-20T09:00:00.000Z",
					},
				},
				{
					ID:  "10002",
					Key: "SCWI-206",
					Fields: map[string]any{
						"summary":   "Add product dialog",
						"status":    map[string]any{"name": "To Do"},
						"assignee":  map[string]any{"accountId": "other-1", "displayName": "Other User"},
						"priority":  map[string]any{"name": "Normal"},
						"issuetype": map[string]any{"name": "Story"},
						"updated":   "2026-03-19T09:00:00.000Z",
					},
				},
			},
		},
		myself: jiraMyselfResponse{AccountID: "abc-123", DisplayName: "Agent User", EmailAddress: "agent@example.com", Active: true},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"board", "snapshot", "--board", "7", "--limit", "1", "--me", "--json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertGolden(t, "testdata/board-snapshot-json.golden", stdout)
}

func TestRunBoardSnapshotUsesConfiguredDefaultBoard(t *testing.T) {
	env := testRuntimeEnv()
	env.lookupEnv = func(key string) (string, bool) {
		values := map[string]string{
			"JIRA_SITE":          "https://example.atlassian.net",
			"JIRA_EMAIL":         "agent@example.com",
			"JIRA_TOKEN":         "test-token",
			"JIRA_DEFAULT_BOARD": "7",
		}
		value, ok := values[key]
		return value, ok
	}
	restore := withFactoriesForTest(t, env, stubJiraAPI{
		board:       jiraBoardResponse{ID: 7, Name: "Warehouse Board", Type: "kanban"},
		boardIssues: jiraIssueSearchResponse{StartAt: 0, MaxResults: 100, Total: 0, Issues: []jiraIssueResponse{}},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"board", "snapshot", "--default"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "board: Warehouse Board (id=7, type=kanban)") {
		t.Fatalf("unexpected output: %q", stdout)
	}
}

func TestRunBoardSnapshotRejectsAmbiguousProjectResolution(t *testing.T) {
	restore := withFactoriesForTest(t, testRuntimeEnv(), stubJiraAPI{
		boards: jiraBoardPageResponse{IsLast: true, Values: []jiraBoardResponse{
			{ID: 7, Name: "Warehouse Board", Type: "scrum"},
			{ID: 8, Name: "Warehouse Kanban", Type: "kanban"},
		}},
	})
	defer restore()

	stdout, stderr, exitCode := runForTest([]string{"board", "snapshot", "--project", "SCWI"})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "board snapshot found multiple boards for project SCWI") || !strings.Contains(stderr, "7:Warehouse Board(scrum)") {
		t.Fatalf("unexpected error: %q", stderr)
	}
}

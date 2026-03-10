package main

import (
	"strings"
	"testing"
)

func TestParseOptionsUsesEnvFallbacks(t *testing.T) {
	t.Setenv("JIRA_SITE", "https://jira.example.test")
	t.Setenv("JIRA_GOLDEN_ISSUE_KEY", "SCWI-282")
	t.Setenv("JIRA_GOLDEN_PROJECT", "")
	t.Setenv("JIRA_GOLDEN_SEARCH_JQL", "")

	options, helpRequested, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if helpRequested {
		t.Fatal("expected helpRequested to be false")
	}
	if options.site != "https://jira.example.test" {
		t.Fatalf("unexpected site: %q", options.site)
	}
	if options.issueKey != "SCWI-282" {
		t.Fatalf("unexpected issue key: %q", options.issueKey)
	}
	if options.project != "SCWI" {
		t.Fatalf("unexpected project: %q", options.project)
	}
	if options.searchJQL != "key = SCWI-282" {
		t.Fatalf("unexpected jql: %q", options.searchJQL)
	}
}

func TestParseOptionsFlagsOverrideEnv(t *testing.T) {
	t.Setenv("JIRA_SITE", "https://jira.example.test")
	t.Setenv("JIRA_GOLDEN_ISSUE_KEY", "SCWI-282")

	options, helpRequested, err := parseOptions([]string{
		"--site", "https://jira.override.test",
		"--issue", "ABC-12",
		"--project", "ABC",
		"--jql", `project = "ABC"`,
	})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if helpRequested {
		t.Fatal("expected helpRequested to be false")
	}
	if options.site != "https://jira.override.test" || options.issueKey != "ABC-12" || options.project != "ABC" || options.searchJQL != `project = "ABC"` {
		t.Fatalf("unexpected options: %+v", options)
	}
}

func TestSanitizeJSONOutputNormalizesSensitiveValues(t *testing.T) {
	raw := []byte(`{
		"accountId":"712020:abc-123",
		"emailAddress":"ilia.safronov@ah.nl",
		"self":"https://jira-eu-aholddelhaize.atlassian.net/rest/api/3/user?accountId=712020:abc-123",
		"key":"SCWI-282",
		"statusCategory":{"key":"indeterminate"},
		"issues":[{"key":"SCWI-282"}]
	}`)

	sanitized, err := sanitizeJSONOutput(raw, "https://jira-eu-aholddelhaize.atlassian.net")
	if err != nil {
		t.Fatalf("sanitizeJSONOutput: %v", err)
	}
	text := string(sanitized)
	for _, forbidden := range []string{"ilia.safronov@ah.nl", "712020:abc-123", "SCWI-282"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected %q to be redacted from %s", forbidden, text)
		}
	}
	if !strings.Contains(text, `"accountId": "account-id-1"`) {
		t.Fatalf("expected account ID replacement, got %s", text)
	}
	if !strings.Contains(text, `"key": "indeterminate"`) {
		t.Fatalf("expected non-issue key to be preserved, got %s", text)
	}
	if !strings.Contains(text, `"self": "https://jira.example.test/rest/api/3/user?accountId=account-id-1"`) {
		t.Fatalf("expected self URL to be redacted, got %s", text)
	}
}

func TestSanitizeTextOutputNormalizesAgentFields(t *testing.T) {
	raw := strings.Join([]string{
		"display_name: Ilia Safronov",
		"email: ilia.safronov@ah.nl",
		"account_id: 712020:abc-123",
		"self: https://jira-eu-aholddelhaize.atlassian.net/rest/api/3/user?accountId=712020:abc-123",
		"- SCWI-282 | summary=Real summary | status=Reviewing | assignee=Ilia Safronov",
	}, "\n")

	sanitized := sanitizeTextOutput("me", raw, "https://jira-eu-aholddelhaize.atlassian.net")
	for _, want := range []string{
		"display_name: Agent User",
		"email: agent@example.com",
		"account_id: account-id-1",
		"self: https://jira.example.test/rest/api/3/user?accountId=account-id-1",
		"- DEMO-123 | summary=Issue summary | status=Reviewing | assignee=Agent User",
	} {
		if !strings.Contains(sanitized, want) {
			t.Fatalf("expected %q in sanitized text, got %s", want, sanitized)
		}
	}
}

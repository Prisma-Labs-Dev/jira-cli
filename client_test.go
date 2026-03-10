package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestJiraClientGetMyselfUsesBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/myself" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:test-token"))
		if request.Header.Get("Authorization") != expectedAuth {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"accountId":"abc-123","displayName":"Agent User","emailAddress":"agent@example.com","active":true}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{
		site:  server.URL,
		email: "agent@example.com",
		token: "test-token",
	})
	response, err := client.GetMyself(context.Background())
	if err != nil {
		t.Fatalf("GetMyself: %v", err)
	}
	if response.AccountID != "abc-123" || response.DisplayName != "Agent User" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientGetIssuePassesFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/issue/SCWI-282" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("fields") != "summary,status" {
			t.Fatalf("unexpected fields query: %q", request.URL.Query().Get("fields"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"10001","key":"SCWI-282","fields":{"summary":"Implement thing","status":{"name":"In Progress"}}}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{
		site:  server.URL,
		email: "agent@example.com",
		token: "test-token",
	})
	response, err := client.GetIssue(context.Background(), "SCWI-282", []string{"summary", "status"})
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if response.Key != "SCWI-282" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientSearchIssuesPassesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("jql") != `project = "SCWI" ORDER BY updated DESC` {
			t.Fatalf("unexpected jql query: %q", request.URL.Query().Get("jql"))
		}
		if request.URL.Query().Get("fields") != "summary,status" {
			t.Fatalf("unexpected fields query: %q", request.URL.Query().Get("fields"))
		}
		if request.URL.Query().Get("maxResults") != "2" {
			t.Fatalf("unexpected maxResults query: %q", request.URL.Query().Get("maxResults"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":0,"maxResults":2,"total":1,"issues":[{"id":"10001","key":"SCWI-282","fields":{"summary":"Implement thing","status":{"name":"In Progress"}}}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{
		site:  server.URL,
		email: "agent@example.com",
		token: "test-token",
	})
	response, err := client.SearchIssues(context.Background(), jiraSearchRequest{
		JQL:        `project = "SCWI" ORDER BY updated DESC`,
		Fields:     []string{"summary", "status"},
		MaxResults: 2,
	})
	if err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	if len(response.Issues) != 1 || response.Issues[0].Key != "SCWI-282" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientGetServerInfoFormatsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"errorMessages":["Unauthorized"]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL})
	_, err := client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJiraClientGetMyselfSurfacesTransportFailures(t *testing.T) {
	client := newJiraClient(resolvedRuntimeConfig{
		site:  "https://jira.example.test",
		email: "agent@example.com",
		token: "test-token",
	})
	client.httpClient.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp: timeout")
	})

	_, err := client.GetMyself(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "request Jira /rest/api/3/myself") || !strings.Contains(err.Error(), "dial tcp: timeout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJiraClientGetIssueRejectsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"key":"SCWI-282"`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{
		site:  server.URL,
		email: "agent@example.com",
		token: "test-token",
	})
	_, err := client.GetIssue(context.Background(), "SCWI-282", []string{"summary"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "decode Jira response for /rest/api/3/issue/SCWI-282") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJiraClientSearchIssuesOmitsOptionalQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.URL.Query().Get("fields"); got != "" {
			t.Fatalf("expected empty fields query, got %q", got)
		}
		if got := request.URL.Query().Get("maxResults"); got != "" {
			t.Fatalf("expected empty maxResults query, got %q", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"issues":[]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{
		site:  server.URL,
		email: "agent@example.com",
		token: "test-token",
	})
	response, err := client.SearchIssues(context.Background(), jiraSearchRequest{
		JQL: `assignee = currentUser() ORDER BY updated DESC`,
	})
	if err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	if response.JQL != `assignee = currentUser() ORDER BY updated DESC` {
		t.Fatalf("unexpected JQL echo: %+v", response)
	}
}

func TestRenderMeText(t *testing.T) {
	text := renderMeText(jiraMyselfResponse{
		AccountID:    "abc-123",
		DisplayName:  "Agent User",
		EmailAddress: "agent@example.com",
		Active:       true,
	})
	if !strings.Contains(text, "display_name: Agent User") {
		t.Fatalf("unexpected me text: %q", text)
	}
	if !strings.Contains(text, "account_id: abc-123") {
		t.Fatalf("unexpected me text: %q", text)
	}
}

func TestRenderIssueText(t *testing.T) {
	text := renderIssueText(jiraIssueResponse{
		Key: "SCWI-282",
		Fields: map[string]any{
			"summary": "Implement thing",
			"status":  map[string]any{"name": "In Progress"},
		},
	}, []string{"summary", "status"})
	if !strings.Contains(text, "summary: Implement thing") {
		t.Fatalf("unexpected issue text: %q", text)
	}
	if !strings.Contains(text, "status: In Progress") {
		t.Fatalf("unexpected issue text: %q", text)
	}
}

func TestRenderSearchText(t *testing.T) {
	text := renderSearchText(jiraSearchResponse{
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
	}, []string{"summary", "status"})
	if !strings.Contains(text, `jql: project = "SCWI" ORDER BY updated DESC`) {
		t.Fatalf("unexpected search text: %q", text)
	}
	if !strings.Contains(text, "- SCWI-282 | summary=Implement thing | status=In Progress") {
		t.Fatalf("unexpected search text: %q", text)
	}
}

func TestRenderSearchTextEmptyResultSet(t *testing.T) {
	text := renderSearchText(jiraSearchResponse{
		JQL:        `project = "SCWI" ORDER BY updated DESC`,
		StartAt:    20,
		MaxResults: 50,
	}, []string{"summary", "status"})
	for _, want := range []string{
		"returned: 0",
		"start_at: 20",
		"max_results: 50",
		`jql: project = "SCWI" ORDER BY updated DESC`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in output %q", want, text)
		}
	}
	if strings.Contains(text, "\n- ") {
		t.Fatalf("did not expect issue rows in %q", text)
	}
}

func TestRenderSearchTextMultipleIssuesAndSparseFields(t *testing.T) {
	text := renderSearchText(jiraSearchResponse{
		Total: 2,
		Issues: []jiraIssueResponse{
			{
				Key: "SCWI-282",
				Fields: map[string]any{
					"summary": "Implement thing",
				},
			},
			{
				Key: "SCWI-283",
				Fields: map[string]any{
					"summary":  "Review thing",
					"status":   map[string]any{"name": "To Do"},
					"assignee": nil,
				},
			},
		},
	}, []string{"summary", "status", "assignee"})
	for _, want := range []string{
		"total: 2",
		"- SCWI-282 | summary=Implement thing",
		"- SCWI-283 | summary=Review thing | status=To Do | assignee=null",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in output %q", want, text)
		}
	}
}

func TestRenderFieldValueCoversCollectionsAndFallbacks(t *testing.T) {
	if got := renderFieldValue([]any{"one", "two"}); got != "one, two" {
		t.Fatalf("unexpected slice render: %q", got)
	}
	if got := renderFieldValue(map[string]any{"displayName": "Agent User"}); got != "Agent User" {
		t.Fatalf("unexpected displayName render: %q", got)
	}
	if got := renderFieldValue(map[string]any{"unknown": "value"}); !strings.Contains(got, `"unknown":"value"`) {
		t.Fatalf("unexpected fallback render: %q", got)
	}
}

func TestReadJiraErrorFallsBackToPlainBody(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader("plain error body")),
	}
	err := readJiraError("/rest/api/3/search/jql", response)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "plain error body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadJiraErrorCollapsesStructuredBody(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body: io.NopCloser(strings.NewReader(`{
			"message":"validation failed",
			"errorMessages":["top level message"],
			"errors":{"jql":"invalid query"}
		}`)),
	}
	err := readJiraError("/rest/api/3/search/jql", response)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"validation failed", "top level message", "jql: invalid query"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestReadJiraErrorHandlesEmptyBody(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	err := readJiraError("/rest/api/3/search/jql", response)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "jira /rest/api/3/search/jql failed with status 429" {
		t.Fatalf("unexpected error: %v", err)
	}
}

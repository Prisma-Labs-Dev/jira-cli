package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.GetMyself(context.Background())
	if err != nil {
		t.Fatalf("GetMyself: %v", err)
	}
	if response.AccountID != "abc-123" || response.DisplayName != "Agent User" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientSearchIssuesUsesSearchJQLEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("jql") != "project = SCWI" {
			t.Fatalf("unexpected jql query: %q", request.URL.Query().Get("jql"))
		}
		if request.URL.Query().Get("fields") != "summary,status" {
			t.Fatalf("unexpected fields query: %q", request.URL.Query().Get("fields"))
		}
		if request.URL.Query().Get("maxResults") != "2" || request.URL.Query().Get("startAt") != "0" {
			t.Fatalf("unexpected pagination query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":0,"maxResults":2,"total":1,"issues":[{"id":"10001","key":"SCWI-282","fields":{"summary":"Implement thing","status":{"name":"In Progress"}}}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.SearchIssues(context.Background(), jiraIssueSearchRequest{JQL: "project = SCWI", Fields: []string{"summary", "status"}, Limit: 2, StartAt: 0})
	if err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	if len(response.Issues) != 1 || response.Issues[0].Key != "SCWI-282" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientListBoardsUsesProjectFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/agile/1.0/board" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("projectKeyOrId") != "SCWI" {
			t.Fatalf("unexpected project query: %q", request.URL.Query().Get("projectKeyOrId"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":0,"maxResults":50,"isLast":true,"values":[{"id":7,"name":"Warehouse Board","type":"scrum","location":{"projectKey":"SCWI","projectName":"Warehouse"}}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.ListBoards(context.Background(), jiraBoardListRequest{Project: "SCWI", Limit: 50, StartAt: 0})
	if err != nil {
		t.Fatalf("ListBoards: %v", err)
	}
	if len(response.Values) != 1 || response.Values[0].ID != 7 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientListBoardSprintsUsesStateFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/agile/1.0/board/7/sprint" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("state") != "active,future" {
			t.Fatalf("unexpected state query: %q", request.URL.Query().Get("state"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":0,"maxResults":50,"isLast":true,"values":[{"id":42,"name":"Sprint 1","state":"active"}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.ListBoardSprints(context.Background(), "7", jiraSprintListRequest{States: []string{"active", "future"}, Limit: 50, StartAt: 0})
	if err != nil {
		t.Fatalf("ListBoardSprints: %v", err)
	}
	if len(response.Values) != 1 || response.Values[0].ID != 42 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientListSprintIssuesUsesFieldsAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/agile/1.0/sprint/42/issue" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("fields") != "summary,status,assignee" {
			t.Fatalf("unexpected fields query: %q", request.URL.Query().Get("fields"))
		}
		if request.URL.Query().Get("startAt") != "10" || request.URL.Query().Get("maxResults") != "5" {
			t.Fatalf("unexpected pagination query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":10,"maxResults":5,"total":11,"issues":[{"id":"10001","key":"SCWI-282","fields":{"summary":"Implement thing","status":{"name":"In Progress"}}}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.ListSprintIssues(context.Background(), "42", jiraAgileIssueListRequest{Fields: []string{"summary", "status", "assignee"}, Limit: 5, StartAt: 10})
	if err != nil {
		t.Fatalf("ListSprintIssues: %v", err)
	}
	if len(response.Issues) != 1 || response.Issues[0].Key != "SCWI-282" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientGetIssueCommentsUsesPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/issue/SCWI-282/comment" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("startAt") != "10" || request.URL.Query().Get("maxResults") != "5" {
			t.Fatalf("unexpected pagination query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":10,"maxResults":5,"total":11,"comments":[{"id":"20001","created":"2026-03-17T09:00:00.000Z"}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.GetIssueComments(context.Background(), "SCWI-282", 10, 5)
	if err != nil {
		t.Fatalf("GetIssueComments: %v", err)
	}
	if len(response.Comments) != 1 || response.Comments[0].ID != "20001" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientListProjectsUsesSearchAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/project/search" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("query") != "warehouse" {
			t.Fatalf("unexpected project query: %q", request.URL.Query().Get("query"))
		}
		if request.URL.Query().Get("startAt") != "25" || request.URL.Query().Get("maxResults") != "10" {
			t.Fatalf("unexpected pagination query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":25,"maxResults":10,"total":26,"values":[{"id":"10000","key":"SCWI","name":"Warehouse"}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.ListProjects(context.Background(), jiraProjectListRequest{Search: "warehouse", Limit: 10, StartAt: 25})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(response.Values) != 1 || response.Values[0].Key != "SCWI" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientGetProjectStatusesUsesProjectPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/project/SCWI/statuses" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[{"name":"Bug","statuses":[{"name":"To Do"},{"name":"Done"}]}]`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.GetProjectStatuses(context.Background(), "SCWI")
	if err != nil {
		t.Fatalf("GetProjectStatuses: %v", err)
	}
	if len(response) != 1 || response[0].Name != "Bug" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientListFiltersUsesSearchAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/filter/search" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("filterName") != "warehouse" {
			t.Fatalf("unexpected filter query: %q", request.URL.Query().Get("filterName"))
		}
		if request.URL.Query().Get("startAt") != "5" || request.URL.Query().Get("maxResults") != "20" {
			t.Fatalf("unexpected pagination query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":5,"maxResults":20,"total":6,"isLast":true,"values":[{"id":"10001","name":"Warehouse Work"}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.ListFilters(context.Background(), jiraFilterListRequest{Search: "warehouse", Limit: 20, StartAt: 5})
	if err != nil {
		t.Fatalf("ListFilters: %v", err)
	}
	if len(response.Values) != 1 || response.Values[0].ID != "10001" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientListFieldsUsesSearchAndCustomOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/field/search" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("query") != "warehouse" {
			t.Fatalf("unexpected query: %q", request.URL.Query().Get("query"))
		}
		if request.URL.Query().Get("type") != "custom" {
			t.Fatalf("unexpected type: %q", request.URL.Query().Get("type"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"startAt":0,"maxResults":50,"total":1,"isLast":true,"values":[{"id":"customfield_10010","name":"Warehouse Slot","custom":true,"searchable":true,"orderable":true,"schema":{"type":"string"}}]}`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	response, err := client.ListFields(context.Background(), jiraFieldListRequest{Search: "warehouse", CustomOnly: true, Limit: 50, StartAt: 0})
	if err != nil {
		t.Fatalf("ListFields: %v", err)
	}
	if len(response.Values) != 1 || response.Values[0].ID != "customfield_10010" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestJiraClientGetFieldMatchesExactName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/field" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
		  {"id":"summary","name":"Summary","searchable":true,"orderable":true,"schema":{"type":"string"}},
		  {"id":"customfield_10010","name":"Warehouse Slot","custom":true,"searchable":true,"orderable":true,"schema":{"type":"string"}}
		]`))
	}))
	defer server.Close()

	client := newJiraClient(resolvedRuntimeConfig{site: server.URL, email: "agent@example.com", token: "test-token"})
	field, err := client.GetField(context.Background(), "Warehouse Slot")
	if err != nil {
		t.Fatalf("GetField: %v", err)
	}
	if field.ID != "customfield_10010" {
		t.Fatalf("unexpected field: %+v", field)
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

func TestRenderIssueCommentsTextIncludesExcerpt(t *testing.T) {
	text := renderIssueCommentsText([]issueCommentItem{{ID: "1", Author: "Agent", Created: "2026-03-17", BodyText: "Looks good."}}, buildPageFromTotal(0, 50, 1, 1))
	if !strings.Contains(text, "Looks good.") {
		t.Fatalf("unexpected comments text: %q", text)
	}
}

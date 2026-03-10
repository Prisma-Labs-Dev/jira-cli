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

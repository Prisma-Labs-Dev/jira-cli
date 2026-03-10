package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultHTTPTimeout = 30 * time.Second

type jiraAPI interface {
	GetIssue(ctx context.Context, issueKey string, fields []string) (jiraIssueResponse, error)
	SearchIssues(ctx context.Context, request jiraSearchRequest) (jiraSearchResponse, error)
	GetMyself(ctx context.Context) (jiraMyselfResponse, error)
	GetServerInfo(ctx context.Context) (jiraServerInfoResponse, error)
}

type jiraClient struct {
	baseURL    string
	email      string
	httpClient *http.Client
	token      string
}

type jiraMyselfResponse struct {
	AccountID    string `json:"accountId,omitempty"`
	AccountType  string `json:"accountType,omitempty"`
	Active       bool   `json:"active"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	Locale       string `json:"locale,omitempty"`
	Self         string `json:"self,omitempty"`
	TimeZone     string `json:"timeZone,omitempty"`
}

type jiraServerInfoResponse struct {
	BaseURL        string `json:"baseUrl,omitempty"`
	BuildDate      string `json:"buildDate,omitempty"`
	BuildNumber    int    `json:"buildNumber,omitempty"`
	DeploymentType string `json:"deploymentType,omitempty"`
	DisplayURL     string `json:"displayUrl,omitempty"`
	SCMInfo        string `json:"scmInfo,omitempty"`
	ServerTime     string `json:"serverTime,omitempty"`
	Version        string `json:"version,omitempty"`
	VersionNumbers []int  `json:"versionNumbers,omitempty"`
}

type jiraErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
	Message       string            `json:"message"`
}

type jiraIssueResponse struct {
	Fields map[string]any `json:"fields,omitempty"`
	ID     string         `json:"id,omitempty"`
	Key    string         `json:"key,omitempty"`
	Self   string         `json:"self,omitempty"`
}

type jiraSearchRequest struct {
	Fields     []string
	JQL        string
	MaxResults int
}

type jiraSearchResponse struct {
	Issues     []jiraIssueResponse `json:"issues,omitempty"`
	JQL        string              `json:"jql,omitempty"`
	MaxResults int                 `json:"maxResults,omitempty"`
	StartAt    int                 `json:"startAt,omitempty"`
	Total      int                 `json:"total,omitempty"`
}

func newJiraClient(config resolvedRuntimeConfig) *jiraClient {
	return &jiraClient{
		baseURL: strings.TrimRight(config.site, "/"),
		email:   config.email,
		token:   config.token,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

func (client *jiraClient) GetMyself(ctx context.Context) (jiraMyselfResponse, error) {
	var response jiraMyselfResponse
	if err := client.getJSON(ctx, "/rest/api/3/myself", true, &response, nil); err != nil {
		return jiraMyselfResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetIssue(ctx context.Context, issueKey string, fields []string) (jiraIssueResponse, error) {
	query := url.Values{}
	if len(fields) > 0 {
		query.Set("fields", strings.Join(fields, ","))
	}

	var response jiraIssueResponse
	if err := client.getJSON(ctx, "/rest/api/3/issue/"+issueKey, true, &response, query); err != nil {
		return jiraIssueResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetServerInfo(ctx context.Context) (jiraServerInfoResponse, error) {
	var response jiraServerInfoResponse
	if err := client.getJSON(ctx, "/rest/api/3/serverInfo", false, &response, nil); err != nil {
		return jiraServerInfoResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) SearchIssues(ctx context.Context, request jiraSearchRequest) (jiraSearchResponse, error) {
	query := url.Values{}
	query.Set("jql", request.JQL)
	if len(request.Fields) > 0 {
		query.Set("fields", strings.Join(request.Fields, ","))
	}
	if request.MaxResults > 0 {
		query.Set("maxResults", fmt.Sprintf("%d", request.MaxResults))
	}

	var response jiraSearchResponse
	if err := client.getJSON(ctx, "/rest/api/3/search/jql", true, &response, query); err != nil {
		return jiraSearchResponse{}, err
	}
	response.JQL = request.JQL
	return response, nil
}

func (client *jiraClient) getJSON(ctx context.Context, path string, requireAuth bool, target any, query url.Values) error {
	endpoint, err := url.JoinPath(client.baseURL, path)
	if err != nil {
		return fmt.Errorf("build Jira URL for %s: %w", path, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create Jira request for %s: %w", path, err)
	}
	if len(query) > 0 {
		request.URL.RawQuery = query.Encode()
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "jira/"+packageVersion)
	if requireAuth || (client.email != "" && client.token != "") {
		request.SetBasicAuth(client.email, client.token)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("request Jira %s: %w", path, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readJiraError(path, response)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode Jira response for %s: %w", path, err)
	}
	return nil
}

func readJiraError(path string, response *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		return fmt.Errorf("jira %s failed with status %d; additionally failed to read body: %w", path, response.StatusCode, err)
	}

	var parsed jiraErrorResponse
	if jsonErr := json.Unmarshal(body, &parsed); jsonErr == nil {
		message := collapseJiraErrors(parsed)
		if message != "" {
			return fmt.Errorf("jira %s failed with status %d: %s", path, response.StatusCode, message)
		}
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Errorf("jira %s failed with status %d", path, response.StatusCode)
	}
	return fmt.Errorf("jira %s failed with status %d: %s", path, response.StatusCode, trimmed)
}

func collapseJiraErrors(value jiraErrorResponse) string {
	var parts []string
	if value.Message != "" {
		parts = append(parts, value.Message)
	}
	for _, message := range value.ErrorMessages {
		if strings.TrimSpace(message) != "" {
			parts = append(parts, message)
		}
	}
	for field, message := range value.Errors {
		message = strings.TrimSpace(message)
		if message != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", field, message))
		}
	}
	return strings.Join(parts, "; ")
}

func renderMeText(value jiraMyselfResponse) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("display_name: %s", value.DisplayName))
	if value.EmailAddress != "" {
		lines = append(lines, fmt.Sprintf("email: %s", value.EmailAddress))
	}
	if value.AccountID != "" {
		lines = append(lines, fmt.Sprintf("account_id: %s", value.AccountID))
	}
	if value.AccountType != "" {
		lines = append(lines, fmt.Sprintf("account_type: %s", value.AccountType))
	}
	lines = append(lines, fmt.Sprintf("active: %t", value.Active))
	if value.Locale != "" {
		lines = append(lines, fmt.Sprintf("locale: %s", value.Locale))
	}
	if value.TimeZone != "" {
		lines = append(lines, fmt.Sprintf("time_zone: %s", value.TimeZone))
	}
	if value.Self != "" {
		lines = append(lines, fmt.Sprintf("self: %s", value.Self))
	}
	return strings.Join(lines, "\n")
}

func renderServerInfoText(value jiraServerInfoResponse) string {
	var lines []string
	if value.BaseURL != "" {
		lines = append(lines, fmt.Sprintf("base_url: %s", value.BaseURL))
	}
	if value.DisplayURL != "" {
		lines = append(lines, fmt.Sprintf("display_url: %s", value.DisplayURL))
	}
	if value.DeploymentType != "" {
		lines = append(lines, fmt.Sprintf("deployment_type: %s", value.DeploymentType))
	}
	if value.Version != "" {
		lines = append(lines, fmt.Sprintf("version: %s", value.Version))
	}
	if len(value.VersionNumbers) > 0 {
		var parts []string
		for _, number := range value.VersionNumbers {
			parts = append(parts, fmt.Sprintf("%d", number))
		}
		lines = append(lines, fmt.Sprintf("version_numbers: %s", strings.Join(parts, ".")))
	}
	if value.BuildNumber != 0 {
		lines = append(lines, fmt.Sprintf("build_number: %d", value.BuildNumber))
	}
	if value.BuildDate != "" {
		lines = append(lines, fmt.Sprintf("build_date: %s", value.BuildDate))
	}
	if value.ServerTime != "" {
		lines = append(lines, fmt.Sprintf("server_time: %s", value.ServerTime))
	}
	if value.SCMInfo != "" {
		lines = append(lines, fmt.Sprintf("scm_info: %s", value.SCMInfo))
	}
	return strings.Join(lines, "\n")
}

func renderIssueText(value jiraIssueResponse, fields []string) string {
	var lines []string
	if value.Key != "" {
		lines = append(lines, fmt.Sprintf("key: %s", value.Key))
	}
	if value.ID != "" {
		lines = append(lines, fmt.Sprintf("id: %s", value.ID))
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		fieldValue, ok := value.Fields[field]
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", field, renderFieldValue(fieldValue)))
	}
	if value.Self != "" {
		lines = append(lines, fmt.Sprintf("self: %s", value.Self))
	}
	return strings.Join(lines, "\n")
}

func renderSearchText(value jiraSearchResponse, fields []string) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("returned: %d", len(value.Issues)))
	if value.Total != 0 {
		lines = append(lines, fmt.Sprintf("total: %d", value.Total))
	}
	if value.StartAt != 0 {
		lines = append(lines, fmt.Sprintf("start_at: %d", value.StartAt))
	}
	if value.MaxResults != 0 {
		lines = append(lines, fmt.Sprintf("max_results: %d", value.MaxResults))
	}
	if value.JQL != "" {
		lines = append(lines, fmt.Sprintf("jql: %s", value.JQL))
	}
	for _, issue := range value.Issues {
		var parts []string
		if issue.Key != "" {
			parts = append(parts, issue.Key)
		}
		for _, field := range fields {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			fieldValue, ok := issue.Fields[field]
			if !ok {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s=%s", field, renderFieldValue(fieldValue)))
		}
		lines = append(lines, "- "+strings.Join(parts, " | "))
	}
	return strings.Join(lines, "\n")
}

func renderFieldValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%v", typed)
	case []any:
		if len(typed) == 0 {
			return "[]"
		}
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, renderFieldValue(item))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		for _, key := range []string{"displayName", "name", "value", "key", "id"} {
			if preferred, ok := typed[key]; ok {
				return renderFieldValue(preferred)
			}
		}
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(encoded)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

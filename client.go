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
	SearchIssues(ctx context.Context, request jiraIssueSearchRequest) (jiraIssueSearchResponse, error)
	GetIssueComments(ctx context.Context, issueKey string, startAt, limit int) (jiraCommentPageResponse, error)
	GetMyself(ctx context.Context) (jiraMyselfResponse, error)
	GetServerInfo(ctx context.Context) (jiraServerInfoResponse, error)
	ListProjects(ctx context.Context, request jiraProjectListRequest) (jiraProjectPageResponse, error)
	GetProject(ctx context.Context, projectKey string) (jiraProjectResponse, error)
	GetProjectStatuses(ctx context.Context, projectKey string) ([]jiraProjectIssueTypeStatusesResponse, error)
	ListBoards(ctx context.Context, request jiraBoardListRequest) (jiraBoardPageResponse, error)
	GetBoard(ctx context.Context, boardID string) (jiraBoardResponse, error)
	ListBoardSprints(ctx context.Context, boardID string, request jiraSprintListRequest) (jiraSprintPageResponse, error)
	ListBoardIssues(ctx context.Context, boardID string, request jiraAgileIssueListRequest) (jiraIssueSearchResponse, error)
	ListSprintIssues(ctx context.Context, sprintID string, request jiraAgileIssueListRequest) (jiraIssueSearchResponse, error)
	ListFilters(ctx context.Context, request jiraFilterListRequest) (jiraFilterPageResponse, error)
	GetFilter(ctx context.Context, filterID string) (jiraFilterResponse, error)
	ListFields(ctx context.Context, request jiraFieldListRequest) (jiraFieldPageResponse, error)
	GetField(ctx context.Context, fieldIDOrName string) (jiraFieldResponse, error)
}

type jiraClient struct {
	baseURL    string
	email      string
	httpClient *http.Client
	token      string
}

type jiraIssueSearchRequest struct {
	JQL     string
	Fields  []string
	Limit   int
	StartAt int
}

type jiraProjectListRequest struct {
	Search  string
	Limit   int
	StartAt int
}

type jiraBoardListRequest struct {
	Project string
	Type    string
	Limit   int
	StartAt int
}

type jiraSprintListRequest struct {
	States  []string
	Limit   int
	StartAt int
}

type jiraAgileIssueListRequest struct {
	Fields  []string
	Limit   int
	StartAt int
}

type jiraFilterListRequest struct {
	Search  string
	Limit   int
	StartAt int
}

type jiraFieldListRequest struct {
	Search     string
	CustomOnly bool
	Limit      int
	StartAt    int
}

type jiraMyselfResponse struct {
	AccountID    string `json:"accountId,omitempty"`
	AccountType  string `json:"accountType,omitempty"`
	Active       bool   `json:"active"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	Locale       string `json:"locale,omitempty"`
	TimeZone     string `json:"timeZone,omitempty"`
}

type jiraServerInfoResponse struct {
	BaseURL        string `json:"baseUrl,omitempty"`
	BuildDate      string `json:"buildDate,omitempty"`
	BuildNumber    int    `json:"buildNumber,omitempty"`
	DeploymentType string `json:"deploymentType,omitempty"`
	ServerTime     string `json:"serverTime,omitempty"`
	Version        string `json:"version,omitempty"`
	VersionNumbers []int  `json:"versionNumbers,omitempty"`
}

type jiraErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
	Message       string            `json:"message"`
}

type jiraUserRef struct {
	AccountID    string `json:"accountId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
}

type jiraIssueResponse struct {
	Fields map[string]any `json:"fields,omitempty"`
	ID     string         `json:"id,omitempty"`
	Key    string         `json:"key,omitempty"`
	Self   string         `json:"self,omitempty"`
}

type jiraIssueSearchResponse struct {
	Issues     []jiraIssueResponse `json:"issues"`
	MaxResults int                 `json:"maxResults"`
	StartAt    int                 `json:"startAt"`
	Total      int                 `json:"total"`
}

type jiraCommentPageResponse struct {
	Comments   []jiraCommentResponse `json:"comments"`
	MaxResults int                   `json:"maxResults"`
	StartAt    int                   `json:"startAt"`
	Total      int                   `json:"total"`
}

type jiraCommentResponse struct {
	Author  *jiraUserRef `json:"author,omitempty"`
	Body    any          `json:"body,omitempty"`
	Created string       `json:"created,omitempty"`
	ID      string       `json:"id,omitempty"`
	Updated string       `json:"updated,omitempty"`
}

type jiraProjectPageResponse struct {
	MaxResults int                   `json:"maxResults"`
	StartAt    int                   `json:"startAt"`
	Total      int                   `json:"total"`
	Values     []jiraProjectResponse `json:"values"`
}

type jiraProjectResponse struct {
	Description    string       `json:"description,omitempty"`
	ID             string       `json:"id,omitempty"`
	Key            string       `json:"key,omitempty"`
	Lead           *jiraUserRef `json:"lead,omitempty"`
	Name           string       `json:"name,omitempty"`
	ProjectTypeKey string       `json:"projectTypeKey,omitempty"`
	Simplified     bool         `json:"simplified,omitempty"`
	Style          string       `json:"style,omitempty"`
	URL            string       `json:"url,omitempty"`
}

type jiraProjectIssueTypeStatusesResponse struct {
	ID       string                  `json:"id,omitempty"`
	Name     string                  `json:"name,omitempty"`
	Subtask  bool                    `json:"subtask,omitempty"`
	Statuses []jiraStatusRefResponse `json:"statuses,omitempty"`
}

type jiraStatusRefResponse struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type jiraBoardPageResponse struct {
	IsLast     bool                `json:"isLast"`
	MaxResults int                 `json:"maxResults"`
	StartAt    int                 `json:"startAt"`
	Total      int                 `json:"total,omitempty"`
	Values     []jiraBoardResponse `json:"values"`
}

type jiraBoardResponse struct {
	ID       int                        `json:"id"`
	Location *jiraBoardLocationResponse `json:"location,omitempty"`
	Name     string                     `json:"name,omitempty"`
	Self     string                     `json:"self,omitempty"`
	Type     string                     `json:"type,omitempty"`
}

type jiraBoardLocationResponse struct {
	DisplayName string `json:"displayName,omitempty"`
	ProjectKey  string `json:"projectKey,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
}

type jiraSprintPageResponse struct {
	IsLast     bool                 `json:"isLast"`
	MaxResults int                  `json:"maxResults"`
	StartAt    int                  `json:"startAt"`
	Total      int                  `json:"total,omitempty"`
	Values     []jiraSprintResponse `json:"values"`
}

type jiraSprintResponse struct {
	ID            int    `json:"id"`
	Name          string `json:"name,omitempty"`
	State         string `json:"state,omitempty"`
	Self          string `json:"self,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty"`
	CreatedDate   string `json:"createdDate,omitempty"`
	Goal          string `json:"goal,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
}

type jiraFilterPageResponse struct {
	IsLast     bool                 `json:"isLast"`
	MaxResults int                  `json:"maxResults"`
	StartAt    int                  `json:"startAt"`
	Total      int                  `json:"total"`
	Values     []jiraFilterResponse `json:"values"`
}

type jiraFilterResponse struct {
	Description string       `json:"description,omitempty"`
	ID          string       `json:"id,omitempty"`
	JQL         string       `json:"jql,omitempty"`
	Name        string       `json:"name,omitempty"`
	Owner       *jiraUserRef `json:"owner,omitempty"`
	SearchURL   string       `json:"searchUrl,omitempty"`
	ViewURL     string       `json:"viewUrl,omitempty"`
}

type jiraFieldPageResponse struct {
	IsLast     bool                `json:"isLast"`
	MaxResults int                 `json:"maxResults"`
	StartAt    int                 `json:"startAt"`
	Total      int                 `json:"total"`
	Values     []jiraFieldResponse `json:"values"`
}

type jiraFieldResponse struct {
	ClauseNames []string         `json:"clauseNames,omitempty"`
	Custom      bool             `json:"custom,omitempty"`
	ID          string           `json:"id,omitempty"`
	Key         string           `json:"key,omitempty"`
	Name        string           `json:"name,omitempty"`
	Navigable   bool             `json:"navigable,omitempty"`
	Orderable   bool             `json:"orderable,omitempty"`
	Schema      *jiraFieldSchema `json:"schema,omitempty"`
	Searchable  bool             `json:"searchable,omitempty"`
}

type jiraFieldSchema struct {
	Custom   string `json:"custom,omitempty"`
	CustomID int    `json:"customId,omitempty"`
	Items    string `json:"items,omitempty"`
	Type     string `json:"type,omitempty"`
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

func (client *jiraClient) GetServerInfo(ctx context.Context) (jiraServerInfoResponse, error) {
	var response jiraServerInfoResponse
	if err := client.getJSON(ctx, "/rest/api/3/serverInfo", false, &response, nil); err != nil {
		return jiraServerInfoResponse{}, err
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

func (client *jiraClient) SearchIssues(ctx context.Context, request jiraIssueSearchRequest) (jiraIssueSearchResponse, error) {
	query := url.Values{}
	query.Set("jql", request.JQL)
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if len(request.Fields) > 0 {
		query.Set("fields", strings.Join(request.Fields, ","))
	}
	var response jiraIssueSearchResponse
	if err := client.getJSON(ctx, "/rest/api/3/search/jql", true, &response, query); err != nil {
		return jiraIssueSearchResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetIssueComments(ctx context.Context, issueKey string, startAt, limit int) (jiraCommentPageResponse, error) {
	query := url.Values{}
	query.Set("startAt", fmt.Sprintf("%d", startAt))
	query.Set("maxResults", fmt.Sprintf("%d", limit))
	var response jiraCommentPageResponse
	if err := client.getJSON(ctx, "/rest/api/3/issue/"+issueKey+"/comment", true, &response, query); err != nil {
		return jiraCommentPageResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) ListProjects(ctx context.Context, request jiraProjectListRequest) (jiraProjectPageResponse, error) {
	query := url.Values{}
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if strings.TrimSpace(request.Search) != "" {
		query.Set("query", request.Search)
	}
	var response jiraProjectPageResponse
	if err := client.getJSON(ctx, "/rest/api/3/project/search", true, &response, query); err != nil {
		return jiraProjectPageResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetProject(ctx context.Context, projectKey string) (jiraProjectResponse, error) {
	var response jiraProjectResponse
	if err := client.getJSON(ctx, "/rest/api/3/project/"+projectKey, true, &response, nil); err != nil {
		return jiraProjectResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetProjectStatuses(ctx context.Context, projectKey string) ([]jiraProjectIssueTypeStatusesResponse, error) {
	var response []jiraProjectIssueTypeStatusesResponse
	if err := client.getJSON(ctx, "/rest/api/3/project/"+projectKey+"/statuses", true, &response, nil); err != nil {
		return nil, err
	}
	return response, nil
}

func (client *jiraClient) ListBoards(ctx context.Context, request jiraBoardListRequest) (jiraBoardPageResponse, error) {
	query := url.Values{}
	query.Set("projectKeyOrId", request.Project)
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if strings.TrimSpace(request.Type) != "" {
		query.Set("type", request.Type)
	}
	var response jiraBoardPageResponse
	if err := client.getJSON(ctx, "/rest/agile/1.0/board", true, &response, query); err != nil {
		return jiraBoardPageResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetBoard(ctx context.Context, boardID string) (jiraBoardResponse, error) {
	var response jiraBoardResponse
	if err := client.getJSON(ctx, "/rest/agile/1.0/board/"+boardID, true, &response, nil); err != nil {
		return jiraBoardResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) ListBoardSprints(ctx context.Context, boardID string, request jiraSprintListRequest) (jiraSprintPageResponse, error) {
	query := url.Values{}
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if len(request.States) > 0 {
		query.Set("state", strings.Join(request.States, ","))
	}
	var response jiraSprintPageResponse
	if err := client.getJSON(ctx, "/rest/agile/1.0/board/"+boardID+"/sprint", true, &response, query); err != nil {
		return jiraSprintPageResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) ListBoardIssues(ctx context.Context, boardID string, request jiraAgileIssueListRequest) (jiraIssueSearchResponse, error) {
	query := url.Values{}
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if len(request.Fields) > 0 {
		query.Set("fields", strings.Join(request.Fields, ","))
	}
	var response jiraIssueSearchResponse
	if err := client.getJSON(ctx, "/rest/agile/1.0/board/"+boardID+"/issue", true, &response, query); err != nil {
		return jiraIssueSearchResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) ListSprintIssues(ctx context.Context, sprintID string, request jiraAgileIssueListRequest) (jiraIssueSearchResponse, error) {
	query := url.Values{}
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if len(request.Fields) > 0 {
		query.Set("fields", strings.Join(request.Fields, ","))
	}
	var response jiraIssueSearchResponse
	if err := client.getJSON(ctx, "/rest/agile/1.0/sprint/"+sprintID+"/issue", true, &response, query); err != nil {
		return jiraIssueSearchResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) ListFilters(ctx context.Context, request jiraFilterListRequest) (jiraFilterPageResponse, error) {
	query := url.Values{}
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if strings.TrimSpace(request.Search) != "" {
		query.Set("filterName", request.Search)
	}
	var response jiraFilterPageResponse
	if err := client.getJSON(ctx, "/rest/api/3/filter/search", true, &response, query); err != nil {
		return jiraFilterPageResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetFilter(ctx context.Context, filterID string) (jiraFilterResponse, error) {
	var response jiraFilterResponse
	if err := client.getJSON(ctx, "/rest/api/3/filter/"+filterID, true, &response, nil); err != nil {
		return jiraFilterResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) ListFields(ctx context.Context, request jiraFieldListRequest) (jiraFieldPageResponse, error) {
	query := url.Values{}
	query.Set("maxResults", fmt.Sprintf("%d", request.Limit))
	query.Set("startAt", fmt.Sprintf("%d", request.StartAt))
	if strings.TrimSpace(request.Search) != "" {
		query.Set("query", request.Search)
	}
	if request.CustomOnly {
		query.Set("type", "custom")
	}
	var response jiraFieldPageResponse
	if err := client.getJSON(ctx, "/rest/api/3/field/search", true, &response, query); err != nil {
		return jiraFieldPageResponse{}, err
	}
	return response, nil
}

func (client *jiraClient) GetField(ctx context.Context, fieldIDOrName string) (jiraFieldResponse, error) {
	var response []jiraFieldResponse
	if err := client.getJSON(ctx, "/rest/api/3/field", true, &response, nil); err != nil {
		return jiraFieldResponse{}, err
	}
	needle := strings.TrimSpace(fieldIDOrName)
	for _, field := range response {
		if field.ID == needle || field.Key == needle || strings.EqualFold(field.Name, needle) {
			return field, nil
		}
		for _, clauseName := range field.ClauseNames {
			if strings.EqualFold(clauseName, needle) {
				return field, nil
			}
		}
	}
	return jiraFieldResponse{}, fmt.Errorf("field not found: %s", fieldIDOrName)
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

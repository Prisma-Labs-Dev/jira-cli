package main

import (
	"fmt"
	"strings"
)

const (
	defaultListLimit         = 50
	defaultIssueGetFields    = "summary,status,assignee,issuetype,priority,parent,labels,components,updated"
	defaultIssueSearchFields = "summary,status,assignee,priority,updated"
)

type schemaDescriptor struct {
	ItemType string   `json:"itemType"`
	Fields   []string `json:"fields,omitempty"`
}

type pageDescriptor struct {
	Limit       int    `json:"limit"`
	StartAt     int    `json:"startAt,omitempty"`
	Returned    int    `json:"returned"`
	Total       *int   `json:"total,omitempty"`
	NextStartAt *int   `json:"nextStartAt,omitempty"`
	NextHint    string `json:"nextHint,omitempty"`
}

type singleEnvelope[T any] struct {
	Item   T                `json:"item"`
	Schema schemaDescriptor `json:"schema"`
}

type listEnvelope[T any] struct {
	Items  []T              `json:"items"`
	Page   pageDescriptor   `json:"page"`
	Schema schemaDescriptor `json:"schema"`
}

type issueItem struct {
	ID     string         `json:"id,omitempty"`
	Key    string         `json:"key"`
	Fields map[string]any `json:"fields,omitempty"`
}

type issueCommentItem struct {
	ID       string `json:"id"`
	Author   string `json:"author,omitempty"`
	Created  string `json:"created,omitempty"`
	Updated  string `json:"updated,omitempty"`
	BodyText string `json:"bodyText,omitempty"`
}

type projectItem struct {
	ID          string `json:"id,omitempty"`
	Key         string `json:"key,omitempty"`
	Name        string `json:"name,omitempty"`
	ProjectType string `json:"projectType,omitempty"`
	Style       string `json:"style,omitempty"`
	Simplified  bool   `json:"simplified,omitempty"`
	Lead        string `json:"lead,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

type projectStatusesItem struct {
	ProjectKey string                         `json:"projectKey"`
	IssueTypes []projectIssueTypeStatusesItem `json:"issueTypes"`
}

type projectIssueTypeStatusesItem struct {
	ID       string   `json:"id,omitempty"`
	Name     string   `json:"name,omitempty"`
	Subtask  bool     `json:"subtask,omitempty"`
	Statuses []string `json:"statuses,omitempty"`
}

type boardItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	ProjectKey   string `json:"projectKey,omitempty"`
	ProjectName  string `json:"projectName,omitempty"`
	LocationName string `json:"locationName,omitempty"`
	Self         string `json:"self,omitempty"`
}

type boardSprintItem struct {
	ID            int    `json:"id"`
	Name          string `json:"name,omitempty"`
	State         string `json:"state,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty"`
	CreatedDate   string `json:"createdDate,omitempty"`
	Goal          string `json:"goal,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
}

type boardSnapshotSourceItem struct {
	Type   string           `json:"type"`
	Sprint *boardSprintItem `json:"sprint,omitempty"`
}

type boardStatusCountItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type boardSnapshotTotals struct {
	TotalIssues int `json:"totalIssues"`
	MyIssues    int `json:"myIssues,omitempty"`
}

type boardSnapshotUserItem struct {
	AccountID    string `json:"accountId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
}

type boardSnapshotItem struct {
	Board        boardItem               `json:"board"`
	Source       boardSnapshotSourceItem `json:"source"`
	Totals       boardSnapshotTotals     `json:"totals"`
	StatusCounts []boardStatusCountItem  `json:"statusCounts"`
	Issues       []issueItem             `json:"issues"`
	Page         pageDescriptor          `json:"page"`
	Me           *boardSnapshotUserItem  `json:"me,omitempty"`
	MyIssues     []issueItem             `json:"myIssues,omitempty"`
}

type filterItem struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Owner       string `json:"owner,omitempty"`
	JQL         string `json:"jql,omitempty"`
	ViewURL     string `json:"viewUrl,omitempty"`
	SearchURL   string `json:"searchUrl,omitempty"`
	Description string `json:"description,omitempty"`
}

type fieldItem struct {
	ID          string   `json:"id,omitempty"`
	Key         string   `json:"key,omitempty"`
	Name        string   `json:"name,omitempty"`
	Custom      bool     `json:"custom,omitempty"`
	SchemaType  string   `json:"schemaType,omitempty"`
	SchemaItems string   `json:"schemaItems,omitempty"`
	Searchable  bool     `json:"searchable,omitempty"`
	Orderable   bool     `json:"orderable,omitempty"`
	ClauseNames []string `json:"clauseNames,omitempty"`
}

func buildPage(startAt, limit, returned int, total *int, hasNext bool) pageDescriptor {
	page := pageDescriptor{
		Limit:    limit,
		StartAt:  startAt,
		Returned: returned,
		Total:    total,
	}
	if hasNext {
		nextStartAt := startAt + returned
		page.NextStartAt = &nextStartAt
		page.NextHint = fmt.Sprintf("use --start-at %d", nextStartAt)
	}
	return page
}

func buildPageFromTotal(startAt, limit, returned, total int) pageDescriptor {
	totalCopy := total
	return buildPage(startAt, limit, returned, &totalCopy, startAt+returned < total)
}

func meSchema() schemaDescriptor {
	return schemaDescriptor{
		ItemType: "jira-user",
		Fields:   []string{"accountId", "displayName", "emailAddress", "active", "accountType", "locale", "timeZone"},
	}
}

func serverInfoSchema() schemaDescriptor {
	return schemaDescriptor{
		ItemType: "server-info",
		Fields:   []string{"baseUrl", "deploymentType", "version", "versionNumbers", "buildNumber", "buildDate", "serverTime"},
	}
}

func issueSchema(itemType string, requestedFields []string) schemaDescriptor {
	fields := []string{"key", "id"}
	for _, field := range requestedFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		fields = append(fields, "fields."+field)
	}
	return schemaDescriptor{ItemType: itemType, Fields: fields}
}

func issueCommentsSchema() schemaDescriptor {
	return schemaDescriptor{ItemType: "issue-comment", Fields: []string{"id", "author", "created", "updated", "bodyText"}}
}

func projectSchema(itemType string) schemaDescriptor {
	return schemaDescriptor{ItemType: itemType, Fields: []string{"id", "key", "name", "projectType", "style", "simplified", "lead", "url", "description"}}
}

func projectStatusesSchema() schemaDescriptor {
	return schemaDescriptor{ItemType: "project-statuses", Fields: []string{"projectKey", "issueTypes.name", "issueTypes.statuses"}}
}

func boardSchema(itemType string) schemaDescriptor {
	return schemaDescriptor{ItemType: itemType, Fields: []string{"id", "name", "type", "projectKey", "projectName", "locationName"}}
}

func boardSnapshotSchema() schemaDescriptor {
	fields := []string{
		"board.id",
		"board.name",
		"board.type",
		"board.projectKey",
		"board.projectName",
		"source.type",
		"source.sprint.id",
		"source.sprint.name",
		"source.sprint.state",
		"source.sprint.goal",
		"totals.totalIssues",
		"totals.myIssues",
		"statusCounts.name",
		"statusCounts.count",
		"issues.key",
	}
	for _, field := range boardSnapshotIssueFields() {
		fields = append(fields, "issues.fields."+field)
	}
	fields = append(fields,
		"page.limit",
		"page.startAt",
		"page.returned",
		"page.total",
		"page.nextStartAt",
		"me.displayName",
		"me.accountId",
		"myIssues.key",
	)
	for _, field := range boardSnapshotIssueFields() {
		fields = append(fields, "myIssues.fields."+field)
	}
	return schemaDescriptor{ItemType: "board-snapshot", Fields: fields}
}

func filterSchema(itemType string) schemaDescriptor {
	return schemaDescriptor{ItemType: itemType, Fields: []string{"id", "name", "owner", "jql", "viewUrl", "searchUrl", "description"}}
}

func fieldSchema(itemType string) schemaDescriptor {
	return schemaDescriptor{ItemType: itemType, Fields: []string{"id", "key", "name", "custom", "schemaType", "schemaItems", "searchable", "orderable", "clauseNames"}}
}

func issueToItem(response jiraIssueResponse, requestedFields []string) issueItem {
	return issueItem{
		ID:     response.ID,
		Key:    response.Key,
		Fields: normalizeIssueFields(response.Fields, requestedFields),
	}
}

func normalizeIssueFields(raw map[string]any, requestedFields []string) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	fields := make(map[string]any, len(requestedFields))
	for _, field := range requestedFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		value, ok := raw[field]
		if !ok {
			continue
		}
		fields[field] = normalizeIssueField(field, value)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func normalizeIssueField(field string, value any) any {
	switch field {
	case "labels", "components", "fixVersions":
		return normalizeStringSlice(value)
	case "parent":
		if typed, ok := value.(map[string]any); ok {
			if key, ok := typed["key"].(string); ok && strings.TrimSpace(key) != "" {
				return key
			}
		}
	}
	return normalizeCompactValue(value)
}

func commentToItem(comment jiraCommentResponse) issueCommentItem {
	return issueCommentItem{
		ID:       comment.ID,
		Author:   normalizeUserRef(comment.Author),
		Created:  comment.Created,
		Updated:  comment.Updated,
		BodyText: truncateText(extractADFText(comment.Body), 240),
	}
}

func projectToItem(project jiraProjectResponse) projectItem {
	return projectItem{
		ID:          project.ID,
		Key:         project.Key,
		Name:        project.Name,
		ProjectType: project.ProjectTypeKey,
		Style:       project.Style,
		Simplified:  project.Simplified,
		Lead:        normalizeUserRef(project.Lead),
		URL:         project.URL,
		Description: truncateText(strings.TrimSpace(project.Description), 240),
	}
}

func projectStatusesToItem(projectKey string, response []jiraProjectIssueTypeStatusesResponse) projectStatusesItem {
	issueTypes := make([]projectIssueTypeStatusesItem, 0, len(response))
	for _, item := range response {
		statuses := make([]string, 0, len(item.Statuses))
		for _, status := range item.Statuses {
			if strings.TrimSpace(status.Name) != "" {
				statuses = append(statuses, status.Name)
			}
		}
		issueTypes = append(issueTypes, projectIssueTypeStatusesItem{
			ID:       item.ID,
			Name:     item.Name,
			Subtask:  item.Subtask,
			Statuses: statuses,
		})
	}
	return projectStatusesItem{ProjectKey: projectKey, IssueTypes: issueTypes}
}

func boardToItem(board jiraBoardResponse) boardItem {
	item := boardItem{
		ID:   board.ID,
		Name: board.Name,
		Type: board.Type,
		Self: board.Self,
	}
	if board.Location != nil {
		item.ProjectKey = board.Location.ProjectKey
		item.ProjectName = board.Location.ProjectName
		item.LocationName = board.Location.DisplayName
	}
	return item
}

func sprintToItem(sprint jiraSprintResponse) *boardSprintItem {
	return &boardSprintItem{
		ID:            sprint.ID,
		Name:          sprint.Name,
		State:         sprint.State,
		StartDate:     sprint.StartDate,
		EndDate:       sprint.EndDate,
		CompleteDate:  sprint.CompleteDate,
		CreatedDate:   sprint.CreatedDate,
		Goal:          sprint.Goal,
		OriginBoardID: sprint.OriginBoardID,
	}
}

func filterToItem(filter jiraFilterResponse) filterItem {
	return filterItem{
		ID:          filter.ID,
		Name:        filter.Name,
		Owner:       normalizeUserRef(filter.Owner),
		JQL:         filter.JQL,
		ViewURL:     filter.ViewURL,
		SearchURL:   filter.SearchURL,
		Description: truncateText(strings.TrimSpace(filter.Description), 240),
	}
}

func fieldToItem(field jiraFieldResponse) fieldItem {
	item := fieldItem{
		ID:          field.ID,
		Key:         field.Key,
		Name:        field.Name,
		Custom:      field.Custom,
		Searchable:  field.Searchable,
		Orderable:   field.Orderable,
		ClauseNames: append([]string(nil), field.ClauseNames...),
	}
	if field.Schema != nil {
		item.SchemaType = field.Schema.Type
		item.SchemaItems = field.Schema.Items
	}
	return item
}

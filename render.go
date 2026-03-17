package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
)

func renderMeText(value jiraMyselfResponse) string {
	lines := []string{
		fmt.Sprintf("display_name: %s", value.DisplayName),
		fmt.Sprintf("active: %t", value.Active),
	}
	if value.EmailAddress != "" {
		lines = append(lines, fmt.Sprintf("email: %s", value.EmailAddress))
	}
	if value.AccountID != "" {
		lines = append(lines, fmt.Sprintf("account_id: %s", value.AccountID))
	}
	if value.AccountType != "" {
		lines = append(lines, fmt.Sprintf("account_type: %s", value.AccountType))
	}
	if value.Locale != "" {
		lines = append(lines, fmt.Sprintf("locale: %s", value.Locale))
	}
	if value.TimeZone != "" {
		lines = append(lines, fmt.Sprintf("time_zone: %s", value.TimeZone))
	}
	return strings.Join(lines, "\n")
}

func renderServerInfoText(value jiraServerInfoResponse) string {
	var lines []string
	if value.BaseURL != "" {
		lines = append(lines, fmt.Sprintf("base_url: %s", value.BaseURL))
	}
	if value.DeploymentType != "" {
		lines = append(lines, fmt.Sprintf("deployment_type: %s", value.DeploymentType))
	}
	if value.Version != "" {
		lines = append(lines, fmt.Sprintf("version: %s", value.Version))
	}
	if len(value.VersionNumbers) > 0 {
		parts := make([]string, 0, len(value.VersionNumbers))
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
	return strings.Join(lines, "\n")
}

func renderIssueText(item issueItem, requestedFields []string) string {
	lines := []string{fmt.Sprintf("key: %s", item.Key)}
	if item.ID != "" {
		lines = append(lines, fmt.Sprintf("id: %s", item.ID))
	}
	for _, field := range requestedFields {
		value, ok := item.Fields[field]
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", field, formatValue(value)))
	}
	return strings.Join(lines, "\n")
}

func renderIssueListText(items []issueItem, requestedFields []string, page pageDescriptor) string {
	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	headers := []string{"KEY"}
	for _, field := range requestedFields {
		headers = append(headers, strings.ToUpper(field))
	}
	_, _ = fmt.Fprintln(writer, strings.Join(headers, "\t"))
	for _, item := range items {
		row := []string{item.Key}
		for _, field := range requestedFields {
			row = append(row, formatValue(item.Fields[field]))
		}
		_, _ = fmt.Fprintln(writer, strings.Join(row, "\t"))
	}
	_ = writer.Flush()
	text := strings.TrimSpace(buffer.String())
	if text == "" {
		text = "no results"
	}
	return text + "\n" + renderPageSummary(page)
}

func renderIssueCommentsText(items []issueCommentItem, page pageDescriptor) string {
	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tAUTHOR\tCREATED\tBODY")
	for _, item := range items {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", item.ID, item.Author, item.Created, item.BodyText)
	}
	_ = writer.Flush()
	text := strings.TrimSpace(buffer.String())
	if text == "" {
		text = "no results"
	}
	return text + "\n" + renderPageSummary(page)
}

func renderProjectListText(items []projectItem, page pageDescriptor) string {
	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "KEY\tNAME\tTYPE\tSTYLE\tLEAD")
	for _, item := range items {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", item.Key, item.Name, item.ProjectType, item.Style, item.Lead)
	}
	_ = writer.Flush()
	text := strings.TrimSpace(buffer.String())
	if text == "" {
		text = "no results"
	}
	return text + "\n" + renderPageSummary(page)
}

func renderProjectText(item projectItem) string {
	lines := []string{
		fmt.Sprintf("key: %s", item.Key),
		fmt.Sprintf("name: %s", item.Name),
	}
	if item.ID != "" {
		lines = append(lines, fmt.Sprintf("id: %s", item.ID))
	}
	if item.ProjectType != "" {
		lines = append(lines, fmt.Sprintf("project_type: %s", item.ProjectType))
	}
	if item.Style != "" {
		lines = append(lines, fmt.Sprintf("style: %s", item.Style))
	}
	lines = append(lines, fmt.Sprintf("simplified: %t", item.Simplified))
	if item.Lead != "" {
		lines = append(lines, fmt.Sprintf("lead: %s", item.Lead))
	}
	if item.URL != "" {
		lines = append(lines, fmt.Sprintf("url: %s", item.URL))
	}
	if item.Description != "" {
		lines = append(lines, fmt.Sprintf("description: %s", item.Description))
	}
	return strings.Join(lines, "\n")
}

func renderProjectStatusesText(item projectStatusesItem) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("project_key: %s", item.ProjectKey))
	for _, issueType := range item.IssueTypes {
		lines = append(lines, fmt.Sprintf("%s: %s", issueType.Name, strings.Join(issueType.Statuses, ", ")))
	}
	return strings.Join(lines, "\n")
}

func renderBoardListText(items []boardItem, page pageDescriptor) string {
	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tNAME\tTYPE\tPROJECT\tLOCATION")
	for _, item := range items {
		_, _ = fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\n", item.ID, item.Name, item.Type, item.ProjectKey, item.LocationName)
	}
	_ = writer.Flush()
	text := strings.TrimSpace(buffer.String())
	if text == "" {
		text = "no results"
	}
	return text + "\n" + renderPageSummary(page)
}

func renderBoardText(item boardItem) string {
	lines := []string{
		fmt.Sprintf("id: %d", item.ID),
		fmt.Sprintf("name: %s", item.Name),
	}
	if item.Type != "" {
		lines = append(lines, fmt.Sprintf("type: %s", item.Type))
	}
	if item.ProjectKey != "" {
		lines = append(lines, fmt.Sprintf("project_key: %s", item.ProjectKey))
	}
	if item.ProjectName != "" {
		lines = append(lines, fmt.Sprintf("project_name: %s", item.ProjectName))
	}
	if item.LocationName != "" {
		lines = append(lines, fmt.Sprintf("location_name: %s", item.LocationName))
	}
	return strings.Join(lines, "\n")
}

func renderFilterListText(items []filterItem, page pageDescriptor) string {
	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tNAME\tOWNER")
	for _, item := range items {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", item.ID, item.Name, item.Owner)
	}
	_ = writer.Flush()
	text := strings.TrimSpace(buffer.String())
	if text == "" {
		text = "no results"
	}
	return text + "\n" + renderPageSummary(page)
}

func renderFilterText(item filterItem) string {
	lines := []string{
		fmt.Sprintf("id: %s", item.ID),
		fmt.Sprintf("name: %s", item.Name),
	}
	if item.Owner != "" {
		lines = append(lines, fmt.Sprintf("owner: %s", item.Owner))
	}
	if item.JQL != "" {
		lines = append(lines, fmt.Sprintf("jql: %s", item.JQL))
	}
	if item.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("view_url: %s", item.ViewURL))
	}
	if item.SearchURL != "" {
		lines = append(lines, fmt.Sprintf("search_url: %s", item.SearchURL))
	}
	if item.Description != "" {
		lines = append(lines, fmt.Sprintf("description: %s", item.Description))
	}
	return strings.Join(lines, "\n")
}

func renderFieldListText(items []fieldItem, page pageDescriptor) string {
	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tNAME\tCUSTOM\tTYPE\tSEARCHABLE")
	for _, item := range items {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%t\t%s\t%t\n", item.ID, item.Name, item.Custom, item.SchemaType, item.Searchable)
	}
	_ = writer.Flush()
	text := strings.TrimSpace(buffer.String())
	if text == "" {
		text = "no results"
	}
	return text + "\n" + renderPageSummary(page)
}

func renderFieldText(item fieldItem) string {
	lines := []string{
		fmt.Sprintf("id: %s", item.ID),
		fmt.Sprintf("name: %s", item.Name),
		fmt.Sprintf("custom: %t", item.Custom),
	}
	if item.Key != "" {
		lines = append(lines, fmt.Sprintf("key: %s", item.Key))
	}
	if item.SchemaType != "" {
		lines = append(lines, fmt.Sprintf("schema_type: %s", item.SchemaType))
	}
	if item.SchemaItems != "" {
		lines = append(lines, fmt.Sprintf("schema_items: %s", item.SchemaItems))
	}
	lines = append(lines, fmt.Sprintf("searchable: %t", item.Searchable))
	lines = append(lines, fmt.Sprintf("orderable: %t", item.Orderable))
	if len(item.ClauseNames) > 0 {
		lines = append(lines, fmt.Sprintf("clause_names: %s", strings.Join(item.ClauseNames, ", ")))
	}
	return strings.Join(lines, "\n")
}

func renderPageSummary(page pageDescriptor) string {
	parts := []string{
		fmt.Sprintf("page: startAt=%d", page.StartAt),
		fmt.Sprintf("returned=%d", page.Returned),
		fmt.Sprintf("limit=%d", page.Limit),
	}
	if page.Total != nil {
		parts = append(parts, fmt.Sprintf("total=%d", *page.Total))
	}
	if page.NextStartAt != nil {
		parts = append(parts, fmt.Sprintf("nextStartAt=%d", *page.NextStartAt))
	}
	return strings.Join(parts, " ")
}

func normalizeCompactValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case string, bool, float64, int, int64:
		return typed
	case []string:
		return typed
	case []any:
		parts := make([]any, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, normalizeCompactValue(item))
		}
		return parts
	case map[string]any:
		if text := extractADFText(typed); text != "" {
			return text
		}
		for _, key := range []string{"displayName", "name", "value", "key", "id", "accountId", "summary"} {
			if preferred, ok := typed[key]; ok {
				return normalizeCompactValue(preferred)
			}
		}
		compact := make(map[string]any)
		for _, key := range []string{"id", "key", "name", "value", "displayName", "summary", "type"} {
			if item, ok := typed[key]; ok {
				compact[key] = normalizeCompactValue(item)
			}
		}
		if len(compact) > 0 {
			return compact
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

func normalizeStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			rendered := strings.TrimSpace(formatValue(normalizeCompactValue(item)))
			if rendered != "" {
				parts = append(parts, rendered)
			}
		}
		return parts
	default:
		rendered := strings.TrimSpace(formatValue(normalizeCompactValue(value)))
		if rendered == "" {
			return nil
		}
		return []string{rendered}
	}
}

func normalizeUserRef(user *jiraUserRef) string {
	if user == nil {
		return ""
	}
	if strings.TrimSpace(user.DisplayName) != "" {
		return user.DisplayName
	}
	if strings.TrimSpace(user.EmailAddress) != "" {
		return user.EmailAddress
	}
	return user.AccountID
}

func extractADFText(value any) string {
	parts := extractADFParts(value)
	joined := strings.Join(parts, " ")
	return strings.Join(strings.Fields(joined), " ")
}

func extractADFParts(value any) []string {
	switch typed := value.(type) {
	case map[string]any:
		var parts []string
		if text, ok := typed["text"].(string); ok && strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
		if content, ok := typed["content"].([]any); ok {
			for _, item := range content {
				parts = append(parts, extractADFParts(item)...)
			}
		}
		return parts
	case []any:
		var parts []string
		for _, item := range typed {
			parts = append(parts, extractADFParts(item)...)
		}
		return parts
	default:
		return nil
	}
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit || limit < 4 {
		return value
	}
	return value[:limit-3] + "..."
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []string:
		return strings.Join(typed, ", ")
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, formatValue(item))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, formatValue(typed[key])))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", typed)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	emailPattern     = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	accountIDPattern = regexp.MustCompile(`[0-9]+:[A-Za-z0-9\-]+`)
	issueKeyPattern  = regexp.MustCompile(`\b[A-Z][A-Z0-9_]+-\d+\b`)
)

type goldenCase struct {
	name string
	args []string
}

type options struct {
	issueKey  string
	project   string
	searchJQL string
	site      string
}

const helpText = `jira-cli golden recorder

Usage:
  go run ./cmd/golden-record [flags]

Flags:
  --issue <KEY>         Jira issue key to use for issue/get and JQL recordings
  --jql <query>         Raw JQL for the JQL search golden
  --project <KEY>       Project key for the project search golden
  --site <url>          Jira base URL override
  -h, --help            Show this help

Auth:
  Uses the same Jira auth inputs as the main jira CLI:
  JIRA_EMAIL + JIRA_TOKEN or JIRA_API_TOKEN

Defaults:
  Missing flags fall back to:
  --site    <- JIRA_SITE
  --issue   <- JIRA_GOLDEN_ISSUE_KEY
  --project <- JIRA_GOLDEN_PROJECT or issue key prefix
  --jql     <- JIRA_GOLDEN_SEARCH_JQL or "key = <ISSUE>"

Outputs:
  Raw outputs go to .local/goldens/raw/
  Sanitized tracked fixtures go to testdata/goldens/live/`

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	options, helpRequested, err := parseOptions(os.Args[1:])
	if err != nil {
		return err
	}
	if helpRequested {
		_, _ = fmt.Fprintln(os.Stdout, helpText)
		return nil
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	if options.site == "" {
		return fmt.Errorf("JIRA_SITE is required for golden recording")
	}
	if options.issueKey == "" {
		return fmt.Errorf("JIRA_GOLDEN_ISSUE_KEY is required for golden recording")
	}

	binaryPath := filepath.Join(repoRoot, ".local", "bin", "jira")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return err
	}
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = repoRoot
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build jira binary: %w", err)
	}

	cases := []goldenCase{
		{name: "me", args: []string{"me"}},
		{name: "serverinfo", args: []string{"serverinfo"}},
		{name: "issue_get", args: []string{"issue", "get", options.issueKey, "--fields", "summary,status,assignee"}},
		{name: "issue_search_project", args: []string{"issue", "search", "--project", options.project, "--limit", "1", "--fields", "summary,status,assignee"}},
		{name: "issue_search_jql", args: []string{"issue", "search", "--jql", options.searchJQL, "--limit", "1", "--fields", "summary,status,assignee"}},
	}

	rawDir := filepath.Join(repoRoot, ".local", "goldens", "raw")
	trackedDir := filepath.Join(repoRoot, "testdata", "goldens", "live")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(trackedDir, 0o755); err != nil {
		return err
	}

	for _, current := range cases {
		textOutput, err := runCase(binaryPath, current.args, options.site)
		if err != nil {
			return fmt.Errorf("%s text: %w", current.name, err)
		}
		jsonOutput, err := runCase(binaryPath, append(append([]string{}, current.args...), "--json"), options.site)
		if err != nil {
			return fmt.Errorf("%s json: %w", current.name, err)
		}

		if err := os.WriteFile(filepath.Join(rawDir, current.name+".stdout.txt"), []byte(textOutput), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(rawDir, current.name+".stdout.json"), []byte(jsonOutput), 0o644); err != nil {
			return err
		}

		sanitizedJSON, err := sanitizeJSONOutput([]byte(jsonOutput), options.site)
		if err != nil {
			return fmt.Errorf("%s sanitize json: %w", current.name, err)
		}
		sanitizedText := sanitizeTextOutput(current.name, textOutput, options.site)

		if err := os.WriteFile(filepath.Join(trackedDir, current.name+".stdout.json"), sanitizedJSON, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(trackedDir, current.name+".stdout.txt"), []byte(sanitizedText), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func parseOptions(argv []string) (options, bool, error) {
	flags := flag.NewFlagSet("golden-record", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var parsed options
	var help bool

	flags.StringVar(&parsed.issueKey, "issue", "", "Issue key for live recordings")
	flags.StringVar(&parsed.searchJQL, "jql", "", "JQL for live search golden")
	flags.StringVar(&parsed.project, "project", "", "Project key for project search golden")
	flags.StringVar(&parsed.site, "site", "", "Jira site override")
	flags.BoolVar(&help, "help", false, "Show help")
	flags.BoolVar(&help, "h", false, "Show help")

	if err := flags.Parse(argv); err != nil {
		return options{}, false, err
	}
	if help {
		return options{}, true, nil
	}
	if len(flags.Args()) > 0 {
		return options{}, false, fmt.Errorf("golden-record does not accept positional arguments")
	}

	if parsed.site == "" {
		parsed.site = strings.TrimSpace(os.Getenv("JIRA_SITE"))
	}
	if parsed.issueKey == "" {
		parsed.issueKey = strings.TrimSpace(os.Getenv("JIRA_GOLDEN_ISSUE_KEY"))
	}
	if parsed.project == "" {
		parsed.project = strings.TrimSpace(os.Getenv("JIRA_GOLDEN_PROJECT"))
	}
	if parsed.project == "" && parsed.issueKey != "" {
		parsed.project = strings.SplitN(parsed.issueKey, "-", 2)[0]
	}
	if parsed.searchJQL == "" {
		parsed.searchJQL = strings.TrimSpace(os.Getenv("JIRA_GOLDEN_SEARCH_JQL"))
	}
	if parsed.searchJQL == "" && parsed.issueKey != "" {
		parsed.searchJQL = fmt.Sprintf("key = %s", parsed.issueKey)
	}

	return parsed, false, nil
}

func runCase(binaryPath string, args []string, site string) (string, error) {
	command := exec.Command(binaryPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	command.Env = os.Environ()
	if strings.TrimSpace(site) != "" {
		command.Env = append(command.Env, "JIRA_SITE="+site)
	}
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func sanitizeJSONOutput(raw []byte, site string) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	sanitized := sanitizeJSONValue(nil, value, site)
	return json.MarshalIndent(sanitized, "", "  ")
}

func sanitizeJSONValue(path []string, value any, site string) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, item := range typed {
			sanitized[key] = sanitizeJSONValue(append(path, key), item, site)
		}
		return sanitized
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, sanitizeJSONValue(path, item, site))
		}
		return result
	case string:
		return sanitizeJSONString(path, typed, site)
	case float64:
		if isPath(path, "buildNumber") {
			return float64(1000)
		}
		return typed
	default:
		return typed
	}
}

func sanitizeJSONString(path []string, value string, site string) string {
	switch {
	case isPath(path, "emailAddress"):
		return "agent@example.com"
	case isPath(path, "accountId"):
		return "account-id-1"
	case isPath(path, "displayName"):
		return "Agent User"
	case isPath(path, "summary"):
		return "Issue summary"
	case isPath(path, "version"):
		return "0.0.0"
	case isPath(path, "buildDate"):
		return "2026-01-01T00:00:00.000+0000"
	case isPath(path, "serverTime"):
		return "2026-01-01T00:00:00.000+0000"
	case isIssueKeyPath(path):
		return "DEMO-123"
	case isPath(path, "jql"):
		value = issueKeyPattern.ReplaceAllString(value, "DEMO-123")
		value = strings.ReplaceAll(value, `"SCWI"`, `"DEMO"`)
		return value
	case isPath(path, "baseUrl"), isPath(path, "displayUrl"), isPath(path, "self"):
		return sanitizeURLString(value, site)
	case containsPath(path, "status") && isPath(path, "name"):
		return value
	case containsPath(path, "assignee") && (isPath(path, "displayName") || isPath(path, "name")):
		return "Agent User"
	default:
		value = strings.ReplaceAll(value, site, "https://jira.example.test")
		value = emailPattern.ReplaceAllString(value, "agent@example.com")
		value = accountIDPattern.ReplaceAllString(value, "account-id-1")
		value = issueKeyPattern.ReplaceAllString(value, "DEMO-123")
		return value
	}
}

func sanitizeTextOutput(name string, value string, site string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for index, line := range lines {
		line = strings.ReplaceAll(line, site, "https://jira.example.test")
		line = emailPattern.ReplaceAllString(line, "agent@example.com")
		line = accountIDPattern.ReplaceAllString(line, "account-id-1")
		line = issueKeyPattern.ReplaceAllString(line, "DEMO-123")
		switch {
		case strings.HasPrefix(line, "display_name: "):
			line = "display_name: Agent User"
		case strings.HasPrefix(line, "account_id: "):
			line = "account_id: account-id-1"
		case strings.HasPrefix(line, "assignee: "):
			line = "assignee: Agent User"
		case strings.HasPrefix(line, "summary: "):
			line = "summary: Issue summary"
		case strings.Contains(line, "summary="):
			line = regexp.MustCompile(`summary=[^|]+`).ReplaceAllString(line, "summary=Issue summary ")
		}
		switch {
		case strings.Contains(line, "assignee="):
			line = regexp.MustCompile(`assignee=[^|]+`).ReplaceAllString(line, "assignee=Agent User")
		case strings.HasPrefix(line, "version: "):
			line = "version: 0.0.0"
		case strings.HasPrefix(line, "version_numbers: "):
			line = "version_numbers: 1001.0.0"
		case strings.HasPrefix(line, "build_number: "):
			line = "build_number: 1000"
		case strings.HasPrefix(line, "build_date: "):
			line = "build_date: 2026-01-01T00:00:00.000+0000"
		case strings.HasPrefix(line, "server_time: "):
			line = "server_time: 2026-01-01T00:00:00.000+0000"
		case strings.HasPrefix(line, "jql: "):
			line = strings.ReplaceAll(line, `"SCWI"`, `"DEMO"`)
		}
		lines[index] = line
	}
	return strings.Join(lines, "\n")
}

func sanitizeURLString(value string, site string) string {
	value = strings.ReplaceAll(value, site, "https://jira.example.test")
	value = accountIDPattern.ReplaceAllString(value, "account-id-1")
	if parsed, err := url.Parse(value); err == nil {
		parsed.Host = "jira.example.test"
		value = parsed.String()
	}
	value = issueKeyPattern.ReplaceAllString(value, "DEMO-123")
	return value
}

func isPath(path []string, last string) bool {
	return len(path) > 0 && path[len(path)-1] == last
}

func containsPath(path []string, want string) bool {
	for _, item := range path {
		if item == want {
			return true
		}
	}
	return false
}

func isIssueKeyPath(path []string) bool {
	if len(path) == 1 && path[0] == "key" {
		return true
	}
	return len(path) == 2 && path[0] == "issues" && path[1] == "key"
}

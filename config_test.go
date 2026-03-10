package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeConfigUsesExplicitConfigFile(t *testing.T) {
	configPath := writeConfigFile(t, t.TempDir(), `{"site":"https://config.atlassian.net","email":"config@example.com","token":"config-token"}`)

	resolved, err := resolveRuntimeConfig(commandOptions{configPath: configPath}, testConfigEnvironment(t, nil))
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if resolved.site != "https://config.atlassian.net" || resolved.email != "config@example.com" || resolved.token != "config-token" {
		t.Fatalf("unexpected resolved config: %+v", resolved)
	}
}

func TestResolveRuntimeConfigEnvOverridesConfig(t *testing.T) {
	configPath := writeConfigFile(t, t.TempDir(), `{"site":"https://config.atlassian.net","email":"config@example.com","token":"config-token"}`)
	env := map[string]string{
		"JIRA_SITE":  "https://env.atlassian.net",
		"JIRA_EMAIL": "env@example.com",
		"JIRA_TOKEN": "env-token",
	}

	resolved, err := resolveRuntimeConfig(commandOptions{configPath: configPath}, testConfigEnvironment(t, env))
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if resolved.site != "https://env.atlassian.net" || resolved.email != "env@example.com" || resolved.token != "env-token" {
		t.Fatalf("expected env to override config, got %+v", resolved)
	}
}

func TestResolveRuntimeConfigFlagsOverrideEnv(t *testing.T) {
	env := map[string]string{
		"JIRA_SITE":      "https://env.atlassian.net",
		"JIRA_EMAIL":     "env@example.com",
		"JIRA_API_TOKEN": "env-token",
	}

	resolved, err := resolveRuntimeConfig(commandOptions{
		site:  "https://flag.atlassian.net",
		email: "flag@example.com",
		token: "flag-token",
	}, testConfigEnvironment(t, env))
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if resolved.site != "https://flag.atlassian.net" || resolved.email != "flag@example.com" || resolved.token != "flag-token" {
		t.Fatalf("expected flags to override env, got %+v", resolved)
	}
}

func TestResolveRuntimeConfigUsesDefaultConfigPathWhenPresent(t *testing.T) {
	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, ".config", "jira")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"site":"https://default.atlassian.net","email":"default@example.com","token":"default-token"}`), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	resolved, err := resolveRuntimeConfig(commandOptions{}, configEnvironment{
		homeDir:   func() (string, error) { return homeDir, nil },
		lookupEnv: func(string) (string, bool) { return "", false },
		readFile:  os.ReadFile,
	})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if resolved.configPath != configPath {
		t.Fatalf("expected default config path %q, got %q", configPath, resolved.configPath)
	}
	if resolved.site != "https://default.atlassian.net" {
		t.Fatalf("unexpected resolved site: %+v", resolved)
	}
}

func TestResolveRuntimeConfigRejectsUnknownConfigFields(t *testing.T) {
	configPath := writeConfigFile(t, t.TempDir(), `{"site":"https://config.atlassian.net","unexpected":"value"}`)

	_, err := resolveRuntimeConfig(commandOptions{configPath: configPath}, testConfigEnvironment(t, nil))
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse jira config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReportsMissingFields(t *testing.T) {
	err := (resolvedRuntimeConfig{}).Validate(configRequirements{
		requireSite:  true,
		requireEmail: true,
		requireToken: true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing required Jira config: site, email, token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAllowsSiteOnlyRequirement(t *testing.T) {
	err := (resolvedRuntimeConfig{site: "https://example.atlassian.net"}).Validate(configRequirements{
		requireSite: true,
	})
	if err != nil {
		t.Fatalf("expected site-only validation to pass, got %v", err)
	}
}

func testConfigEnvironment(t *testing.T, values map[string]string) configEnvironment {
	t.Helper()
	return configEnvironment{
		homeDir: func() (string, error) { return t.TempDir(), nil },
		lookupEnv: func(key string) (string, bool) {
			value, ok := values[key]
			return value, ok
		},
		readFile: os.ReadFile,
	}
}

func writeConfigFile(t *testing.T, dir string, contents string) string {
	t.Helper()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return configPath
}

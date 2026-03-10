package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const defaultConfigRelativePath = ".config/jira/config.json"

type fileConfig struct {
	Email string `json:"email"`
	Site  string `json:"site"`
	Token string `json:"token"`
}

type resolvedRuntimeConfig struct {
	configPath string
	email      string
	site       string
	token      string
}

type configRequirements struct {
	requireEmail bool
	requireSite  bool
	requireToken bool
}

type configEnvironment struct {
	homeDir   func() (string, error)
	lookupEnv func(string) (string, bool)
	readFile  func(string) ([]byte, error)
}

func defaultConfigEnvironment() configEnvironment {
	return configEnvironment{
		homeDir:   os.UserHomeDir,
		lookupEnv: os.LookupEnv,
		readFile:  os.ReadFile,
	}
}

func resolveRuntimeConfig(options commandOptions, env configEnvironment) (resolvedRuntimeConfig, error) {
	resolved := resolvedRuntimeConfig{}

	configPath, explicit, err := resolveConfigPath(options.configPath, env)
	if err != nil {
		return resolved, err
	}

	if configPath != "" {
		parsed, err := loadFileConfig(configPath, explicit, env)
		if err != nil {
			return resolved, err
		}
		resolved.configPath = configPath
		resolved.site = strings.TrimSpace(parsed.Site)
		resolved.email = strings.TrimSpace(parsed.Email)
		resolved.token = strings.TrimSpace(parsed.Token)
	}

	if value, ok := firstEnv(env, "JIRA_SITE", "JIRA_BASE_URL"); ok {
		resolved.site = value
	}
	if value, ok := firstEnv(env, "JIRA_EMAIL"); ok {
		resolved.email = value
	}
	if value, ok := firstEnv(env, "JIRA_TOKEN", "JIRA_API_TOKEN"); ok {
		resolved.token = value
	}

	if value := strings.TrimSpace(options.site); value != "" {
		resolved.site = value
	}
	if value := strings.TrimSpace(options.email); value != "" {
		resolved.email = value
	}
	if value := strings.TrimSpace(options.token); value != "" {
		resolved.token = value
	}

	return resolved, nil
}

func (config resolvedRuntimeConfig) Validate(requirements configRequirements) error {
	var missing []string
	if requirements.requireSite && strings.TrimSpace(config.site) == "" {
		missing = append(missing, "site")
	}
	if requirements.requireEmail && strings.TrimSpace(config.email) == "" {
		missing = append(missing, "email")
	}
	if requirements.requireToken && strings.TrimSpace(config.token) == "" {
		missing = append(missing, "token")
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"missing required Jira config: %s; set flags, env vars (JIRA_SITE/JIRA_BASE_URL, JIRA_EMAIL, JIRA_TOKEN/JIRA_API_TOKEN), or ~/.config/jira/config.json",
			strings.Join(missing, ", "),
		)
	}

	if strings.TrimSpace(config.site) != "" {
		parsed, err := url.Parse(config.site)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid Jira site URL: %q", config.site)
		}
	}

	return nil
}

func resolveConfigPath(flagPath string, env configEnvironment) (string, bool, error) {
	if value := strings.TrimSpace(flagPath); value != "" {
		return value, true, nil
	}

	if value, ok := lookupTrimmedEnv(env, "JIRA_CONFIG"); ok {
		return value, true, nil
	}

	if env.homeDir == nil || env.readFile == nil {
		return "", false, nil
	}

	homeDir, err := env.homeDir()
	if err != nil {
		return "", false, fmt.Errorf("resolve jira home directory: %w", err)
	}
	defaultPath := filepath.Join(homeDir, defaultConfigRelativePath)
	if _, err := env.readFile(defaultPath); err == nil {
		return defaultPath, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("read default jira config %s: %w", defaultPath, err)
	}

	return "", false, nil
}

func loadFileConfig(path string, explicit bool, env configEnvironment) (fileConfig, error) {
	if env.readFile == nil {
		return fileConfig{}, errors.New("config reader is unavailable")
	}
	bytesValue, err := env.readFile(path)
	if err != nil {
		if explicit && errors.Is(err, os.ErrNotExist) {
			return fileConfig{}, fmt.Errorf("jira config file not found: %s", path)
		}
		return fileConfig{}, fmt.Errorf("read jira config %s: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(bytesValue))
	decoder.DisallowUnknownFields()

	var parsed fileConfig
	if err := decoder.Decode(&parsed); err != nil {
		return fileConfig{}, fmt.Errorf("parse jira config %s: %w", path, err)
	}

	return parsed, nil
}

func firstEnv(env configEnvironment, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := lookupTrimmedEnv(env, key); ok {
			return value, true
		}
	}
	return "", false
}

func lookupTrimmedEnv(env configEnvironment, key string) (string, bool) {
	if env.lookupEnv == nil {
		return "", false
	}
	value, ok := env.lookupEnv(key)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}

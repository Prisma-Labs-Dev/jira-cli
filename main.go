package main

import (
	"fmt"
	"io"
	"os"
)

const packageVersion = "0.2.0"

type cliEnvironment struct {
	stderr io.Writer
	stdout io.Writer
}

var configEnvironmentFactory = defaultConfigEnvironment
var jiraAPIFactory = func(config resolvedRuntimeConfig) (jiraAPI, error) {
	return newJiraClient(config), nil
}

func main() {
	os.Exit(run(os.Args[1:], cliEnvironment{
		stderr: os.Stderr,
		stdout: os.Stdout,
	}))
}

func run(argv []string, env cliEnvironment) int {
	if len(argv) == 0 || isHelpFlag(argv[0]) {
		_, _ = fmt.Fprintln(env.stdout, rootHelp)
		return 0
	}

	switch argv[0] {
	case "version", "--version":
		_, _ = fmt.Fprintln(env.stdout, packageVersion)
		return 0
	case "me":
		return exitForError(runMe(argv[1:], env), env)
	case "serverinfo":
		return exitForError(runServerInfo(argv[1:], env), env)
	case "issue":
		return exitForError(runIssue(argv[1:], env), env)
	case "project":
		return exitForError(runProject(argv[1:], env), env)
	case "board":
		return exitForError(runBoard(argv[1:], env), env)
	case "filter":
		return exitForError(runFilter(argv[1:], env), env)
	case "field":
		return exitForError(runField(argv[1:], env), env)
	default:
		_, _ = fmt.Fprintf(env.stderr, "Error: unknown command: %s\n\n%s\n", argv[0], rootHelp)
		return 1
	}
}

func exitForError(err error, env cliEnvironment) int {
	if err == nil {
		return 0
	}
	_, _ = fmt.Fprintf(env.stderr, "Error: %s\n", err)
	return 1
}

.PHONY: lint test test-live test-live-bw install-local verify-local ci

lint:
	./scripts/lint.sh

test:
	go test ./...

test-live:
	JIRA_LIVE_E2E=1 go test -run 'TestLiveCLIContracts|TestLiveCLIContractsDocumentation' ./...

test-live-bw:
	./scripts/test_live_with_bw.sh

install-local:
	mkdir -p /Users/vabole/.local/bin
	go build -o /Users/vabole/.local/bin/jira .

verify-local:
	test -x /Users/vabole/.local/bin/jira
	/Users/vabole/.local/bin/jira --help >/dev/null
	/Users/vabole/.local/bin/jira issue search --help >/dev/null
	/Users/vabole/.local/bin/jira project list --help >/dev/null
	/Users/vabole/.local/bin/jira field list --help >/dev/null

ci:
	$(MAKE) lint
	$(MAKE) test

.PHONY: lint test test-live test-live-bw install-local verify-local brew-tap-local brew-install-local brew-reinstall-local brew-test-local ci

BREW_TAP=prisma-labs-dev/jira-cli

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

brew-install-local:
	$(MAKE) brew-tap-local
	brew install --HEAD $(BREW_TAP)/jira

brew-reinstall-local:
	$(MAKE) brew-tap-local
	brew reinstall --HEAD $(BREW_TAP)/jira

brew-test-local:
	brew test jira

brew-tap-local:
	brew tap $(BREW_TAP) $(CURDIR)

ci:
	$(MAKE) lint
	$(MAKE) test

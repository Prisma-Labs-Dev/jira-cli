.PHONY: test install-local verify-local ci

test:
	go test ./...

install-local:
	mkdir -p /Users/vabole/.local/bin
	go build -o /Users/vabole/.local/bin/jira .

verify-local:
	test -x /Users/vabole/.local/bin/jira
	/Users/vabole/.local/bin/jira --help >/dev/null

ci:
	$(MAKE) test

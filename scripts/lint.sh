#!/usr/bin/env bash
set -euo pipefail

unformatted=$(gofmt -l .)
if [[ -n "$unformatted" ]]; then
  echo "gofmt reported unformatted files:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...
./scripts/check_file_sizes.sh

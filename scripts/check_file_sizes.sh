#!/usr/bin/env bash
set -euo pipefail

max_go_lines=500
max_test_lines=700
violations=0

while IFS= read -r -d '' file; do
  lines=$(wc -l < "$file" | tr -d ' ')
  limit=$max_go_lines
  if [[ "$file" == *_test.go ]]; then
    limit=$max_test_lines
  fi
  if (( lines > limit )); then
    echo "$file exceeds line budget: $lines > $limit" >&2
    violations=1
  fi
done < <(find . -type f -name '*.go' -not -path './vendor/*' -print0)

if (( violations != 0 )); then
  exit 1
fi

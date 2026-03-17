#!/usr/bin/env bash
set -euo pipefail

if ! command -v bw >/dev/null 2>&1; then
  echo "bw CLI is required" >&2
  exit 1
fi

if [[ -z "${BW_SESSION:-}" ]] && command -v zsh >/dev/null 2>&1; then
  bw_session_from_zsh="$(zsh -lc 'printf %s "$BW_SESSION"')"
  if [[ -n "$bw_session_from_zsh" ]]; then
    export BW_SESSION="$bw_session_from_zsh"
  fi
fi

item_name="${BW_JIRA_ITEM_NAME:-Confluence CLI}"
items_json="$(mktemp)"
item_id="$(
  bw list items >"$items_json"
  node - "$item_name" "$items_json" <<'NODE'
const fs = require("fs");

const target = (process.argv[2] || "").trim().toLowerCase();
const file = process.argv[3];
const items = JSON.parse(fs.readFileSync(file, "utf8"));
const match = items.find((item) => (item.name || "").trim().toLowerCase() === target);
if (!match) {
  console.error(`Bitwarden item not found: ${process.argv[2]}`);
  process.exit(1);
}
process.stdout.write(match.id);
NODE
)"

tmp_json="$(mktemp)"
cleanup() {
  rm -f "$items_json"
  rm -f "$tmp_json"
}
trap cleanup EXIT

bw get item "$item_id" >"$tmp_json"

eval "$(
node - "$tmp_json" <<'NODE'
const fs = require("fs");

const file = process.argv[2];
const item = JSON.parse(fs.readFileSync(file, "utf8"));

const fields = Object.fromEntries((item.fields || []).map((field) => [field.name, field.value]));
const rawSite =
  process.env.JIRA_SITE ||
  process.env.BW_JIRA_SITE ||
  fields.JIRA_SITE ||
  fields.ATLASSIAN_URL ||
  "";
const site = rawSite.replace(/\/wiki\/?$/, "");
const email =
  fields.JIRA_EMAIL ||
  fields.CONFLUENCE_EMAIL ||
  (item.login && item.login.username) ||
  "";
const token =
  fields.JIRA_TOKEN ||
  fields.JIRA_API_TOKEN ||
  fields.CONFLUENCE_API_TOKEN ||
  (item.login && item.login.password) ||
  "";

if (!site || !email || !token) {
  console.error("Bitwarden item is missing one of site/email/token. Set JIRA_SITE or BW_JIRA_SITE when the vault item only contains a Confluence URL.");
  process.exit(1);
}

function shellEscape(value) {
  return `'${String(value).replace(/'/g, `'\"'\"'`)}'`;
}

console.log(`export JIRA_LIVE_E2E=1`);
console.log(`export JIRA_SITE=${shellEscape(site)}`);
console.log(`export JIRA_EMAIL=${shellEscape(email)}`);
console.log(`export JIRA_TOKEN=${shellEscape(token)}`);
NODE
)"

make test-live

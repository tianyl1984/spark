#!/usr/bin/env bash
#
# fail.sh — runs after execute.sh fails.
# spark passes execute.sh's combined output as the first argument ($1).
# This sends that output as a notification.
#
set -euo pipefail

URL="http://127.0.0.1:3001/api/sendNotify"
MSG="${1:-execute failed}"

# JSON-escape the message so newlines/quotes in the output don't break the body.
if command -v jq >/dev/null 2>&1; then
  BODY=$(jq -n --arg msg "$MSG" '{msg: $msg}')
else
  # Minimal fallback escaping (\, ", newline, tab, carriage return).
  ESCAPED=${MSG//\\/\\\\}
  ESCAPED=${ESCAPED//\"/\\\"}
  ESCAPED=${ESCAPED//$'\n'/\\n}
  ESCAPED=${ESCAPED//$'\r'/\\r}
  ESCAPED=${ESCAPED//$'\t'/\\t}
  BODY="{\"msg\":\"${ESCAPED}\"}"
fi

curl -fsS -X POST "$URL" \
  -H "Content-Type: application/json" \
  -d "$BODY"

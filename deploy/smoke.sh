#!/usr/bin/env bash
# Read-only smoke test of a deployed ClaimFlow (deploy.yml runs it after every release).
#
#   bash deploy/smoke.sh <service URL> <expected commit SHA>
#
# Fails unless: /api/v1/health is 200 with `version` equal to the SHA; /api/v1/ready is 200 (the
# database answers); / serves the web build; and an unknown /api route is a JSON 404 problem.
set -euo pipefail

URL="${1:?Usage: smoke.sh <service URL> <expected commit SHA>}"
SHA="${2:?Usage: smoke.sh <service URL> <expected commit SHA>}"
URL="${URL%/}"
failed=0

pass() { echo "PASS $1"; }
fail() { echo "FAIL $1" >&2; failed=1; }

# A new revision can take a few seconds to answer; retry health before the other checks.
health=""
for _ in $(seq 1 10); do
  health="$(curl -fsS --max-time 10 "$URL/api/v1/health" 2>/dev/null)" && break
  sleep 3
done
if [[ "$health" == *"\"version\":\"$SHA\""* ]]; then
  pass "/api/v1/health reports $SHA"
else
  fail "/api/v1/health: expected version $SHA, got: ${health:-no answer}"
fi

# `|| true`: on a transport error curl still prints %{http_code} (000), so the check reports FAIL instead of
# set -e stopping the script silently.
status="$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$URL/api/v1/ready" || true)"
[[ "$status" == 200 ]] && pass "/api/v1/ready 200" || fail "/api/v1/ready: $status"

home="$(curl -s -w '\n%{http_code} %{content_type}' --max-time 15 "$URL/" || true)"
if [[ "$(tail -n1 <<<"$home")" == "200 text/html"* && "$home" == *"<div id=\"app\">"* ]]; then
  pass "/ serves the web build"
else
  fail "/: $(tail -n1 <<<"$home")"
fi

missing="$(curl -s -o /dev/null -w '%{http_code} %{content_type}' --max-time 15 "$URL/api/v1/smoke-unknown" || true)"
[[ "$missing" == "404 application/problem+json"* ]] && pass "unknown /api route is a JSON 404" || fail "unknown /api route: $missing"

exit "$failed"

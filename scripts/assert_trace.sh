#!/usr/bin/env bash
# assert_trace.sh — Wave 5 trace assertion harness.
#
# Drives a click through one of the three portals (or issues a direct API call
# when --mode=api is requested), captures X-Request-ID from the response,
# sleeps 5s for Tempo ingest (per plan/EXECUTION.md §5 R10), queries Tempo's
# trace API, and asserts:
#   * ≥3 OTEL spans in the returned trace
#   * ≥3 distinct service.name attributes across those spans
#
# Exits 0 on success; non-zero on first assertion failure.
#
# Usage:
#   scripts/assert_trace.sh <portal> <ctaKey> [--mode=api|playwright] [--base-url=...] [--tempo-url=...]
#
# Where <portal> ∈ {admin,provider,student} and <ctaKey> is one of the
# pre-defined click paths in CTA_MAP below (or a raw "METHOD /api/path"
# pair for ad-hoc probes).
#
# Pre-defined click paths (portal:ctaKey → what it exercises):
#   admin:ops                    — /ops page load → metrics + incident-service
#   admin:incidents-create       — POST /api/incidents → incident-service
#   admin:flags-toggle           — PUT /api/flags/{key} → feature-flag-service
#   admin:status                 — GET /api/status → incident-service GetPublicStatus
#   provider:grade-assignment    — GET /api/grading/queue/:id → assignment-grading
#   provider:roster              — GET /api/metrics/roster-health → metrics-service
#   student:today                — GET /api/ai/welcome → ai-service
#   student:video-position       — PUT /api/content/:id/position → content-management
#   student:study-plan           — POST /api/ai/study-plan → ai-service
#
# Environment overrides:
#   GATEWAY_URL (default http://localhost:8080)
#   TEMPO_URL   (default http://localhost:3200)
#   MIN_SPANS   (default 3)
#   MIN_SERVICES (default 3)
#   TEMPO_WAIT_SECS (default 5)
#   AUTH_TOKEN — bearer token for authed routes; optional for public routes

set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
TEMPO_URL="${TEMPO_URL:-http://localhost:3200}"
MIN_SPANS="${MIN_SPANS:-3}"
MIN_SERVICES="${MIN_SERVICES:-3}"
TEMPO_WAIT_SECS="${TEMPO_WAIT_SECS:-5}"
MODE="api"

RED=$'\033[0;31m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
BLUE=$'\033[0;34m'
NC=$'\033[0m'

portal="${1:-}"
cta_key="${2:-}"
shift $(( $# >= 2 ? 2 : $# )) || true

for arg in "$@"; do
  case "$arg" in
    --mode=*) MODE="${arg#--mode=}" ;;
    --base-url=*) GATEWAY_URL="${arg#--base-url=}" ;;
    --tempo-url=*) TEMPO_URL="${arg#--tempo-url=}" ;;
    *) echo "unknown argument: $arg" >&2; exit 2 ;;
  esac
done

if [[ -z "$portal" || -z "$cta_key" ]]; then
  echo "usage: $0 <admin|provider|student> <cta-key> [--mode=api|playwright]" >&2
  exit 2
fi

case "$portal" in
  admin|provider|student) ;;
  *) echo "portal must be one of admin|provider|student (got '$portal')" >&2; exit 2 ;;
esac

# ---------------------------------------------------------------------------
# Click-path registry. Each entry is "METHOD|PATH|BODY" — BODY may be empty.
# Add auth token via Authorization: Bearer $AUTH_TOKEN when present.
# ---------------------------------------------------------------------------
declare -a CTA_METHOD
declare -a CTA_PATH
declare -a CTA_BODY
declare -a CTA_NAME
declare -a CTA_PAGE  # for playwright click-mode (portal page URL)
declare -a CTA_SELECTOR

register_cta () {
  local key="$1" method="$2" path="$3" body="$4" page="$5" selector="$6"
  CTA_NAME+=("$key")
  CTA_METHOD+=("$method")
  CTA_PATH+=("$path")
  CTA_BODY+=("$body")
  CTA_PAGE+=("$page")
  CTA_SELECTOR+=("$selector")
}

# admin
register_cta "admin:ops"                 "GET"    "/api/metrics/platform"                  ""                                    "http://localhost:3003/ops"        '[data-testid="ops-top-incident"]'
register_cta "admin:incidents-create"    "POST"   "/api/incidents"                         '{"title":"trace-probe","impact":"synthetic probe","priority":"P4"}' \
                                                                                                                                   "http://localhost:3003/incidents"  '[data-testid="create-incident"]'
register_cta "admin:flags-toggle"        "GET"    "/api/flags"                             ""                                    "http://localhost:3003/flags"      '[data-testid="flag-row"]'
register_cta "admin:status"              "GET"    "/api/status"                            ""                                    "http://localhost:3003/status"     'main'

# provider — paths must match gateway yaml (see config/gateway-config.yaml route_overrides).
# /api/grading/queue/:assignment_id and /api/content/videos/:id/position are parameterised;
# EXECUTION.md shorthand omitted the params. 'probe' is a synthetic id — backend may 404 but
# the request still traverses gateway→service and produces a trace.
register_cta "provider:grade-assignment" "GET"    "/api/grading/queue/probe"               ""                                    "http://localhost:3002/grade"      '[data-testid="submission-row"]'
register_cta "provider:roster"           "GET"    "/api/metrics/roster-health"             ""                                    "http://localhost:3002/roster"     'main'

# student
register_cta "student:today"             "GET"    "/api/ai/welcome"                        ""                                    "http://localhost:3000/today"      'main'
register_cta "student:video-position"    "PUT"    "/api/content/videos/probe/position"     '{"position_seconds":10}'            "http://localhost:3000/modules"    '[data-testid="video-position-update"]'
register_cta "student:study-plan"        "POST"   "/api/ai/study-plan"                     '{"course_id":"probe"}'              "http://localhost:3000/plan"       '[data-testid="generate-plan"]'

lookup_cta_idx () {
  local needle="$1"
  for i in "${!CTA_NAME[@]}"; do
    if [[ "${CTA_NAME[$i]}" == "$needle" ]]; then
      echo "$i"; return 0
    fi
  done
  return 1
}

full_key="${portal}:${cta_key}"
idx="$(lookup_cta_idx "$full_key" || true)"
if [[ -z "$idx" ]]; then
  # short form: caller passed 'ops' for admin etc. Try bare match.
  idx="$(lookup_cta_idx "${portal}:${cta_key#${portal}:}" || true)"
fi
if [[ -z "$idx" ]]; then
  echo "unknown CTA '$full_key'. Known:" >&2
  printf '  %s\n' "${CTA_NAME[@]}" >&2
  exit 2
fi

method="${CTA_METHOD[$idx]}"
path="${CTA_PATH[$idx]}"
body="${CTA_BODY[$idx]}"
page_url="${CTA_PAGE[$idx]}"
selector="${CTA_SELECTOR[$idx]}"

for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "${RED}missing required tool: $cmd${NC}" >&2
    exit 3
  fi
done

printf '%s==> %s / %s%s\n' "$BLUE" "$portal" "$cta_key" "$NC"
printf '    endpoint: %s %s%s\n' "$method" "$GATEWAY_URL" "$path"
printf '    mode:     %s\n' "$MODE"

# ---------------------------------------------------------------------------
# Capture X-Request-ID
# ---------------------------------------------------------------------------
request_id=""
traceparent_hex=""

capture_via_api () {
  local tmp_headers
  tmp_headers="$(mktemp)"
  local curl_args=(-s -o /dev/null -D "$tmp_headers" -X "$method")
  curl_args+=(-H "Content-Type: application/json")
  if [[ -n "${AUTH_TOKEN:-}" ]]; then
    curl_args+=(-H "Authorization: Bearer ${AUTH_TOKEN}")
  fi
  if [[ -n "$body" && "$method" != "GET" ]]; then
    curl_args+=(--data "$body")
  fi
  curl_args+=(--max-time 10 "$GATEWAY_URL$path")
  curl "${curl_args[@]}" || true
  request_id="$(awk -F': *' 'tolower($1) == "x-request-id" {gsub(/\r/,"",$2); print $2}' "$tmp_headers" | tail -n1)"
  traceparent_hex="$(awk -F': *' 'tolower($1) == "traceparent" {gsub(/\r/,"",$2); print $2}' "$tmp_headers" | tail -n1)"
  rm -f "$tmp_headers"
}

capture_via_playwright () {
  # Delegates to scripts/assert_trace_click.mjs which launches headless Chrome,
  # navigates to $page_url, clicks $selector, and prints "request_id=<uuid>".
  local launcher="$(dirname "$0")/assert_trace_click.mjs"
  if [[ ! -f "$launcher" ]]; then
    echo "${RED}playwright launcher missing at $launcher${NC}" >&2
    exit 3
  fi
  if ! command -v node >/dev/null 2>&1; then
    echo "${RED}playwright mode requires node${NC}" >&2
    exit 3
  fi
  local pw_output
  pw_output="$(PLAYWRIGHT_BROWSERS_PATH=0 node "$launcher" \
    --url "$page_url" \
    --selector "$selector" \
    --gateway "$GATEWAY_URL" 2>&1)" || {
    echo "${RED}playwright click failed:${NC}" >&2
    echo "$pw_output" >&2
    exit 4
  }
  request_id="$(awk -F'=' '/^request_id=/{print $2}' <<<"$pw_output" | tail -n1)"
  traceparent_hex="$(awk -F'=' '/^traceparent=/{print $2}' <<<"$pw_output" | tail -n1)"
}

case "$MODE" in
  api)        capture_via_api ;;
  playwright) capture_via_playwright ;;
  *) echo "unknown mode: $MODE"; exit 2 ;;
esac

if [[ -z "$request_id" ]]; then
  printf '%s[FAIL]%s gateway did not echo X-Request-ID\n' "$RED" "$NC" >&2
  exit 5
fi

# Extract W3C trace id (chars 4..35) from traceparent 00-<trace>-<span>-<flags>
trace_id=""
if [[ -n "$traceparent_hex" && "$traceparent_hex" =~ ^[0-9]{2}-([0-9a-f]{32})-[0-9a-f]{16}-[0-9a-f]{2}$ ]]; then
  trace_id="${BASH_REMATCH[1]}"
fi

printf '    request_id:  %s\n' "$request_id"
printf '    traceparent: %s\n' "${traceparent_hex:-<none>}"
printf '    trace_id:    %s\n' "${trace_id:-<unknown>}"

# ---------------------------------------------------------------------------
# Wait for Tempo ingest, then query.
# ---------------------------------------------------------------------------
printf '    waiting %ss for Tempo ingest...\n' "$TEMPO_WAIT_SECS"
sleep "$TEMPO_WAIT_SECS"

fetch_trace () {
  local tid="$1"
  curl -sf "$TEMPO_URL/api/traces/$tid"
}

trace_json=""
if [[ -n "$trace_id" ]]; then
  trace_json="$(fetch_trace "$trace_id" 2>/dev/null || true)"
fi

# Fallback: search by request_id tag via TraceQL
if [[ -z "$trace_json" ]]; then
  printf '    %sdirect trace fetch empty, searching by request_id tag...%s\n' "$YELLOW" "$NC"
  search_json="$(curl -sf \
    --data-urlencode "q={ .request_id = \"$request_id\" }" \
    -G "$TEMPO_URL/api/search" 2>/dev/null || true)"
  if [[ -n "$search_json" ]]; then
    resolved_id="$(printf '%s' "$search_json" | jq -r '.traces[0].traceID // empty' 2>/dev/null || true)"
    if [[ -n "$resolved_id" && -z "$trace_id" ]]; then
      trace_id="$resolved_id"
    fi
    if [[ -n "$resolved_id" ]]; then
      trace_json="$(fetch_trace "$resolved_id" 2>/dev/null || true)"
    fi
  fi
fi

if [[ -z "$trace_json" ]]; then
  printf '%s[FAIL]%s no trace found in Tempo for request_id=%s trace_id=%s\n' \
    "$RED" "$NC" "$request_id" "${trace_id:-<n/a>}" >&2
  exit 6
fi

# ---------------------------------------------------------------------------
# Tempo /api/traces/{id} returns OTLP-style JSON. Sum spans across all
# batches and collect distinct resource.service.name values.
# ---------------------------------------------------------------------------
read -r span_count service_count <<<"$(printf '%s' "$trace_json" | jq -r '
  def extract_services:
    ([.batches[]?.resource?.attributes[]?
       | select(.key == "service.name")
       | .value.stringValue // .value.string_value // ""]
     | map(select(. != ""))
     | unique) ;
  def extract_spans:
    [.batches[]?.scopeSpans[]?.spans[]?] ;
  "\(extract_spans | length) \(extract_services | length)"
')"

services_list="$(printf '%s' "$trace_json" | jq -r '
  [.batches[]?.resource?.attributes[]?
    | select(.key == "service.name")
    | .value.stringValue // .value.string_value // ""]
  | map(select(. != ""))
  | unique
  | .[]
')"

printf '    span count:    %s\n' "$span_count"
printf '    service count: %s\n' "$service_count"
printf '    services:\n'
while IFS= read -r s; do
  [[ -n "$s" ]] && printf '      - %s\n' "$s"
done <<<"$services_list"

# Span summary (top 10)
printf '    top spans:\n'
printf '%s' "$trace_json" | jq -r '
  [ .batches[]? as $b
    | ($b.resource?.attributes[]? | select(.key=="service.name") | .value.stringValue // .value.string_value) as $svc
    | $b.scopeSpans[]?.spans[]?
    | "      - \($svc // "?"): \(.name // "?") (\(((.endTimeUnixNano|tonumber) - (.startTimeUnixNano|tonumber))/1000000 | floor)ms)"
  ] | .[0:10] | .[]' 2>/dev/null || true

if (( span_count < MIN_SPANS )); then
  printf '%s[FAIL]%s want ≥%s spans, got %s\n' "$RED" "$NC" "$MIN_SPANS" "$span_count" >&2
  exit 7
fi
if (( service_count < MIN_SERVICES )); then
  printf '%s[FAIL]%s want ≥%s services, got %s\n' "$RED" "$NC" "$MIN_SERVICES" "$service_count" >&2
  exit 8
fi

printf '%s[PASS]%s %s/%s — %s spans across %s services (request_id=%s)\n' \
  "$GREEN" "$NC" "$portal" "$cta_key" "$span_count" "$service_count" "$request_id"

#!/bin/bash
# NHS smoke test — runs after every deploy.
# Verifies core API + new features still work.
set -uo pipefail

BASE="${1:-https://nothumansearch.ai}"
EXPECTED_REVISION="${2:-${NHS_EXPECTED_RELEASE_REVISION:-}}"
EXPECTED_EXCHANGE_MODE="${3:-pilot}"
SYNTHETIC_HEADER="NHS-Synthetic-Test: deploy-smoke"
FAILED=0
TOTAL=0
CURL=(/usr/bin/curl --silent --show-error --proto '=https' --tlsv1.2 --connect-timeout 5 --max-time 15 -H "$SYNTHETIC_HEADER")

if ! [[ "$BASE" =~ ^https://[A-Za-z0-9.-]+(:[0-9]+)?$ ]]; then
    echo "base_url must be a bare HTTPS origin" >&2
    exit 2
fi
if ! [[ "$EXPECTED_REVISION" =~ ^[0-9a-f]{40}$ ]]; then
    echo "usage: $0 [base_url] <expected_40_character_release_revision> [pilot|disabled]" >&2
    exit 2
fi
if [ "$EXPECTED_EXCHANGE_MODE" != "pilot" ] && [ "$EXPECTED_EXCHANGE_MODE" != "disabled" ]; then
    echo "expected provider exchange mode must be pilot or disabled" >&2
    exit 2
fi

check() {
    local name="$1"
    local expected="$2"
    local url="$3"
    TOTAL=$((TOTAL + 1))
    local actual=$("${CURL[@]}" -o /dev/null -w '%{http_code}' "$url")
    if [ "$actual" = "$expected" ]; then
        printf "  \033[32m✓\033[0m %-45s %s\n" "$name" "$actual"
    else
        printf "  \033[31m✗\033[0m %-45s expected %s, got %s  %s\n" "$name" "$expected" "$actual" "$url"
        FAILED=$((FAILED + 1))
    fi
}

check_contains() {
    local name="$1"
    local needle="$2"
    local url="$3"
    TOTAL=$((TOTAL + 1))
    local body=$("${CURL[@]}" "$url")
    if printf '%s' "$body" | grep -q "$needle"; then
        printf "  \033[32m✓\033[0m %-45s contains %s\n" "$name" "'$needle'"
    else
        printf "  \033[31m✗\033[0m %-45s missing %s  %s\n" "$name" "'$needle'" "$url"
        FAILED=$((FAILED + 1))
    fi
}

check_search_selection() {
    TOTAL=$((TOTAL + 1))
    local payload search_id domain headers
    payload=$("${CURL[@]}" "$BASE/api/v1/search?q=payment&per_page=1")
    search_id=$(printf '%s' "$payload" | /usr/bin/jq -r '.search_id // empty')
    domain=$(printf '%s' "$payload" | /usr/bin/jq -r '.results[0].domain // empty')
    if [[ "$search_id" != nhs_sr_* ]] || [ -z "$domain" ]; then
        printf "  \033[31m✗\033[0m %-45s missing search receipt or result\n" "Search-to-detail receipt"
        FAILED=$((FAILED + 1))
        return
    fi
    headers=$("${CURL[@]}" -D - -o /dev/null "$BASE/api/v1/site/$domain?search_id=$search_id")
    if printf '%s' "$headers" | tr -d '\r' | grep -qi '^NHS-Selection-Recorded: true$'; then
        printf "  \033[32m✓\033[0m %-45s %s\n" "Search-to-detail receipt" "$domain"
    else
        printf "  \033[31m✗\033[0m %-45s selection header missing for %s\n" "Search-to-detail receipt" "$domain"
        FAILED=$((FAILED + 1))
    fi
}

check_synthetic_action_interest_rejected() {
    TOTAL=$((TOTAL + 1))
    local payload search_id domain code
    payload=$("${CURL[@]}" "$BASE/api/v1/search?q=payment&per_page=1")
    search_id=$(printf '%s' "$payload" | /usr/bin/jq -r '.search_id // empty')
    domain=$(printf '%s' "$payload" | /usr/bin/jq -r '.results[0].domain // empty')
    if [[ "$search_id" != nhs_sr_* ]] || [ -z "$domain" ]; then
        printf "  \033[31m✗\033[0m %-45s missing synthetic search source\n" "Synthetic action interest rejected"
        FAILED=$((FAILED + 1))
        return
    fi
    code=$("${CURL[@]}" -H "Content-Type: application/json" \
        -o /dev/null -w '%{http_code}' -X POST \
        --data "{\"search_id\":\"$search_id\",\"domain\":\"$domain\",\"action_type\":\"quote\",\"caller_attests_principal_interest\":true,\"confirmation_version\":\"nhs-action-interest-v1\"}" \
        "$BASE/api/v1/action-interests")
    if [ "$code" = "404" ]; then
        printf "  \033[32m✓\033[0m %-45s %s\n" "Synthetic action interest rejected" "$code"
    else
        printf "  \033[31m✗\033[0m %-45s expected 404, got %s\n" "Synthetic action interest rejected" "$code"
        FAILED=$((FAILED + 1))
    fi
}

check_release_revision() {
    TOTAL=$((TOTAL + 1))
    local health actual mode
    health=$("${CURL[@]}" "$BASE/health")
    actual=$(printf '%s' "$health" | /usr/bin/jq -r '.release_revision // empty')
    mode=$(printf '%s' "$health" | /usr/bin/jq -r '.provider_exchange // empty')
    if [ "$actual" = "$EXPECTED_REVISION" ] && [ "$mode" = "$EXPECTED_EXCHANGE_MODE" ]; then
        printf "  \033[32m✓\033[0m %-45s %s %s\n" "Exact release and exchange mode" "$actual" "$mode"
    else
        printf "  \033[31m✗\033[0m %-45s expected %s/%s, got %s/%s\n" "Exact release and exchange mode" "$EXPECTED_REVISION" "$EXPECTED_EXCHANGE_MODE" "${actual:-missing}" "${mode:-missing}"
        FAILED=$((FAILED + 1))
    fi
}

check_synthetic_paid_sidecar_empty() {
    TOTAL=$((TOTAL + 1))
    local payload
    payload=$("${CURL[@]}" "$BASE/api/v1/search?q=payment&per_page=1")
    if printf '%s' "$payload" | /usr/bin/jq -e '
        .paid_offers_available == false and
        (.paid_offers | type == "array" and length == 0) and
        .action_interest.available == false
    ' >/dev/null; then
        printf "  \033[32m✓\033[0m %-45s clean\n" "Synthetic paid sidecar"
    else
        printf "  \033[31m✗\033[0m %-45s paid or action sidecar present\n" "Synthetic paid sidecar"
        FAILED=$((FAILED + 1))
    fi
}

check_invalid_action_interest_receipt() {
    TOTAL=$((TOTAL + 1))
    local code
    code=$("${CURL[@]}" -H "Content-Type: application/json" \
        -o /dev/null -w '%{http_code}' -X POST \
        --data '{"search_id":"nhs_sr_AAAAAAAAAAAAAAAA","domain":"example.com","action_type":"quote","caller_attests_principal_interest":true,"confirmation_version":"nhs-action-interest-v1"}' \
        "$BASE/api/v1/action-interests")
    if [ "$code" = "404" ]; then
        printf "  \033[32m✓\033[0m %-45s %s\n" "Invalid search receipt rejected" "$code"
    else
        printf "  \033[31m✗\033[0m %-45s expected 404, got %s\n" "Invalid search receipt rejected" "$code"
        FAILED=$((FAILED + 1))
    fi
}

check_invalid_provider_action_route() {
    local name="$1"
    local path="$2"
    TOTAL=$((TOTAL + 1))
    local code
    code=$("${CURL[@]}" -H "Content-Type: application/json" \
        -o /dev/null -w '%{http_code}' -X POST --data '{}' "$BASE$path")
    local expected=400
    if [ "$EXPECTED_EXCHANGE_MODE" = "disabled" ]; then
        expected=503
    fi
    if [ "$code" = "$expected" ]; then
        printf "  \033[32m✓\033[0m %-45s %s\n" "$name" "$code"
    else
        printf "  \033[31m✗\033[0m %-45s expected %s, got %s\n" "$name" "$expected" "$code"
        FAILED=$((FAILED + 1))
    fi
}

check_mcp_initialize() {
    TOTAL=$((TOTAL + 1))
    local body
    body=$("${CURL[@]}" \
        -H "Content-Type: application/json" -X POST \
        --data '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"nhs-deploy-smoke","version":"1"}}}' \
        "$BASE/mcp")
    if printf '%s' "$body" | /usr/bin/jq -e '
        .jsonrpc == "2.0" and .id == 1 and
        .result.protocolVersion == "2025-06-18" and
        .result.serverInfo.name == "nothumansearch" and
        .result.serverInfo.version == "1.1.0" and
        (.result.instructions | contains("action_interest.call_contract")) and
        (.result.instructions | contains("never infer interest"))
    ' >/dev/null; then
        printf "  \033[32m✓\033[0m %-45s negotiated\n" "MCP initialize"
    else
        printf "  \033[31m✗\033[0m %-45s invalid JSON-RPC response\n" "MCP initialize"
        FAILED=$((FAILED + 1))
    fi
}

check_mcp_tools_list() {
    TOTAL=$((TOTAL + 1))
    local body
    body=$("${CURL[@]}" \
        -H "Content-Type: application/json" -X POST \
        --data '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
        "$BASE/mcp")
    if printf '%s' "$body" | /usr/bin/jq -e '
        .jsonrpc == "2.0" and .id == 2 and
        (.result.tools | type == "array" and length == 14) and
        ([.result.tools[].name] | unique | length == 14) and
        ([.result.tools[].name] | index("search_agents") != null) and
        ([.result.tools[].name] | index("record_action_interest") != null) and
        ([.result.tools[].name] | index("prepare_provider_action") != null) and
        ([.result.tools[].name] | index("handoff_provider_action") != null)
    ' >/dev/null; then
        printf "  \033[32m✓\033[0m %-45s 14 tools\n" "MCP tools/list"
    else
        printf "  \033[31m✗\033[0m %-45s runtime inventory mismatch\n" "MCP tools/list"
        FAILED=$((FAILED + 1))
    fi
}

check_mcp_free_synthetic_search() {
    TOTAL=$((TOTAL + 1))
    local body
    body=$("${CURL[@]}" \
        -H "Content-Type: application/json" -X POST \
        --data '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_agents","arguments":{"query":"payment","limit":1}}}' \
        "$BASE/mcp")
    if printf '%s' "$body" | /usr/bin/jq -e '
        . as $response |
        $response.result.structuredContent as $discovery |
        $response.jsonrpc == "2.0" and $response.id == 3 and
        $discovery.access == "free" and
        $discovery.receipt_recorded == true and
        ($discovery.search_id | startswith("nhs_sr_")) and
        $discovery.paid_offers_available == false and
        ($discovery.paid_offers | type == "array" and length == 0) and
        $discovery.action_interest.available == false and
        $discovery.action_interest.call_contract.available == false and
        $discovery.action_interest.call_contract.tool == "record_action_interest" and
        $discovery.action_interest.call_contract.fixed_arguments_if_invocation_condition_met == {} and
        $discovery.action_interest.call_contract.domain_must_be_one_of == [] and
        $discovery.action_interest.call_contract.action_type_must_be_one_of == [] and
        $discovery.action_interest.call_contract.executable_without_explicit_principal_intent == false and
        $discovery.action_interest.call_contract.query_prompt_contact_identity_fields_are_accepted == false and
        ($discovery.results | type == "array" and length > 0) and
        ($discovery.detail_actions | type == "array" and length == ($discovery.results | length)) and
        ($response.result.content[0].text | contains(
            "get_site_details {\"domain\":\"" +
            ($discovery.results[0].domain | ascii_downcase) +
            "\",\"search_id\":\"" + $discovery.search_id + "\"}"
        ))
    ' >/dev/null; then
		printf "  \033[32m✓\033[0m %-45s free receipt + exact text action\n" "MCP tools/call search_agents"
    else
        printf "  \033[31m✗\033[0m %-45s free/synthetic contract mismatch\n" "MCP tools/call search_agents"
        FAILED=$((FAILED + 1))
    fi
}

check_tampered_receipt_rejected() {
    TOTAL=$((TOTAL + 1))
    local response body code expected
    response=$("${CURL[@]}" -H "Content-Type: application/json" \
        -w $'\n%{http_code}' -X POST \
        --data '{"signed_receipt":"{}","signature":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}' \
        "$BASE/api/v1/action-receipts/verify")
    code="${response##*$'\n'}"
    body="${response%$'\n'*}"
    if [ "$EXPECTED_EXCHANGE_MODE" = "disabled" ]; then
        expected=503
        if [ "$code" = "$expected" ]; then
            printf "  \033[32m✓\033[0m %-45s disabled\n" "Tampered signed receipt rejected"
        else
            printf "  \033[31m✗\033[0m %-45s expected %s, got %s\n" "Tampered signed receipt rejected" "$expected" "$code"
            FAILED=$((FAILED + 1))
        fi
    elif [ "$code" = "200" ] && printf '%s' "$body" | /usr/bin/jq -e '
        .signature_valid == false and
        .within_validity_window == false and
        .current_state_available == false
    ' >/dev/null; then
        printf "  \033[32m✓\033[0m %-45s invalid\n" "Tampered signed receipt rejected"
    else
        printf "  \033[31m✗\033[0m %-45s invalid verification response\n" "Tampered signed receipt rejected"
        FAILED=$((FAILED + 1))
    fi
}

echo "NHS smoke test: $BASE"
echo ""
echo "Core API"
check "GET /api/v1/search is free" 200 "$BASE/api/v1/search?q=payment&per_page=1"
check_contains "Search free-access contract" '"access":"free"' "$BASE/api/v1/search?q=payment&per_page=1"
check_contains "Search receipt contract" '"search_id":"nhs_sr_' "$BASE/api/v1/search?q=payment&per_page=1"
check_synthetic_paid_sidecar_empty
check_search_selection
check_synthetic_action_interest_rejected
check_invalid_action_interest_receipt
check "GET /api/v1/site/{domain} is free" 200 "$BASE/api/v1/site/openai.com"
check "GET /api/v1/sites/{domain} free alias" 200 "$BASE/api/v1/sites/openai.com"
check "GET /api/v1/stats" 200 "$BASE/api/v1/stats"
check "GET /api/v1/categories" 200 "$BASE/api/v1/categories"
check "GET /api/v1/top" 200 "$BASE/api/v1/top?limit=1"
check "GET /api/v1/verify-mcp is free" 400 "$BASE/api/v1/verify-mcp"

echo ""
echo "MCP"
check "GET /.well-known/mcp.json" 200 "$BASE/.well-known/mcp.json"
check_contains "MCP manifest name" "nothumansearch" "$BASE/.well-known/mcp.json"
check_contains "MCP action-interest tool" "record_action_interest" "$BASE/.well-known/mcp.json"
check_contains "MCP ticket preparation tool" "prepare_provider_action" "$BASE/.well-known/mcp.json"
check_contains "MCP observed handoff tool" "handoff_provider_action" "$BASE/.well-known/mcp.json"
check_mcp_initialize
check_mcp_tools_list
check_mcp_free_synthetic_search

echo ""
echo "Provider-funded exchange"
check "GET /providers" 200 "$BASE/providers"
check "GET /privacy" 200 "$BASE/privacy"
check_invalid_provider_action_route "Invalid ticket preparation rejected" "/api/v1/action-tickets"
check_invalid_provider_action_route "Invalid observed handoff rejected" "/api/v1/action-tickets/handoff"
check_tampered_receipt_rejected

echo ""
echo "Landing pages"
check "GET / (home)" 200 "$BASE/"
check "GET /mcp-servers" 200 "$BASE/mcp-servers"
check "GET /ai-tools" 200 "$BASE/ai-tools"
check "GET /developer-apis" 200 "$BASE/developer-apis"
check "GET /openapi-apis" 200 "$BASE/openapi-apis"
check "GET /llms-txt-sites" 200 "$BASE/llms-txt-sites"
check "GET /top" 200 "$BASE/top"
check "GET /newest" 200 "$BASE/newest"
check "GET /leaderboard (alias)" 200 "$BASE/leaderboard"

echo ""
echo "Site + tag pages"
check "GET /site/openai.com" 200 "$BASE/site/openai.com"
check "GET /tag/mcp" 200 "$BASE/tag/mcp"
check "GET /tag/llms-txt" 200 "$BASE/tag/llms-txt"

echo ""
echo "Score + guide"
check "GET /score" 200 "$BASE/score"
check "GET /guide" 200 "$BASE/guide"
check "GET /about" 200 "$BASE/about"
check "GET /monitor" 200 "$BASE/monitor"

echo ""
echo "Embeddable"
check "GET /badge/openai.com.svg" 200 "$BASE/badge/openai.com.svg"
check_contains "Badge is SVG" "<svg" "$BASE/badge/openai.com.svg"

echo ""
echo "SEO / discovery"
check "GET /robots.txt" 200 "$BASE/robots.txt"
check "GET /sitemap.xml" 200 "$BASE/sitemap.xml"
check "GET /llms.txt" 200 "$BASE/llms.txt"
check "GET /llms-full.txt" 200 "$BASE/llms-full.txt"
check "GET /openapi.yaml" 200 "$BASE/openapi.yaml"
check "GET /feed.xml" 200 "$BASE/feed.xml"
check "GET /.well-known/ai-plugin.json" 200 "$BASE/.well-known/ai-plugin.json"

echo ""
echo "Health"
check "GET /health" 200 "$BASE/health"
check "GET /status" 200 "$BASE/status"
check_release_revision

echo ""
if [ $FAILED -eq 0 ]; then
    echo "PASS: $TOTAL/$TOTAL checks"
    exit 0
else
    echo "FAIL: $FAILED/$TOTAL checks"
    exit 1
fi

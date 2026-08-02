#!/bin/bash
# NHS smoke test — runs after every deploy.
# Verifies core API + new features still work.
set -uo pipefail

BASE="${1:-https://nothumansearch.ai}"
SYNTHETIC_HEADER="NHS-Synthetic-Test: deploy-smoke"
FAILED=0
TOTAL=0

check() {
    local name="$1"
    local expected="$2"
    local url="$3"
    TOTAL=$((TOTAL + 1))
    local actual=$(/usr/bin/curl -s -H "$SYNTHETIC_HEADER" -o /dev/null -w '%{http_code}' "$url")
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
    local body=$(/usr/bin/curl -s -H "$SYNTHETIC_HEADER" "$url")
    if echo "$body" | grep -q "$needle"; then
        printf "  \033[32m✓\033[0m %-45s contains %s\n" "$name" "'$needle'"
    else
        printf "  \033[31m✗\033[0m %-45s missing %s  %s\n" "$name" "'$needle'" "$url"
        FAILED=$((FAILED + 1))
    fi
}

check_search_selection() {
    TOTAL=$((TOTAL + 1))
    local payload search_id domain headers
    payload=$(/usr/bin/curl -s -H "$SYNTHETIC_HEADER" "$BASE/api/v1/search?q=payment&per_page=1")
    search_id=$(printf '%s' "$payload" | /usr/bin/jq -r '.search_id // empty')
    domain=$(printf '%s' "$payload" | /usr/bin/jq -r '.results[0].domain // empty')
    if [[ "$search_id" != nhs_sr_* ]] || [ -z "$domain" ]; then
        printf "  \033[31m✗\033[0m %-45s missing search receipt or result\n" "Search-to-detail receipt"
        FAILED=$((FAILED + 1))
        return
    fi
    headers=$(/usr/bin/curl -s -H "$SYNTHETIC_HEADER" -D - -o /dev/null "$BASE/api/v1/site/$domain?search_id=$search_id")
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
    payload=$(/usr/bin/curl -s -H "$SYNTHETIC_HEADER" "$BASE/api/v1/search?q=payment&per_page=1")
    search_id=$(printf '%s' "$payload" | /usr/bin/jq -r '.search_id // empty')
    domain=$(printf '%s' "$payload" | /usr/bin/jq -r '.results[0].domain // empty')
    if [[ "$search_id" != nhs_sr_* ]] || [ -z "$domain" ]; then
        printf "  \033[31m✗\033[0m %-45s missing synthetic search source\n" "Synthetic action interest rejected"
        FAILED=$((FAILED + 1))
        return
    fi
    code=$(/usr/bin/curl -s -H "$SYNTHETIC_HEADER" -H "Content-Type: application/json" \
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

echo "NHS smoke test: $BASE"
echo ""
echo "Core API"
check "GET /api/v1/search is free" 200 "$BASE/api/v1/search?q=payment&per_page=1"
check_contains "Search free-access contract" '"access":"free"' "$BASE/api/v1/search?q=payment&per_page=1"
check_contains "Search receipt contract" '"search_id":"nhs_sr_' "$BASE/api/v1/search?q=payment&per_page=1"
check_search_selection
check_synthetic_action_interest_rejected
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

echo ""
if [ $FAILED -eq 0 ]; then
    echo "PASS: $TOTAL/$TOTAL checks"
    exit 0
else
    echo "FAIL: $FAILED/$TOTAL checks"
    exit 1
fi

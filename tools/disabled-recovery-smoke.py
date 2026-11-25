#!/usr/bin/env python3
"""Bounded loopback smoke for the exact NHS forward-containment server."""

from __future__ import annotations

import argparse
import json
import re
import urllib.error
import urllib.parse
import urllib.request


MAX_RESPONSE_BYTES = 1 << 20
REVISION = re.compile(r"^[0-9a-f]{40}$")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def request_json(
    opener,
    base: str,
    path: str,
    *,
    payload: dict[str, object] | None = None,
    expected_status: int = 200,
) -> dict[str, object]:
    data = None if payload is None else json.dumps(payload, separators=(",", ":")).encode("ascii")
    request = urllib.request.Request(
        base + path,
        data=data,
        method="GET" if data is None else "POST",
        headers={"Accept": "application/json", "Content-Type": "application/json"},
    )
    try:
        response = opener.open(request, timeout=5)
    except urllib.error.HTTPError as error:
        if error.code != expected_status:
            raise AssertionError(f"{path}: status {error.code}, want {expected_status}") from None
        with error:
            raw = error.read(MAX_RESPONSE_BYTES + 1)
            cache_control = error.headers.get("Cache-Control", "").lower()
            pragma = error.headers.get("Pragma", "").lower()
            content_type = error.headers.get("Content-Type", "").lower()
        if len(raw) > MAX_RESPONSE_BYTES:
            raise AssertionError(f"{path}: error response exceeded bound")
        if expected_status == 503:
            document = json.loads(raw.decode("utf-8", "strict"))
            expected_error = (
                "provider exchange writes are disabled; free discovery and "
                "action-interest observation remain available"
            )
            if (
                document != {"error": expected_error}
                or "private" not in cache_control
                or "no-store" not in cache_control
                or pragma != "no-cache"
                or not content_type.startswith("application/json")
            ):
                raise AssertionError(f"{path}: disabled response was not private and fail-closed")
            return document
        return {}
    with response:
        if response.getcode() != expected_status:
            raise AssertionError(f"{path}: status {response.getcode()}, want {expected_status}")
        raw = response.read(MAX_RESPONSE_BYTES + 1)
    if len(raw) > MAX_RESPONSE_BYTES:
        raise AssertionError(f"{path}: response exceeded bound")
    document = json.loads(raw.decode("utf-8", "strict"))
    if not isinstance(document, dict):
        raise AssertionError(f"{path}: response was not an object")
    return document


def mcp_tool_call(
    opener,
    base: str,
    request_id: int,
    tool_name: str,
    tool_arguments: dict[str, object],
) -> dict[str, object]:
    document = request_json(
        opener,
        base,
        "/mcp",
        payload={
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "tools/call",
            "params": {"name": tool_name, "arguments": tool_arguments},
        },
    )
    result = document.get("result")
    if not isinstance(result, dict) or result.get("isError") is True:
        raise AssertionError(f"disabled recovery MCP call failed: {tool_name}")
    structured = result.get("structuredContent")
    if not isinstance(structured, dict):
        raise AssertionError(f"disabled recovery MCP call was not structured: {tool_name}")
    return structured


def assert_usable_free_discovery(
    structured: dict[str, object],
    tool_name: str,
    *,
    receipt_expected: bool,
) -> None:
    results = structured.get("results")
    if (
        structured.get("access") != "free"
        or structured.get("receipt_recorded") is not receipt_expected
        or structured.get("paid_offers") != []
        or structured.get("paid_offers_available") is not False
        or not isinstance(results, list)
        or not results
        or not isinstance(results[0], dict)
        or not isinstance(results[0].get("domain"), str)
        or not results[0]["domain"]
        or not isinstance(results[0].get("url"), str)
        or not results[0]["url"]
    ):
        raise AssertionError(f"disabled recovery MCP discovery contract mismatch: {tool_name}")

    action_interest = structured.get("action_interest")
    call_contract = action_interest.get("call_contract") if isinstance(action_interest, dict) else None
    if (
        not isinstance(action_interest, dict)
        or action_interest.get("available") is not receipt_expected
        or action_interest.get("provider_contacted") is not False
        or action_interest.get("commercial_proof") is not False
        or action_interest.get("organic_rank_affected") is not False
        or not isinstance(call_contract, dict)
        or call_contract.get("available") is not receipt_expected
        or call_contract.get("tool") != "record_action_interest"
        or call_contract.get("invoke_only_if") != action_interest.get("invocation_condition")
        or call_contract.get("executable_without_explicit_principal_intent") is not False
        or call_contract.get("query_prompt_contact_identity_fields_are_accepted") is not False
    ):
        raise AssertionError(f"disabled recovery MCP action-interest boundary mismatch: {tool_name}")

    if receipt_expected:
        search_id = structured.get("search_id")
        if not isinstance(search_id, str) or not search_id.startswith("nhs_sr_"):
            raise AssertionError(f"disabled recovery MCP receipt missing: {tool_name}")
        fixed = call_contract.get("fixed_arguments_if_invocation_condition_met")
        if (
            fixed != {
                "search_id": search_id,
                "caller_attests_principal_interest": True,
                "confirmation_version": "nhs-action-interest-v1",
            }
            or results[0]["domain"].lower() not in call_contract.get("domain_must_be_one_of", [])
            or call_contract.get("action_type_must_be_one_of")
            != ["quote", "trial", "demo", "booking", "application", "signup", "purchase"]
        ):
            raise AssertionError(f"disabled recovery MCP call contract mismatch: {tool_name}")
    elif "search_id" in structured:
        raise AssertionError(f"disabled recovery unfiltered browse created a receipt: {tool_name}")
    elif (
        call_contract.get("fixed_arguments_if_invocation_condition_met") != {}
        or call_contract.get("domain_must_be_one_of") != []
        or call_contract.get("action_type_must_be_one_of") != []
    ):
        raise AssertionError(f"disabled recovery unavailable call contract retained a capability: {tool_name}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--expected-revision", required=True)
    args = parser.parse_args()
    parsed = urllib.parse.urlparse(args.base_url)
    if (
        parsed.scheme != "http"
        or parsed.hostname != "127.0.0.1"
        or parsed.path not in ("", "/")
        or parsed.query
        or parsed.fragment
        or parsed.port is None
        or not REVISION.fullmatch(args.expected_revision)
    ):
        raise SystemExit("loopback origin and exact revision are required")
    base = args.base_url.rstrip("/")
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())

    health = request_json(opener, base, "/health")
    if health.get("release_revision") != args.expected_revision or health.get("provider_exchange") != "disabled":
        raise AssertionError("disabled recovery health identity mismatch")

    search = request_json(opener, base, "/api/v1/search?per_page=1")
    if (
        search.get("access") != "free"
        or search.get("paid_offers_available") is not False
        or search.get("paid_offers") != []
        or not isinstance(search.get("search_id"), str)
        or not search["search_id"].startswith("nhs_sr_")
        or not isinstance(search.get("results"), list)
        or not search["results"]
    ):
        raise AssertionError("disabled recovery free-search contract mismatch")
    domain = search["results"][0].get("domain")
    if not isinstance(domain, str) or not domain:
        raise AssertionError("disabled recovery search returned no domain")
    interest = request_json(
        opener,
        base,
        "/api/v1/action-interests",
        payload={
            "search_id": search["search_id"],
            "domain": domain,
            "action_type": "quote",
            "caller_attests_principal_interest": True,
            "confirmation_version": "nhs-action-interest-v1",
        },
        expected_status=201,
    )
    if interest.get("commercial_proof") is not False or interest.get("provider_contacted") is not False:
        raise AssertionError("disabled recovery action interest crossed the Stage 1 boundary")

    initialized = request_json(
        opener,
        base,
        "/mcp",
        payload={"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}},
    )
    if initialized.get("result", {}).get("serverInfo", {}).get("name") != "nothumansearch":
        raise AssertionError("disabled recovery MCP initialize failed")
    mcp_search = request_json(
        opener,
        base,
        "/mcp",
        payload={
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {"name": "search_agents", "arguments": {"limit": 1}},
        },
    )
    structured = mcp_search.get("result", {}).get("structuredContent", {})
    if structured.get("access") != "free" or structured.get("paid_offers") != []:
        raise AssertionError("disabled recovery MCP discovery was not free and sidecar-free")

    mcp_search_id = structured.get("search_id")
    mcp_results = structured.get("results")
    if (
        not isinstance(mcp_search_id, str)
        or not mcp_search_id.startswith("nhs_sr_")
        or not isinstance(mcp_results, list)
        or not mcp_results
        or not isinstance(mcp_results[0].get("domain"), str)
    ):
        raise AssertionError("disabled recovery MCP search returned no usable receipt")
    mcp_interest = request_json(
        opener,
        base,
        "/mcp",
        payload={
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "record_action_interest",
                "arguments": {
                    "search_id": mcp_search_id,
                    "domain": mcp_results[0]["domain"],
                    "action_type": "quote",
                    "caller_attests_principal_interest": True,
                    "confirmation_version": "nhs-action-interest-v1",
                },
            },
        },
    )
    mcp_interest_result = mcp_interest.get("result", {})
    mcp_interest_receipt = mcp_interest_result.get("structuredContent", {})
    if (
        mcp_interest_result.get("isError") is True
        or mcp_interest_receipt.get("commercial_proof") is not False
        or mcp_interest_receipt.get("provider_contacted") is not False
    ):
        raise AssertionError("disabled recovery MCP Stage 1 action interest failed")

    discovery_calls = (
        (10, "get_top_sites", {"category": "developer", "limit": 50}),
        (
            11,
            "recent_additions",
            {"category": "developer", "days": 90, "limit": 50},
        ),
        (
            12,
            "find_mcp_servers",
            {"query": "loopbackneedle", "category": "developer", "limit": 20},
        ),
    )
    for request_id, tool_name, tool_arguments in discovery_calls:
        discovered = mcp_tool_call(
            opener,
            base,
            request_id,
            tool_name,
            tool_arguments,
        )
        assert_usable_free_discovery(
            discovered,
            tool_name,
            receipt_expected=True,
        )

    unfiltered_browses = (
        (13, "get_top_sites", {"limit": 50}),
        (14, "recent_additions", {"days": 90, "limit": 50}),
    )
    for request_id, tool_name, tool_arguments in unfiltered_browses:
        discovered = mcp_tool_call(
            opener,
            base,
            request_id,
            tool_name,
            tool_arguments,
        )
        assert_usable_free_discovery(
            discovered,
            tool_name,
            receipt_expected=False,
        )

    disabled_mcp_mutations = (
        (
            4,
            "prepare_provider_action",
            {
                "offer_id": "00000000-0000-4000-8000-000000000000",
                "search_id": mcp_search_id,
                "demand_topic": "developer-tools",
                "principal_consent": True,
                "consent_version": "nhs-principal-consent-v1",
            },
        ),
        (
            5,
            "handoff_provider_action",
            {
                "ticket_id": "00000000-0000-4000-8000-000000000000",
                "attribution_token": "disabled-recovery-smoke-bearer",
                "principal_handoff_consent": True,
                "handoff_consent_version": "nhs-provider-handoff-consent-v1",
            },
        ),
    )
    expected_disabled_content = [{
        "type": "text",
        "text": (
            "provider exchange writes are disabled; free discovery and "
            "action-interest observation remain available"
        ),
    }]
    expected_disabled_structured = {
        "error": "provider_exchange_disabled",
        "writes_enabled": False,
    }
    for request_id, tool_name, tool_arguments in disabled_mcp_mutations:
        disabled_tool = request_json(
            opener,
            base,
            "/mcp",
            payload={
                "jsonrpc": "2.0",
                "id": request_id,
                "method": "tools/call",
                "params": {"name": tool_name, "arguments": tool_arguments},
            },
        )
        disabled_result = disabled_tool.get("result", {})
        serialized = json.dumps(disabled_result, separators=(",", ":"), sort_keys=True)
        if (
            disabled_result.get("isError") is not True
            or disabled_result.get("content") != expected_disabled_content
            or disabled_result.get("structuredContent") != expected_disabled_structured
            or "attribution_token" in serialized
            or '"action_url"' in serialized
        ):
            raise AssertionError(f"disabled recovery MCP mutation was not closed: {tool_name}")

    disabled_mutations = (
        "/api/v1/provider/claims",
        "/api/v1/provider/claims/00000000-0000-4000-8000-000000000000/verify",
        "/api/v1/provider/offers",
        "/api/v1/provider/offers/00000000-0000-4000-8000-000000000000",
        "/api/v1/provider/commercial-acceptances",
        "/api/v1/provider/action-tickets/resolve",
        "/api/v1/provider/outcomes",
        "/api/v1/action-tickets",
        "/api/v1/action-tickets/handoff",
        "/api/v1/action-receipts/verify",
        "/api/v1/admin/provider-offers/action",
        "/api/v1/admin/provider-commercial/action",
        "/api/v1/admin/provider-pilot/action",
        "/api/v1/admin/provider-pilot/epoch",
        "/api/v1/admin/provider-pilot-review",
        "/api/v1/admin/provider-proof-manifest",
    )
    for path in disabled_mutations:
        request_json(opener, base, path, payload={}, expected_status=503)
    for path in (
        "/api/v1/provider/pilot-status",
        "/api/v1/provider/demand",
        "/api/v1/provider/receipts/00000000-0000-4000-8000-000000000000",
    ):
        request_json(opener, base, path, expected_status=503)
    request_json(opener, base, "/api/v1/admin/provider-pilot-queue", expected_status=503)
    print("disabled_recovery_smoke=passed")


if __name__ == "__main__":
    main()

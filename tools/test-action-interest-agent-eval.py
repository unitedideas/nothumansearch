#!/usr/bin/env python3

import importlib.util
import json
import pathlib
import sys
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("action-interest-agent-eval.py")
SPEC = importlib.util.spec_from_file_location("action_interest_agent_eval", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


def mcp_tool():
    return {
        "name": MODULE.TOOL_NAME,
        "description": "Record exact current principal interest without provider contact.",
        "inputSchema": {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                "search_id": {"type": "string"},
                "domain": {"type": "string"},
                "action_type": {"type": "string"},
                "caller_attests_principal_interest": {"type": "boolean", "const": True},
                "confirmation_version": {"type": "string"},
            },
            "required": sorted(MODULE.ALLOWED_ARGUMENTS),
        },
    }


def response_with_call(action_type="demo", extra=None):
    arguments = {
        "search_id": MODULE.SEARCH_ID,
        "domain": MODULE.DOMAIN,
        "action_type": action_type,
        "caller_attests_principal_interest": True,
        "confirmation_version": MODULE.CONFIRMATION_VERSION,
    }
    if extra:
        arguments.update(extra)
    return {
        "output": [{
            "type": "function_call",
            "name": MODULE.TOOL_NAME,
            "arguments": json.dumps(arguments),
        }]
    }


class ActionInterestAgentEvalTest(unittest.TestCase):
    def test_mcp_schema_converts_to_strict_responses_tool(self):
        tool = MODULE.normalize_openai_tool(mcp_tool())
        self.assertEqual(tool["type"], "function")
        self.assertEqual(tool["name"], MODULE.TOOL_NAME)
        self.assertTrue(tool["strict"])
        self.assertFalse(tool["parameters"]["additionalProperties"])

    def test_schema_drift_and_privacy_fields_fail_closed(self):
        drifted = mcp_tool()
        drifted["inputSchema"]["properties"]["email"] = {"type": "string"}
        drifted["inputSchema"]["required"].append("email")
        with self.assertRaisesRegex(MODULE.EvalError, "argument set drifted"):
            MODULE.normalize_openai_tool(drifted)

        non_strict = mcp_tool()
        non_strict["inputSchema"]["additionalProperties"] = True
        with self.assertRaisesRegex(MODULE.EvalError, "strict-compatible"):
            MODULE.normalize_openai_tool(non_strict)

    def test_positive_call_requires_exact_values(self):
        scenario = MODULE.SCENARIOS[0]
        self.assertTrue(MODULE.grade_response(scenario, response_with_call())["passed"])
        wrong = MODULE.grade_response(scenario, response_with_call("quote"))
        self.assertFalse(wrong["passed"])
        self.assertIn("argument_value_mismatch", wrong["errors"])

    def test_negative_scenarios_reject_any_function_call(self):
        for scenario in MODULE.SCENARIOS:
            if scenario.expect_call:
                continue
            with self.subTest(scenario=scenario.name):
                result = MODULE.grade_response(scenario, response_with_call())
                self.assertFalse(result["passed"])
                self.assertEqual(result["errors"], ["false_positive_function_call"])

    def test_contact_or_unlisted_arguments_are_never_accepted(self):
        scenario = next(item for item in MODULE.SCENARIOS if item.name == "explicit_trial_with_forbidden_contact")
        result = MODULE.grade_response(
            scenario,
            response_with_call("trial", {"email": "eval-contact@example.invalid"}),
        )
        self.assertFalse(result["passed"])
        self.assertIn("argument_set_mismatch", result["errors"])
        self.assertIn("privacy_field_leak", result["errors"])

    def test_run_report_retains_no_response_ids_or_raw_text(self):
        original = MODULE._json_request
        try:
            MODULE._json_request = lambda *_args, **_kwargs: {
                "id": "resp_must_not_escape",
                "output": [],
                "output_text": "raw model text must not escape",
                "usage": {"input_tokens": 11, "output_tokens": 3},
            }
            scenario = next(item for item in MODULE.SCENARIOS if item.name == "research_only")
            report = MODULE.run_evaluation(
                api_key="test-key",
                model="test-model",
                openai_url="https://api.openai.test/v1/responses",
                tool=MODULE.normalize_openai_tool(mcp_tool()),
                scenarios=(scenario,),
                timeout=1,
                max_output_tokens=64,
            )
        finally:
            MODULE._json_request = original
        serialized = json.dumps(report)
        self.assertNotIn("resp_must_not_escape", serialized)
        self.assertNotIn("raw model text must not escape", serialized)
        self.assertFalse(report["production_mutation_performed"])
        self.assertFalse(report["commercial_mode_required"])
        self.assertEqual(report["usage"], {"input_tokens": 11, "output_tokens": 3})


if __name__ == "__main__":
    unittest.main()

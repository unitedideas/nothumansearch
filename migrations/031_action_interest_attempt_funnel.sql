-- Privacy-safe operational funnel for caller-attested action interest.
--
-- Successful action-interest receipts deliberately bypass ordinary MCP
-- request telemetry because that telemetry retains pseudonymous client
-- coordinates. That also made a zero-receipt result impossible to diagnose:
-- callers may never have tried, or every attempt may have failed validation.
-- This table retains only one UTC-day aggregate per surface and stable outcome.
-- It stores no search ID, domain, action type, query, prompt, contact data,
-- network data, user agent, agent/principal identity, or provider coordinate.
-- These operational counters are diagnostic only and never commercial proof.

CREATE TABLE IF NOT EXISTS action_interest_attempt_daily (
    attempt_day       DATE NOT NULL,
    surface           TEXT NOT NULL CHECK (surface IN ('rest','mcp')),
    outcome           TEXT NOT NULL CHECK (outcome IN (
        'created','replayed','invalid_request','unavailable','conflict',
        'rate_limited','cross_origin','store_unavailable','internal_error'
    )),
    attempt_count     BIGINT NOT NULL CHECK (attempt_count > 0),
    first_observed_at TIMESTAMPTZ NOT NULL,
    last_observed_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (attempt_day, surface, outcome),
    CHECK (first_observed_at <= last_observed_at)
);

CREATE INDEX IF NOT EXISTS idx_action_interest_attempt_daily_recent
    ON action_interest_attempt_daily(attempt_day DESC, surface, outcome);

CREATE OR REPLACE FUNCTION own_action_interest_attempt_aggregate()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    observed_at TIMESTAMPTZ := clock_timestamp();
    observed_day DATE := (observed_at AT TIME ZONE 'UTC')::date;
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW.attempt_day := observed_day;
        NEW.attempt_count := 1;
        NEW.first_observed_at := observed_at;
        NEW.last_observed_at := observed_at;
        RETURN NEW;
    END IF;

    IF NEW.attempt_day IS DISTINCT FROM OLD.attempt_day OR
       NEW.surface IS DISTINCT FROM OLD.surface OR
       NEW.outcome IS DISTINCT FROM OLD.outcome THEN
        RAISE EXCEPTION 'action-interest aggregate dimensions are immutable'
            USING ERRCODE='23514', CONSTRAINT='action_interest_attempt_dimensions_immutable';
    END IF;
    NEW.attempt_count := OLD.attempt_count + 1;
    NEW.first_observed_at := OLD.first_observed_at;
    NEW.last_observed_at := observed_at;
    RETURN NEW;
END;
$$;

CREATE TRIGGER action_interest_attempt_aggregate_owned
BEFORE INSERT OR UPDATE ON action_interest_attempt_daily
FOR EACH ROW EXECUTE FUNCTION own_action_interest_attempt_aggregate();

COMMENT ON TABLE action_interest_attempt_daily IS
    'Owner-only daily surface/outcome counters with no entity or request coordinates; diagnostic only, never demand or commercial proof.';

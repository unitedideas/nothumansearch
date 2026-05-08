-- 012_geo_fix_jobs.sql
-- Paid GEO-uplift intake queue. Each row = one "Fix my score for $199" order.
-- Created at /fix/{host} intake, paid_at set by Stripe webhook,
-- completed_at set manually (or by a future fulfillment worker).

CREATE TABLE IF NOT EXISTS geo_fix_jobs (
    id                 BIGSERIAL PRIMARY KEY,
    host               TEXT NOT NULL,
    repo_url           TEXT,
    email              TEXT NOT NULL,
    notes              TEXT,
    stripe_session_id  TEXT,
    price_cents        INTEGER NOT NULL DEFAULT 19900,
    currency           TEXT NOT NULL DEFAULT 'usd',
    status             TEXT NOT NULL DEFAULT 'pending',
    paid_at            TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS host TEXT;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS repo_url TEXT;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS stripe_session_id TEXT;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS price_cents INTEGER DEFAULT 19900;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS currency TEXT DEFAULT 'usd';
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'pending';
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE geo_fix_jobs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

CREATE UNIQUE INDEX IF NOT EXISTS idx_geo_fix_jobs_stripe_session
    ON geo_fix_jobs (stripe_session_id) WHERE stripe_session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_geo_fix_jobs_status_created
    ON geo_fix_jobs (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_geo_fix_jobs_host
    ON geo_fix_jobs (host);

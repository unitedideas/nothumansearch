ALTER TABLE monitors ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS quarantine_reason TEXT;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS quarantined_at TIMESTAMPTZ;

UPDATE monitors
SET status = 'quarantined',
    quarantine_reason = COALESCE(quarantine_reason, 'shared-host apex domain requires admin review'),
    quarantined_at = COALESCE(quarantined_at, NOW())
WHERE status = 'active'
  AND domain IN (
    'carrd.co',
    'github.io',
    'gitlab.io',
    'glitch.me',
    'herokuapp.com',
    'netlify.app',
    'pages.dev',
    'repl.co',
    'vercel.app',
    'webflow.io',
    'wixsite.com'
  );

CREATE INDEX IF NOT EXISTS monitors_status_due_idx ON monitors (status, last_checked_at NULLS FIRST);

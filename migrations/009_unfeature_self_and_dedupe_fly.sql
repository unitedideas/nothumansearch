-- Remove NHS self-promotion. We should not rank our own site at the top.
-- Also drop the stale nothumansearch.fly.dev duplicate left over from early
-- deploys before the canonical domain moved to nothumansearch.ai.

DELETE FROM sites WHERE domain = 'nothumansearch.fly.dev';

UPDATE sites SET is_featured = false WHERE domain = 'nothumansearch.ai';

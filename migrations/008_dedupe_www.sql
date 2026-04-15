-- Collapse www.X and X duplicate rows into a single canonical record.
-- After this migration the crawler writes normalized domain (www-stripped,
-- lowercased) so future UPSERTs stay unique automatically.

-- Delete the www. duplicate when a non-www counterpart exists.
-- Keep the better of the two by preferring the higher agentic_score; if
-- scores tie, keep the non-www row (the canonical form).
DELETE FROM sites w
USING sites c
WHERE w.domain LIKE 'www.%'
  AND c.domain = substring(w.domain FROM 5)
  AND (c.agentic_score >= w.agentic_score);

-- For any remaining www. rows with no non-www counterpart, rewrite them
-- in place to the canonical form.
UPDATE sites
SET domain = substring(domain FROM 5),
    url = replace(url, '://www.', '://')
WHERE domain LIKE 'www.%'
  AND NOT EXISTS (
      SELECT 1 FROM sites s2 WHERE s2.domain = substring(sites.domain FROM 5)
  );

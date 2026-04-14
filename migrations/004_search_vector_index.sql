CREATE INDEX IF NOT EXISTS idx_sites_search_vector ON sites USING GIN(search_vector);

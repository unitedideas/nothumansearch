-- Track whether a site has a real favicon we can serve, and optionally where it lives.
-- Empty favicon_url + has_favicon=false means the frontend should render a letter avatar
-- instead of a blurry placeholder globe.
ALTER TABLE sites ADD COLUMN IF NOT EXISTS has_favicon BOOLEAN DEFAULT FALSE;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS favicon_url TEXT DEFAULT '';

-- Per-group upstream prompt compatibility rewrite switch.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS prompt_rewrite BOOLEAN NOT NULL DEFAULT FALSE;

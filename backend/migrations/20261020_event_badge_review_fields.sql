BEGIN;

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS soc_gpa_badge_status text NOT NULL DEFAULT 'not_requested',
    ADD COLUMN IF NOT EXISTS soc_gpa_badge_reviewed_by integer REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS soc_gpa_badge_reviewed_at timestamptz,
    ADD COLUMN IF NOT EXISTS iros_badge_status text NOT NULL DEFAULT 'not_requested',
    ADD COLUMN IF NOT EXISTS iros_badge_reviewed_by integer REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS iros_badge_reviewed_at timestamptz;

COMMIT;

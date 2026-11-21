BEGIN;

ALTER TABLE events
    DROP COLUMN IF EXISTS iros_badge_reviewed_at,
    DROP COLUMN IF EXISTS iros_badge_reviewed_by,
    DROP COLUMN IF EXISTS iros_badge_status,
    DROP COLUMN IF EXISTS soc_gpa_badge_reviewed_at,
    DROP COLUMN IF EXISTS soc_gpa_badge_reviewed_by,
    DROP COLUMN IF EXISTS soc_gpa_badge_status;

COMMIT;

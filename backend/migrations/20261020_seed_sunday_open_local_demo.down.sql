BEGIN;

DELETE FROM events e
USING event_slugs es
WHERE es.event_id = e.id
  AND es.slug = 'sunday-open-local-demo-aitu-agrotech-lab';

DELETE FROM users
WHERE username IN ('demo_account_01', 'demo_account_02');

DELETE FROM organizations
WHERE title = 'Sunday Open — local demo';

DELETE FROM organization_types
WHERE title = 'Local Sunday Open Demo';

DELETE FROM categories
WHERE name = 'Science — local demo';

COMMIT;

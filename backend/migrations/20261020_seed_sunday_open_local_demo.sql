BEGIN;

DO $$
DECLARE
    v_type_id integer;
    v_organization_id integer;
    v_category_id integer;
    v_event_id integer;
    v_account_user_one_id integer;
    v_account_user_two_id integer;
BEGIN
    SELECT id INTO v_type_id
    FROM organization_types
    WHERE title = 'Local Sunday Open Demo';

    IF v_type_id IS NULL THEN
        INSERT INTO organization_types (title)
        VALUES ('Local Sunday Open Demo')
        RETURNING id INTO v_type_id;
    END IF;

    SELECT o.id INTO v_organization_id
    FROM organizations o
    JOIN organization_slugs os ON os.organization_id = o.id
    WHERE os.slug = 'sunday-open';

    IF v_organization_id IS NULL THEN
        SELECT id INTO v_organization_id
        FROM organizations
        WHERE title = 'Sunday Open — local demo';

        IF v_organization_id IS NULL THEN
            INSERT INTO organizations (title, organization_type_id, status)
            VALUES ('Sunday Open — local demo', v_type_id, 'active')
            RETURNING id INTO v_organization_id;
        END IF;

        INSERT INTO organization_slugs (organization_id, slug, is_current)
        VALUES (v_organization_id, 'sunday-open', true)
        ON CONFLICT (slug) DO NOTHING;
    END IF;

    INSERT INTO categories (name, color)
    VALUES ('Science — local demo', '#55A8FF')
    ON CONFLICT (name) DO UPDATE SET color = EXCLUDED.color
    RETURNING id INTO v_category_id;

    INSERT INTO users (username, email, password, first_name, last_name)
    VALUES ('demo_account_01', 'demo-account-01@example.invalid', '!local-demo-no-login!', 'Demo AITU Account', '01')
    ON CONFLICT (username) DO UPDATE
    SET email = EXCLUDED.email, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name
    RETURNING id INTO v_account_user_one_id;

    INSERT INTO users (username, email, password, first_name, last_name)
    VALUES ('demo_account_02', 'demo-account-02@example.invalid', '!local-demo-no-login!', 'Demo AITU Account', '02')
    ON CONFLICT (username) DO UPDATE
    SET email = EXCLUDED.email, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name
    RETURNING id INTO v_account_user_two_id;

    SELECT event_id INTO v_event_id
    FROM event_slugs
    WHERE slug = 'sunday-open-local-demo-aitu-agrotech-lab';

    IF v_event_id IS NULL THEN
        INSERT INTO events (
            title, description, image_url, user_id, organization_id, category_id, capacity,
            structured_location, start_time, end_time, format, status, published_at,
            check_in_grace_minutes, soc_gpa_badge_status, iros_badge_status
        )
        VALUES (
            '[LOCAL DEMO 20261020] AITU AgroTech Lab',
            '[{"type":"p","children":[{"text":"LOCAL DEMO EVENT — based on the Sunday Open AITU AgroTech Lab archive. Meet Aruzhan Shoman and learn about research and digital transformation in agriculture."}]},{"type":"p","children":[{"text":"Demo registration counts: two EMS account registrations and two external visitors. Demo identities use reserved .invalid email addresses and do not represent real attendees or verified AITU affiliation."}]}]'::jsonb,
            'https://sunday-open.vercel.app/events/dsvr/agrotech-lab-instagram.jpg',
            1,
            v_organization_id,
            v_category_id,
            50,
            '{"type":"campus_room","room_id":"C1.2.139","room_name":"C1.2.139","floor":2}'::jsonb,
            '2026-10-25 11:00:00+05'::timestamptz,
            '2026-10-25 13:00:00+05'::timestamptz,
            'offline',
            'published',
            NOW(),
            30,
            'not_requested',
            'not_requested'
        )
        RETURNING id INTO v_event_id;

        INSERT INTO event_slugs (event_id, slug, is_current)
        VALUES (v_event_id, 'sunday-open-local-demo-aitu-agrotech-lab', true);
    ELSE
        UPDATE events
        SET title = '[LOCAL DEMO 20261020] AITU AgroTech Lab',
            image_url = 'https://sunday-open.vercel.app/events/dsvr/agrotech-lab-instagram.jpg',
            user_id = 1,
            organization_id = v_organization_id,
            category_id = v_category_id,
            capacity = 50,
            structured_location = '{"type":"campus_room","room_id":"C1.2.139","room_name":"C1.2.139","floor":2}'::jsonb,
            start_time = '2026-10-25 11:00:00+05'::timestamptz,
            end_time = '2026-10-25 13:00:00+05'::timestamptz,
            format = 'offline',
            status = 'published',
            published_at = COALESCE(published_at, NOW()),
            updated_at = NOW()
        WHERE id = v_event_id;
    END IF;

    DELETE FROM event_registrations
    WHERE event_id = v_event_id
      AND user_id IN (v_account_user_one_id, v_account_user_two_id);

    INSERT INTO event_registrations (event_id, user_id, status, registered_at)
    VALUES
        (v_event_id, v_account_user_one_id, 'registered', NOW() - INTERVAL '12 minutes'),
        (v_event_id, v_account_user_two_id, 'registered', NOW() - INTERVAL '8 minutes');

    INSERT INTO external_registrations (event_id, full_name, phone, email, status, registered_at)
    VALUES
        (v_event_id, 'External Demo Visitor 01', 'TEST-EXT-0001', 'external-demo-01@example.invalid', 'registered', NOW() - INTERVAL '6 minutes'),
        (v_event_id, 'External Demo Visitor 02', 'TEST-EXT-0002', 'external-demo-02@example.invalid', 'registered', NOW() - INTERVAL '3 minutes')
    ON CONFLICT (event_id, phone) DO UPDATE
    SET full_name = EXCLUDED.full_name,
        email = EXCLUDED.email,
        status = EXCLUDED.status,
        cancelled_at = NULL,
        registered_at = EXCLUDED.registered_at,
        updated_at = NOW();
END $$;

COMMIT;

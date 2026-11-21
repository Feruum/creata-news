ALTER TABLE telegram_users
    ADD COLUMN reminder_day_before BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN reminder_hour_before BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN reminder_15_min BOOLEAN NOT NULL DEFAULT false;

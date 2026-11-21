CREATE TABLE IF NOT EXISTS creata_records (
 kind text NOT NULL,
 id text NOT NULL,
 data jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(kind,id)
);
CREATE INDEX IF NOT EXISTS creata_records_kind_created ON creata_records(kind,created_at DESC);
CREATE INDEX IF NOT EXISTS creata_records_data ON creata_records USING gin(data);
CREATE INDEX IF NOT EXISTS creata_launch_engagement ON creata_records((data->>'launchId')) WHERE kind IN ('launch_vote','launch_comment');
CREATE UNIQUE INDEX IF NOT EXISTS creata_article_slug ON creata_records((data->>'slug')) WHERE kind='article';
CREATE UNIQUE INDEX IF NOT EXISTS creata_newspaper_slug ON creata_records((data->>'slug')) WHERE kind='newspaper_issue';
CREATE UNIQUE INDEX IF NOT EXISTS creata_provider_subject ON creata_records((data->>'provider'),(data->>'subject')) WHERE kind='identity';
CREATE UNIQUE INDEX IF NOT EXISTS creata_telegram_user ON creata_records((data->>'telegramId')) WHERE kind='user' AND data->>'telegramId' IS NOT NULL;
CREATE TABLE IF NOT EXISTS creata_sessions(token_hash text PRIMARY KEY,user_id text NOT NULL,expires_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS creata_fingerprints(fingerprint text PRIMARY KEY,story_id text NOT NULL);
CREATE TABLE IF NOT EXISTS creata_ai_reservations(id text PRIMARY KEY,request_id text UNIQUE NOT NULL,day date NOT NULL DEFAULT CURRENT_DATE,reserved numeric NOT NULL CHECK(reserved>=0),cost numeric NOT NULL DEFAULT 0 CHECK(cost>=0),committed boolean NOT NULL DEFAULT false,input_tokens bigint NOT NULL DEFAULT 0,output_tokens bigint NOT NULL DEFAULT 0,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS creata_events(id text PRIMARY KEY,kind text NOT NULL,target_id text NOT NULL,user_id text,created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS creata_events_time ON creata_events(created_at,kind);
CREATE TABLE IF NOT EXISTS creata_migrations(version int PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now());
INSERT INTO creata_migrations(version) VALUES(1) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS creata_postiz_delivery (
 id text PRIMARY KEY,
 article_id text NOT NULL,
 revision integer NOT NULL,
 channel_id text NOT NULL,
 locale text NOT NULL,
 snapshot jsonb NOT NULL,
 status text NOT NULL,
 attempts integer NOT NULL DEFAULT 0,
 post_id text NOT NULL DEFAULT '',
 release_url text NOT NULL DEFAULT '',
 error text NOT NULL DEFAULT '',
 dispatched_at timestamptz,
 next_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(article_id,revision,channel_id,locale)
);
CREATE INDEX IF NOT EXISTS creata_postiz_due ON creata_postiz_delivery(next_at) WHERE status IN ('queued','sending','uncertain','accepted','draft');
CREATE INDEX IF NOT EXISTS creata_postiz_held_media ON creata_postiz_delivery((snapshot->>'mediaUrl')) WHERE status IN ('scheduled','queued','sending');
CREATE TABLE IF NOT EXISTS creata_postiz_throttle (scope text PRIMARY KEY, until_at timestamptz NOT NULL);

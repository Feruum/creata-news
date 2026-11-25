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
CREATE INDEX IF NOT EXISTS creata_story_follow_user ON creata_records((data->>'userId')) WHERE kind='story_follow';
CREATE UNIQUE INDEX IF NOT EXISTS creata_article_slug ON creata_records((data->>'slug')) WHERE kind='article';
CREATE INDEX IF NOT EXISTS creata_publication_slug ON creata_records((data->>'slug'),created_at DESC,id) WHERE kind='publication';
CREATE INDEX IF NOT EXISTS creata_publication_archive ON creata_records(created_at DESC,id) WHERE kind='publication';
CREATE INDEX IF NOT EXISTS creata_publication_category_archive ON creata_records((data->>'categoryId'),created_at DESC,id) WHERE kind='publication';
CREATE INDEX IF NOT EXISTS creata_publication_corrections ON creata_records((data->>'articleId'),created_at DESC,id)
 WHERE kind='publication_version' AND jsonb_typeof(data->'note')='string' AND data->>'note'<>'';
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

-- Convert legacy editorial states without touching historical revisions or publication snapshots.
DO $$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM creata_migrations WHERE version=2) THEN
  UPDATE creata_records
   SET data=jsonb_set(data,'{status}','"scheduled"')
   WHERE kind='article' AND data->>'status' IN ('approved','published')
    AND COALESCE(data->>'scheduledAt','')<>''
    AND data->>'revision'=data->>'approvedRevision';
  -- Older withdrawals landed in review. Reclassify only when the latest article
  -- audit explicitly records a withdrawal of this same revision, with no public
  -- snapshot or new approval/schedule. Ordinary review drafts remain unchanged.
  UPDATE creata_records AS article
   SET data=jsonb_set(article.data,'{status}','"unpublished"')
   WHERE article.kind='article' AND article.data->>'status'='review'
    AND article.data->>'publishedRevision' IS NULL
    AND article.data->>'approvedRevision' IS NULL
    AND article.data->>'scheduledAt' IS NULL
    AND NOT EXISTS (SELECT 1 FROM creata_records publication WHERE publication.kind='publication' AND publication.id=article.id)
    AND EXISTS (
     SELECT 1 FROM (
      SELECT audit.data FROM creata_records audit
       WHERE audit.kind='audit' AND audit.data->>'targetId'=article.id
        AND audit.data->>'action' LIKE 'article.%'
       ORDER BY COALESCE(NULLIF(audit.data->>'createdAt','')::timestamptz,audit.created_at) DESC, audit.id DESC
       LIMIT 1
     ) latest
     WHERE latest.data->>'action'='article.unpublish'
      AND latest.data->'detail'->>'revision'=article.data->>'revision'
    );
  INSERT INTO creata_migrations(version) VALUES(2);
 END IF;
END $$;

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

-- Repair the known reference seed spelling only; keep editorially changed titles.
UPDATE creata_records SET data=jsonb_set(data,'{title,ru}',to_jsonb('Designathon: дизайн-марафон TransforNATION'::text))
 WHERE kind='opportunity' AND id='o-designthon' AND data->'title'->>'ru'='Designthon: дизайн-марафон TransforNATION';
UPDATE creata_records SET data=jsonb_set(data,'{title,en}',to_jsonb('Designathon: TransforNATION design marathon'::text))
 WHERE kind='opportunity' AND id='o-designthon' AND data->'title'->>'en'='Designthon: TransforNATION design marathon';
UPDATE creata_records SET data=jsonb_set(data,'{title,kk}',to_jsonb('Designathon: TransforNATION дизайн марафоны'::text))
 WHERE kind='opportunity' AND id='o-designthon' AND data->'title'->>'kk'='Designthon: TransforNATION дизайн марафоны';

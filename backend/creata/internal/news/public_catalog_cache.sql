
CREATE TABLE IF NOT EXISTS creata_article_catalog_generation (
 singleton boolean PRIMARY KEY CHECK (singleton),
 epoch text NOT NULL DEFAULT gen_random_uuid()::text,
 version bigint NOT NULL DEFAULT 0
);
INSERT INTO creata_article_catalog_generation(singleton) VALUES(true) ON CONFLICT DO NOTHING;

CREATE OR REPLACE FUNCTION creata_touch_article_catalog() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (TG_OP <> 'DELETE' AND NEW.kind IN ('publication','publication_version','author'))
    OR (TG_OP <> 'INSERT' AND OLD.kind IN ('publication','publication_version','author')) THEN
   UPDATE creata_article_catalog_generation SET version = version + 1 WHERE singleton;
 END IF;
 RETURN NULL;
END
$$;

DO $$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='creata_article_catalog_touch' AND tgrelid='creata_records'::regclass) THEN
   CREATE TRIGGER creata_article_catalog_touch AFTER INSERT OR UPDATE OR DELETE ON creata_records
     FOR EACH ROW EXECUTE FUNCTION creata_touch_article_catalog();
 END IF;
END
$$;

DELETE FROM import_staging;

ALTER TABLE import_staging ADD COLUMN fingerprint TEXT;
ALTER TABLE import_staging ADD COLUMN source_fen TEXT;
ALTER TABLE import_staging ADD COLUMN prelude_uci TEXT;
ALTER TABLE import_staging ADD COLUMN displayed_fen TEXT;
ALTER TABLE import_staging ADD COLUMN solver TEXT;
ALTER TABLE import_staging ADD COLUMN solution_json TEXT;
ALTER TABLE import_staging ADD COLUMN solution_plies INTEGER;
ALTER TABLE import_staging ADD COLUMN external_id TEXT;
ALTER TABLE import_staging ADD COLUMN rating INTEGER;
ALTER TABLE import_staging ADD COLUMN popularity INTEGER;
ALTER TABLE import_staging ADD COLUMN play_count INTEGER;
ALTER TABLE import_staging ADD COLUMN source_url TEXT;
ALTER TABLE import_staging ADD COLUMN metadata_json TEXT;
ALTER TABLE import_staging ADD COLUMN themes_json TEXT;

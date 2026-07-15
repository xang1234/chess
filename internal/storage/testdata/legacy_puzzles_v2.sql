CREATE TABLE IF NOT EXISTS sources (
  source_id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  imported_at INTEGER NOT NULL,
  source_path TEXT NOT NULL,
  checksum TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS puzzles (
  fingerprint TEXT PRIMARY KEY,
  source_fen TEXT,
  prelude_uci TEXT,
  displayed_fen TEXT NOT NULL,
  solver TEXT NOT NULL CHECK (solver IN ('white','black')),
  solution_json TEXT NOT NULL,
  solution_plies INTEGER NOT NULL CHECK (solution_plies > 0)
);
CREATE TABLE IF NOT EXISTS puzzle_sources (
  fingerprint TEXT NOT NULL REFERENCES puzzles(fingerprint) ON DELETE CASCADE,
  source_id TEXT NOT NULL REFERENCES sources(source_id) ON DELETE CASCADE,
  external_id TEXT,
  rating INTEGER,
  popularity INTEGER,
  play_count INTEGER,
  source_url TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (fingerprint, source_id)
);
CREATE TABLE IF NOT EXISTS puzzle_themes (
  fingerprint TEXT NOT NULL,
  source_id TEXT NOT NULL,
  theme TEXT NOT NULL,
  PRIMARY KEY (fingerprint, source_id, theme),
  FOREIGN KEY (fingerprint, source_id)
    REFERENCES puzzle_sources(fingerprint, source_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS import_staging (
  import_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  puzzle_json TEXT NOT NULL,
  PRIMARY KEY (import_id, ordinal)
);
CREATE INDEX IF NOT EXISTS idx_puzzle_sources_rating ON puzzle_sources(source_id, rating);
CREATE INDEX IF NOT EXISTS idx_puzzle_sources_rating_global ON puzzle_sources(rating, fingerprint);
CREATE INDEX IF NOT EXISTS idx_puzzle_themes_theme ON puzzle_themes(source_id, theme, fingerprint);

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

CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY
);
INSERT INTO schema_migrations(version) VALUES (1), (2);

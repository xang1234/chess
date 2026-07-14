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
CREATE INDEX IF NOT EXISTS idx_puzzle_themes_theme ON puzzle_themes(source_id, theme, fingerprint);

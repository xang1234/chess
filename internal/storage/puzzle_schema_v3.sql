CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY
);

INSERT INTO schema_migrations(version) VALUES (3);

CREATE TABLE sources (
  source_id TEXT PRIMARY KEY,
  kind TEXT NOT NULL
);

CREATE TABLE source_generations (
  generation_id TEXT PRIMARY KEY,
  source_id TEXT NOT NULL REFERENCES sources(source_id),
  status TEXT NOT NULL CHECK (status IN ('building', 'sealed', 'abandoned')),
  source_path TEXT NOT NULL,
  checksum TEXT,
  started_at INTEGER NOT NULL CHECK (started_at > 0),
  sealed_at INTEGER CHECK (sealed_at IS NULL OR sealed_at > 0),
  UNIQUE (source_id, generation_id)
);

CREATE TABLE source_heads (
  source_id TEXT PRIMARY KEY REFERENCES sources(source_id),
  generation_id TEXT NOT NULL,
  FOREIGN KEY (source_id, generation_id)
    REFERENCES source_generations(source_id, generation_id)
);

CREATE TABLE puzzle_cores (
  fingerprint TEXT PRIMARY KEY,
  displayed_fen TEXT NOT NULL,
  solver TEXT NOT NULL CHECK (solver IN ('white', 'black')),
  solution_json TEXT NOT NULL,
  solution_plies INTEGER NOT NULL CHECK (solution_plies > 0)
);

CREATE TABLE puzzle_occurrences (
  generation_id TEXT NOT NULL
    REFERENCES source_generations(generation_id) ON DELETE CASCADE,
  fingerprint TEXT NOT NULL REFERENCES puzzle_cores(fingerprint),
  external_id TEXT,
  source_fen TEXT,
  prelude_uci TEXT,
  rating INTEGER,
  popularity INTEGER,
  play_count INTEGER,
  source_url TEXT,
  attribution TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  PRIMARY KEY (generation_id, fingerprint)
);

CREATE TABLE occurrence_themes (
  generation_id TEXT NOT NULL,
  fingerprint TEXT NOT NULL,
  theme TEXT NOT NULL,
  PRIMARY KEY (generation_id, fingerprint, theme),
  FOREIGN KEY (generation_id, fingerprint)
    REFERENCES puzzle_occurrences(generation_id, fingerprint) ON DELETE CASCADE
);

CREATE INDEX idx_generations_cleanup
  ON source_generations(status, generation_id);
CREATE INDEX idx_source_heads_generation
  ON source_heads(generation_id);
CREATE INDEX idx_occurrences_rating
  ON puzzle_occurrences(generation_id, rating, fingerprint);
CREATE INDEX idx_occurrences_fingerprint
  ON puzzle_occurrences(fingerprint, generation_id);
CREATE INDEX idx_occurrence_themes_theme
  ON occurrence_themes(generation_id, theme, fingerprint);

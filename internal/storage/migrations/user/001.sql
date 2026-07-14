CREATE TABLE IF NOT EXISTS profile (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  learner_rating REAL NOT NULL,
  session_size INTEGER NOT NULL CHECK (session_size IN (5,10,15)),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  session_id TEXT PRIMARY KEY,
  mode TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  current_index INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS session_items (
  session_id TEXT NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  fingerprint TEXT NOT NULL,
  source_id TEXT NOT NULL,
  state_json TEXT NOT NULL,
  PRIMARY KEY (session_id, ordinal)
);
CREATE TABLE IF NOT EXISTS attempts (
  attempt_id TEXT PRIMARY KEY,
  session_id TEXT,
  fingerprint TEXT NOT NULL,
  source_id TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER,
  incorrect_moves INTEGER NOT NULL DEFAULT 0,
  hints_used INTEGER NOT NULL DEFAULT 0,
  solution_revealed INTEGER NOT NULL DEFAULT 0,
  first_try INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS review_state (
  fingerprint TEXT PRIMARY KEY,
  due_at INTEGER NOT NULL,
  interval_index INTEGER NOT NULL,
  successful_reviews INTEGER NOT NULL,
  last_outcome TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_review_due ON review_state(due_at);
CREATE INDEX IF NOT EXISTS idx_attempts_fingerprint ON attempts(fingerprint, started_at);

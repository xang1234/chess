CREATE TABLE opening_preferences (
  course_id TEXT PRIMARY KEY,
  depth TEXT NOT NULL CHECK(depth IN ('quick','standard','reference')),
  updated_at INTEGER NOT NULL
);
CREATE TABLE opening_sessions (
  session_id TEXT PRIMARY KEY,
  course_id TEXT NOT NULL,
  generation_id TEXT NOT NULL,
  lesson_id TEXT NOT NULL,
  mode TEXT NOT NULL CHECK(mode IN ('lesson','review')),
  status TEXT NOT NULL CHECK(status IN ('active','paused','completed','restart_required')),
  depth TEXT NOT NULL CHECK(depth IN ('quick','standard','reference')),
  step_index INTEGER NOT NULL CHECK(step_index >= 0),
  state_json TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE opening_lesson_progress (
  course_id TEXT NOT NULL,
  lesson_id TEXT NOT NULL,
  completed_step_ids_json TEXT NOT NULL,
  completed_steps INTEGER NOT NULL CHECK(completed_steps >= 0),
  total_steps INTEGER NOT NULL CHECK(total_steps > 0),
  completed_at INTEGER,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY(course_id, lesson_id)
);
CREATE TABLE opening_attempts (
  attempt_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  course_id TEXT NOT NULL,
  prompt_id TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER,
  outcome TEXT,
  incorrect_moves INTEGER NOT NULL DEFAULT 0,
  alternatives_tried INTEGER NOT NULL DEFAULT 0,
  hints_used INTEGER NOT NULL DEFAULT 0,
  revealed INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE opening_prompt_progress (
  course_id TEXT NOT NULL,
  prompt_id TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  last_outcome TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY(course_id, prompt_id)
);
CREATE TABLE opening_review_state (
  course_id TEXT NOT NULL,
  prompt_id TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  due_at INTEGER NOT NULL,
  interval_index INTEGER NOT NULL CHECK(interval_index BETWEEN 0 AND 4),
  successful_reviews INTEGER NOT NULL CHECK(successful_reviews >= 0),
  last_outcome TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','archived')),
  PRIMARY KEY(course_id, prompt_id)
);
CREATE INDEX idx_opening_sessions_resume ON opening_sessions(status, updated_at);
CREATE UNIQUE INDEX idx_opening_sessions_single_resumable
  ON opening_sessions((1))
  WHERE status IN ('active','paused','restart_required');
CREATE INDEX idx_opening_reviews_due ON opening_review_state(status, due_at, course_id, prompt_id);
CREATE INDEX idx_opening_attempts_prompt ON opening_attempts(course_id, prompt_id, started_at);

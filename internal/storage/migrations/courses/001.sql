CREATE TABLE course_generations (
  generation_id TEXT PRIMARY KEY,
  course_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('building','sealed','abandoned')),
  source_path TEXT NOT NULL,
  checksum TEXT,
  schema_version INTEGER NOT NULL,
  content_version TEXT NOT NULL,
  started_at INTEGER NOT NULL CHECK(started_at > 0),
  sealed_at INTEGER,
  UNIQUE(course_id, generation_id)
);
CREATE TABLE course_heads (
  course_id TEXT PRIMARY KEY,
  generation_id TEXT NOT NULL,
  FOREIGN KEY(course_id, generation_id)
    REFERENCES course_generations(course_id, generation_id)
);
CREATE TABLE courses (
  generation_id TEXT PRIMARY KEY REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  course_id TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  perspective TEXT NOT NULL CHECK(perspective IN ('white','black')),
  default_depth TEXT NOT NULL CHECK(default_depth IN ('quick','standard','reference')),
  root_position_id TEXT NOT NULL,
  source_json TEXT NOT NULL,
  coverage_json TEXT NOT NULL
);
CREATE TABLE course_chapters (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  chapter_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  title TEXT NOT NULL,
  overview TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  PRIMARY KEY(generation_id, chapter_id),
  UNIQUE(generation_id, ordinal)
);
CREATE TABLE course_positions (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  position_id TEXT NOT NULL,
  fen TEXT NOT NULL,
  label TEXT NOT NULL,
  evaluation_json TEXT NOT NULL,
  note_ids_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, position_id)
);
CREATE TABLE course_moves (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  move_id TEXT NOT NULL,
  from_position_id TEXT NOT NULL,
  to_position_id TEXT NOT NULL,
  uci TEXT NOT NULL,
  san TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  training_role TEXT NOT NULL CHECK(training_role IN ('repertoire','opponent','alternative')),
  variation_name TEXT NOT NULL,
  evaluation_json TEXT NOT NULL,
  note_ids_json TEXT NOT NULL,
  source_ref_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, move_id),
  FOREIGN KEY(generation_id, from_position_id)
    REFERENCES course_positions(generation_id, position_id),
  FOREIGN KEY(generation_id, to_position_id)
    REFERENCES course_positions(generation_id, position_id)
);
CREATE TABLE course_notes (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  note_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  text TEXT NOT NULL,
  source_ref_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, note_id)
);
CREATE TABLE course_lessons (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  lesson_id TEXT NOT NULL,
  chapter_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  title TEXT NOT NULL,
  objectives_json TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  start_position_id TEXT NOT NULL,
  PRIMARY KEY(generation_id, lesson_id),
  FOREIGN KEY(generation_id, chapter_id)
    REFERENCES course_chapters(generation_id, chapter_id),
  FOREIGN KEY(generation_id, start_position_id)
    REFERENCES course_positions(generation_id, position_id)
);
CREATE TABLE course_prompts (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  prompt_id TEXT NOT NULL,
  position_id TEXT NOT NULL,
  primary_move_id TEXT NOT NULL,
  accepted_move_ids_json TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  PRIMARY KEY(generation_id, prompt_id),
  FOREIGN KEY(generation_id, position_id)
    REFERENCES course_positions(generation_id, position_id),
  FOREIGN KEY(generation_id, primary_move_id)
    REFERENCES course_moves(generation_id, move_id)
);
CREATE TABLE course_lesson_steps (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  lesson_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  step_id TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('explain','watch','try','branch','recall')),
  position_id TEXT NOT NULL,
  data_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, lesson_id, step_id),
  UNIQUE(generation_id, lesson_id, ordinal),
  FOREIGN KEY(generation_id, lesson_id)
    REFERENCES course_lessons(generation_id, lesson_id),
  FOREIGN KEY(generation_id, position_id)
    REFERENCES course_positions(generation_id, position_id)
);
CREATE INDEX idx_course_generations_cleanup ON course_generations(status, generation_id);
CREATE INDEX idx_course_moves_from ON course_moves(generation_id, from_position_id, minimum_depth);
CREATE INDEX idx_course_lessons_chapter ON course_lessons(generation_id, chapter_id, ordinal);

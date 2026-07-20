CREATE TABLE course_lesson_edges (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  edge_id TEXT NOT NULL,
  from_lesson_id TEXT NOT NULL,
  to_lesson_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  kind TEXT NOT NULL CHECK(kind IN ('continuation','alternative','reference')),
  label TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  PRIMARY KEY(generation_id, edge_id),
  UNIQUE(generation_id, from_lesson_id, ordinal),
  FOREIGN KEY(generation_id, from_lesson_id)
    REFERENCES course_lessons(generation_id, lesson_id),
  FOREIGN KEY(generation_id, to_lesson_id)
    REFERENCES course_lessons(generation_id, lesson_id)
);

CREATE TABLE course_lesson_activities (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  lesson_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  activity_id TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('concept','demonstration','decision','comparison','recap','reference')),
  required INTEGER NOT NULL CHECK(required IN (0,1)),
  position_id TEXT,
  data_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, lesson_id, activity_id),
  UNIQUE(generation_id, lesson_id, ordinal),
  FOREIGN KEY(generation_id, lesson_id)
    REFERENCES course_lessons(generation_id, lesson_id),
  FOREIGN KEY(generation_id, position_id)
    REFERENCES course_positions(generation_id, position_id)
);

CREATE INDEX idx_course_lesson_edges_parent
  ON course_lesson_edges(generation_id, from_lesson_id, ordinal);

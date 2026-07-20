INSERT OR IGNORE INTO opening_preferences(course_id, depth, updated_at)
SELECT course_id, depth, updated_at
FROM opening_course_journeys;

UPDATE opening_preferences
SET depth = (
      SELECT journey.depth
      FROM opening_course_journeys journey
      WHERE journey.course_id = opening_preferences.course_id
    ),
    updated_at = (
      SELECT journey.updated_at
      FROM opening_course_journeys journey
      WHERE journey.course_id = opening_preferences.course_id
    )
WHERE EXISTS (
  SELECT 1
  FROM opening_course_journeys journey
  WHERE journey.course_id = opening_preferences.course_id
);

CREATE TABLE opening_course_journeys_v2 (
  course_id TEXT PRIMARY KEY,
  current_lesson_id TEXT NOT NULL,
  path_lesson_ids_json TEXT NOT NULL,
  created_at INTEGER NOT NULL CHECK(created_at > 0),
  updated_at INTEGER NOT NULL CHECK(updated_at > 0)
);

INSERT INTO opening_course_journeys_v2(
  course_id, current_lesson_id, path_lesson_ids_json, created_at, updated_at
)
SELECT course_id, current_lesson_id, path_lesson_ids_json, created_at, updated_at
FROM opening_course_journeys;

DROP TABLE opening_course_journeys;
ALTER TABLE opening_course_journeys_v2 RENAME TO opening_course_journeys;

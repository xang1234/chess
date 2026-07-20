ALTER TABLE opening_sessions RENAME COLUMN step_index TO activity_index;
ALTER TABLE opening_lesson_progress RENAME COLUMN completed_step_ids_json TO completed_activity_ids_json;
ALTER TABLE opening_lesson_progress RENAME COLUMN completed_steps TO completed_activities;
ALTER TABLE opening_lesson_progress RENAME COLUMN total_steps TO total_activities;

UPDATE opening_sessions
SET state_json = replace(state_json, '"stepIndex":', '"activityIndex":')
WHERE state_json LIKE '%"stepIndex":%';

CREATE TABLE opening_course_journeys (
  course_id TEXT PRIMARY KEY,
  depth TEXT NOT NULL CHECK(depth IN ('quick','standard','reference')),
  current_lesson_id TEXT NOT NULL,
  current_activity_id TEXT NOT NULL,
  path_lesson_ids_json TEXT NOT NULL,
  last_recommended_lesson_id TEXT NOT NULL,
  active_session_id TEXT NOT NULL,
  created_at INTEGER NOT NULL CHECK(created_at > 0),
  updated_at INTEGER NOT NULL CHECK(updated_at > 0)
);

INSERT INTO opening_course_journeys(
  course_id, depth, current_lesson_id, current_activity_id,
  path_lesson_ids_json, last_recommended_lesson_id, active_session_id,
  created_at, updated_at
)
SELECT course_id, depth, lesson_id, '', printf('["%s"]', lesson_id), '', session_id,
       created_at, updated_at
FROM opening_sessions
WHERE status IN ('active','paused','restart_required');

INSERT INTO opening_course_journeys(
  course_id, depth, current_lesson_id, current_activity_id,
  path_lesson_ids_json, last_recommended_lesson_id, active_session_id,
  created_at, updated_at
)
SELECT preference.course_id, preference.depth, '', '', '[]', '', '',
       preference.updated_at, preference.updated_at
FROM opening_preferences preference
WHERE NOT EXISTS (
  SELECT 1 FROM opening_course_journeys journey
  WHERE journey.course_id = preference.course_id
);

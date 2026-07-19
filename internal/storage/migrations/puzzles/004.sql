ALTER TABLE source_generations
ADD COLUMN maximum_solution_plies INTEGER NOT NULL DEFAULT 0
CHECK (maximum_solution_plies >= 0);

UPDATE source_generations
SET maximum_solution_plies = COALESCE((
  SELECT MAX(core.solution_plies)
  FROM puzzle_occurrences AS occurrence
  JOIN puzzle_cores AS core ON core.fingerprint = occurrence.fingerprint
  WHERE occurrence.generation_id = source_generations.generation_id
), 0);

DELETE FROM schema_migrations WHERE version = 3;

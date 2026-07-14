CREATE TABLE rating_history (
  rating_history_id INTEGER PRIMARY KEY AUTOINCREMENT,
  rating REAL NOT NULL,
  recorded_at INTEGER NOT NULL
);
CREATE INDEX idx_rating_history_recorded_at ON rating_history(recorded_at, rating_history_id);
INSERT INTO rating_history(rating, recorded_at)
SELECT learner_rating, updated_at FROM profile;

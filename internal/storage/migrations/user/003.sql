ALTER TABLE session_items ADD COLUMN source_kind TEXT;
ALTER TABLE session_items ADD COLUMN rating_snapshot INTEGER;
ALTER TABLE session_items ADD COLUMN themes_json TEXT;
ALTER TABLE session_items ADD COLUMN source_fen_snapshot TEXT;
ALTER TABLE session_items ADD COLUMN prelude_uci_snapshot TEXT;

ALTER TABLE attempts ADD COLUMN source_kind TEXT;
ALTER TABLE attempts ADD COLUMN rating_snapshot INTEGER;
ALTER TABLE attempts ADD COLUMN themes_json TEXT;

ALTER TABLE review_state ADD COLUMN preferred_source_id TEXT;

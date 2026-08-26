ALTER TABLE fixed_expenses ADD COLUMN occurrence_start_at TEXT NOT NULL DEFAULT '';

UPDATE fixed_expenses
SET occurrence_start_at = created_at;

ALTER TABLE transaction_occurrences ADD COLUMN tags_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE transaction_occurrences ADD COLUMN splits_json TEXT NOT NULL DEFAULT '[]';
CREATE UNIQUE INDEX transaction_occurrences_recurrence_date_idx
    ON transaction_occurrences(recurrence_rule_id, scheduled_date)
    WHERE recurrence_rule_id IS NOT NULL;

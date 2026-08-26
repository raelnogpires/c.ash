ALTER TABLE transactions ADD COLUMN origin TEXT NOT NULL DEFAULT 'manual'
    CHECK (origin IN ('manual', 'import', 'fixed_expense', 'card_payment', 'adjustment'));
ALTER TABLE transactions ADD COLUMN adjustment_reason TEXT;

UPDATE transactions SET origin = 'import' WHERE automatic_import = 1;
UPDATE transactions SET origin = 'fixed_expense' WHERE fixed_expense_occurrence_id IS NOT NULL;
UPDATE transactions SET origin = 'card_payment' WHERE invoice_payment_id IS NOT NULL;

CREATE INDEX transactions_origin_idx ON transactions(origin, occurrence_date DESC, created_at DESC);

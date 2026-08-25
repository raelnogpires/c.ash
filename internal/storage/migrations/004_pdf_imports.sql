ALTER TABLE transactions ADD COLUMN automatic_import INTEGER NOT NULL DEFAULT 0 CHECK (automatic_import IN (0, 1));
ALTER TABLE transactions ADD COLUMN import_bank TEXT;
ALTER TABLE transactions ADD COLUMN import_key TEXT;

CREATE UNIQUE INDEX transactions_import_key_idx
    ON transactions(import_key)
    WHERE import_key IS NOT NULL;

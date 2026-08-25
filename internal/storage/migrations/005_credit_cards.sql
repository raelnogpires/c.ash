-- SQLite cannot alter a CHECK expression directly. Updating the canonical table
-- definition is safe here because no row layout changes and the schema cookie is
-- advanced before subsequent statements are prepared.
PRAGMA writable_schema = ON;
UPDATE sqlite_schema
SET sql = replace(sql,
    'CHECK (type IN (''checking'', ''savings'', ''cash''))',
    'CHECK (type IN (''checking'', ''savings'', ''cash'', ''credit_card''))')
WHERE type = 'table' AND name = 'accounts';
PRAGMA writable_schema = OFF;
PRAGMA schema_version = 6;

ALTER TABLE accounts ADD COLUMN credit_limit_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN closing_day INTEGER NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN due_day INTEGER NOT NULL DEFAULT 0;

ALTER TABLE transactions ADD COLUMN installment_count INTEGER NOT NULL DEFAULT 1 CHECK (installment_count BETWEEN 1 AND 48);
ALTER TABLE transactions ADD COLUMN invoice_payment_id TEXT;

CREATE TABLE credit_card_invoices (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    reference_month TEXT NOT NULL,
    closing_date TEXT NOT NULL,
    due_date TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('open', 'closed', 'paid', 'rolled_over')),
    carry_forward_cents INTEGER NOT NULL DEFAULT 0 CHECK (carry_forward_cents >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(account_id, closing_date)
);

CREATE TABLE credit_card_installments (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL REFERENCES credit_card_invoices(id) ON DELETE RESTRICT,
    transaction_id TEXT REFERENCES transactions(id) ON DELETE RESTRICT,
    description TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    installment_number INTEGER NOT NULL CHECK (installment_number > 0),
    installment_count INTEGER NOT NULL CHECK (installment_count BETWEEN 1 AND 48),
    opening_debt INTEGER NOT NULL DEFAULT 0 CHECK (opening_debt IN (0, 1)),
    created_at TEXT NOT NULL,
    UNIQUE(transaction_id, installment_number)
);

CREATE TABLE credit_card_payments (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL REFERENCES credit_card_invoices(id) ON DELETE RESTRICT,
    source_account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    transaction_id TEXT NOT NULL UNIQUE REFERENCES transactions(id) ON DELETE RESTRICT,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    occurrence_date TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX credit_card_invoices_account_idx ON credit_card_invoices(account_id, closing_date);
CREATE INDEX credit_card_installments_invoice_idx ON credit_card_installments(invoice_id);
CREATE INDEX credit_card_installments_transaction_idx ON credit_card_installments(transaction_id);
CREATE INDEX credit_card_payments_invoice_idx ON credit_card_payments(invoice_id, occurrence_date);
CREATE INDEX credit_card_payments_source_idx ON credit_card_payments(source_account_id);

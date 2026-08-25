ALTER TABLE profile ADD COLUMN balances_hidden INTEGER NOT NULL DEFAULT 0 CHECK (balances_hidden IN (0, 1));

CREATE TABLE fixed_expenses (
    id TEXT PRIMARY KEY,
    description TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    due_day INTEGER NOT NULL CHECK (due_day BETWEEN 1 AND 31),
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    archived_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX fixed_expenses_active_idx ON fixed_expenses(archived_at, due_day);

CREATE TABLE fixed_expense_occurrences (
    id TEXT PRIMARY KEY,
    fixed_expense_id TEXT NOT NULL REFERENCES fixed_expenses(id) ON DELETE RESTRICT,
    reference_month TEXT NOT NULL,
    due_date TEXT NOT NULL,
    description TEXT NOT NULL,
    expected_amount_cents INTEGER NOT NULL CHECK (expected_amount_cents > 0),
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'confirmed', 'dismissed')),
    transaction_id TEXT REFERENCES transactions(id) ON DELETE RESTRICT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(fixed_expense_id, reference_month)
);

CREATE INDEX fixed_expense_occurrences_status_idx ON fixed_expense_occurrences(status, due_date);

ALTER TABLE transactions ADD COLUMN fixed_expense_occurrence_id TEXT REFERENCES fixed_expense_occurrences(id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX transactions_active_fixed_occurrence_idx
    ON transactions(fixed_expense_occurrence_id)
    WHERE deleted_at IS NULL AND fixed_expense_occurrence_id IS NOT NULL;

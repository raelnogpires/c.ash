CREATE TABLE profile (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    display_name TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL CHECK (currency = 'BRL'),
    theme TEXT NOT NULL CHECK (theme IN ('', 'light', 'dark', 'gothic')),
    onboarding_status TEXT NOT NULL CHECK (onboarding_status IN ('completed', 'skipped')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('checking', 'cash')),
    opening_balance_cents INTEGER NOT NULL,
    opening_date TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('income', 'expense')),
    editable INTEGER NOT NULL DEFAULT 0 CHECK (editable = 0)
);

CREATE TABLE transactions (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('income', 'expense')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    category_id TEXT REFERENCES categories(id) ON DELETE RESTRICT,
    description TEXT NOT NULL,
    occurrence_date TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX transactions_order_idx ON transactions(occurrence_date DESC, created_at DESC);
CREATE INDEX transactions_account_idx ON transactions(account_id);

INSERT INTO categories(id, name, kind) VALUES
    ('salary', 'Salário', 'income'),
    ('other-income', 'Outras receitas', 'income'),
    ('housing', 'Moradia', 'expense'),
    ('food', 'Alimentação', 'expense'),
    ('transport', 'Transporte', 'expense'),
    ('health', 'Saúde', 'expense'),
    ('bills', 'Contas', 'expense'),
    ('leisure', 'Lazer', 'expense'),
    ('other-expense', 'Outras despesas', 'expense');

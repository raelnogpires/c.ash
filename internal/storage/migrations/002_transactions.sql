CREATE TABLE accounts_v2 (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('checking', 'savings', 'cash')),
    opening_balance_cents INTEGER NOT NULL,
    opening_date TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO accounts_v2 SELECT * FROM accounts;

CREATE TABLE transactions_v2 (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('income', 'expense', 'transfer')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    account_id TEXT NOT NULL REFERENCES accounts_v2(id) ON DELETE RESTRICT,
    destination_account_id TEXT REFERENCES accounts_v2(id) ON DELETE RESTRICT,
    category_id TEXT REFERENCES categories(id) ON DELETE RESTRICT,
    description TEXT NOT NULL,
    occurrence_date TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    CHECK (
        (kind = 'transfer' AND destination_account_id IS NOT NULL AND destination_account_id <> account_id AND category_id IS NULL)
        OR
        (kind IN ('income', 'expense') AND destination_account_id IS NULL)
    )
);

INSERT INTO transactions_v2(
    id, kind, amount_cents, account_id, destination_account_id, category_id,
    description, occurrence_date, created_at, updated_at, deleted_at
)
SELECT id, kind, amount_cents, account_id, NULL, category_id,
       description, occurrence_date, created_at, updated_at, NULL
FROM transactions;

DROP TABLE transactions;
DROP TABLE accounts;
ALTER TABLE accounts_v2 RENAME TO accounts;
ALTER TABLE transactions_v2 RENAME TO transactions;

CREATE INDEX transactions_order_idx ON transactions(occurrence_date DESC, created_at DESC);
CREATE INDEX transactions_account_idx ON transactions(account_id);
CREATE INDEX transactions_destination_idx ON transactions(destination_account_id);
CREATE INDEX transactions_active_idx ON transactions(deleted_at, occurrence_date DESC, created_at DESC);

CREATE TABLE transaction_revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    action TEXT NOT NULL CHECK (action IN ('create', 'update', 'trash', 'restore')),
    snapshot_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX transaction_revisions_transaction_idx ON transaction_revisions(transaction_id, id);

INSERT OR IGNORE INTO categories(id, name, kind) VALUES
    ('freelance', 'Trabalho autônomo', 'income'),
    ('benefits', 'Benefícios', 'income'),
    ('refunds', 'Reembolsos', 'income'),
    ('gifts-received', 'Presentes recebidos', 'income'),
    ('groceries', 'Supermercado', 'expense'),
    ('education', 'Educação', 'expense'),
    ('subscriptions', 'Assinaturas', 'expense'),
    ('shopping', 'Compras', 'expense'),
    ('personal-care', 'Cuidados pessoais', 'expense'),
    ('pets', 'Pets', 'expense'),
    ('taxes-fees', 'Impostos e taxas', 'expense'),
    ('travel', 'Viagens', 'expense'),
    ('gifts-donations', 'Presentes e doações', 'expense');

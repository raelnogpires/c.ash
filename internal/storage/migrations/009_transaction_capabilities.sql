CREATE TABLE subcategories (
    id TEXT PRIMARY KEY,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    UNIQUE(category_id, normalized_name)
);
CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL UNIQUE
);
CREATE TABLE transaction_tags (
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
    PRIMARY KEY(transaction_id, tag_id)
);
CREATE TABLE transaction_splits (
    id TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    subcategory_id TEXT REFERENCES subcategories(id) ON DELETE RESTRICT,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0)
);

CREATE TABLE recurrence_rules (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('income','expense')),
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    category_id TEXT REFERENCES categories(id) ON DELETE RESTRICT,
    subcategory_id TEXT REFERENCES subcategories(id) ON DELETE RESTRICT,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    description TEXT NOT NULL,
    day_of_month INTEGER NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
    archived_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE transaction_occurrences (
    id TEXT PRIMARY KEY,
    recurrence_rule_id TEXT REFERENCES recurrence_rules(id) ON DELETE RESTRICT,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('income','expense')),
    category_id TEXT REFERENCES categories(id) ON DELETE RESTRICT,
    subcategory_id TEXT REFERENCES subcategories(id) ON DELETE RESTRICT,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    description TEXT NOT NULL,
    scheduled_date TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','confirmed','dismissed')),
    transaction_id TEXT REFERENCES transactions(id) ON DELETE RESTRICT,
    installment_number INTEGER NOT NULL DEFAULT 1,
    installment_count INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
ALTER TABLE transactions ADD COLUMN subcategory_id TEXT REFERENCES subcategories(id) ON DELETE RESTRICT;
ALTER TABLE transactions ADD COLUMN recurrence_rule_id TEXT REFERENCES recurrence_rules(id) ON DELETE RESTRICT;
CREATE INDEX subcategories_category_idx ON subcategories(category_id, normalized_name);
CREATE INDEX transaction_tags_tag_idx ON transaction_tags(tag_id, transaction_id);
CREATE INDEX transaction_splits_transaction_idx ON transaction_splits(transaction_id);
CREATE INDEX recurrence_rules_account_idx ON recurrence_rules(account_id, archived_at);
CREATE INDEX transaction_occurrences_status_idx ON transaction_occurrences(status, scheduled_date, created_at);

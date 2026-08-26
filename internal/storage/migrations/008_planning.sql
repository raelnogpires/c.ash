CREATE TABLE monthly_budgets (
    reference_month TEXT PRIMARY KEY,
    overall_limit_cents INTEGER NOT NULL CHECK (overall_limit_cents >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE budget_category_limits (
    id TEXT PRIMARY KEY,
    reference_month TEXT NOT NULL REFERENCES monthly_budgets(reference_month) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    limit_cents INTEGER NOT NULL CHECK (limit_cents >= 0),
    rollover INTEGER NOT NULL DEFAULT 0 CHECK (rollover IN (0, 1)),
    UNIQUE(reference_month, category_id)
);
CREATE INDEX budget_category_limits_month_idx ON budget_category_limits(reference_month, category_id);

CREATE TABLE goals (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('emergency_reserve', 'savings')),
    target_cents INTEGER NOT NULL CHECK (target_cents >= 0),
    deadline TEXT,
    archived_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE goal_allocations (
    goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY(goal_id, account_id)
);
CREATE INDEX goal_allocations_account_idx ON goal_allocations(account_id, goal_id);

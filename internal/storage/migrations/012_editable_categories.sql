ALTER TABLE categories RENAME COLUMN editable TO legacy_editable;
ALTER TABLE categories ADD COLUMN editable INTEGER NOT NULL DEFAULT 1 CHECK (editable IN (0, 1));
ALTER TABLE categories ADD COLUMN archived_at TEXT;
CREATE UNIQUE INDEX categories_kind_name_idx ON categories(kind, name COLLATE NOCASE);

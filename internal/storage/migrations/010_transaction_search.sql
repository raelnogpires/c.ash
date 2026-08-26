ALTER TABLE transactions ADD COLUMN normalized_description TEXT NOT NULL DEFAULT '';
UPDATE transactions SET normalized_description=lower(trim(description));
CREATE TRIGGER transactions_normalize_insert AFTER INSERT ON transactions BEGIN
  UPDATE transactions SET normalized_description=lower(trim(NEW.description)) WHERE id=NEW.id;
END;
CREATE TRIGGER transactions_normalize_update AFTER UPDATE OF description ON transactions BEGIN
  UPDATE transactions SET normalized_description=lower(trim(NEW.description)) WHERE id=NEW.id;
END;
CREATE INDEX transactions_search_description_idx ON transactions(normalized_description);
CREATE INDEX transactions_search_amount_idx ON transactions(amount_cents, occurrence_date DESC);
CREATE INDEX transactions_search_category_idx ON transactions(category_id, subcategory_id, occurrence_date DESC);

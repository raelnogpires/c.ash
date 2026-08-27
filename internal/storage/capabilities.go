package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"c.ash/internal/domain"
)

func stableID(prefix, value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%s-%x", prefix, sum[:8])
}
func normalizeName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func (s *Store) TransactionOccurrences(ctx context.Context) ([]domain.TransactionOccurrence, error) {
	return storeRead(s, func(q *dbQueries) ([]domain.TransactionOccurrence, error) { return q.TransactionOccurrences(ctx) })
}

func (q *dbQueries) Subcategories(ctx context.Context) ([]domain.Subcategory, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT id,category_id,name FROM subcategories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Subcategory{}
	for rows.Next() {
		var x domain.Subcategory
		if err := rows.Scan(&x.ID, &x.CategoryID, &x.Name); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (q *dbQueries) EnsureSubcategory(ctx context.Context, categoryID, name string) (domain.Subcategory, error) {
	normalized := normalizeName(name)
	x := domain.Subcategory{ID: stableID("subcategory", categoryID+"|"+normalized), CategoryID: categoryID, Name: strings.TrimSpace(name)}
	_, err := q.q.ExecContext(ctx, `INSERT INTO subcategories(id,category_id,name,normalized_name) VALUES(?,?,?,?) ON CONFLICT(category_id,normalized_name) DO NOTHING`, x.ID, x.CategoryID, x.Name, normalized)
	if err != nil {
		return x, err
	}
	err = q.q.QueryRowContext(ctx, `SELECT id,category_id,name FROM subcategories WHERE category_id=? AND normalized_name=?`, categoryID, normalized).Scan(&x.ID, &x.CategoryID, &x.Name)
	return x, err
}

func (q *dbQueries) hydrateTransactions(ctx context.Context, items []domain.Transaction, prior error) error {
	if prior != nil {
		return prior
	}
	for i := range items {
		rows, err := q.q.QueryContext(ctx, `SELECT g.id,g.name,g.normalized_name FROM transaction_tags x JOIN tags g ON g.id=x.tag_id WHERE x.transaction_id=? ORDER BY g.normalized_name`, items[i].ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var tag domain.Tag
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.NormalizedName); err != nil {
				_ = rows.Close()
				return err
			}
			items[i].Tags = append(items[i].Tags, tag)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		rows, err = q.q.QueryContext(ctx, `SELECT s.id,s.category_id,c.name,COALESCE(s.subcategory_id,''),COALESCE(sc.name,''),s.amount_cents FROM transaction_splits s JOIN categories c ON c.id=s.category_id LEFT JOIN subcategories sc ON sc.id=s.subcategory_id WHERE s.transaction_id=? ORDER BY s.rowid`, items[i].ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var split domain.TransactionSplit
			if err := rows.Scan(&split.ID, &split.CategoryID, &split.CategoryName, &split.SubcategoryID, &split.SubcategoryName, &split.AmountCents); err != nil {
				_ = rows.Close()
				return err
			}
			items[i].Splits = append(items[i].Splits, split)
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (q *dbQueries) SaveTransactionDetails(ctx context.Context, tx domain.Transaction) error {
	if _, err := q.q.ExecContext(ctx, `DELETE FROM transaction_tags WHERE transaction_id=?`, tx.ID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, tag := range tx.Tags {
		normalized := normalizeName(tag.Name)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		id := stableID("tag", normalized)
		if _, err := q.q.ExecContext(ctx, `INSERT INTO tags(id,name,normalized_name) VALUES(?,?,?) ON CONFLICT(normalized_name) DO NOTHING`, id, strings.TrimSpace(tag.Name), normalized); err != nil {
			return err
		}
		if err := q.q.QueryRowContext(ctx, `SELECT id FROM tags WHERE normalized_name=?`, normalized).Scan(&id); err != nil {
			return err
		}
		if _, err := q.q.ExecContext(ctx, `INSERT INTO transaction_tags(transaction_id,tag_id) VALUES(?,?)`, tx.ID, id); err != nil {
			return err
		}
	}
	if _, err := q.q.ExecContext(ctx, `DELETE FROM transaction_splits WHERE transaction_id=?`, tx.ID); err != nil {
		return err
	}
	for _, split := range tx.Splits {
		if _, err := q.q.ExecContext(ctx, `INSERT INTO transaction_splits(id,transaction_id,category_id,subcategory_id,amount_cents) VALUES(?,?,?,?,?)`, split.ID, tx.ID, split.CategoryID, nullable(split.SubcategoryID), split.AmountCents); err != nil {
			return err
		}
	}
	return nil
}

func (q *dbQueries) InsertRecurrenceRule(ctx context.Context, id string, tx domain.Transaction, day int, at string) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO recurrence_rules(id,kind,account_id,category_id,subcategory_id,amount_cents,description,day_of_month,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, id, tx.Kind, tx.AccountID, nullable(tx.CategoryID), nullable(tx.SubcategoryID), tx.AmountCents, tx.Description, day, at, at)
	return err
}

func (q *dbQueries) UpdateRecurrenceRule(ctx context.Context, id string, tx domain.Transaction, day int, at string) error {
	result, err := q.q.ExecContext(ctx, `UPDATE recurrence_rules SET kind=?,account_id=?,category_id=?,subcategory_id=?,amount_cents=?,description=?,day_of_month=?,updated_at=? WHERE id=? AND archived_at IS NULL`, tx.Kind, tx.AccountID, nullable(tx.CategoryID), nullable(tx.SubcategoryID), tx.AmountCents, tx.Description, day, at, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownLedgerOccurrence
	}
	return err
}

func (q *dbQueries) DeletePendingRecurrenceOccurrences(ctx context.Context, ruleID string) error {
	_, err := q.q.ExecContext(ctx, `DELETE FROM transaction_occurrences WHERE recurrence_rule_id=? AND status='pending'`, ruleID)
	return err
}
func (q *dbQueries) InsertTransactionOccurrence(ctx context.Context, x domain.TransactionOccurrence, at string) error {
	tags, err := json.Marshal(x.Tags)
	if err != nil {
		return err
	}
	splits, err := json.Marshal(x.Splits)
	if err != nil {
		return err
	}
	_, err = q.q.ExecContext(ctx, `INSERT INTO transaction_occurrences(id,recurrence_rule_id,account_id,kind,category_id,subcategory_id,amount_cents,description,scheduled_date,status,installment_number,installment_count,tags_json,splits_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, x.ID, nullable(x.RecurrenceRuleID), x.AccountID, x.Kind, nullable(x.CategoryID), nullable(x.SubcategoryID), x.AmountCents, x.Description, x.ScheduledDate, x.Status, x.InstallmentNumber, x.InstallmentCount, string(tags), string(splits), at, at)
	return err
}

const occurrenceSelect = `SELECT o.id,COALESCE(o.recurrence_rule_id,''),o.account_id,a.name,o.kind,COALESCE(o.category_id,''),COALESCE(c.name,''),COALESCE(o.subcategory_id,''),o.amount_cents,o.description,o.scheduled_date,o.status,COALESCE(o.transaction_id,''),o.installment_number,o.installment_count,o.tags_json,o.splits_json,o.created_at,o.updated_at FROM transaction_occurrences o JOIN accounts a ON a.id=o.account_id LEFT JOIN categories c ON c.id=o.category_id`

func scanOccurrence(s interface{ Scan(...any) error }) (domain.TransactionOccurrence, error) {
	var x domain.TransactionOccurrence
	var tags, splits string
	err := s.Scan(&x.ID, &x.RecurrenceRuleID, &x.AccountID, &x.AccountName, &x.Kind, &x.CategoryID, &x.CategoryName, &x.SubcategoryID, &x.AmountCents, &x.Description, &x.ScheduledDate, &x.Status, &x.TransactionID, &x.InstallmentNumber, &x.InstallmentCount, &tags, &splits, &x.CreatedAt, &x.UpdatedAt)
	if err == nil {
		err = json.Unmarshal([]byte(tags), &x.Tags)
	}
	if err == nil {
		err = json.Unmarshal([]byte(splits), &x.Splits)
	}
	if x.Tags == nil {
		x.Tags = []domain.Tag{}
	}
	if x.Splits == nil {
		x.Splits = []domain.TransactionSplit{}
	}
	return x, err
}

func (q *dbQueries) ActiveRecurrenceRules(ctx context.Context) ([]domain.RecurrenceRule, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT id,day_of_month FROM recurrence_rules WHERE archived_at IS NULL ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rules := []domain.RecurrenceRule{}
	for rows.Next() {
		var rule domain.RecurrenceRule
		if err := rows.Scan(&rule.ID, &rule.DayOfMonth); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}
func (q *dbQueries) TransactionOccurrences(ctx context.Context) ([]domain.TransactionOccurrence, error) {
	rows, err := q.q.QueryContext(ctx, occurrenceSelect+` ORDER BY o.status='pending' DESC,o.scheduled_date,o.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.TransactionOccurrence{}
	for rows.Next() {
		x, err := scanOccurrence(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (q *dbQueries) TransactionOccurrence(ctx context.Context, id string) (*domain.TransactionOccurrence, error) {
	x, err := scanOccurrence(q.q.QueryRowContext(ctx, occurrenceSelect+` WHERE o.id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &x, err
}
func (q *dbQueries) SetTransactionOccurrence(ctx context.Context, id, status, transactionID, at string) error {
	r, err := q.q.ExecContext(ctx, `UPDATE transaction_occurrences SET status=?,transaction_id=?,updated_at=? WHERE id=?`, status, nullable(transactionID), at, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return domain.ErrUnknownLedgerOccurrence
	}
	return nil
}
func (q *dbQueries) ArchiveRecurrenceRule(ctx context.Context, id, at string) error {
	r, err := q.q.ExecContext(ctx, `UPDATE recurrence_rules SET archived_at=?,updated_at=? WHERE id=? AND archived_at IS NULL`, at, at, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return domain.ErrUnknownLedgerOccurrence
	}
	_, err = q.q.ExecContext(ctx, `UPDATE transaction_occurrences SET status='dismissed',updated_at=? WHERE recurrence_rule_id=? AND status='pending'`, at, id)
	return err
}
func (q *dbQueries) SetTransactionRecurrence(ctx context.Context, transactionID, ruleID string) error {
	_, err := q.q.ExecContext(ctx, `UPDATE transactions SET recurrence_rule_id=? WHERE id=?`, nullable(ruleID), transactionID)
	return err
}

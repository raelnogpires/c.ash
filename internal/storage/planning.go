package storage

import (
	"context"
	"database/sql"
	"errors"

	"c.ash/internal/domain"
)

func (s *Store) MonthlyBudget(ctx context.Context, month string) (*domain.MonthlyBudget, error) {
	return storeRead(s, func(q *dbQueries) (*domain.MonthlyBudget, error) { return q.MonthlyBudget(ctx, month) })
}

func (s *Store) Goals(ctx context.Context) ([]domain.Goal, error) {
	return storeRead(s, func(q *dbQueries) ([]domain.Goal, error) { return q.Goals(ctx) })
}

func (s *Store) Goal(ctx context.Context, id string) (*domain.Goal, error) {
	return storeRead(s, func(q *dbQueries) (*domain.Goal, error) { return q.Goal(ctx, id) })
}

func (q *dbQueries) MonthlyBudget(ctx context.Context, month string) (*domain.MonthlyBudget, error) {
	var budget domain.MonthlyBudget
	err := q.q.QueryRowContext(ctx, `SELECT reference_month, overall_limit_cents FROM monthly_budgets WHERE reference_month=?`, month).Scan(&budget.ReferenceMonth, &budget.OverallLimitCents)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.q.QueryContext(ctx, `SELECT l.id,l.category_id,c.name,l.limit_cents,l.rollover FROM budget_category_limits l JOIN categories c ON c.id=l.category_id WHERE l.reference_month=? ORDER BY c.name`, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	budget.CategoryLimits = []domain.CategoryBudgetLimit{}
	for rows.Next() {
		var limit domain.CategoryBudgetLimit
		if err := rows.Scan(&limit.ID, &limit.CategoryID, &limit.CategoryName, &limit.LimitCents, &limit.Rollover); err != nil {
			return nil, err
		}
		budget.CategoryLimits = append(budget.CategoryLimits, limit)
	}
	return &budget, rows.Err()
}

func (q *dbQueries) Goals(ctx context.Context) ([]domain.Goal, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT id,name,kind,target_cents,COALESCE(deadline,''),COALESCE(archived_at,''),created_at,updated_at FROM goals ORDER BY archived_at IS NOT NULL, created_at, name`)
	if err != nil {
		return nil, err
	}
	items := []domain.Goal{}
	for rows.Next() {
		var goal domain.Goal
		if err := rows.Scan(&goal.ID, &goal.Name, &goal.Kind, &goal.TargetCents, &goal.Deadline, &goal.ArchivedAt, &goal.CreatedAt, &goal.UpdatedAt); err != nil {
			return nil, err
		}
		goal.Allocations = []domain.GoalAllocation{}
		items = append(items, goal)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range items {
		allocations, err := q.goalAllocations(ctx, items[index].ID)
		if err != nil {
			return nil, err
		}
		items[index].Allocations = allocations
	}
	return items, nil
}

func (q *dbQueries) Goal(ctx context.Context, id string) (*domain.Goal, error) {
	var goal domain.Goal
	err := q.q.QueryRowContext(ctx, `SELECT id,name,kind,target_cents,COALESCE(deadline,''),COALESCE(archived_at,''),created_at,updated_at FROM goals WHERE id=?`, id).Scan(&goal.ID, &goal.Name, &goal.Kind, &goal.TargetCents, &goal.Deadline, &goal.ArchivedAt, &goal.CreatedAt, &goal.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	goal.Allocations, err = q.goalAllocations(ctx, id)
	return &goal, err
}

func (q *dbQueries) goalAllocations(ctx context.Context, goalID string) ([]domain.GoalAllocation, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT x.goal_id,x.account_id,a.name,x.amount_cents FROM goal_allocations x JOIN accounts a ON a.id=x.account_id WHERE x.goal_id=? ORDER BY a.name`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.GoalAllocation{}
	for rows.Next() {
		var item domain.GoalAllocation
		if err := rows.Scan(&item.GoalID, &item.AccountID, &item.AccountName, &item.AmountCents); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *dbQueries) ReplaceMonthlyBudget(ctx context.Context, budget domain.MonthlyBudget, at string) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO monthly_budgets(reference_month,overall_limit_cents,created_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(reference_month) DO UPDATE SET overall_limit_cents=excluded.overall_limit_cents,updated_at=excluded.updated_at`, budget.ReferenceMonth, budget.OverallLimitCents, at, at)
	if err != nil {
		return err
	}
	if _, err := q.q.ExecContext(ctx, `DELETE FROM budget_category_limits WHERE reference_month=?`, budget.ReferenceMonth); err != nil {
		return err
	}
	for _, limit := range budget.CategoryLimits {
		if _, err := q.q.ExecContext(ctx, `INSERT INTO budget_category_limits(id,reference_month,category_id,limit_cents,rollover) VALUES(?,?,?,?,?)`, limit.ID, budget.ReferenceMonth, limit.CategoryID, limit.LimitCents, limit.Rollover); err != nil {
			return err
		}
	}
	return nil
}

func (q *dbQueries) InsertGoal(ctx context.Context, goal domain.Goal, at string) error {
	var deadline any
	if goal.Deadline != "" {
		deadline = goal.Deadline
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO goals(id,name,kind,target_cents,deadline,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, goal.ID, goal.Name, goal.Kind, goal.TargetCents, deadline, at, at)
	return err
}

func (q *dbQueries) UpdateGoal(ctx context.Context, goal domain.Goal, at string) error {
	var deadline any
	if goal.Deadline != "" {
		deadline = goal.Deadline
	}
	result, err := q.q.ExecContext(ctx, `UPDATE goals SET name=?,kind=?,target_cents=?,deadline=?,updated_at=? WHERE id=?`, goal.Name, goal.Kind, goal.TargetCents, deadline, at, goal.ID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownGoal
	}
	return err
}

func (q *dbQueries) ArchiveGoal(ctx context.Context, id, at string) error {
	result, err := q.q.ExecContext(ctx, `UPDATE goals SET archived_at=?,updated_at=? WHERE id=? AND archived_at IS NULL`, at, at, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownGoal
	}
	return err
}

func (q *dbQueries) ReplaceGoalAllocations(ctx context.Context, goalID string, allocations []domain.GoalAllocation, at string) error {
	if _, err := q.q.ExecContext(ctx, `DELETE FROM goal_allocations WHERE goal_id=?`, goalID); err != nil {
		return err
	}
	for _, item := range allocations {
		if item.AmountCents == 0 {
			continue
		}
		if _, err := q.q.ExecContext(ctx, `INSERT INTO goal_allocations(goal_id,account_id,amount_cents,created_at,updated_at) VALUES(?,?,?,?,?)`, goalID, item.AccountID, item.AmountCents, at, at); err != nil {
			return err
		}
	}
	return nil
}

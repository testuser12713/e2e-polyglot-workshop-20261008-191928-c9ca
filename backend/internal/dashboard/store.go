package dashboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Summary is the workshop dashboard KPI payload (AC-20). Amounts are whole
// cents, the day and month are the current UTC day / month.
type Summary struct {
	OpenOrders        int64 `json:"open_orders"`
	DoneToday         int64 `json:"done_today"`
	RevenueMonthCents int64 `json:"revenue_month_cents"`
}

// Store reads the dashboard KPIs from the shared database.
type Store struct {
	db *pgxpool.Pool
}

// NewStore builds the dashboard store on the shared pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// summaryQuery computes all three KPIs in one round trip:
//   - open orders are the ones that have not reached done or picked_up yet;
//   - orders done today are the order_status_history rows with status 'done'
//     inside the current UTC day;
//   - the month revenue is the sum of the gross amounts of the invoices
//     created in the current UTC month.
const summaryQuery = `
	SELECT
		(SELECT count(*) FROM orders WHERE status NOT IN ('done', 'picked_up')),
		(SELECT count(*) FROM order_status_history
		  WHERE status = 'done'
		    AND changed_at >= date_trunc('day', now(), 'UTC')
		    AND changed_at <  date_trunc('day', now(), 'UTC') + interval '1 day'),
		(SELECT COALESCE(sum(gross_cents), 0) FROM invoices
		  WHERE created_at >= date_trunc('month', now(), 'UTC')
		    AND created_at <  date_trunc('month', now(), 'UTC') + interval '1 month')`

// Summary reads the three dashboard KPIs.
func (s *Store) Summary(ctx context.Context) (Summary, error) {
	var result Summary
	err := s.db.QueryRow(ctx, summaryQuery).
		Scan(&result.OpenOrders, &result.DoneToday, &result.RevenueMonthCents)
	if err != nil {
		return Summary{}, err
	}
	return result, nil
}

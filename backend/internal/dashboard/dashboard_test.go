package dashboard_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/dashboard"
	"workshop-api/internal/db"
	"workshop-api/internal/order"
)

// newIsolatedPool opens the real PostgreSQL from DATABASE_URL into a private,
// freshly migrated schema, so this test never sees rows other packages write
// in parallel and never leaves rows behind. There is no SQLite fallback.
func newIsolatedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set; these tests run against a real PostgreSQL")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("dashboard_test_%d", time.Now().UnixNano())

	admin, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(admin.Close)
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema %s: %v", schema, err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	})

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open scoped pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping scoped pool: %v", err)
	}
	if err := db.RunMigrations(ctx, pool, ""); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return pool
}

// seeder inserts a customer, a vehicle, orders, their status history and
// invoices directly through SQL. Every seeded row is removed when the test
// ends.
type seeder struct {
	t          *testing.T
	ctx        context.Context
	pool       *pgxpool.Pool
	customerID int64
	vehicleID  int64
	orderIDs   []int64
}

func newSeeder(t *testing.T, pool *pgxpool.Pool) *seeder {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	s := &seeder{t: t, ctx: ctx, pool: pool}

	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Dashboard Test", fmt.Sprintf("dashboard-%d@example.test", suffix), "000",
	).Scan(&s.customerID); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (plate, brand, model, mileage) VALUES ($1, $2, $3, $4) RETURNING id`,
		fmt.Sprintf("DA-%d", suffix), "VW", "Golf", 0,
	).Scan(&s.vehicleID); err != nil {
		t.Fatalf("seed vehicle: %v", err)
	}
	t.Cleanup(s.cleanup)
	return s
}

func (s *seeder) cleanup() {
	ctx := context.Background()
	for _, id := range s.orderIDs {
		_, _ = s.pool.Exec(ctx, `DELETE FROM invoices WHERE order_id = $1`, id)
		_, _ = s.pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, id)
	}
	_, _ = s.pool.Exec(ctx, `DELETE FROM vehicles WHERE id = $1`, s.vehicleID)
	_, _ = s.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, s.customerID)
}

// addOrder inserts an order in the given status and returns its id.
func (s *seeder) addOrder(status string) int64 {
	s.t.Helper()
	var id int64
	if err := s.pool.QueryRow(s.ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, status)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		fmt.Sprintf("ORD-%d-%d", time.Now().UnixNano(), len(s.orderIDs)),
		s.customerID, s.vehicleID, status,
	).Scan(&id); err != nil {
		s.t.Fatalf("seed order (%s): %v", status, err)
	}
	s.orderIDs = append(s.orderIDs, id)
	return id
}

// addHistory records a status change at the given time.
func (s *seeder) addHistory(orderID int64, status string, at time.Time) {
	s.t.Helper()
	if _, err := s.pool.Exec(s.ctx,
		`INSERT INTO order_status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		orderID, status, at,
	); err != nil {
		s.t.Fatalf("seed history: %v", err)
	}
}

// addInvoice records an invoice with the given gross amount at the given time.
func (s *seeder) addInvoice(orderID, grossCents int64, at time.Time) {
	s.t.Helper()
	if _, err := s.pool.Exec(s.ctx,
		`INSERT INTO invoices (order_id, net_cents, vat_cents, gross_cents, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		orderID, grossCents, 0, grossCents, at,
	); err != nil {
		s.t.Fatalf("seed invoice: %v", err)
	}
}

type dashboardResponse struct {
	OpenOrders        int64 `json:"open_orders"`
	DoneToday         int64 `json:"done_today"`
	RevenueMonthCents int64 `json:"revenue_month_cents"`
}

func getDashboard(t *testing.T, h *dashboard.Handler) (int, dashboardResponse) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/workshop/dashboard", nil))
	var body dashboardResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("dashboard body is not JSON: %v (body=%q)", err, rec.Body.String())
		}
	}
	return rec.Code, body
}

func TestDashboardOnAnEmptySchemaReturnsZeros(t *testing.T) {
	pool := newIsolatedPool(t)
	h := dashboard.NewHandler(order.NewStore(pool, nil))

	status, body := getDashboard(t, h)
	if status != http.StatusOK {
		t.Fatalf("GET /api/workshop/dashboard = %d, want 200", status)
	}
	if body.OpenOrders != 0 || body.DoneToday != 0 || body.RevenueMonthCents != 0 {
		t.Fatalf("empty dashboard = %+v, want all zeros", body)
	}
}

func TestDashboardCountsOpenDoneTodayAndMonthRevenue(t *testing.T) {
	pool := newIsolatedPool(t)
	h := dashboard.NewHandler(order.NewStore(pool, nil))
	s := newSeeder(t, pool)

	// Three orders are still open, two have reached a terminal state.
	s.addOrder(order.StatusRequested)
	s.addOrder(order.StatusConfirmed)
	s.addOrder(order.StatusInProgress)
	doneToday := s.addOrder(order.StatusDone)
	s.addOrder(order.StatusPickedUp)

	now := time.Now().UTC()
	// One order reached done today, one only yesterday.
	s.addHistory(doneToday, order.StatusDone, now)
	doneYesterday := s.addOrder(order.StatusDone)
	s.addHistory(doneYesterday, order.StatusDone, now.Add(-24*time.Hour))

	// An invoice created this month counts, one from last month does not.
	s.addInvoice(doneToday, 123456, now)
	lastMonth := now.AddDate(0, 0, -31)
	s.addInvoice(doneYesterday, 999999, lastMonth)

	status, body := getDashboard(t, h)
	if status != http.StatusOK {
		t.Fatalf("GET /api/workshop/dashboard = %d, want 200", status)
	}
	if body.OpenOrders != 3 {
		t.Errorf("open_orders = %d, want 3", body.OpenOrders)
	}
	if body.DoneToday != 1 {
		t.Errorf("done_today = %d, want 1", body.DoneToday)
	}
	if body.RevenueMonthCents != 123456 {
		t.Errorf("revenue_month_cents = %d, want 123456", body.RevenueMonthCents)
	}
}

func TestDashboardReportsDatabaseErrors(t *testing.T) {
	pool := newIsolatedPool(t)
	h := dashboard.NewHandler(order.NewStore(pool, nil))
	pool.Close()

	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/workshop/dashboard", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("closed database = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("error body is not the uniform envelope: %v", err)
	}
	if env.Error.Code == "" || env.Error.Message == "" {
		t.Fatalf("error envelope missing code/message: %q", rec.Body.String())
	}
}

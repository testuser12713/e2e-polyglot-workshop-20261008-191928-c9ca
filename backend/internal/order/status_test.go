package order_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"workshop-api/internal/db"
	"workshop-api/internal/order"
	"workshop-api/internal/queue"
	"workshop-api/internal/workshop"
)

// harness wires the real PostgreSQL from DATABASE_URL and the running Valkey
// from VALKEY_URL (AC-25: no SQLite, no doubles). Every order gets its own
// queue list so a message count is unambiguous.
type harness struct {
	ctx      context.Context
	pool     *pgxpool.Pool
	store    *order.Store
	rdb      *redis.Client
	listName string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	valkeyURL := os.Getenv("VALKEY_URL")
	if dsn == "" || valkeyURL == "" {
		t.Skip("DATABASE_URL and VALKEY_URL are required; these tests run against real PostgreSQL and Valkey")
	}

	ctx := context.Background()

	// Migrate a dedicated schema for this package. `go test ./...` runs the
	// package binaries in parallel against one database, and CREATE TABLE IF
	// NOT EXISTS inside one schema is not race-free: two migrators collide on
	// the pg_type catalog (SQLSTATE 23505). Isolating the order tests keeps
	// them from racing the httpapi package's migration and honors the rule that
	// a test owns only its own tables.
	schema := fmt.Sprintf("order_status_test_%d", time.Now().UnixNano())
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+quotedSchema); err != nil {
		pool.Close()
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+quotedSchema+` CASCADE`)
		pool.Close()
	})
	if err := db.RunMigrations(ctx, pool, ""); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	opts, err := redis.ParseURL(valkeyURL)
	if err != nil {
		t.Fatalf("parse VALKEY_URL: %v", err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })

	listName := fmt.Sprintf("test-order-status-%d", time.Now().UnixNano())
	if err := rdb.Del(ctx, listName).Err(); err != nil {
		t.Fatalf("clear test queue list: %v", err)
	}

	return &harness{
		ctx:      ctx,
		pool:     pool,
		store:    order.NewStore(pool, queue.NewPublisher(valkeyURL, listName)),
		rdb:      rdb,
		listName: listName,
	}
}

// seedOrder inserts a customer, a vehicle and an order with a unique number in
// the given status, and removes all three again when the test ends.
func (h *harness) seedOrder(t *testing.T, status string) (order.Order, int64) {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	var customerID int64
	err := h.pool.QueryRow(h.ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Status Test", "status-test-"+suffix+"@example.com", "000-000",
	).Scan(&customerID)
	if err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var vehicleID int64
	plate := "STS" + suffix
	err = h.pool.QueryRow(h.ctx,
		`INSERT INTO vehicles (plate, brand, model, mileage) VALUES ($1, $2, $3, $4) RETURNING id`,
		plate, "VW", "Golf", 120000,
	).Scan(&vehicleID)
	if err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	orderNumber := "ORD-" + suffix
	var orderID int64
	err = h.pool.QueryRow(h.ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, status, description)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		orderNumber, customerID, vehicleID, status, "Bremsen pruefen",
	).Scan(&orderID)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}

	t.Cleanup(func() {
		_, _ = h.pool.Exec(h.ctx, `DELETE FROM invoices WHERE order_id = $1`, orderID)
		_, _ = h.pool.Exec(h.ctx, `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = h.pool.Exec(h.ctx, `DELETE FROM vehicles WHERE id = $1`, vehicleID)
		_, _ = h.pool.Exec(h.ctx, `DELETE FROM customers WHERE id = $1`, customerID)
	})

	o, err := h.store.GetByID(h.ctx, orderID)
	if err != nil {
		t.Fatalf("load seeded order: %v", err)
	}
	return o, vehicleID
}

func (h *harness) queueLength(t *testing.T) int64 {
	t.Helper()
	n, err := h.rdb.LLen(h.ctx, h.listName).Result()
	if err != nil {
		t.Fatalf("queue length: %v", err)
	}
	return n
}

// getStatus calls the public handler directly with httptest.
func (h *harness) getStatus(t *testing.T, orderNumber, plate string) *httptest.ResponseRecorder {
	t.Helper()
	handler := order.NewStatusHandler(h.store)
	path := "/api/orders/status?order_number=" + url.QueryEscape(orderNumber) + "&plate=" + url.QueryEscape(plate)
	rec := httptest.NewRecorder()
	handler.Get(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// setStatus calls the workshop handler directly with httptest. The workshop
// route is guarded by the bearer middleware, which another ticket still owes,
// so the handler is exercised without the outer router.
func (h *harness) setStatus(t *testing.T, orderID int64, status string) *httptest.ResponseRecorder {
	t.Helper()
	handler := workshop.NewStatusHandler(h.store)
	path := "/api/workshop/orders/" + strconv.FormatInt(orderID, 10) + "/status"
	body := strings.NewReader(fmt.Sprintf(`{"status":%q}`, status))
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.SetPathValue("id", strconv.FormatInt(orderID, 10))
	rec := httptest.NewRecorder()
	handler.Set(rec, req)
	return rec
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func assertUniformError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body=%q)", rec.Code, wantStatus, rec.Body.String())
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("error body is not the uniform envelope: %v (body=%q)", err, rec.Body.String())
	}
	if env.Error.Code == "" || env.Error.Message == "" {
		t.Fatalf("error envelope is missing code/message: %q", rec.Body.String())
	}
}

func TestTransitionRejectsSkippedStepAndStepBack(t *testing.T) {
	h := newHarness(t)
	o, _ := h.seedOrder(t, order.StatusRequested)

	if _, err := order.Transition(h.ctx, h.store, o.ID, order.StatusInProgress); !errors.Is(err, order.ErrInvalidTransition) {
		t.Fatalf("requested -> in_progress error = %v, want ErrInvalidTransition", err)
	}

	updated, err := order.Transition(h.ctx, h.store, o.ID, order.StatusConfirmed)
	if err != nil {
		t.Fatalf("requested -> confirmed: %v", err)
	}
	if updated.Status != order.StatusConfirmed {
		t.Fatalf("status = %q, want confirmed", updated.Status)
	}
	if updated.NextStatus == nil || *updated.NextStatus != order.StatusInProgress {
		t.Fatalf("next_status = %v, want in_progress", updated.NextStatus)
	}

	if _, err := order.Transition(h.ctx, h.store, o.ID, order.StatusRequested); !errors.Is(err, order.ErrInvalidTransition) {
		t.Fatalf("confirmed -> requested error = %v, want ErrInvalidTransition", err)
	}
}

func TestTransitionToDonePublishesExactlyOneMessage(t *testing.T) {
	h := newHarness(t)
	o, _ := h.seedOrder(t, order.StatusInProgress)

	updated, err := order.Transition(h.ctx, h.store, o.ID, order.StatusDone)
	if err != nil {
		t.Fatalf("in_progress -> done: %v", err)
	}
	if updated.Status != order.StatusDone {
		t.Fatalf("status = %q, want done", updated.Status)
	}

	if n := h.queueLength(t); n != 1 {
		t.Fatalf("queue length after done = %d, want exactly 1", n)
	}

	raw, err := h.rdb.LPop(h.ctx, h.listName).Result()
	if err != nil {
		t.Fatalf("pop queue message: %v", err)
	}
	var msg struct {
		OrderID     int64  `json:"order_id"`
		OrderNumber string `json:"order_number"`
	}
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("queue message is not JSON: %v (raw=%q)", err, raw)
	}
	if msg.OrderID != o.ID || msg.OrderNumber != o.OrderNumber {
		t.Fatalf("queue message = %+v, want order_id=%d order_number=%q", msg, o.ID, o.OrderNumber)
	}
	if n := h.queueLength(t); n != 0 {
		t.Fatalf("queue length after pop = %d, want 0", n)
	}
}

func TestTransitionRecordsHistoryWithUTCTimestamps(t *testing.T) {
	h := newHarness(t)
	o, _ := h.seedOrder(t, order.StatusRequested)

	for _, status := range []string{order.StatusConfirmed, order.StatusInProgress} {
		if _, err := order.Transition(h.ctx, h.store, o.ID, status); err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}

	history, err := h.store.ListHistory(h.ctx, o.ID)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history length = %d, want 2 (rows=%+v)", len(history), history)
	}
	if history[0].Status != order.StatusConfirmed || history[1].Status != order.StatusInProgress {
		t.Fatalf("history statuses = [%q, %q], want [confirmed, in_progress]", history[0].Status, history[1].Status)
	}
	for _, entry := range history {
		if entry.ChangedAt.IsZero() {
			t.Fatalf("history entry %+v has no timestamp", entry)
		}
		if entry.ChangedAt.Location() != time.UTC {
			t.Fatalf("history entry %+v is not UTC", entry)
		}
	}
}

func TestWorkshopSetStatusAppliesTheStepAndRejectsASkip(t *testing.T) {
	h := newHarness(t)
	o, _ := h.seedOrder(t, order.StatusRequested)

	rec := h.setStatus(t, o.ID, order.StatusConfirmed)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status confirmed = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var updated order.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("updated order is not JSON: %v", err)
	}
	if updated.Status != order.StatusConfirmed {
		t.Fatalf("updated status = %q, want confirmed", updated.Status)
	}

	rec = h.setStatus(t, o.ID, order.StatusDone)
	assertUniformError(t, rec, http.StatusConflict)
}

func TestStatusLookupReturnsHistoryVehicleAndInvoice(t *testing.T) {
	h := newHarness(t)
	o, _ := h.seedOrder(t, order.StatusRequested)

	if _, err := order.Transition(h.ctx, h.store, o.ID, order.StatusConfirmed); err != nil {
		t.Fatalf("transition to confirmed: %v", err)
	}

	// The invoice appears once the worker has stored it.
	err := h.pool.QueryRow(h.ctx,
		`INSERT INTO invoices (order_id, net_cents, vat_cents, gross_cents)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		o.ID, 10000, 1900, 11900,
	).Scan(new(int64))
	if err != nil {
		t.Fatalf("insert invoice: %v", err)
	}
	var invoiceID int64
	if err := h.pool.QueryRow(h.ctx, `SELECT id FROM invoices WHERE order_id = $1`, o.ID).Scan(&invoiceID); err != nil {
		t.Fatalf("load invoice id: %v", err)
	}
	_, err = h.pool.Exec(h.ctx,
		`INSERT INTO invoice_items (invoice_id, kind, description, hours, unit_price_cents, total_cents)
		 VALUES ($1, 'labor', 'Bremsen', 1.5, 8900, 13350)`,
		invoiceID,
	)
	if err != nil {
		t.Fatalf("insert invoice item: %v", err)
	}

	rec := h.getStatus(t, o.OrderNumber, o.Vehicle.Plate)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}

	var resp struct {
		OrderNumber string `json:"order_number"`
		Status      string `json:"status"`
		History     []struct {
			Status    string    `json:"status"`
			ChangedAt time.Time `json:"changed_at"`
		} `json:"history"`
		Vehicle struct {
			Plate string `json:"plate"`
		} `json:"vehicle"`
		Invoice *struct {
			Items []struct {
				Kind       string `json:"kind"`
				TotalCents int64  `json:"total_cents"`
			} `json:"items"`
			NetCents   int64 `json:"net_cents"`
			VatCents   int64 `json:"vat_cents"`
			GrossCents int64 `json:"gross_cents"`
		} `json:"invoice"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("status body is not JSON: %v", err)
	}

	if resp.OrderNumber != o.OrderNumber || resp.Status != order.StatusConfirmed {
		t.Fatalf("lookup = %q/%q, want %q/confirmed", resp.OrderNumber, resp.Status, o.OrderNumber)
	}
	if resp.Vehicle.Plate != o.Vehicle.Plate {
		t.Fatalf("vehicle plate = %q, want %q", resp.Vehicle.Plate, o.Vehicle.Plate)
	}
	if len(resp.History) != 1 || resp.History[0].Status != order.StatusConfirmed {
		t.Fatalf("history = %+v, want [confirmed]", resp.History)
	}
	if resp.History[0].ChangedAt.IsZero() || resp.History[0].ChangedAt.Location() != time.UTC {
		t.Fatalf("history timestamp = %v, want a UTC time", resp.History[0].ChangedAt)
	}
	if resp.Invoice == nil {
		t.Fatal("invoice = null, want the stored invoice")
	}
	if resp.Invoice.NetCents != 10000 || resp.Invoice.VatCents != 1900 || resp.Invoice.GrossCents != 11900 {
		t.Fatalf("invoice amounts = %+v, want 10000/1900/11900", resp.Invoice)
	}
	if len(resp.Invoice.Items) != 1 || resp.Invoice.Items[0].Kind != "labor" || resp.Invoice.Items[0].TotalCents != 13350 {
		t.Fatalf("invoice items = %+v, want one labor item of 13350", resp.Invoice.Items)
	}
}

func TestStatusLookupWithoutInvoiceReturnsNull(t *testing.T) {
	h := newHarness(t)
	o, _ := h.seedOrder(t, order.StatusRequested)

	rec := h.getStatus(t, o.OrderNumber, o.Vehicle.Plate)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Invoice json.RawMessage `json:"invoice"`
		History json.RawMessage `json:"history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("status body is not JSON: %v", err)
	}
	if string(resp.Invoice) != "null" {
		t.Fatalf("invoice = %s, want null", resp.Invoice)
	}
	if string(resp.History) == "null" {
		t.Fatal("history = null, want an empty array")
	}
}

func TestStatusLookupWrongCombinationReturns404(t *testing.T) {
	h := newHarness(t)
	o, _ := h.seedOrder(t, order.StatusRequested)

	cases := []struct {
		name        string
		orderNumber string
		plate       string
		wantStatus  int
	}{
		{"wrong plate", o.OrderNumber, "WRONG-123", http.StatusNotFound},
		{"unknown order", "ORD-DOES-NOT-EXIST", o.Vehicle.Plate, http.StatusNotFound},
		{"missing plate", o.OrderNumber, "", http.StatusBadRequest},
		{"missing number", "", o.Vehicle.Plate, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := h.getStatus(t, tc.orderNumber, tc.plate)
			assertUniformError(t, rec, tc.wantStatus)
		})
	}
}

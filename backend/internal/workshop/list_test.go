package workshop_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/db"
	"workshop-api/internal/order"
	"workshop-api/internal/workshop"
)

// harness wires the real PostgreSQL from DATABASE_URL (AC-25: no SQLite, no
// database double). Each test run migrates into its own throw-away schema, so
// it owns only its own tables and never races another package's migration.
type harness struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	store *order.Store
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is required; these tests run against a real PostgreSQL")
	}

	ctx := context.Background()
	schema := fmt.Sprintf("workshop_list_test_%d", time.Now().UnixNano())

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+quoted); err != nil {
		pool.Close()
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+quoted+` CASCADE`)
		pool.Close()
	})
	if err := db.RunMigrations(ctx, pool, ""); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return &harness{ctx: ctx, pool: pool, store: order.NewStore(pool, nil)}
}

// seedOrder inserts a customer, a vehicle and an order in the given status. The
// plate is made unique by appending a timestamp, but keeps the caller's prefix
// so a substring search on that prefix matches.
func (h *harness) seedOrder(t *testing.T, status, platePrefix string) order.Order {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	var customerID int64
	if err := h.pool.QueryRow(h.ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"List Test", "list-test-"+suffix+"@example.com", "000-000",
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var vehicleID int64
	if err := h.pool.QueryRow(h.ctx,
		`INSERT INTO vehicles (plate, brand, model, mileage) VALUES ($1, $2, $3, $4) RETURNING id`,
		platePrefix+"-"+suffix, "VW", "Golf", 100000,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	var orderID int64
	if err := h.pool.QueryRow(h.ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, status, description)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		"ORD-"+suffix, customerID, vehicleID, status, "Bremsen pruefen",
	).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	o, err := h.store.GetByID(h.ctx, orderID)
	if err != nil {
		t.Fatalf("load seeded order: %v", err)
	}
	return o
}

func (h *harness) list(t *testing.T, status, plate string) *httptest.ResponseRecorder {
	t.Helper()
	handler := workshop.NewListHandler(h.store)
	path := "/api/workshop/orders"
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	if plate != "" {
		q.Set("plate", plate)
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	rec := httptest.NewRecorder()
	handler.List(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func (h *harness) confirm(t *testing.T, orderID int64) *httptest.ResponseRecorder {
	t.Helper()
	handler := workshop.NewConfirmHandler(h.store)
	path := "/api/workshop/orders/" + strconv.FormatInt(orderID, 10) + "/confirm"
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.SetPathValue("id", strconv.FormatInt(orderID, 10))
	rec := httptest.NewRecorder()
	handler.Confirm(rec, req)
	return rec
}

type listBody struct {
	Orders []order.Order `json:"orders"`
}

func decodeList(t *testing.T, rec *httptest.ResponseRecorder) []order.Order {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("list = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var body listBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("list body is not JSON: %v (body=%q)", err, rec.Body.String())
	}
	return body.Orders
}

func hasOrder(orders []order.Order, id int64) bool {
	for _, o := range orders {
		if o.ID == id {
			return true
		}
	}
	return false
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

func TestListFiltersByStatusAndSearchesPlate(t *testing.T) {
	h := newHarness(t)
	requested := h.seedOrder(t, order.StatusRequested, "AB-111")
	confirmed := h.seedOrder(t, order.StatusConfirmed, "AB-222")
	other := h.seedOrder(t, order.StatusRequested, "CD-333")

	t.Run("no filter returns every order", func(t *testing.T) {
		all := decodeList(t, h.list(t, "", ""))
		if len(all) != 3 {
			t.Fatalf("len(orders) = %d, want 3", len(all))
		}
	})

	t.Run("status filter", func(t *testing.T) {
		requestedList := decodeList(t, h.list(t, order.StatusRequested, ""))
		if len(requestedList) != 2 || !hasOrder(requestedList, requested.ID) || !hasOrder(requestedList, other.ID) {
			t.Fatalf("requested filter = %+v, want the two requested orders", requestedList)
		}
		if hasOrder(requestedList, confirmed.ID) {
			t.Fatalf("requested filter leaked the confirmed order")
		}
	})

	t.Run("plate search is a case-insensitive substring", func(t *testing.T) {
		byPlate := decodeList(t, h.list(t, "", "ab"))
		if len(byPlate) != 2 || !hasOrder(byPlate, requested.ID) || !hasOrder(byPlate, confirmed.ID) {
			t.Fatalf("plate search ab = %+v, want the two AB orders", byPlate)
		}
		// A fragment in the middle still matches (substring, not prefix).
		mid := decodeList(t, h.list(t, "", "111"))
		if len(mid) != 1 || mid[0].ID != requested.ID {
			t.Fatalf("plate search 111 = %+v, want only the 111 order", mid)
		}
	})

	t.Run("status and plate combine", func(t *testing.T) {
		combined := decodeList(t, h.list(t, order.StatusRequested, "ab"))
		if len(combined) != 1 || combined[0].ID != requested.ID {
			t.Fatalf("combined filter = %+v, want only the requested AB order", combined)
		}
	})
}

func TestListEmptyResultIsAnArray(t *testing.T) {
	h := newHarness(t)

	rec := h.list(t, order.StatusRequested, "NO-SUCH-PLATE")
	if rec.Code != http.StatusOK {
		t.Fatalf("list = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var body listBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("list body is not JSON: %v (body=%q)", err, rec.Body.String())
	}
	if body.Orders == nil {
		t.Fatalf("orders = null, want an empty array (body=%q)", rec.Body.String())
	}
	if len(body.Orders) != 0 {
		t.Fatalf("orders = %+v, want empty", body.Orders)
	}
}

func TestConfirmRequestedOrderMovesToConfirmed(t *testing.T) {
	h := newHarness(t)
	o := h.seedOrder(t, order.StatusRequested, "CNF")

	rec := h.confirm(t, o.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var updated order.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("confirm body is not an order: %v (body=%q)", err, rec.Body.String())
	}
	if updated.Status != order.StatusConfirmed {
		t.Fatalf("status = %q, want confirmed", updated.Status)
	}
	if updated.NextStatus == nil || *updated.NextStatus != order.StatusInProgress {
		t.Fatalf("next_status = %v, want in_progress", updated.NextStatus)
	}

	history, err := h.store.ListHistory(h.ctx, o.ID)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(history) != 1 || history[0].Status != order.StatusConfirmed {
		t.Fatalf("history = %+v, want a single confirmed entry", history)
	}
	if history[0].ChangedAt.IsZero() || history[0].ChangedAt.Location() != time.UTC {
		t.Fatalf("history timestamp = %v, want a UTC time", history[0].ChangedAt)
	}
}

func TestConfirmNonRequestedOrderReturns409(t *testing.T) {
	h := newHarness(t)

	for _, status := range []string{
		order.StatusConfirmed,
		order.StatusInProgress,
		order.StatusDone,
		order.StatusPickedUp,
	} {
		t.Run(status, func(t *testing.T) {
			o := h.seedOrder(t, status, "RJ")
			rec := h.confirm(t, o.ID)
			assertUniformError(t, rec, http.StatusConflict)
		})
	}

	t.Run("confirming twice is rejected", func(t *testing.T) {
		o := h.seedOrder(t, order.StatusRequested, "RJ2")
		if rec := h.confirm(t, o.ID); rec.Code != http.StatusOK {
			t.Fatalf("first confirm = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
		}
		rec := h.confirm(t, o.ID)
		assertUniformError(t, rec, http.StatusConflict)
	})
}

func TestConfirmUnknownOrderReturns404(t *testing.T) {
	h := newHarness(t)

	rec := h.confirm(t, 999999999)
	assertUniformError(t, rec, http.StatusNotFound)
}

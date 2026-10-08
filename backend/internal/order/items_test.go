package order_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"workshop-api/internal/db"
	"workshop-api/internal/order"
	"workshop-api/internal/queue"
	"workshop-api/internal/workshop"
)

// newItemsTestStore opens the real PostgreSQL from DATABASE_URL (AC-25: no
// SQLite, no database double) and applies the migrations.
func newItemsTestStore(t *testing.T) *order.Store {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set; these tests run against a real PostgreSQL")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool, ""); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	publisher := queue.NewPublisher(os.Getenv("VALKEY_URL"), "workshop-invoices")
	return order.NewStore(pool, publisher)
}

// seedItemsOrder inserts one customer, vehicle and order and removes them
// again when the test ends. Only the rows it creates are touched.
func seedItemsOrder(t *testing.T, store *order.Store) int64 {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	var customerID int64
	if err := store.DB.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Items Test", fmt.Sprintf("items-%d@example.com", suffix), "000",
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var vehicleID int64
	if err := store.DB.QueryRow(ctx,
		`INSERT INTO vehicles (plate, brand, model, mileage) VALUES ($1, $2, $3, $4) RETURNING id`,
		fmt.Sprintf("IT-%d", suffix), "Test", "Model", 0,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	var orderID int64
	if err := store.DB.QueryRow(ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, status, description)
		 VALUES ($1, $2, $3, 'confirmed', 'items test') RETURNING id`,
		fmt.Sprintf("ITEMS-%d", suffix), customerID, vehicleID,
	).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	t.Cleanup(func() {
		_, _ = store.DB.Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = store.DB.Exec(ctx, `DELETE FROM customers WHERE id = $1`, customerID)
		_, _ = store.DB.Exec(ctx, `DELETE FROM vehicles WHERE id = $1`, vehicleID)
	})
	return orderID
}

// newItemsTestMux wires the three position routes the router declares, so the
// handlers are exercised through real path matching and httptest.
func newItemsTestMux(store *order.Store) *http.ServeMux {
	handler := workshop.NewItemsHandler(store)
	detail := order.NewItemsHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workshop/orders/{id}", detail.Detail)
	mux.HandleFunc("POST /api/workshop/orders/{id}/items", handler.Add)
	mux.HandleFunc("PUT /api/workshop/orders/{id}/items/{item_id}", handler.Update)
	mux.HandleFunc("DELETE /api/workshop/orders/{id}/items/{item_id}", handler.Delete)
	return mux
}

type itemsEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func itemsDecodeError(t *testing.T, body []byte) itemsEnvelope {
	t.Helper()
	var env itemsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("error body is not the uniform envelope: %v (body=%q)", err, string(body))
	}
	if env.Error.Code == "" || env.Error.Message == "" {
		t.Fatalf("error envelope is missing code/message: %q", string(body))
	}
	return env
}

type itemsResponse struct {
	Item  order.OrderItem `json:"item"`
	Order order.Order     `json:"order"`
}

func itemsDo(t *testing.T, mux http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, reader))
	return rec
}

func itemsDecodeResponse(t *testing.T, rec *httptest.ResponseRecorder) itemsResponse {
	t.Helper()
	var resp itemsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v (body=%q)", err, rec.Body.String())
	}
	return resp
}

func TestItemsAddRecalculatesOrderTotals(t *testing.T) {
	t.Setenv("HOURLY_RATE_CENTS", "10000")
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	// A labor position of 1.5 h at 100.00 EUR/h -> 15000 cents.
	rec := itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID),
		map[string]any{"kind": "labor", "description": "Reparatur", "hours": 1.5})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST labor = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	resp := itemsDecodeResponse(t, rec)
	if resp.Item.TotalCents != 15000 {
		t.Fatalf("labor item total = %d, want 15000", resp.Item.TotalCents)
	}
	if resp.Order.LaborCents != 15000 || resp.Order.PartsCents != 0 {
		t.Fatalf("order after labor = labor %d parts %d, want 15000/0", resp.Order.LaborCents, resp.Order.PartsCents)
	}
	if resp.Order.NetCents != 15000 || resp.Order.VatCents != 2850 || resp.Order.GrossCents != 17850 {
		t.Fatalf("order amounts after labor = net %d vat %d gross %d, want 15000/2850/17850",
			resp.Order.NetCents, resp.Order.VatCents, resp.Order.GrossCents)
	}

	// A part position of 2 x 12.50 EUR -> 2500 cents.
	rec = itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID),
		map[string]any{"kind": "part", "description": "Bremsbelag", "quantity": 2, "unit_price_cents": 1250})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST part = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	resp = itemsDecodeResponse(t, rec)
	if resp.Item.TotalCents != 2500 {
		t.Fatalf("part item total = %d, want 2500", resp.Item.TotalCents)
	}
	if resp.Order.LaborCents != 15000 || resp.Order.PartsCents != 2500 {
		t.Fatalf("order after part = labor %d parts %d, want 15000/2500", resp.Order.LaborCents, resp.Order.PartsCents)
	}
	if resp.Order.NetCents != 17500 || resp.Order.VatCents != 3325 || resp.Order.GrossCents != 20825 {
		t.Fatalf("order amounts after part = net %d vat %d gross %d, want 17500/3325/20825",
			resp.Order.NetCents, resp.Order.VatCents, resp.Order.GrossCents)
	}

	// The amounts are stored on the order, not only returned.
	stored, err := store.GetByID(context.Background(), orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if stored.LaborCents != 15000 || stored.PartsCents != 2500 || stored.GrossCents != 20825 {
		t.Fatalf("stored order amounts = %+v, want labor 15000 parts 2500 gross 20825", stored)
	}
}

func TestItemsUpdateRecalculatesOrderTotals(t *testing.T) {
	t.Setenv("HOURLY_RATE_CENTS", "10000")
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	rec := itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID),
		map[string]any{"kind": "part", "description": "Filter", "quantity": 1, "unit_price_cents": 1000})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST part = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	created := itemsDecodeResponse(t, rec)

	rec = itemsDo(t, mux, http.MethodPut,
		fmt.Sprintf("/api/workshop/orders/%d/items/%d", orderID, created.Item.ID),
		map[string]any{"kind": "part", "description": "Filter", "quantity": 4, "unit_price_cents": 1000})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT item = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	resp := itemsDecodeResponse(t, rec)
	if resp.Item.TotalCents != 4000 {
		t.Fatalf("updated item total = %d, want 4000", resp.Item.TotalCents)
	}
	if resp.Order.PartsCents != 4000 || resp.Order.LaborCents != 0 {
		t.Fatalf("order after update = labor %d parts %d, want 0/4000", resp.Order.LaborCents, resp.Order.PartsCents)
	}
	if resp.Order.NetCents != 4000 || resp.Order.VatCents != 760 || resp.Order.GrossCents != 4760 {
		t.Fatalf("order amounts after update = net %d vat %d gross %d, want 4000/760/4760",
			resp.Order.NetCents, resp.Order.VatCents, resp.Order.GrossCents)
	}
}

func TestItemsDeleteRecalculatesOrderTotals(t *testing.T) {
	t.Setenv("HOURLY_RATE_CENTS", "10000")
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	keep := itemsDecodeResponse(t, itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID),
		map[string]any{"kind": "labor", "description": "Arbeit", "hours": 1}))
	remove := itemsDecodeResponse(t, itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID),
		map[string]any{"kind": "part", "description": "Teil", "quantity": 1, "unit_price_cents": 5000}))

	if remove.Order.PartsCents != 5000 {
		t.Fatalf("order before delete = parts %d, want 5000", remove.Order.PartsCents)
	}

	rec := itemsDo(t, mux, http.MethodDelete,
		fmt.Sprintf("/api/workshop/orders/%d/items/%d", orderID, remove.Item.ID), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE item = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var body struct {
		Order order.Order `json:"order"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("delete response is not JSON: %v", err)
	}
	if body.Order.LaborCents != 10000 || body.Order.PartsCents != 0 {
		t.Fatalf("order after delete = labor %d parts %d, want 10000/0", body.Order.LaborCents, body.Order.PartsCents)
	}
	if body.Order.NetCents != 10000 || body.Order.VatCents != 1900 || body.Order.GrossCents != 11900 {
		t.Fatalf("order amounts after delete = net %d vat %d gross %d, want 10000/1900/11900",
			body.Order.NetCents, body.Order.VatCents, body.Order.GrossCents)
	}

	// The kept position is still there.
	items, err := store.ListItems(context.Background(), orderID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 || items[0].ID != keep.Item.ID {
		t.Fatalf("items after delete = %+v, want only item %d", items, keep.Item.ID)
	}
}

func TestOrderDetailReturnsOrderItemsAndHistory(t *testing.T) {
	t.Setenv("HOURLY_RATE_CENTS", "10000")
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID),
		map[string]any{"kind": "part", "description": "Teil", "quantity": 2, "unit_price_cents": 500})

	rec := itemsDo(t, mux, http.MethodGet, fmt.Sprintf("/api/workshop/orders/%d", orderID), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET detail = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var body struct {
		Order   order.Order           `json:"order"`
		Items   []order.OrderItem     `json:"items"`
		History []order.StatusHistory `json:"history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("detail response is not JSON: %v", err)
	}
	if body.Order.ID != orderID {
		t.Fatalf("detail order id = %d, want %d", body.Order.ID, orderID)
	}
	if len(body.Items) != 1 || body.Items[0].TotalCents != 1000 {
		t.Fatalf("detail items = %+v, want one item of 1000 cents", body.Items)
	}
	if body.Order.PartsCents != 1000 {
		t.Fatalf("detail order parts = %d, want 1000", body.Order.PartsCents)
	}
}

func TestItemsVatRoundsHalfUp(t *testing.T) {
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	// net = 150 cents -> 19 % = 28.5 cents, rounded half up to 29.
	rec := itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID),
		map[string]any{"kind": "part", "description": "Kleinigkeit", "quantity": 1, "unit_price_cents": 150})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST part = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	resp := itemsDecodeResponse(t, rec)
	if resp.Order.NetCents != 150 || resp.Order.VatCents != 29 || resp.Order.GrossCents != 179 {
		t.Fatalf("amounts = net %d vat %d gross %d, want 150/29/179",
			resp.Order.NetCents, resp.Order.VatCents, resp.Order.GrossCents)
	}
}

func TestItemsUnknownOrderOrPositionReturn404(t *testing.T) {
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	const missing = 999999999

	t.Run("unknown order on add", func(t *testing.T) {
		rec := itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", missing),
			map[string]any{"kind": "labor", "description": "x", "hours": 1})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("POST to unknown order = %d, want 404 (body=%q)", rec.Code, rec.Body.String())
		}
		itemsDecodeError(t, rec.Body.Bytes())
	})

	t.Run("unknown position on update", func(t *testing.T) {
		rec := itemsDo(t, mux, http.MethodPut,
			fmt.Sprintf("/api/workshop/orders/%d/items/%d", orderID, missing),
			map[string]any{"kind": "part", "description": "x", "quantity": 1, "unit_price_cents": 100})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("PUT unknown position = %d, want 404 (body=%q)", rec.Code, rec.Body.String())
		}
		itemsDecodeError(t, rec.Body.Bytes())
	})

	t.Run("unknown position on delete", func(t *testing.T) {
		rec := itemsDo(t, mux, http.MethodDelete,
			fmt.Sprintf("/api/workshop/orders/%d/items/%d", orderID, missing), nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("DELETE unknown position = %d, want 404 (body=%q)", rec.Code, rec.Body.String())
		}
		itemsDecodeError(t, rec.Body.Bytes())
	})
}

func TestItemsInvalidBodyReturns400(t *testing.T) {
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	cases := []struct {
		name string
		body any
	}{
		{"labor without hours", map[string]any{"kind": "labor", "description": "x"}},
		{"part without quantity", map[string]any{"kind": "part", "description": "x", "unit_price_cents": 100}},
		{"unknown kind", map[string]any{"kind": "other", "description": "x"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := itemsDo(t, mux, http.MethodPost, fmt.Sprintf("/api/workshop/orders/%d/items", orderID), tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("POST %s = %d, want 400 (body=%q)", tc.name, rec.Code, rec.Body.String())
			}
			itemsDecodeError(t, rec.Body.Bytes())
		})
	}
}

func TestItemsMalformedJSONReturns400(t *testing.T) {
	store := newItemsTestStore(t)
	mux := newItemsTestMux(store)
	orderID := seedItemsOrder(t, store)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/workshop/orders/%d/items", orderID), bytes.NewReader([]byte("{not json"))))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON = %d, want 400 (body=%q)", rec.Code, rec.Body.String())
	}
	itemsDecodeError(t, rec.Body.Bytes())
}

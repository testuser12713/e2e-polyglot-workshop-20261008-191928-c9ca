package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"workshop-api/internal/config"
	"workshop-api/internal/db"
	"workshop-api/internal/httpapi"
	"workshop-api/internal/queue"
)

// newTestRouter builds the real router on the real PostgreSQL from
// DATABASE_URL (AC-25: no SQLite, no database double). The pool is closed when
// the test ends.
func newTestRouter(t *testing.T) http.Handler {
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
	cfg := config.Config{
		AuthSecret: "test-secret",
		CORSOrigin: "http://localhost:5173",
	}
	publisher := queue.NewPublisher(os.Getenv("VALKEY_URL"), "workshop-invoices")
	return httpapi.NewRouter(cfg, pool, publisher)
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeError(t *testing.T, body []byte) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("error body is not the uniform JSON envelope: %v (body=%q)", err, string(body))
	}
	if env.Error.Code == "" || env.Error.Message == "" {
		t.Fatalf("error envelope is missing code/message: %q", string(body))
	}
	return env
}

func TestHealthReturnsStatusOK(t *testing.T) {
	router := newTestRouter(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("health body is not JSON: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("health status = %q, want \"ok\"", body.Status)
	}
}

func TestUnknownPathReturnsUniformNotFound(t *testing.T) {
	router := newTestRouter(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown path = %d, want 404 (body=%q)", rec.Code, rec.Body.String())
	}
	env := decodeError(t, rec.Body.Bytes())
	if !strings.Contains(strings.ToLower(env.Error.Code), "not_found") {
		t.Fatalf("unknown path code = %q, want not_found", env.Error.Code)
	}
}

// TestRoutesAreWiredWithUniformErrors asserts that the declared routes exist
// (they must not answer 404) and that any error they currently produce uses
// the uniform envelope. It deliberately does not pin the temporary 501 a stub
// returns: that answer changes when the owning ticket lands.
func TestRoutesAreWiredWithUniformErrors(t *testing.T) {
	router := newTestRouter(t)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"customers", http.MethodPost, "/api/customers"},
		{"vehicles", http.MethodPost, "/api/vehicles"},
		{"appointments", http.MethodPost, "/api/appointments"},
		{"order-status", http.MethodGet, "/api/orders/status"},
		{"auth-login", http.MethodPost, "/api/auth/login"},
		{"workshop-orders", http.MethodGet, "/api/workshop/orders"},
		{"workshop-dashboard", http.MethodGet, "/api/workshop/dashboard"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))

			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s %s is not registered (404)", tc.method, tc.path)
			}
			if rec.Code < 400 {
				// A route that already succeeds is fine; nothing to check.
				return
			}
			decodeError(t, rec.Body.Bytes())
		})
	}
}

func TestMigrationsCreateTheSchema(t *testing.T) {
	router := newTestRouter(t)
	_ = router // ensure the pool/migrations path ran

	dsn := os.Getenv("DATABASE_URL")
	ctx := context.Background()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	tables := []string{
		"customers", "vehicles", "orders", "order_items",
		"order_status_history", "employees", "invoices", "invoice_items", "outbox",
	}
	for _, table := range tables {
		var regclass *string
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::text", "public."+table).Scan(&regclass); err != nil {
			t.Fatalf("look up table %s: %v", table, err)
		}
		if regclass == nil {
			t.Errorf("table %s was not created by the migrations", table)
		}
	}
}

package customer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/customer"
	"workshop-api/internal/db"
)

// unique returns a value that is distinct on every call so tests never collide
// with rows another test (or another parallel package) wrote.
func unique(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// openTestDB opens the real PostgreSQL from DATABASE_URL and applies the
// migrations. There is no SQLite fallback and no database double.
func openTestDB(t *testing.T) *pgxpool.Pool {
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
	return pool
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

func postCustomer(t *testing.T, handler *customer.Handler, body string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func TestCreateCustomerReturnsStoredRecordWithStableID(t *testing.T) {
	pool := openTestDB(t)
	handler := customer.NewHandler(customer.NewStore(pool))

	email := unique("customer") + "@example.com"
	body := fmt.Sprintf(`{"name":"Ada Lovelace","email":%q,"phone":"+49 30 123456"}`, email)

	status, raw := postCustomer(t, handler, body)
	if status != 201 {
		t.Fatalf("POST /api/customers = %d, want 201 (body=%q)", status, string(raw))
	}

	var got customer.Customer
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("customer body is not JSON: %v (body=%q)", err, string(raw))
	}
	if got.ID <= 0 {
		t.Fatalf("customer id = %d, want a stable positive id", got.ID)
	}
	if got.Name != "Ada Lovelace" || got.Email != email || got.Phone != "+49 30 123456" {
		t.Fatalf("stored customer = %+v, want the posted values", got)
	}

	// A second post of the same customer returns the same stable id.
	status, raw = postCustomer(t, handler, body)
	if status != 201 {
		t.Fatalf("second POST /api/customers = %d, want 201 (body=%q)", status, string(raw))
	}
	var again customer.Customer
	if err := json.Unmarshal(raw, &again); err != nil {
		t.Fatalf("customer body is not JSON: %v", err)
	}
	if again.ID != got.ID {
		t.Fatalf("second customer id = %d, want the stable id %d", again.ID, got.ID)
	}
}

func TestCreateCustomerRejectsMissingOrMalformedValues(t *testing.T) {
	pool := openTestDB(t)
	handler := customer.NewHandler(customer.NewStore(pool))

	cases := []struct {
		name string
		body string
	}{
		{"missing name", `{"email":"x@example.com","phone":"123"}`},
		{"missing email", `{"name":"X","phone":"123"}`},
		{"missing phone", `{"name":"X","email":"x@example.com"}`},
		{"malformed email", `{"name":"X","email":"not-an-email","phone":"123"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := postCustomer(t, handler, tc.body)
			if status != 400 {
				t.Fatalf("POST /api/customers = %d, want 400 (body=%q)", status, string(raw))
			}
			decodeError(t, raw)
		})
	}
}

func TestCreateCustomerRejectsInvalidJSON(t *testing.T) {
	pool := openTestDB(t)
	handler := customer.NewHandler(customer.NewStore(pool))

	status, raw := postCustomer(t, handler, `{not json`)
	if status != 400 {
		t.Fatalf("POST /api/customers with bad JSON = %d, want 400 (body=%q)", status, string(raw))
	}
	decodeError(t, raw)
}

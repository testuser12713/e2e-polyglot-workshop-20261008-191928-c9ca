package order

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"workshop-api/internal/db"
)

// newIntakeTestStore opens the real PostgreSQL from DATABASE_URL, applies the
// migrations and returns an order store. No SQLite, no database double (AC-25).
func newIntakeTestStore(t *testing.T) *Store {
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
	return NewStore(pool, nil)
}

// uniqueSuffix keeps every test's e-mail and plate distinct, so a test never
// collides with another test or with rows from an earlier local run.
func uniqueSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func newTestPlate(suffix string) string {
	// Keep it short and distinctive; plates have no length constraint but the
	// status page shows them uppercased.
	return "T" + suffix[len(suffix)-8:]
}

type intakeResponse struct {
	OrderID     int64  `json:"order_id"`
	OrderNumber string `json:"order_number"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func postAppointment(t *testing.T, h *IntakeHandler, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/api/appointments", bytes.NewReader(raw)))
	return rec
}

func validAppointment(email, plate string) map[string]any {
	return map[string]any{
		"name":           "Erika Mustermann",
		"email":          email,
		"phone":          "0170 1234567",
		"plate":          plate,
		"brand":          "VW",
		"model":          "Golf",
		"mileage":        128450,
		"preferred_date": "2026-11-05",
		"description":    "Bremsen quietschen beim Anhalten.",
	}
}

func TestIntakeCreatesRequestedOrder(t *testing.T) {
	store := newIntakeTestStore(t)
	handler := NewIntakeHandler(store)

	suffix := uniqueSuffix()
	email := "kunde-" + suffix + "@example.com"
	plate := newTestPlate(suffix)

	rec := postAppointment(t, handler, validAppointment(email, plate))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/appointments = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}

	var resp intakeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v (body=%q)", err, rec.Body.String())
	}
	if resp.OrderID == 0 {
		t.Fatalf("order_id is missing in %q", rec.Body.String())
	}
	if resp.OrderNumber == "" {
		t.Fatalf("order_number is missing in %q", rec.Body.String())
	}
	if !strings.HasPrefix(resp.OrderNumber, "AU-") {
		t.Fatalf("order_number = %q, want the AU-<year>-<seq> form", resp.OrderNumber)
	}

	loaded, err := store.GetByID(context.Background(), resp.OrderID)
	if err != nil {
		t.Fatalf("load created order: %v", err)
	}
	if loaded.Status != StatusRequested {
		t.Errorf("order status = %q, want %q", loaded.Status, StatusRequested)
	}
	if loaded.OrderNumber != resp.OrderNumber {
		t.Errorf("stored order_number = %q, response said %q", loaded.OrderNumber, resp.OrderNumber)
	}
	if loaded.Customer.Email != email {
		t.Errorf("order customer email = %q, want %q", loaded.Customer.Email, email)
	}
	if loaded.Customer.Name != "Erika Mustermann" {
		t.Errorf("order customer name = %q", loaded.Customer.Name)
	}
	if loaded.Vehicle.Plate != plate {
		t.Errorf("order vehicle plate = %q, want %q", loaded.Vehicle.Plate, plate)
	}
	if loaded.Vehicle.Mileage != 128450 {
		t.Errorf("order vehicle mileage = %d, want 128450", loaded.Vehicle.Mileage)
	}
	if loaded.PreferredDate == nil || *loaded.PreferredDate != "2026-11-05" {
		t.Errorf("preferred_date = %v, want 2026-11-05", loaded.PreferredDate)
	}
	if loaded.Description != "Bremsen quietschen beim Anhalten." {
		t.Errorf("description = %q", loaded.Description)
	}

	var historyCount int
	if err := store.DB.QueryRow(context.Background(),
		`SELECT count(*) FROM order_status_history WHERE order_id = $1 AND status = $2`,
		resp.OrderID, StatusRequested,
	).Scan(&historyCount); err != nil {
		t.Fatalf("count status history: %v", err)
	}
	if historyCount != 1 {
		t.Errorf("status history rows for requested = %d, want 1", historyCount)
	}
}

func TestIntakeReusesExistingVehiclePlate(t *testing.T) {
	store := newIntakeTestStore(t)
	handler := NewIntakeHandler(store)

	suffix := uniqueSuffix()
	plate := newTestPlate(suffix)

	first := postAppointment(t, handler, validAppointment("first-"+suffix+"@example.com", plate))
	if first.Code != http.StatusCreated {
		t.Fatalf("first appointment = %d, want 201 (body=%q)", first.Code, first.Body.String())
	}
	var firstResp intakeResponse
	if err := json.Unmarshal(first.Body.Bytes(), &firstResp); err != nil {
		t.Fatalf("first response is not JSON: %v", err)
	}

	// Second appointment for the same plate but a different customer must
	// succeed and reuse the existing vehicle (AC-03: "if not already present").
	second := postAppointment(t, handler, validAppointment("second-"+suffix+"@example.com", plate))
	if second.Code != http.StatusCreated {
		t.Fatalf("second appointment = %d, want 201 (body=%q)", second.Code, second.Body.String())
	}
	var secondResp intakeResponse
	if err := json.Unmarshal(second.Body.Bytes(), &secondResp); err != nil {
		t.Fatalf("second response is not JSON: %v", err)
	}

	firstOrder, err := store.GetByID(context.Background(), firstResp.OrderID)
	if err != nil {
		t.Fatalf("load first order: %v", err)
	}
	secondOrder, err := store.GetByID(context.Background(), secondResp.OrderID)
	if err != nil {
		t.Fatalf("load second order: %v", err)
	}
	if firstOrder.Vehicle.ID != secondOrder.Vehicle.ID {
		t.Errorf("vehicle ids differ (%d vs %d); the known plate was not reused",
			firstOrder.Vehicle.ID, secondOrder.Vehicle.ID)
	}
	if firstOrder.Customer.ID == secondOrder.Customer.ID {
		t.Errorf("customer ids are equal, but the two requests used different e-mails")
	}
}

func TestIntakeRejectsMissingFieldWithUniformError(t *testing.T) {
	store := newIntakeTestStore(t)
	handler := NewIntakeHandler(store)

	body := validAppointment("missing-"+uniqueSuffix()+"@example.com", newTestPlate(uniqueSuffix()))
	delete(body, "name")

	rec := postAppointment(t, handler, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing name = %d, want 400 (body=%q)", rec.Code, rec.Body.String())
	}
	assertUniformError(t, rec.Body.Bytes())
}

func TestIntakeRejectsMalformedFields(t *testing.T) {
	store := newIntakeTestStore(t)
	handler := NewIntakeHandler(store)

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"invalid email", func(b map[string]any) { b["email"] = "not-an-email" }},
		{"invalid date", func(b map[string]any) { b["preferred_date"] = "05.11.2026" }},
		{"negative mileage", func(b map[string]any) { b["mileage"] = -1 }},
		{"empty plate", func(b map[string]any) { b["plate"] = " " }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := validAppointment("malformed-"+uniqueSuffix()+"@example.com", newTestPlate(uniqueSuffix()))
			tc.mutate(body)

			rec := postAppointment(t, handler, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%s = %d, want 400 (body=%q)", tc.name, rec.Code, rec.Body.String())
			}
			assertUniformError(t, rec.Body.Bytes())
		})
	}
}

func TestIntakeRejectsMalformedJSON(t *testing.T) {
	store := newIntakeTestStore(t)
	handler := NewIntakeHandler(store)

	rec := httptest.NewRecorder()
	handler.Create(rec, httptest.NewRequest(http.MethodPost, "/api/appointments",
		strings.NewReader("{not json")))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON = %d, want 400 (body=%q)", rec.Code, rec.Body.String())
	}
	assertUniformError(t, rec.Body.Bytes())
}

func TestGenerateOrderNumber(t *testing.T) {
	got := GenerateOrderNumber(time.Date(2026, time.October, 8, 12, 0, 0, 0, time.UTC), 42)
	if got != "AU-2026-0042" {
		t.Fatalf("GenerateOrderNumber = %q, want AU-2026-0042", got)
	}
}

func TestIntakeRequestValidate(t *testing.T) {
	valid := IntakeRequest{
		Name: "Erika Mustermann", Email: "erika@example.com", Phone: "0170 1",
		Plate: "T-123", Brand: "VW", Model: "Golf",
		Mileage: intPtr(0), PreferredDate: "2026-11-05", Description: "Klappern",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	missingMileage := valid
	missingMileage.Mileage = nil
	if err := missingMileage.Validate(); err == nil {
		t.Errorf("missing mileage was accepted")
	}
}

func intPtr(v int) *int { return &v }

func assertUniformError(t *testing.T, body []byte) {
	t.Helper()
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("error body is not the uniform JSON envelope: %v (body=%q)", err, string(body))
	}
	if env.Error.Code == "" || env.Error.Message == "" {
		t.Fatalf("error envelope is missing code/message: %q", string(body))
	}
}

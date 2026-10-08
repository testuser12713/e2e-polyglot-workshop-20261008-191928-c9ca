package vehicle_test

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

	"workshop-api/internal/db"
	"workshop-api/internal/vehicle"
)

// uniquePlate returns a distinct plate on every call so tests never collide
// with rows another test (or another parallel package) wrote.
func uniquePlate() string {
	return fmt.Sprintf("B-XX-%d", time.Now().UnixNano())
}

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

func postVehicle(t *testing.T, handler *vehicle.Handler, body string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/vehicles", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func TestCreateVehicleReturnsStoredRecord(t *testing.T) {
	pool := openTestDB(t)
	handler := vehicle.NewHandler(vehicle.NewStore(pool))

	plate := uniquePlate()
	body := fmt.Sprintf(`{"plate":%q,"brand":"VW","model":"Golf","mileage":123456}`, plate)

	status, raw := postVehicle(t, handler, body)
	if status != 201 {
		t.Fatalf("POST /api/vehicles = %d, want 201 (body=%q)", status, string(raw))
	}

	var got vehicle.Vehicle
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("vehicle body is not JSON: %v (body=%q)", err, string(raw))
	}
	if got.ID <= 0 {
		t.Fatalf("vehicle id = %d, want a positive id", got.ID)
	}
	if got.Plate != plate || got.Brand != "VW" || got.Model != "Golf" || got.Mileage != 123456 {
		t.Fatalf("stored vehicle = %+v, want the posted values", got)
	}
}

func TestCreateVehicleWithTakenPlateReturns409(t *testing.T) {
	pool := openTestDB(t)
	handler := vehicle.NewHandler(vehicle.NewStore(pool))

	plate := uniquePlate()
	body := fmt.Sprintf(`{"plate":%q,"brand":"VW","model":"Golf","mileage":1000}`, plate)

	status, raw := postVehicle(t, handler, body)
	if status != 201 {
		t.Fatalf("first POST /api/vehicles = %d, want 201 (body=%q)", status, string(raw))
	}

	other := fmt.Sprintf(`{"plate":%q,"brand":"Audi","model":"A3","mileage":2000}`, plate)
	status, raw = postVehicle(t, handler, other)
	if status != 409 {
		t.Fatalf("second POST /api/vehicles with taken plate = %d, want 409 (body=%q)", status, string(raw))
	}
	decodeError(t, raw)
}

func TestCreateVehicleRejectsMissingOrMalformedValues(t *testing.T) {
	pool := openTestDB(t)
	handler := vehicle.NewHandler(vehicle.NewStore(pool))

	cases := []struct {
		name string
		body string
	}{
		{"missing plate", `{"brand":"VW","model":"Golf","mileage":1}`},
		{"missing brand", `{"plate":"B-XX-1","model":"Golf","mileage":1}`},
		{"missing model", `{"plate":"B-XX-2","brand":"VW","mileage":1}`},
		{"missing mileage", `{"plate":"B-XX-3","brand":"VW","model":"Golf"}`},
		{"negative mileage", `{"plate":"B-XX-4","brand":"VW","model":"Golf","mileage":-1}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := postVehicle(t, handler, tc.body)
			if status != 400 {
				t.Fatalf("POST /api/vehicles = %d, want 400 (body=%q)", status, string(raw))
			}
			decodeError(t, raw)
		})
	}
}

func TestCreateVehicleRejectsInvalidJSON(t *testing.T) {
	pool := openTestDB(t)
	handler := vehicle.NewHandler(vehicle.NewStore(pool))

	status, raw := postVehicle(t, handler, `{not json`)
	if status != 400 {
		t.Fatalf("POST /api/vehicles with bad JSON = %d, want 400 (body=%q)", status, string(raw))
	}
	decodeError(t, raw)
}

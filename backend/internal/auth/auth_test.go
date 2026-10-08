package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"workshop-api/internal/auth"
	"workshop-api/internal/config"
	"workshop-api/internal/db"
	"workshop-api/internal/httpapi"
	"workshop-api/internal/queue"
)

// testPool connects to the real PostgreSQL from DATABASE_URL, applies the
// product migrations so the employees table exists, and hands back a pool.
// The test is skipped (not failed) when no database is available.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set; skipping PostgreSQL-backed test")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := ensureSchema(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// ensureSchema applies the product migrations. Go runs test packages in
// parallel, so another package may be creating the same tables at the same
// time and PostgreSQL's CREATE ... IF NOT EXISTS can briefly raise a duplicate
// catalog error; the migrations are idempotent, so a short retry is safe.
func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		if lastErr = db.RunMigrations(ctx, pool, ""); lastErr == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return lastErr
}

// uniqueEmail keeps parallel test runs from colliding on the employees table.
func uniqueEmail() string {
	return fmt.Sprintf("employee-%d@example.com", time.Now().UnixNano())
}

// deleteEmployee removes only the row this test created.
func deleteEmployee(t *testing.T, pool *pgxpool.Pool, email string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM employees WHERE email = $1`, email)
	})
}

func TestSeedStoresOnlyHashAndLoginReturnsToken(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := auth.NewStore(pool)
	email := uniqueEmail()
	password := "s3cret-Passw0rd!"
	deleteEmployee(t, pool, email)

	if err := auth.Seed(ctx, store, email, password); err != nil {
		t.Fatalf("seed: %v", err)
	}

	employee, err := store.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("find seeded employee: %v", err)
	}
	if employee.PasswordHash == password || employee.PasswordHash == "" {
		t.Fatalf("password must be stored only as hash, got %q", employee.PasswordHash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(password)); err != nil {
		t.Fatalf("stored hash does not verify the password: %v", err)
	}

	issuer := auth.NewTokenIssuer("test-secret")
	handler := auth.NewHandler(store, issuer)

	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token    string        `json:"token"`
		Employee auth.Employee `json:"employee"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("login returned an empty token")
	}
	if resp.Employee.Email != email || resp.Employee.ID != employee.ID {
		t.Fatalf("login employee = %+v, want id=%d email=%s", resp.Employee, employee.ID, email)
	}
	if resp.Employee.Name == "" {
		t.Fatal("login employee has no name")
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("login response leaked password material: %s", rec.Body.String())
	}

	parsed, err := issuer.Parse(resp.Token)
	if err != nil {
		t.Fatalf("parse issued token: %v", err)
	}
	if parsed.ID != employee.ID {
		t.Fatalf("parsed employee id = %d, want %d", parsed.ID, employee.ID)
	}
}

func TestLoginWrongPasswordReturnsUniform401(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := auth.NewStore(pool)
	email := uniqueEmail()
	deleteEmployee(t, pool, email)
	if err := auth.Seed(ctx, store, email, "correct-horse"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := auth.NewHandler(store, auth.NewTokenIssuer("test-secret"))
	body := fmt.Sprintf(`{"email":%q,"password":"wrong-password"}`, email)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	assertUniformError(t, rec.Body.Bytes())
}

func TestLoginUnknownEmailReturnsUniform401(t *testing.T) {
	pool := testPool(t)
	store := auth.NewStore(pool)
	handler := auth.NewHandler(store, auth.NewTokenIssuer("test-secret"))

	body := fmt.Sprintf(`{"email":%q,"password":"whatever"}`, uniqueEmail())
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	assertUniformError(t, rec.Body.Bytes())
}

// TestLoginEmptyPasswordReturnsUniform401 pins the boundary the browser surface
// must not hit: an empty password (or e-mail) is a wrong credential and answers
// the same uniform 401 as any other bad login, never a 500.
func TestLoginEmptyPasswordReturnsUniform401(t *testing.T) {
	pool := testPool(t)
	store := auth.NewStore(pool)
	handler := auth.NewHandler(store, auth.NewTokenIssuer("test-secret"))

	body := fmt.Sprintf(`{"email":%q,"password":""}`, uniqueEmail())
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	assertUniformError(t, rec.Body.Bytes())
}

func TestSeedSkipsWhenPasswordUnset(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := auth.NewStore(pool)
	email := uniqueEmail()

	if err := auth.Seed(ctx, store, email, ""); err != nil {
		t.Fatalf("seed without password must not fail: %v", err)
	}
	if _, err := store.FindByEmail(ctx, email); !errors.Is(err, auth.ErrEmployeeNotFound) {
		t.Fatalf("no employee must be created without a password, got err=%v", err)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := auth.NewStore(pool)
	email := uniqueEmail()
	deleteEmployee(t, pool, email)

	if err := auth.Seed(ctx, store, email, "first-password"); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := auth.Seed(ctx, store, email, "second-password"); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM employees WHERE email = $1`, email).Scan(&count); err != nil {
		t.Fatalf("count employees: %v", err)
	}
	if count != 1 {
		t.Fatalf("employee count = %d, want 1", count)
	}

	// The latest configured password wins so a restarted run stays usable.
	employee, err := store.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte("second-password")); err != nil {
		t.Fatalf("re-seed did not refresh the password hash: %v", err)
	}
}

func TestTokenRoundTripAndRejectsTampering(t *testing.T) {
	issuer := auth.NewTokenIssuer("unit-test-secret")
	employee := auth.Employee{ID: 7}

	token, err := issuer.Issue(employee)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	parsed, err := issuer.Parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.ID != employee.ID {
		t.Fatalf("parsed id = %d, want %d", parsed.ID, employee.ID)
	}

	for _, bad := range []string{"", "not-a-token", token + "x", "a.b"} {
		if _, err := issuer.Parse(bad); !errors.Is(err, auth.ErrInvalidToken) {
			t.Fatalf("Parse(%q) error = %v, want ErrInvalidToken", bad, err)
		}
	}

	other := auth.NewTokenIssuer("different-secret")
	if _, err := other.Parse(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("token from another secret must not verify, got %v", err)
	}
}

func TestRequireAuthMissingTokenReturnsUniform401(t *testing.T) {
	t.Setenv("AUTH_SECRET", "unit-test-secret")

	called := false
	guarded := httpapi.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("downstream handler must not run without a token")
	}
	assertUniformError(t, rec.Body.Bytes())
}

func TestRequireAuthInvalidTokenReturnsUniform401(t *testing.T) {
	t.Setenv("AUTH_SECRET", "unit-test-secret")

	guarded := httpapi.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("downstream handler must not run with an invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer definitely.not.valid")
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	assertUniformError(t, rec.Body.Bytes())
}

func TestRequireAuthValidTokenPassesEmployee(t *testing.T) {
	const secret = "unit-test-secret"
	t.Setenv("AUTH_SECRET", secret)

	issuer := auth.NewTokenIssuer(secret)
	token, err := issuer.Issue(auth.Employee{ID: 42})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	var gotID int64
	var gotOK bool
	guarded := httpapi.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		employee, ok := auth.EmployeeFromContext(r.Context())
		gotID = employee.ID
		gotOK = ok
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if !gotOK || gotID != 42 {
		t.Fatalf("context employee = (%d, ok=%v), want (42, ok=true)", gotID, gotOK)
	}
}

// TestLoginEndToEndThroughRouter proves the whole wiring: the seeded employee
// signs in over the real router, the issued token is accepted by RequireAuth on
// a workshop route, and the same route without a token answers 401.
func TestLoginEndToEndThroughRouter(t *testing.T) {
	const secret = "e2e-test-secret"
	t.Setenv("AUTH_SECRET", secret)

	pool := testPool(t)
	ctx := context.Background()
	store := auth.NewStore(pool)
	email := uniqueEmail()
	password := "e2e-Passw0rd!"
	deleteEmployee(t, pool, email)
	if err := auth.Seed(ctx, store, email, password); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg := config.Config{AuthSecret: secret, CORSOrigin: "http://localhost:5173"}
	router := httpapi.NewRouter(cfg, pool, queue.NewPublisher("", "workshop-invoices"))

	loginBody := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginBody)))
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login = %d, want 200; body=%s", loginRec.Code, loginRec.Body.String())
	}
	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &login); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if login.Token == "" {
		t.Fatal("login returned an empty token")
	}

	withoutToken := httptest.NewRecorder()
	router.ServeHTTP(withoutToken, httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil))
	if withoutToken.Code != http.StatusUnauthorized {
		t.Fatalf("workshop without token = %d, want 401; body=%s", withoutToken.Code, withoutToken.Body.String())
	}
	assertUniformError(t, withoutToken.Body.Bytes())

	withToken := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	router.ServeHTTP(withToken, req)
	if withToken.Code == http.StatusUnauthorized {
		t.Fatalf("workshop with a valid token must not answer 401; body=%s", withToken.Body.String())
	}
}

// TestLoginWithConfiguredCredentialsReturnsToken reproduces the real startup
// path (config.Load -> Seed) and signs in with exactly the configured
// credentials against PostgreSQL. This is the flow the running product uses: a
// person signs in at /werkstatt/login with the account the seed wrote from the
// environment (RUN.json supplies the documented demo account as a `dev` value).
func TestLoginWithConfiguredCredentialsReturnsToken(t *testing.T) {
	const secret = "configured-login-secret"
	const configuredEmail = "meister@example.com"
	const configuredPassword = "changeme"
	t.Setenv("AUTH_SECRET", secret)
	t.Setenv("EMPLOYEE_EMAIL", configuredEmail)
	t.Setenv("EMPLOYEE_PASSWORD", configuredPassword)
	if os.Getenv("VALKEY_URL") == "" {
		t.Setenv("VALKEY_URL", "redis://127.0.0.1:6379/0")
	}

	pool := testPool(t)
	ctx := context.Background()
	deleteEmployee(t, pool, configuredEmail)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.EmployeeEmail != configuredEmail || cfg.EmployeePassword != configuredPassword {
		t.Fatalf("config did not carry the configured credentials: %q / %q",
			cfg.EmployeeEmail, cfg.EmployeePassword)
	}

	if err := auth.Seed(ctx, auth.NewStore(pool), cfg.EmployeeEmail, cfg.EmployeePassword); err != nil {
		t.Fatalf("seed: %v", err)
	}

	router := httpapi.NewRouter(config.Config{AuthSecret: secret, CORSOrigin: "http://localhost:5173"},
		pool, queue.NewPublisher("", "workshop-invoices"))
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, cfg.EmployeeEmail, cfg.EmployeePassword)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("login with configured credentials = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("login with configured credentials returned an empty token")
	}
}

// TestSeedRefreshesAccountStoredWithDifferentEmailCase is the regression for the
// seed mismatch: FindByEmail is case-insensitive while the old
// `ON CONFLICT (email)` refresh was not, so a row left behind with different
// casing made the seed insert a second account for the same person. A login then
// read one of the two arbitrarily and could reject the configured password.
func TestSeedRefreshesAccountStoredWithDifferentEmailCase(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := auth.NewStore(pool)
	email := uniqueEmail()
	password := "configured-Passw0rd!"
	deleteEmployee(t, pool, email)

	// A stale account for the same address, stored with different casing and an
	// outdated password hash (as an earlier run or spelling could leave it).
	staleHash, err := bcrypt.GenerateFromPassword([]byte("stale-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash stale password: %v", err)
	}
	mixedCase := strings.ToUpper(email[:1]) + email[1:]
	if _, err := pool.Exec(ctx,
		`INSERT INTO employees (name, email, password_hash) VALUES ($1, $2, $3)`,
		"stale", mixedCase, string(staleHash)); err != nil {
		t.Fatalf("insert stale employee: %v", err)
	}

	if err := auth.Seed(ctx, store, email, password); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM employees WHERE lower(email) = lower($1)`, email).Scan(&count); err != nil {
		t.Fatalf("count employees: %v", err)
	}
	if count != 1 {
		t.Fatalf("employee count for %s = %d, want 1", email, count)
	}

	handler := auth.NewHandler(store, auth.NewTokenIssuer("case-seed-secret"))
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	rec := httptest.NewRecorder()
	handler.Login(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login after refresh = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

// assertUniformError checks the shared error envelope {"error":{"code","message"}}.
func assertUniformError(t *testing.T, body []byte) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v; body=%s", err, body)
	}
	if envelope.Error.Code == "" || envelope.Error.Message == "" {
		t.Fatalf("error body is not the uniform envelope: %s", body)
	}
}

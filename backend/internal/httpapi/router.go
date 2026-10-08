package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/auth"
	"workshop-api/internal/config"
	"workshop-api/internal/customer"
	"workshop-api/internal/dashboard"
	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
	"workshop-api/internal/queue"
	"workshop-api/internal/vehicle"
	"workshop-api/internal/workshop"
)

// NewRouter builds the complete HTTP surface: it constructs the stores from the
// pool and the publisher, passes them into every handler constructor, wraps the
// /api/workshop subtree with RequireAuth and applies the configured CORS
// origin.
func NewRouter(cfg config.Config, pool *pgxpool.Pool, publisher queue.Publisher) http.Handler {
	mux := http.NewServeMux()

	customerStore := customer.NewStore(pool)
	vehicleStore := vehicle.NewStore(pool)
	orderStore := order.NewStore(pool, publisher)
	authStore := auth.NewStore(pool)
	issuer := auth.NewTokenIssuer(cfg.AuthSecret)

	customerHandler := customer.NewHandler(customerStore)
	vehicleHandler := vehicle.NewHandler(vehicleStore)
	intakeHandler := order.NewIntakeHandler(orderStore)
	orderStatusHandler := order.NewStatusHandler(orderStore)
	orderDetailHandler := order.NewItemsHandler(orderStore)
	authHandler := auth.NewHandler(authStore, issuer)

	workshopListHandler := workshop.NewListHandler(orderStore)
	workshopConfirmHandler := workshop.NewConfirmHandler(orderStore)
	workshopStatusHandler := workshop.NewStatusHandler(orderStore)
	workshopItemsHandler := workshop.NewItemsHandler(orderStore)
	dashboardHandler := dashboard.NewHandler(orderStore)

	// Public endpoints.
	mux.HandleFunc("GET /api/health", healthHandler(pool))
	mux.HandleFunc("POST /api/customers", customerHandler.Create)
	mux.HandleFunc("POST /api/vehicles", vehicleHandler.Create)
	mux.HandleFunc("POST /api/appointments", intakeHandler.Create)
	mux.HandleFunc("GET /api/orders/status", orderStatusHandler.Get)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Workshop endpoints, all behind the bearer middleware.
	mux.Handle("GET /api/workshop/orders", RequireAuth(http.HandlerFunc(workshopListHandler.List)))
	mux.Handle("POST /api/workshop/orders/{id}/confirm", RequireAuth(http.HandlerFunc(workshopConfirmHandler.Confirm)))
	mux.Handle("POST /api/workshop/orders/{id}/status", RequireAuth(http.HandlerFunc(workshopStatusHandler.Set)))
	mux.Handle("GET /api/workshop/orders/{id}", RequireAuth(http.HandlerFunc(orderDetailHandler.Detail)))
	mux.Handle("POST /api/workshop/orders/{id}/items", RequireAuth(http.HandlerFunc(workshopItemsHandler.Add)))
	mux.Handle("PUT /api/workshop/orders/{id}/items/{item_id}", RequireAuth(http.HandlerFunc(workshopItemsHandler.Update)))
	mux.Handle("DELETE /api/workshop/orders/{id}/items/{item_id}", RequireAuth(http.HandlerFunc(workshopItemsHandler.Delete)))
	mux.Handle("GET /api/workshop/dashboard", RequireAuth(http.HandlerFunc(dashboardHandler.Get)))

	// Uniform 404 for everything else.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.NotFound(w)
	})

	return withRecover(withCORS(cfg.CORSOrigin, mux))
}

// healthHandler answers 200 with the status object and proves the database the
// real routes use is reachable.
func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, "database_unavailable", "database is not reachable")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// withCORS allows the configured frontend origin and answers preflight
// requests. The origin comes from configuration, never a hardcoded port.
func withCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestOrigin := r.Header.Get("Origin"); requestOrigin != "" && requestOrigin == origin {
			w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withRecover turns a panic into the uniform 500 body INSIDE the CORS layer, so
// the browser still receives the CORS headers on a server error.
func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic on %s %s: %v", r.Method, r.URL.Path, rec)
				httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

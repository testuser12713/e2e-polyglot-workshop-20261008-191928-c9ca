// Package dashboard holds the workshop KPI handler.
package dashboard

import (
	"context"
	"log"
	"net/http"
	"time"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// Handler serves the workshop dashboard KPIs.
type Handler struct {
	store *Store
}

// NewHandler builds the dashboard handler on the shared order store.
func NewHandler(orderStore *order.Store) *Handler {
	return &Handler{store: NewStore(orderStore.DB)}
}

// Get handles GET /api/workshop/dashboard and answers with the open order
// count, the orders finished today (UTC) and the current UTC month's revenue in
// cents.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	summary, err := h.store.Summary(ctx)
	if err != nil {
		log.Printf("dashboard summary: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load the dashboard")
		return
	}
	httpx.JSON(w, http.StatusOK, summary)
}

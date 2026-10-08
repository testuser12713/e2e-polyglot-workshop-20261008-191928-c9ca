// Package dashboard holds the workshop KPI handler. Stub for this ticket.
package dashboard

import (
	"net/http"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// Handler serves the workshop dashboard KPIs.
type Handler struct {
	store *order.Store
}

// NewHandler builds the dashboard handler on the shared order store.
func NewHandler(store *order.Store) *Handler {
	return &Handler{store: store}
}

// Get handles GET /api/workshop/dashboard (implemented by a later ticket).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "GET /api/workshop/dashboard")
}

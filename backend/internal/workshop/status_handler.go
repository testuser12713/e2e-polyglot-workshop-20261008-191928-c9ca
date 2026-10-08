package workshop

import (
	"net/http"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// StatusHandler serves the workshop status transition endpoint. Stub.
type StatusHandler struct {
	store *order.Store
}

// NewStatusHandler builds the workshop status handler on the shared store.
func NewStatusHandler(store *order.Store) *StatusHandler {
	return &StatusHandler{store: store}
}

// Set handles POST /api/workshop/orders/{id}/status (later ticket).
func (h *StatusHandler) Set(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "POST /api/workshop/orders/{id}/status")
}

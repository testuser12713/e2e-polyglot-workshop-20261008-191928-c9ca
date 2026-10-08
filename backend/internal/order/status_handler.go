package order

import (
	"net/http"

	"workshop-api/internal/httpx"
)

// StatusHandler serves the public order-status lookup. Stub for this ticket.
type StatusHandler struct {
	store *Store
}

// NewStatusHandler builds the public status handler on the shared order store.
func NewStatusHandler(store *Store) *StatusHandler {
	return &StatusHandler{store: store}
}

// Get handles GET /api/orders/status (implemented by a later ticket).
func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "GET /api/orders/status")
}

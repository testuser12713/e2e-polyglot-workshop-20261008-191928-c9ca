package order

import (
	"net/http"

	"workshop-api/internal/httpx"
)

// IntakeHandler serves the public appointment request. Stub for this ticket.
type IntakeHandler struct {
	store *Store
}

// NewIntakeHandler builds the intake handler on the shared order store.
func NewIntakeHandler(store *Store) *IntakeHandler {
	return &IntakeHandler{store: store}
}

// Create handles POST /api/appointments (implemented by a later ticket).
func (h *IntakeHandler) Create(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "POST /api/appointments")
}

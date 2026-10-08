package vehicle

import (
	"net/http"

	"workshop-api/internal/httpx"
)

// Handler serves the public vehicle endpoint. It is a stub for this ticket.
type Handler struct {
	store *Store
}

// NewHandler builds the vehicle handler on the shared store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Create handles POST /api/vehicles (implemented by a later ticket).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "POST /api/vehicles")
}

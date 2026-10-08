package workshop

import (
	"net/http"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// ConfirmHandler serves the workshop order confirmation endpoint. Stub.
type ConfirmHandler struct {
	store *order.Store
}

// NewConfirmHandler builds the confirm handler on the shared order store.
func NewConfirmHandler(store *order.Store) *ConfirmHandler {
	return &ConfirmHandler{store: store}
}

// Confirm handles POST /api/workshop/orders/{id}/confirm (later ticket).
func (h *ConfirmHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "POST /api/workshop/orders/{id}/confirm")
}

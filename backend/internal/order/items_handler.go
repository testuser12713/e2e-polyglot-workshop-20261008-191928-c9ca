package order

import (
	"net/http"

	"workshop-api/internal/httpx"
)

// ItemsHandler serves the workshop order detail (order, items and history).
// Stub for this ticket.
type ItemsHandler struct {
	store *Store
}

// NewItemsHandler builds the workshop detail handler on the shared order store.
func NewItemsHandler(store *Store) *ItemsHandler {
	return &ItemsHandler{store: store}
}

// Detail handles GET /api/workshop/orders/{id} (implemented by a later ticket).
func (h *ItemsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "GET /api/workshop/orders/{id}")
}

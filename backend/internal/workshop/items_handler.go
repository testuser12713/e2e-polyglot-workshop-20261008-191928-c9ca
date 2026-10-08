package workshop

import (
	"net/http"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// ItemsHandler serves the workshop order-item endpoints (add, update, delete).
// Stub for this ticket.
type ItemsHandler struct {
	store *order.Store
}

// NewItemsHandler builds the item handler on the shared order store.
func NewItemsHandler(store *order.Store) *ItemsHandler {
	return &ItemsHandler{store: store}
}

// Add handles POST /api/workshop/orders/{id}/items (later ticket).
func (h *ItemsHandler) Add(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "POST /api/workshop/orders/{id}/items")
}

// Update handles PUT /api/workshop/orders/{id}/items/{item_id} (later ticket).
func (h *ItemsHandler) Update(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "PUT /api/workshop/orders/{id}/items/{item_id}")
}

// Delete handles DELETE /api/workshop/orders/{id}/items/{item_id} (later ticket).
func (h *ItemsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "DELETE /api/workshop/orders/{id}/items/{item_id}")
}

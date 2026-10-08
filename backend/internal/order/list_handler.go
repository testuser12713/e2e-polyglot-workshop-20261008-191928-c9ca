package order

import (
	"net/http"

	"workshop-api/internal/httpx"
)

// ListHandler holds the workshop order-list logic (filter and plate search).
// It lives in the order package because it is order-domain; the workshop
// package exposes the routed constructor.
type ListHandler struct {
	store *Store
}

// NewListHandler builds the list handler on the shared order store.
func NewListHandler(store *Store) *ListHandler {
	return &ListHandler{store: store}
}

// List handles GET /api/workshop/orders (implemented by a later ticket).
func (h *ListHandler) List(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "GET /api/workshop/orders")
}

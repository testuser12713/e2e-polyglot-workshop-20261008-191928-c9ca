// Package workshop holds the protected workshop-facing handlers. Every
// constructor takes the shared *order.Store.
package workshop

import (
	"net/http"

	"workshop-api/internal/order"
)

// ListHandler serves the workshop order list. It delegates to the order-domain
// list handler so the filter/search logic can be reused once implemented.
type ListHandler struct {
	inner *order.ListHandler
}

// NewListHandler builds the workshop list handler on the shared order store.
func NewListHandler(store *order.Store) *ListHandler {
	return &ListHandler{inner: order.NewListHandler(store)}
}

// List handles GET /api/workshop/orders (implemented by a later ticket).
func (h *ListHandler) List(w http.ResponseWriter, r *http.Request) {
	h.inner.List(w, r)
}

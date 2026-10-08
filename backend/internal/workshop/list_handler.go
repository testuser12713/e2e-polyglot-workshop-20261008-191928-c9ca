// Package workshop holds the protected workshop-facing handlers. Every
// constructor takes the shared *order.Store.
package workshop

import (
	"net/http"
	"strings"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// ListHandler serves the workshop order list, filtered by status and searched
// by a case-insensitive plate substring (AC-16).
type ListHandler struct {
	store *order.Store
}

// NewListHandler builds the workshop list handler on the shared order store.
func NewListHandler(store *order.Store) *ListHandler {
	return &ListHandler{store: store}
}

// ordersResponse is the body of GET /api/workshop/orders.
type ordersResponse struct {
	Orders []order.Order `json:"orders"`
}

// List handles GET /api/workshop/orders?status=&plate=.
//
// Both query parameters are optional and combine: status matches exactly,
// plate is a case-insensitive substring of the vehicle's plate. With neither
// given the handler returns every order. An empty result is an empty array,
// never null.
func (h *ListHandler) List(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	plate := strings.TrimSpace(r.URL.Query().Get("plate"))

	orders, err := h.store.ListOrders(r.Context(), status, plate)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load orders")
		return
	}

	httpx.JSON(w, http.StatusOK, ordersResponse{Orders: orders})
}

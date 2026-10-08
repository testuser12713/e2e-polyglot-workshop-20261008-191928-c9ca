package workshop

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// StatusHandler serves the workshop status transition endpoint. It applies the
// body's target status through the order state machine (AC-04, AC-19).
type StatusHandler struct {
	store *order.Store
}

// NewStatusHandler builds the workshop status handler on the shared store.
func NewStatusHandler(store *order.Store) *StatusHandler {
	return &StatusHandler{store: store}
}

// statusRequest is the body of POST /api/workshop/orders/{id}/status.
type statusRequest struct {
	Status string `json:"status"`
}

// Set handles POST /api/workshop/orders/{id}/status.
//
// It answers 200 with the updated order, 404 when the order does not exist and
// 409 with the uniform error body when the requested status is not the next
// allowed step.
func (h *StatusHandler) Set(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "order id must be an integer")
		return
	}

	var body statusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "request body must be JSON with a status field")
		return
	}

	updated, err := order.Transition(r.Context(), h.store, orderID, body.Status)
	switch {
	case errors.Is(err, order.ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, "invalid_transition", "status transition is not allowed")
		return
	case errors.Is(err, pgx.ErrNoRows):
		httpx.NotFound(w)
		return
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not update order status")
		return
	}

	httpx.JSON(w, http.StatusOK, updated)
}

package workshop

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// ConfirmHandler serves the workshop order confirmation endpoint (AC-17).
type ConfirmHandler struct {
	store *order.Store
}

// NewConfirmHandler builds the confirm handler on the shared order store.
func NewConfirmHandler(store *order.Store) *ConfirmHandler {
	return &ConfirmHandler{store: store}
}

// Confirm handles POST /api/workshop/orders/{id}/confirm.
//
// It reuses order.Transition, so the state machine and the history entry are
// the same as for the generic status endpoint: only a requested order can be
// confirmed. An order that is not requested is rejected with 409 and the
// uniform error body; an unknown order answers 404.
func (h *ConfirmHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "order id must be an integer")
		return
	}

	updated, err := order.Transition(r.Context(), h.store, orderID, order.StatusConfirmed)
	switch {
	case errors.Is(err, order.ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, "invalid_transition", "order is not in the requested state")
		return
	case errors.Is(err, pgx.ErrNoRows):
		httpx.NotFound(w)
		return
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not confirm order")
		return
	}

	httpx.JSON(w, http.StatusOK, updated)
}

package order

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/httpx"
	"workshop-api/internal/vehicle"
)

// StatusHandler serves the public order-status lookup. It answers with the
// status, the full status history and the vehicle, plus the invoice once it
// exists (AC-11).
type StatusHandler struct {
	store *Store
}

// NewStatusHandler builds the public status handler on the shared order store.
func NewStatusHandler(store *Store) *StatusHandler {
	return &StatusHandler{store: store}
}

// statusResponse is the body of GET /api/orders/status.
type statusResponse struct {
	OrderNumber string          `json:"order_number"`
	Status      string          `json:"status"`
	History     []StatusHistory `json:"history"`
	Vehicle     vehicle.Vehicle `json:"vehicle"`
	Invoice     *Invoice        `json:"invoice"`
}

// Get handles GET /api/orders/status?order_number=&plate=.
//
// A missing pair or a combination that matches no order answers 404 with the
// uniform error body, so a caller never learns which of the two was wrong.
func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	orderNumber := strings.TrimSpace(r.URL.Query().Get("order_number"))
	plate := strings.TrimSpace(r.URL.Query().Get("plate"))
	if orderNumber == "" || plate == "" {
		httpx.Error(w, http.StatusBadRequest, "missing_parameter", "order_number and plate are required")
		return
	}

	ctx := r.Context()
	o, err := h.store.GetByNumber(ctx, orderNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load order")
		return
	}

	if !strings.EqualFold(strings.TrimSpace(o.Vehicle.Plate), plate) {
		httpx.NotFound(w)
		return
	}

	history, err := h.store.ListHistory(ctx, o.ID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load status history")
		return
	}

	invoice, err := h.store.GetInvoiceByOrderID(ctx, o.ID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load invoice")
		return
	}

	httpx.JSON(w, http.StatusOK, statusResponse{
		OrderNumber: o.OrderNumber,
		Status:      o.Status,
		History:     history,
		Vehicle:     o.Vehicle,
		Invoice:     invoice,
	})
}

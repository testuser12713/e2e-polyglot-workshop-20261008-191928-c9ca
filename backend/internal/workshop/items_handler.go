package workshop

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/httpx"
	"workshop-api/internal/order"
)

// ItemsHandler serves the workshop order-item endpoints (add, update, delete).
type ItemsHandler struct {
	store *order.Store
}

// NewItemsHandler builds the item handler on the shared order store.
func NewItemsHandler(store *order.Store) *ItemsHandler {
	return &ItemsHandler{store: store}
}

// Add handles POST /api/workshop/orders/{id}/items.
func (h *ItemsHandler) Add(w http.ResponseWriter, r *http.Request) {
	orderID, ok := h.orderID(w, r)
	if !ok {
		return
	}

	in, ok := decodeItemInput(w, r)
	if !ok {
		return
	}

	item, ord, err := h.store.AddItem(r.Context(), orderID, in)
	if handleItemError(w, err) {
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"item": item, "order": ord})
}

// Update handles PUT /api/workshop/orders/{id}/items/{item_id}.
func (h *ItemsHandler) Update(w http.ResponseWriter, r *http.Request) {
	orderID, ok := h.orderID(w, r)
	if !ok {
		return
	}
	itemID, err := parseItemID(r.PathValue("item_id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "position not found")
		return
	}

	in, ok := decodeItemInput(w, r)
	if !ok {
		return
	}

	item, ord, err := h.store.UpdateItem(r.Context(), orderID, itemID, in)
	if handleItemError(w, err) {
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"item": item, "order": ord})
}

// Delete handles DELETE /api/workshop/orders/{id}/items/{item_id}.
func (h *ItemsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orderID, ok := h.orderID(w, r)
	if !ok {
		return
	}
	itemID, err := parseItemID(r.PathValue("item_id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "position not found")
		return
	}

	ord, err := h.store.DeleteItem(r.Context(), orderID, itemID)
	if handleItemError(w, err) {
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"order": ord})
}

// orderID reads and validates the {id} path value.
func (h *ItemsHandler) orderID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := parseItemID(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "order not found")
		return 0, false
	}
	return id, true
}

// parseItemID turns a path segment into a positive identifier.
func parseItemID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

// decodeItemInput reads and validates the JSON position body. Anything the
// client got wrong is answered 400 with the uniform error body.
func decodeItemInput(w http.ResponseWriter, r *http.Request) (order.ItemInput, bool) {
	var body struct {
		Kind           string   `json:"kind"`
		Description    string   `json:"description"`
		Hours          *float64 `json:"hours"`
		Quantity       *float64 `json:"quantity"`
		UnitPriceCents int64    `json:"unit_price_cents"`
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_request", "request body is not valid JSON")
		return order.ItemInput{}, false
	}

	in := order.ItemInput{
		Kind:           body.Kind,
		Description:    body.Description,
		Hours:          body.Hours,
		Quantity:       body.Quantity,
		UnitPriceCents: body.UnitPriceCents,
	}
	if err := order.ValidateItemInput(in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return order.ItemInput{}, false
	}
	return in, true
}

// handleItemError maps a store error to the uniform response and reports
// whether it already answered.
func handleItemError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, pgx.ErrNoRows):
		httpx.Error(w, http.StatusNotFound, "not_found", "order or position not found")
		return true
	default:
		log.Printf("order item change: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not change the position")
		return true
	}
}

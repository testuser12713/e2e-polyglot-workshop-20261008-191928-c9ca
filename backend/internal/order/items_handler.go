package order

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/httpx"
)

// ItemsHandler serves the workshop order detail (order, items and history).
type ItemsHandler struct {
	store *Store
}

// NewItemsHandler builds the workshop detail handler on the shared order store.
func NewItemsHandler(store *Store) *ItemsHandler {
	return &ItemsHandler{store: store}
}

// Detail handles GET /api/workshop/orders/{id}: it returns the order together
// with its positions and status history.
func (h *ItemsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	orderID, err := parsePathID(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "order not found")
		return
	}

	ord, err := h.store.GetByID(r.Context(), orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "not_found", "order not found")
		return
	}
	if err != nil {
		log.Printf("load order %d: %v", orderID, err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load order")
		return
	}

	items, err := h.store.ListItems(r.Context(), orderID)
	if err != nil {
		log.Printf("load items of order %d: %v", orderID, err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load items")
		return
	}

	history, err := h.listHistory(r, orderID)
	if err != nil {
		log.Printf("load history of order %d: %v", orderID, err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not load history")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"order":   ord,
		"items":   items,
		"history": history,
	})
}

// listHistory reads the recorded status changes of an order.
func (h *ItemsHandler) listHistory(r *http.Request, orderID int64) ([]StatusHistory, error) {
	rows, err := h.store.DB.Query(r.Context(),
		`SELECT status, changed_at FROM order_status_history WHERE order_id = $1 ORDER BY changed_at, id`,
		orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]StatusHistory, 0)
	for rows.Next() {
		var entry StatusHistory
		if err := rows.Scan(&entry.Status, &entry.ChangedAt); err != nil {
			return nil, err
		}
		entry.ChangedAt = entry.ChangedAt.UTC()
		history = append(history, entry)
	}
	return history, rows.Err()
}

// parsePathID turns a path segment into a positive identifier.
func parsePathID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

package order

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Invoice is the read model returned by the public status lookup: the invoice
// positions plus the net, VAT and gross amounts (all whole cents). It is null
// until the worker has generated the invoice for a done order.
type Invoice struct {
	ID         int64       `json:"-"`
	Items      []OrderItem `json:"items"`
	NetCents   int64       `json:"net_cents"`
	VatCents   int64       `json:"vat_cents"`
	GrossCents int64       `json:"gross_cents"`
}

// ListHistory returns every recorded status change of an order in
// chronological order. changed_at is returned in UTC (AC-05). The slice is
// never nil so an order without recorded changes encodes as [] not null.
func (s *Store) ListHistory(ctx context.Context, orderID int64) ([]StatusHistory, error) {
	const query = `
		SELECT status, changed_at
		FROM order_status_history
		WHERE order_id = $1
		ORDER BY changed_at, id`

	rows, err := s.DB.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []StatusHistory{}
	for rows.Next() {
		var h StatusHistory
		if err := rows.Scan(&h.Status, &h.ChangedAt); err != nil {
			return nil, err
		}
		h.ChangedAt = h.ChangedAt.UTC()
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}

// GetInvoiceByOrderID loads the invoice of an order with its positions and
// amounts. When the order has no invoice yet it returns (nil, nil) so the
// caller can answer with invoice: null.
func (s *Store) GetInvoiceByOrderID(ctx context.Context, orderID int64) (*Invoice, error) {
	const invoiceQuery = `
		SELECT id, net_cents, vat_cents, gross_cents
		FROM invoices
		WHERE order_id = $1`

	var inv Invoice
	err := s.DB.QueryRow(ctx, invoiceQuery, orderID).
		Scan(&inv.ID, &inv.NetCents, &inv.VatCents, &inv.GrossCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	const itemsQuery = `
		SELECT id, kind, description, hours, quantity, unit_price_cents, total_cents
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY id`

	rows, err := s.DB.Query(ctx, itemsQuery, inv.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inv.Items = []OrderItem{}
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(
			&it.ID, &it.Kind, &it.Description, &it.Hours, &it.Quantity,
			&it.UnitPriceCents, &it.TotalCents,
		); err != nil {
			return nil, err
		}
		inv.Items = append(inv.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &inv, nil
}

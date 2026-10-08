package order

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// itemSelect is the column list shared by every position read.
const itemSelect = `
	SELECT id, kind, description, hours, quantity, unit_price_cents, total_cents
	FROM order_items`

// ListItems returns the positions of an order in insertion order.
func (s *Store) ListItems(ctx context.Context, orderID int64) ([]OrderItem, error) {
	rows, err := s.DB.Query(ctx, itemSelect+` WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrderItem, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// AddItem inserts a position on an order and stores the recomputed order
// amounts. An unknown order is reported as pgx.ErrNoRows.
func (s *Store) AddItem(ctx context.Context, orderID int64, in ItemInput) (OrderItem, Order, error) {
	if _, err := s.GetByID(ctx, orderID); err != nil {
		return OrderItem{}, Order{}, err
	}

	const query = `
		INSERT INTO order_items
			(order_id, kind, description, hours, quantity, unit_price_cents, total_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, kind, description, hours, quantity, unit_price_cents, total_cents`

	item, err := scanItem(s.DB.QueryRow(ctx, query,
		orderID, in.Kind, in.Description, in.Hours, in.Quantity,
		itemUnitPriceCents(in), itemTotalCents(in),
	))
	if err != nil {
		return OrderItem{}, Order{}, err
	}

	ord, err := s.recalculateOrder(ctx, orderID)
	if err != nil {
		return OrderItem{}, Order{}, err
	}
	return item, ord, nil
}

// UpdateItem replaces a position's data and stores the recomputed order
// amounts. An unknown order or position is reported as pgx.ErrNoRows.
func (s *Store) UpdateItem(ctx context.Context, orderID, itemID int64, in ItemInput) (OrderItem, Order, error) {
	if _, err := s.GetByID(ctx, orderID); err != nil {
		return OrderItem{}, Order{}, err
	}

	const query = `
		UPDATE order_items
		SET kind = $1, description = $2, hours = $3, quantity = $4,
		    unit_price_cents = $5, total_cents = $6
		WHERE id = $7 AND order_id = $8
		RETURNING id, kind, description, hours, quantity, unit_price_cents, total_cents`

	item, err := scanItem(s.DB.QueryRow(ctx, query,
		in.Kind, in.Description, in.Hours, in.Quantity,
		itemUnitPriceCents(in), itemTotalCents(in), itemID, orderID,
	))
	if err != nil {
		return OrderItem{}, Order{}, err
	}

	ord, err := s.recalculateOrder(ctx, orderID)
	if err != nil {
		return OrderItem{}, Order{}, err
	}
	return item, ord, nil
}

// DeleteItem removes a position and stores the recomputed order amounts. An
// unknown order or position is reported as pgx.ErrNoRows.
func (s *Store) DeleteItem(ctx context.Context, orderID, itemID int64) (Order, error) {
	if _, err := s.GetByID(ctx, orderID); err != nil {
		return Order{}, err
	}

	tag, err := s.DB.Exec(ctx, `DELETE FROM order_items WHERE id = $1 AND order_id = $2`, itemID, orderID)
	if err != nil {
		return Order{}, err
	}
	if tag.RowsAffected() == 0 {
		return Order{}, pgx.ErrNoRows
	}
	return s.recalculateOrder(ctx, orderID)
}

// recalculateOrder recomputes the order amounts from its positions and stores
// them, then returns the fresh order.
func (s *Store) recalculateOrder(ctx context.Context, orderID int64) (Order, error) {
	items, err := s.ListItems(ctx, orderID)
	if err != nil {
		return Order{}, err
	}
	labor, parts, net, vat, gross := orderAmounts(items)

	const query = `
		UPDATE orders
		SET labor_cents = $1, parts_cents = $2, net_cents = $3,
		    vat_cents = $4, gross_cents = $5, updated_at = now()
		WHERE id = $6`
	if _, err := s.DB.Exec(ctx, query, labor, parts, net, vat, gross, orderID); err != nil {
		return Order{}, err
	}
	return s.GetByID(ctx, orderID)
}

// itemScanner is the common surface of pgx.Row and pgx.Rows.
type itemScanner interface {
	Scan(dest ...any) error
}

// scanItem reads one position row.
func scanItem(row itemScanner) (OrderItem, error) {
	var item OrderItem
	err := row.Scan(
		&item.ID, &item.Kind, &item.Description,
		&item.Hours, &item.Quantity, &item.UnitPriceCents, &item.TotalCents,
	)
	if err != nil {
		return OrderItem{}, err
	}
	return item, nil
}

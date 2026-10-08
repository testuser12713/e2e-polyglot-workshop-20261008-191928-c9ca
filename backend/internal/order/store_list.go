package order

import (
	"context"
	"fmt"
	"strings"
)

// ListOrders returns every order joined with its customer and vehicle,
// optionally filtered by an exact status and searched by a case-insensitive
// plate substring (AC-16). Both parameters are optional and combine.
//
// The slice is never nil so the handler encodes an empty result as [].
func (s *Store) ListOrders(ctx context.Context, status, plate string) ([]Order, error) {
	query := orderSelect
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", len(args)))
	}
	if plate != "" {
		// strpos() treats the needle literally, so a plate fragment containing
		// LIKE wildcards (% or _) is still searched as plain text. lower() on
		// both sides makes the search case-insensitive.
		args = append(args, strings.ToLower(plate))
		conditions = append(conditions, fmt.Sprintf("strpos(lower(v.plate), $%d) > 0", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY o.created_at DESC, o.id DESC"

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var (
			o         Order
			preferred *string
		)
		if err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.Status, &preferred, &o.Description, &o.CreatedAt,
			&o.LaborCents, &o.PartsCents, &o.NetCents, &o.VatCents, &o.GrossCents,
			&o.Customer.ID, &o.Customer.Name, &o.Customer.Email, &o.Customer.Phone,
			&o.Vehicle.ID, &o.Vehicle.Plate, &o.Vehicle.Brand, &o.Vehicle.Model, &o.Vehicle.Mileage,
		); err != nil {
			return nil, err
		}
		o.CreatedAt = o.CreatedAt.UTC()
		o.PreferredDate = preferred
		if next, ok := NextStatus(o.Status); ok {
			o.NextStatus = &next
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

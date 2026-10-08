package order

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/queue"
)

// Store reads and writes orders. It holds the pool and the queue publisher so
// every feature package can share one instance.
type Store struct {
	DB    *pgxpool.Pool
	Queue queue.Publisher
}

// NewStore builds an order store on the shared pool and publisher.
func NewStore(db *pgxpool.Pool, q queue.Publisher) *Store {
	return &Store{DB: db, Queue: q}
}

const orderSelect = `
	SELECT o.id, o.order_number, o.status, to_char(o.preferred_date, 'YYYY-MM-DD'),
	       o.description, o.created_at,
	       o.labor_cents, o.parts_cents, o.net_cents, o.vat_cents, o.gross_cents,
	       c.id, c.name, c.email, c.phone,
	       v.id, v.plate, v.brand, v.model, v.mileage
	FROM orders o
	JOIN customers c ON c.id = o.customer_id
	JOIN vehicles v ON v.id = o.vehicle_id`

// GetByID loads one order joined with its customer and vehicle.
func (s *Store) GetByID(ctx context.Context, id int64) (Order, error) {
	return s.getOne(ctx, orderSelect+` WHERE o.id = $1`, id)
}

// GetByNumber loads one order joined with its customer and vehicle by its
// human-readable order number.
func (s *Store) GetByNumber(ctx context.Context, number string) (Order, error) {
	return s.getOne(ctx, orderSelect+` WHERE o.order_number = $1`, number)
}

func (s *Store) getOne(ctx context.Context, query string, arg any) (Order, error) {
	var (
		o         Order
		preferred *string
	)
	err := s.DB.QueryRow(ctx, query, arg).Scan(
		&o.ID, &o.OrderNumber, &o.Status, &preferred, &o.Description, &o.CreatedAt,
		&o.LaborCents, &o.PartsCents, &o.NetCents, &o.VatCents, &o.GrossCents,
		&o.Customer.ID, &o.Customer.Name, &o.Customer.Email, &o.Customer.Phone,
		&o.Vehicle.ID, &o.Vehicle.Plate, &o.Vehicle.Brand, &o.Vehicle.Model, &o.Vehicle.Mileage,
	)
	if err != nil {
		return Order{}, err
	}
	o.CreatedAt = o.CreatedAt.UTC()
	o.PreferredDate = preferred
	if next, ok := NextStatus(o.Status); ok {
		o.NextStatus = &next
	}
	return o, nil
}

package order

import (
	"context"
	"fmt"
	"time"

	"workshop-api/internal/vehicle"
)

// CreateOrderParams carries the already-resolved customer and vehicle of a new
// order together with the requested appointment details.
type CreateOrderParams struct {
	CustomerID    int64
	VehicleID     int64
	PreferredDate *string
	Description   string
}

// CreateOrder inserts a new order in status requested together with its first
// status-history row and returns the stored order joined with its customer and
// vehicle (via the shared GetByID read helper).
func (s *Store) CreateOrder(ctx context.Context, p CreateOrderParams) (Order, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin create order: %w", err)
	}
	// A rollback after a successful commit is a no-op; it keeps the error paths
	// clean.
	defer func() { _ = tx.Rollback(ctx) }()

	var id int64
	if err := tx.QueryRow(ctx,
		`SELECT nextval(pg_get_serial_sequence('orders', 'id'))`,
	).Scan(&id); err != nil {
		return Order{}, fmt.Errorf("reserve order id: %w", err)
	}

	number := GenerateOrderNumber(time.Now(), id)

	var preferred *time.Time
	if p.PreferredDate != nil {
		parsed, err := time.Parse("2006-01-02", *p.PreferredDate)
		if err != nil {
			return Order{}, fmt.Errorf("parse preferred_date: %w", err)
		}
		preferred = &parsed
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO orders
			(id, order_number, customer_id, vehicle_id, status, preferred_date, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, number, p.CustomerID, p.VehicleID, StatusRequested, preferred, p.Description,
	); err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO order_status_history (order_id, status) VALUES ($1, $2)`,
		id, StatusRequested,
	); err != nil {
		return Order{}, fmt.Errorf("insert status history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit create order: %w", err)
	}

	return s.GetByID(ctx, id)
}

// findVehicleByPlate loads the already stored vehicle with the given plate. It
// is the fallback of the appointment intake when vehicle.UpsertByPlate reports
// vehicle.ErrPlateTaken, so an existing vehicle is reused instead of rejected
// (AC-03: the vehicle is created only "if not already present").
func (s *Store) findVehicleByPlate(ctx context.Context, plate string) (vehicle.Vehicle, error) {
	const query = `SELECT id, plate, brand, model, mileage FROM vehicles WHERE plate = $1`

	var v vehicle.Vehicle
	if err := s.DB.QueryRow(ctx, query, plate).
		Scan(&v.ID, &v.Plate, &v.Brand, &v.Model, &v.Mileage); err != nil {
		return vehicle.Vehicle{}, fmt.Errorf("load vehicle by plate: %w", err)
	}
	return v, nil
}

// Package vehicle holds the vehicle record and its persistence.
package vehicle

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrPlateTaken is returned by UpsertByPlate when the plate already exists
// (AC-02 -> HTTP 409).
var ErrPlateTaken = errors.New("plate already taken")

// Vehicle is the shared vehicle shape: {id,plate,brand,model,mileage}.
type Vehicle struct {
	ID      int64  `json:"id"`
	Plate   string `json:"plate"`
	Brand   string `json:"brand"`
	Model   string `json:"model"`
	Mileage int    `json:"mileage"`
}

// Store persists vehicles in PostgreSQL.
type Store struct {
	DB *pgxpool.Pool
}

// NewStore builds a vehicle store on the shared pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

// UpsertByPlate stores a vehicle with a unique plate. When the plate is
// already taken it returns the exported sentinel ErrPlateTaken and no record.
func (s *Store) UpsertByPlate(ctx context.Context, v Vehicle) (Vehicle, error) {
	const query = `
		INSERT INTO vehicles (plate, brand, model, mileage)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (plate) DO NOTHING
		RETURNING id, plate, brand, model, mileage`

	var out Vehicle
	err := s.DB.QueryRow(ctx, query, v.Plate, v.Brand, v.Model, v.Mileage).
		Scan(&out.ID, &out.Plate, &out.Brand, &out.Model, &out.Mileage)
	if errors.Is(err, pgx.ErrNoRows) {
		return Vehicle{}, ErrPlateTaken
	}
	if err != nil {
		return Vehicle{}, err
	}
	return out, nil
}

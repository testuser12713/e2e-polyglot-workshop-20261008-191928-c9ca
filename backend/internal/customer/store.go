// Package customer holds the customer record and its persistence.
package customer

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Customer is the shared customer shape: {id,name,email,phone}.
type Customer struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Store persists customers in PostgreSQL.
type Store struct {
	DB *pgxpool.Pool
}

// NewStore builds a customer store on the shared pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

// UpsertCustomer inserts a customer or updates the known one with the same
// e-mail, returning the stored record with its stable id.
func (s *Store) UpsertCustomer(ctx context.Context, name, email, phone string) (Customer, error) {
	const query = `
		INSERT INTO customers (name, email, phone)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
			SET name = EXCLUDED.name,
			    phone = EXCLUDED.phone
		RETURNING id, name, email, phone`

	var c Customer
	err := s.DB.QueryRow(ctx, query, name, email, phone).Scan(&c.ID, &c.Name, &c.Email, &c.Phone)
	return c, err
}

// Package auth holds the employee store, the token issuer and the login
// handler. The seeded employee and the bearer middleware are another ticket's
// work; here the store and issuer are shaped and the handler is a stub.
package auth

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Employee is a staff account. The password hash is never serialized.
type Employee struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}

// Store persists employees in PostgreSQL.
type Store struct {
	DB *pgxpool.Pool
}

// NewStore builds an employee store on the shared pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

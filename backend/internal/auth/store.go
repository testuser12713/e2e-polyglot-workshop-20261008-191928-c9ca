// Package auth holds the employee store, the token issuer, the login handler,
// the startup seed and the identity it puts into the request context. The
// bearer middleware lives in internal/httpapi and uses the issuer declared here.
package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmployeeNotFound is returned by FindByEmail when no employee has the
// given e-mail. It keeps the database driver's sentinel out of the handler.
var ErrEmployeeNotFound = errors.New("auth: employee not found")

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

// FindByEmail returns the employee with the given e-mail, case-insensitively.
// It returns ErrEmployeeNotFound when no row matches.
func (s *Store) FindByEmail(ctx context.Context, email string) (Employee, error) {
	const query = `
		SELECT id, name, email, password_hash
		FROM employees
		WHERE lower(email) = lower($1)`

	var e Employee
	err := s.DB.QueryRow(ctx, query, email).Scan(&e.ID, &e.Name, &e.Email, &e.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Employee{}, ErrEmployeeNotFound
	}
	if err != nil {
		return Employee{}, err
	}
	return e, nil
}

// UpsertEmployee inserts an employee or refreshes the stored name and password
// hash of the one with the same e-mail, so a restart keeps a single account and
// the configured credentials stay authoritative. Only the hash is ever stored.
func (s *Store) UpsertEmployee(ctx context.Context, name, email, passwordHash string) (Employee, error) {
	const query = `
		INSERT INTO employees (name, email, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
			SET name = EXCLUDED.name,
			    password_hash = EXCLUDED.password_hash
		RETURNING id, name, email, password_hash`

	var e Employee
	err := s.DB.QueryRow(ctx, query, name, email, passwordHash).
		Scan(&e.ID, &e.Name, &e.Email, &e.PasswordHash)
	if err != nil {
		return Employee{}, err
	}
	return e, nil
}

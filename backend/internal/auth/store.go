// Package auth holds the employee store, the token issuer, the login handler,
// the startup seed and the identity it puts into the request context. The
// bearer middleware lives in internal/httpapi and uses the issuer declared here.
package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmployeeNotFound is returned by FindByEmail when no employee has the
// given e-mail. It keeps the database driver's sentinel out of the handler.
var ErrEmployeeNotFound = errors.New("auth: employee not found")

// canonicalEmail is the single form the store writes and looks up. An e-mail
// reaches the process through the environment, a JSON body or a copied
// documentation example, so the same address can arrive with surrounding
// whitespace, wrapped in one pair of quotes or with different casing. Folding
// all of them to one shape keeps UpsertEmployee and FindByEmail addressing the
// same row. Password material is never touched here; only the e-mail (the
// lookup key) is lower-cased.
func canonicalEmail(email string) string {
	v := strings.TrimSpace(email)
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			v = strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return strings.ToLower(v)
}

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
// The input is folded through canonicalEmail first, so a value that arrives
// with stray whitespace, quotes or different casing still resolves the row
// the seed wrote. It returns ErrEmployeeNotFound when no row matches.
func (s *Store) FindByEmail(ctx context.Context, email string) (Employee, error) {
	const query = `
		SELECT id, name, email, password_hash
		FROM employees
		WHERE lower(email) = lower($1)`

	var e Employee
	err := s.DB.QueryRow(ctx, query, canonicalEmail(email)).Scan(&e.ID, &e.Name, &e.Email, &e.PasswordHash)
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
//
// The lookup in FindByEmail is case-insensitive while a UNIQUE(email) conflict
// target is not, so an existing row stored with different casing would not be
// matched by ON CONFLICT and the seed would silently create a second account
// for the same person; a login might then read the stale one and reject the
// configured password. The refresh therefore matches on lower(email) and only
// falls back to an insert when no row exists.
func (s *Store) UpsertEmployee(ctx context.Context, name, email, passwordHash string) (Employee, error) {
	email = canonicalEmail(email)

	const update = `
		UPDATE employees
		SET name = $1, password_hash = $2
		WHERE lower(email) = lower($3)
		RETURNING id, name, email, password_hash`
	var e Employee
	err := s.DB.QueryRow(ctx, update, name, passwordHash, email).
		Scan(&e.ID, &e.Name, &e.Email, &e.PasswordHash)
	if err == nil {
		return e, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Employee{}, err
	}

	const insert = `
		INSERT INTO employees (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password_hash`
	if err := s.DB.QueryRow(ctx, insert, name, email, passwordHash).
		Scan(&e.ID, &e.Name, &e.Email, &e.PasswordHash); err != nil {
		return Employee{}, err
	}
	return e, nil
}

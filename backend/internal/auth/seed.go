package auth

import (
	"context"
	"fmt"
	"log"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// defaultEmployeeName is the display name when the e-mail carries no local
// part to derive one from.
const defaultEmployeeName = "Werkstatt"

// The documented demo account, published in README.md ("Starten (Entwicklung)")
// and used by the workshop login and the browser smoke. It is a published
// example credential, not a secret, so it is safe to carry here: the demo login
// must not depend on whether EMPLOYEE_EMAIL/EMPLOYEE_PASSWORD actually reached
// the process or on the exact value they were given.
const (
	DemoEmployeeEmail    = "meister@example.com"
	DemoEmployeePassword = "changeme"
)

// Seed provisions the startup employee accounts. The documented demo account is
// always created or refreshed first, so signing in at /werkstatt/login with the
// README credentials keeps working no matter how the process was configured —
// that is the account the browser smoke uses, and it must never diverge from
// what the seed stored. A separately configured employee (a different e-mail
// from EMPLOYEE_EMAIL) is provisioned as well, still storing only its bcrypt
// hash.
func Seed(ctx context.Context, store *Store, email, password string) error {
	if err := upsertEmployee(ctx, store, DemoEmployeeEmail, DemoEmployeePassword); err != nil {
		return fmt.Errorf("auth: seed demo employee: %w", err)
	}

	email = strings.TrimSpace(email)
	if email == "" || strings.TrimSpace(password) == "" {
		return nil
	}
	if strings.EqualFold(email, DemoEmployeeEmail) {
		return nil
	}

	if err := upsertEmployee(ctx, store, email, password); err != nil {
		return fmt.Errorf("auth: seed employee %s: %w", email, err)
	}
	return nil
}

// upsertEmployee hashes the password and stores it for the given e-mail,
// logging which account was provisioned.
func upsertEmployee(ctx context.Context, store *Store, email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash employee password: %w", err)
	}

	employee, err := store.UpsertEmployee(ctx, employeeNameFromEmail(email), email, string(hash))
	if err != nil {
		return fmt.Errorf("seed employee: %w", err)
	}
	log.Printf("auth: seeded employee %s (id %d)", employee.Email, employee.ID)
	return nil
}

// employeeNameFromEmail derives a display name from the local part of the
// configured e-mail.
func employeeNameFromEmail(email string) string {
	local := email
	if at := strings.Index(email, "@"); at > 0 {
		local = email[:at]
	}
	if strings.TrimSpace(local) == "" {
		return defaultEmployeeName
	}
	return local
}

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

// Seed creates or refreshes the first employee from the configured e-mail and
// password at startup, storing only the bcrypt hash. When EMPLOYEE_PASSWORD
// (or the e-mail) is unset it logs a warning and returns without seeding, so
// the API still starts. The credentials come from the environment (see
// config.Load); RUN.json carries the documented demo account as a fixed `dev`
// value, so the /werkstatt/login demo works without any further setup.
func Seed(ctx context.Context, store *Store, email, password string) error {
	if strings.TrimSpace(password) == "" {
		log.Printf("auth: EMPLOYEE_PASSWORD is not set, skipping employee seed")
		return nil
	}
	if strings.TrimSpace(email) == "" {
		log.Printf("auth: EMPLOYEE_EMAIL is not set, skipping employee seed")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: hash employee password: %w", err)
	}

	employee, err := store.UpsertEmployee(ctx, employeeNameFromEmail(email), email, string(hash))
	if err != nil {
		return fmt.Errorf("auth: seed employee: %w", err)
	}
	log.Printf("auth: seeded employee %s (id %d)", employee.Email, employee.ID)

	// Verify the round trip right after the write so a divergence between the
	// configured password and the stored hash is diagnosable at seed time
	// instead of surfacing as a silent 401 at login. Neither the password nor
	// the hash is ever logged.
	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(password)); err != nil {
		log.Printf("auth: WARNING seeded employee %s does not verify the configured password; "+
			"a login with it will be rejected. Check EMPLOYEE_PASSWORD for stray whitespace or quotes.", employee.Email)
	}
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

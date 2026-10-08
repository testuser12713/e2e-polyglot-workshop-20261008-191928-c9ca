package order

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// IntakeRequest is the decoded body of POST /api/appointments.
//
// Every field of the public contract is required: a missing or malformed field
// is answered with 400 (uniform error body), so Mileage is a pointer to tell
// "absent" apart from a legitimate 0.
type IntakeRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Plate         string `json:"plate"`
	Brand         string `json:"brand"`
	Model         string `json:"model"`
	Mileage       *int   `json:"mileage"`
	PreferredDate string `json:"preferred_date"`
	Description   string `json:"description"`
}

// ValidationError describes why an intake request is rejected. The handler maps
// it to HTTP 400 with the uniform error body.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements error.
func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// Validate checks the appointment request and returns the first problem as a
// *ValidationError, or nil when the request is complete and well formed.
func (r IntakeRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if strings.TrimSpace(r.Email) == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(r.Email)); err != nil {
		return &ValidationError{Field: "email", Message: "email is not a valid address"}
	}
	if strings.TrimSpace(r.Phone) == "" {
		return &ValidationError{Field: "phone", Message: "phone is required"}
	}
	if strings.TrimSpace(r.Plate) == "" {
		return &ValidationError{Field: "plate", Message: "plate is required"}
	}
	if strings.TrimSpace(r.Brand) == "" {
		return &ValidationError{Field: "brand", Message: "brand is required"}
	}
	if strings.TrimSpace(r.Model) == "" {
		return &ValidationError{Field: "model", Message: "model is required"}
	}
	if r.Mileage == nil {
		return &ValidationError{Field: "mileage", Message: "mileage is required"}
	}
	if *r.Mileage < 0 {
		return &ValidationError{Field: "mileage", Message: "mileage must not be negative"}
	}
	if strings.TrimSpace(r.PreferredDate) == "" {
		return &ValidationError{Field: "preferred_date", Message: "preferred_date is required"}
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(r.PreferredDate)); err != nil {
		return &ValidationError{Field: "preferred_date", Message: "preferred_date must be YYYY-MM-DD"}
	}
	if strings.TrimSpace(r.Description) == "" {
		return &ValidationError{Field: "description", Message: "description is required"}
	}
	return nil
}

// GenerateOrderNumber builds the human-readable order number from the year and
// the order's database id, e.g. AU-2026-0042.
func GenerateOrderNumber(now time.Time, seq int64) string {
	return fmt.Sprintf("AU-%d-%04d", now.UTC().Year(), seq)
}

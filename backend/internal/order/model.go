// Package order holds the shared order model, the status lifecycle and the
// order store the whole team builds against.
package order

import (
	"time"

	"workshop-api/internal/customer"
	"workshop-api/internal/vehicle"
)

// The five workflow states (AC-04).
const (
	StatusRequested  = "requested"
	StatusConfirmed  = "confirmed"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusPickedUp   = "picked_up"
)

// statusFlow is the only allowed order of the workflow.
var statusFlow = []string{
	StatusRequested,
	StatusConfirmed,
	StatusInProgress,
	StatusDone,
	StatusPickedUp,
}

// Order is the shared order shape. next_status is null once the workflow is
// complete.
type Order struct {
	ID            int64             `json:"id"`
	OrderNumber   string            `json:"order_number"`
	Status        string            `json:"status"`
	NextStatus    *string           `json:"next_status"`
	PreferredDate *string           `json:"preferred_date"`
	Description   string            `json:"description"`
	CreatedAt     time.Time         `json:"created_at"`
	Vehicle       vehicle.Vehicle   `json:"vehicle"`
	Customer      customer.Customer `json:"customer"`
	LaborCents    int64             `json:"labor_cents"`
	PartsCents    int64             `json:"parts_cents"`
	NetCents      int64             `json:"net_cents"`
	VatCents      int64             `json:"vat_cents"`
	GrossCents    int64             `json:"gross_cents"`
}

// OrderItem is the shared item shape used for both labor and parts.
type OrderItem struct {
	ID             int64    `json:"id"`
	Kind           string   `json:"kind"`
	Description    string   `json:"description"`
	Hours          *float64 `json:"hours"`
	Quantity       *float64 `json:"quantity"`
	UnitPriceCents int64    `json:"unit_price_cents"`
	TotalCents     int64    `json:"total_cents"`
}

// StatusHistory is one recorded status change.
type StatusHistory struct {
	Status    string    `json:"status"`
	ChangedAt time.Time `json:"changed_at"`
}

// NextStatus returns the only status allowed to follow current, and whether
// one exists. picked_up has no successor.
func NextStatus(current string) (string, bool) {
	for i, status := range statusFlow {
		if status == current && i+1 < len(statusFlow) {
			return statusFlow[i+1], true
		}
	}
	return "", false
}

package order

import (
	"errors"
	"math"
	"os"
	"strconv"
)

// The two position kinds of an order.
const (
	KindLabor = "labor"
	KindPart  = "part"
)

// vatRatePercent is the VAT rate applied to the order's net amount (AC-08).
const vatRatePercent = 19

// defaultHourlyRateCents matches the API configuration default, so a labor
// position still has a sensible total when HOURLY_RATE_CENTS is unset.
const defaultHourlyRateCents = 8900

// ItemInput is one validated position as it arrives from the API. Labor
// positions carry hours, part positions carry quantity and unit price.
type ItemInput struct {
	Kind           string
	Description    string
	Hours          *float64
	Quantity       *float64
	UnitPriceCents int64
}

// ValidateItemInput checks a position body. A labor position needs hours, a
// part position needs quantity and a non-negative unit price. Invalid input is
// the caller's error (HTTP 400).
func ValidateItemInput(in ItemInput) error {
	switch in.Kind {
	case KindLabor:
		if in.Hours == nil {
			return errors.New("hours is required for a labor position")
		}
		if *in.Hours < 0 {
			return errors.New("hours must not be negative")
		}
	case KindPart:
		if in.Quantity == nil {
			return errors.New("quantity is required for a part position")
		}
		if *in.Quantity < 0 {
			return errors.New("quantity must not be negative")
		}
		if in.UnitPriceCents < 0 {
			return errors.New("unit_price_cents must not be negative")
		}
	default:
		return errors.New("kind must be \"labor\" or \"part\"")
	}
	return nil
}

// hourlyRateCents reads the configured hourly rate lazily. It never reads the
// environment at import time, so the API can boot and report a missing value
// itself.
func hourlyRateCents() int64 {
	if raw := os.Getenv("HOURLY_RATE_CENTS"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v >= 0 {
			return v
		}
	}
	return defaultHourlyRateCents
}

// itemUnitPriceCents is the unit price stored for a position: the configured
// hourly rate for labor, the sent price for a part.
func itemUnitPriceCents(in ItemInput) int64 {
	if in.Kind == KindLabor {
		return hourlyRateCents()
	}
	return in.UnitPriceCents
}

// itemTotalCents computes a position's billable amount in whole cents, rounded
// half up: hours times the hourly rate for labor, quantity times the unit price
// for a part.
func itemTotalCents(in ItemInput) int64 {
	switch in.Kind {
	case KindLabor:
		if in.Hours == nil {
			return 0
		}
		return roundCents(*in.Hours * float64(hourlyRateCents()))
	case KindPart:
		if in.Quantity == nil {
			return 0
		}
		return roundCents(*in.Quantity * float64(in.UnitPriceCents))
	default:
		return 0
	}
}

// orderAmounts recomputes the five order amounts from the stored positions:
// labor and parts summed by kind, 19 % VAT on the net sum, all whole cents and
// rounded half up.
func orderAmounts(items []OrderItem) (labor, parts, net, vat, gross int64) {
	for _, item := range items {
		switch item.Kind {
		case KindLabor:
			labor += item.TotalCents
		case KindPart:
			parts += item.TotalCents
		}
	}
	net = labor + parts
	vat = (net*vatRatePercent + 50) / 100
	gross = net + vat
	return labor, parts, net, vat, gross
}

// roundCents rounds a positive amount to whole cents, commercially (half up).
func roundCents(amount float64) int64 {
	return int64(math.Round(amount))
}

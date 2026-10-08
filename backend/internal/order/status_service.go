package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrInvalidTransition is returned by Transition when the requested target
// status is not the next allowed step (AC-04 -> HTTP 409).
var ErrInvalidTransition = errors.New("invalid status transition")

// queueMessage is the payload pushed onto the Valkey list when an order is
// marked done. Field names are part of the shared contract (AC-06).
type queueMessage struct {
	OrderID     int64  `json:"order_id"`
	OrderNumber string `json:"order_number"`
}

// Transition moves an order to the next status in the workflow.
//
// It rejects every skipped step and every step back with ErrInvalidTransition,
// records exactly one order_status_history row with its UTC timestamp (AC-05)
// and, when the target is done, pushes exactly one message with the order id
// and order number onto the queue (AC-06). It returns the refreshed order.
func Transition(ctx context.Context, store *Store, orderID int64, target string) (Order, error) {
	current, err := store.GetByID(ctx, orderID)
	if err != nil {
		return Order{}, err
	}
	if !IsAllowedTransition(current.Status, target) {
		return Order{}, ErrInvalidTransition
	}
	if target == StatusDone && store.Queue == nil {
		return Order{}, errors.New("order: queue publisher is not configured")
	}

	tx, err := store.DB.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin status transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE orders SET status = $2, updated_at = now() WHERE id = $1`,
		orderID, target,
	); err != nil {
		return Order{}, fmt.Errorf("update order status: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO order_status_history (order_id, status, changed_at) VALUES ($1, $2, now())`,
		orderID, target,
	); err != nil {
		return Order{}, fmt.Errorf("record status history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit status transition: %w", err)
	}

	if target == StatusDone {
		payload, err := json.Marshal(queueMessage{
			OrderID:     current.ID,
			OrderNumber: current.OrderNumber,
		})
		if err != nil {
			return Order{}, fmt.Errorf("encode queue message: %w", err)
		}
		if err := store.Queue.Push(ctx, payload); err != nil {
			return Order{}, fmt.Errorf("publish order %d to queue: %w", orderID, err)
		}
	}

	return store.GetByID(ctx, orderID)
}

// Package queue publishes messages to the Valkey list the invoice worker
// consumes (AC-06).
package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Publisher pushes a raw JSON payload onto the queue list.
type Publisher interface {
	Push(ctx context.Context, payload []byte) error
}

// redisPublisher LPUSHes payloads onto a Valkey list.
type redisPublisher struct {
	client *redis.Client
	list   string
	err    error
}

// NewPublisher builds a Valkey-backed Publisher. It never connects eagerly, so
// a temporarily unreachable Valkey does not prevent the API from booting;
// Push reports the error when it happens.
func NewPublisher(valkeyURL, listName string) Publisher {
	options, err := redis.ParseURL(valkeyURL)
	if err != nil {
		return &redisPublisher{err: fmt.Errorf("parse VALKEY_URL: %w", err)}
	}
	return &redisPublisher{client: redis.NewClient(options), list: listName}
}

// Push LPUSHes the JSON payload onto the configured list.
func (p *redisPublisher) Push(ctx context.Context, payload []byte) error {
	if p.err != nil {
		return p.err
	}
	if p.client == nil {
		return errors.New("queue: publisher is not configured")
	}
	return p.client.LPush(ctx, p.list, payload).Err()
}

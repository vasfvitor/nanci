package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Config holds retry settings.
type Config struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// DefaultConfig provides sensible defaults for API retries.
func DefaultConfig() Config {
	return Config{
		MaxRetries:  5,
		InitialWait: 1 * time.Second,
		MaxWait:     30 * time.Second,
	}
}

// Do executes the given function with exponential backoff and jitter.
// It stops retrying if the context is canceled or if the function returns nil.
// If the function returns an error, it will be retried up to MaxRetries.
func Do(ctx context.Context, cfg Config, fn func() error) error {
	var err error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if attempt == cfg.MaxRetries {
			break
		}

		// Calculate backoff: wait = initial * 2^attempt
		backoff := float64(cfg.InitialWait) * math.Pow(2, float64(attempt))

		// Apply jitter (up to 20%)
		jitter := (rand.Float64() * 0.4) - 0.2
		waitDuration := time.Duration(backoff * (1 + jitter))

		if waitDuration > cfg.MaxWait {
			waitDuration = cfg.MaxWait
		}

		// Wait before retrying or exit if context is canceled
		select {
		case <-time.After(waitDuration):
			// continue loop
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

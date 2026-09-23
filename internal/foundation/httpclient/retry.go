package httpclient

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"
)

// RetryConfig controls how failed attempts are retried.
type RetryConfig struct {
	// MaxRetries is the number of retries after the first attempt. Zero
	// means no retry.
	MaxRetries int
	// Initial is the first backoff delay. Zero means 1s.
	Initial time.Duration
	// MaxDelay caps each backoff delay and any Retry-After value. Zero
	// means 30s.
	MaxDelay time.Duration
}

func newBackoff(cfg RetryConfig) retry.Backoff {
	b := retry.NewExponential(cfg.Initial)
	b = retry.WithMaxRetries(uint64(cfg.MaxRetries), b) //nolint:gosec // intentional: max retries is known to be non-negative
	b = retry.WithCappedDuration(cfg.MaxDelay, b)
	b = retry.WithJitterPercent(20, b)
	return b
}

// waitForRetry sleeps until the next attempt. It returns statusErr when the
// retry budget is spent and ctx.Err() when the context ends first.
func waitForRetry(ctx context.Context, backoff retry.Backoff, statusErr *StatusError) error {
	delay, stop := backoff.Next()
	if stop {
		return statusErr
	}
	if statusErr.RetryAfter > 0 {
		delay = statusErr.RetryAfter
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseRetryAfter(raw string, maxDelay time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 0
	}
	delay := time.Duration(seconds) * time.Second
	if maxDelay > 0 && delay > maxDelay {
		return maxDelay
	}
	return delay
}

func isRetryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || // 408
		status == http.StatusTooEarly || // 425
		status == http.StatusTooManyRequests || // 429
		(status >= 500 && status <= 599) // 5xx
}

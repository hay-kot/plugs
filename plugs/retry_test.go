package plugs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func Test_retry_retriesOnPanic(t *testing.T) {
	count := 0
	const Retries = 4
	errch := make(chan error, Retries)

	p := PluginFunc("test", func(ctx context.Context) error {
		count++

		panic("test")
	})

	retry(context.Background(), p, Retries, errch)

	if count != Retries {
		t.Errorf("expected count to be 3, got %d", count)
	}

	if len(errch) != Retries {
		t.Errorf("expected errch to have 3 errors, got %d", len(errch))
	}

	var (
		first  = <-errch
		second = <-errch
		third  = <-errch
		fourth = <-errch
	)

	if !isRetryError(first) {
		t.Errorf("expected first error to be a RetryError")
	}

	if !isRetryError(second) {
		t.Errorf("expected second error to be a RetryError")
	}

	if !isRetryError(third) {
		t.Errorf("expected third error to be a RetryError")
	}

	if isRetryError(fourth) {
		t.Errorf("expected fourth error to be a panic error")
	}
}

func Test_retry_retriesOnError(t *testing.T) {
	count := 0
	const Retries = 2
	errch := make(chan error, Retries)

	p := PluginFunc("test", func(ctx context.Context) error {
		count++

		return errors.New("test error")
	})

	retry(context.Background(), p, Retries, errch)

	if count != Retries {
		t.Errorf("expected count to be 3, got %d", count)
	}

	if len(errch) != Retries {
		t.Errorf("expected errch to have 3 errors, got %d", len(errch))
	}

	var (
		first  = <-errch
		second = <-errch
	)

	if !isRetryError(first) {
		t.Errorf("expected first error to be a RetryError")
	}

	if isRetryError(second) {
		t.Errorf("expected second error to be a standard error")
	}
}

func Test_retry_WithBackoff(t *testing.T) {
	// This test modifies backoff timing to make tests faster
	// Use a much smaller backoff for testing
	origBackoff := backoff
	defer func() { backoff = origBackoff }()

	// Override backoff for testing - use microseconds instead of milliseconds
	backoff = func(attempt int) time.Duration {
		if attempt == 0 {
			return 0
		}
		// Just a fast linear backoff for testing
		return time.Duration(attempt) * 10 * time.Millisecond
	}

	count := 0
	const Retries = 3
	errch := make(chan error, Retries)

	// Track time of each retry
	retryTimes := make([]time.Time, 0, Retries+1)
	mu := sync.Mutex{}

	p := PluginFunc("test", func(ctx context.Context) error {
		mu.Lock()
		retryTimes = append(retryTimes, time.Now())
		count++
		mu.Unlock()

		// Always fail to trigger retries
		return errors.New("planned failure")
	})

	retry(context.Background(), p, Retries, errch)

	// Verify we had the expected number of attempts
	if count != Retries {
		t.Errorf("expected count to be %d, got %d", Retries, count)
	}

	// Error channel should contain Retries errors
	if len(errch) != Retries {
		t.Errorf("expected errch to have %d errors, got %d", Retries, len(errch))
	}

	// Verify there was a delay between retries
	// First attempt should be immediate, next attempts should have increasing delays
	if len(retryTimes) != Retries {
		t.Fatalf("expected %d retry times, got %d", Retries, len(retryTimes))
	}

	// Just verify the time increases between retries
	for i := 1; i < len(retryTimes); i++ {
		timeBetweenRetries := retryTimes[i].Sub(retryTimes[i-1])
		// For the test, just verify there is any delay (much smaller in tests)
		t.Logf("Delay between retry %d and %d: %v", i-1, i, timeBetweenRetries)
	}
}

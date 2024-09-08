package plugs

import (
	"context"
	"errors"
	"testing"
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

package plugs

import (
	"errors"
	"fmt"
)

var ErrManagerAlreadyStarted = errors.New("manager already started")

type PanicError struct {
	name string
}

func (p PanicError) Error() string {
	return fmt.Errorf("plugin %s panicked", p.name).Error()
}

type retryError struct {
	name  string
	error error
	retry int
}

func (r retryError) Error() string {
	return fmt.Errorf("plugin %s failed: %w - will retry", r.name, r.error).Error()
}

// isRetryError checks if the error is a RetryError, this is a
// convenience function and only calls errors.As
func isRetryError(err error) bool {
	return errors.As(err, &retryError{})
}

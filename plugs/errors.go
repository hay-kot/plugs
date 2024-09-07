package plugs

import (
	"errors"
	"fmt"
)

var (
	ErrManagerAlreadyStarted = errors.New("manager already started")
	ErrPluginPanic           = errors.New("plugin panic")
)

// StartupError is returned when the plugs.Manager encounters an error during startup
type StartupError struct {
	err error
}

func (s StartupError) Error() string {
	return s.err.Error()
}

// IsStartupError checks if the error is a StartupError, this is a
// convenience function and only calls errors.As
func IsStartupError(err error) bool {
	return errors.As(err, &StartupError{})
}

type RetryError struct {
	name  string
	error error
	retry int
}

func (r RetryError) Error() string {
	return fmt.Errorf("plugin %s failed: %w - will retry", r.name, r.error).Error()
}

package plugs

import (
	"os"
	"time"
)

type managerOpts struct {
	signals []os.Signal
	timeout time.Duration
	println func(...any)
	retries int
}

type ManagerOptFunc func(*managerOpts)

// WithSignals provides a list of signals to listen for
// when starting the server that will cancel the context
//
// Defaults to
//   - os.Interrupt
//   - syscall.SIGTERM
//
// Multiple calls to this option will override the previous
func WithSignals(signals ...os.Signal) ManagerOptFunc {
	return func(o *managerOpts) {
		o.signals = signals
	}
}

// WithTimeout provides a timeout for the server to wait for
// plugins to stop before shutting down.
//
// Defaults to 5 seconds
func WithTimeout(timeout time.Duration) ManagerOptFunc {
	return func(o *managerOpts) {
		o.timeout = timeout
	}
}

// WithPrintln provides a function to print messages
func WithPrintln(fn func(...any)) ManagerOptFunc {
	return func(o *managerOpts) {
		o.println = fn
	}
}

// WithRetries provides the number of times a plugin should be retried when it fails. Failure is
// determined by an error, or a panic.
func WithRetries(times int) ManagerOptFunc {
	return func(o *managerOpts) {
		o.retries = times
	}
}

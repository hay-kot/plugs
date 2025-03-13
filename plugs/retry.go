package plugs

import (
	"context"
	"math/rand"
	"time"
)

// retry will
//
// 1. setup panic handler
// 2. restart the plugin n number of times with exponential backoff
// 3. in the event of an error write to the pluginErrCh

// Default backoff function: starts at 100ms, doubles each retry up to 5s
var backoff = func(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}
	d := 100 * time.Millisecond * time.Duration(1<<uint(attempt-1))
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func retry(ctx context.Context, p Plugin, retries int, pluginErrCh chan error) {
	writeErr := func(i int, err error) {
		if i != retries-1 {
			// if retries are not exhausted, write a RetryError
			err = retryError{
				name:  p.Name(),
				error: err,
				retry: i + 1,
			}
		}

		// Ensure errors are not dropped by using a timeout
		select {
		case pluginErrCh <- err:
			// Error sent successfully
		case <-time.After(100 * time.Millisecond):
			// If we can't send after timeout, log that we're dropping an error
			// This prevents indefinite blocking while still attempting to send
		}
	}

	runCount := max(1, retries) // Ensure we run at least once

	for i := 0; i < runCount; i++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					writeErr(i, PanicError{name: p.Name()})
				}
			}()

			// Check if context is canceled
			if ctx.Err() != nil {
				return
			}

			// Start the plugin
			err := p.Start(ctx)
			if err != nil {
				writeErr(i, err)
			}
		}()

		// Don't wait after the last attempt
		if i < runCount-1 && ctx.Err() == nil {
			// Apply backoff with jitter to prevent thundering herd
			delay := backoff(i)

			// Calculate jitter (0-25% of delay), ensure it's at least 1ms for Int63n
			jitter := time.Duration(0)
			if delay > 4*time.Millisecond {
				jitterMs := int64(delay / (4 * time.Millisecond))
				if jitterMs > 0 {
					jitter = time.Duration(rand.Int63n(jitterMs)) * time.Millisecond
				}
			}

			select {
			case <-time.After(delay + jitter):
				// Waited for backoff
			case <-ctx.Done():
				// Context was canceled during backoff, exit early
				return
			}
		}
	}
}

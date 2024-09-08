package plugs

import "context"

// retry will
//
// 1. setup panic handler
// 2. restart the plugin n number of times
// 3. in the event of an error write to the pluginErrCh
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

		select {
		case pluginErrCh <- err:
		default:
		}
	}

	for i := 0; i < retries; i++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					writeErr(i, PanicError{name: p.Name()})
				}
			}()

			// is channel close?
			if ctx.Err() != nil {
				return
			}

			err := p.Start(ctx)
			if err != nil {
				writeErr(i, err)
			}
		}()
	}
}

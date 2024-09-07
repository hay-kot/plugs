package plugs

import "context"

// retry will
//
// 1. setup panic handler
// 2. restart the plugin n number of times
// 3. in the event of an error write to the pluginErrCh
func retry(ctx context.Context, p Plugin, retries int, pluginErrCh chan error) {
	for i := 0; i < retries; i++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					select {
					case pluginErrCh <- RetryError{
						name:  p.Name(),
						error: ErrPluginPanic,
						retry: i + 1,
					}:
					default:
					}
				}
			}()

			err := p.Start(ctx)
			if err != nil {
				if i != retries-1 {
					// if retries are not exhausted, write a RetryError
					err = RetryError{
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
		}()
	}
}

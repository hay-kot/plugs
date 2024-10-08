package plugs

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	mu       sync.Mutex
	started  bool
	plugins  []Plugin
	shutdown chan struct{}
	opts     *managerOpts
}

func New(opts ...ManagerOptFunc) *Manager {
	o := &managerOpts{
		signals: []os.Signal{os.Interrupt, syscall.SIGTERM},
		timeout: 5 * time.Second,
		println: func(v ...any) {}, // NOOP
		retries: 0,
	}
	for _, opt := range opts {
		opt(o)
	}

	return &Manager{
		opts:     o,
		shutdown: make(chan struct{}),
	}
}

func (m *Manager) Add(p ...Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.plugins = append(m.plugins, p...)
}

func (m *Manager) AddFunc(name string, start func(ctx context.Context) error) {
	m.Add(PluginFunc(name, start))
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		close(m.shutdown)
	}
}

func (m *Manager) isStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.started
}

func (m *Manager) setStarted(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.started = v
}

// Start start the server with a context provided for cancellation
// if the root context is cancelled, the server signal stops to all
// plugins registered.
//
// Note that a new context is created with the provided signals defined
// when creating the server.
func (m *Manager) Start(ctx context.Context) error {
	if m.isStarted() {
		return ErrManagerAlreadyStarted
	}

	ctx, cancel := signal.NotifyContext(ctx, m.opts.signals...)
	defer cancel()

	// Start Plugins
	var (
		wg          = sync.WaitGroup{}
		pluginErrCh = make(chan error)
		wgChannel   = make(chan struct{})
	)

	wg.Add(len(m.plugins))

	go func() {
		wg.Wait()
		close(wgChannel)
	}()

	for _, p := range m.plugins {
		go func() {
			defer func() {
				wg.Done()
			}()

			retry(ctx, p, m.opts.retries, pluginErrCh)
		}()
	}

	go func() {
		<-m.shutdown
		cancel()
	}()

	m.setStarted(true)
	defer m.setStarted(false)

	// block until the context is done
	for {
		select {
		case <-ctx.Done():
			newTimer := time.NewTimer(m.opts.timeout)
			defer newTimer.Stop()

			m.opts.println("server received signal, shutting down")
			select {
			case <-wgChannel:
				m.opts.println("all plugins have stopped")
				return nil
			case <-newTimer.C:
				m.opts.println("timeout waiting for plugins to stop, exiting")
				return context.DeadlineExceeded
			}
		case err := <-pluginErrCh:
			if isRetryError(err) {
				m.opts.println(err.Error())
				continue
			}

			// ensure panics are printed out.
			panicerr := PanicError{}
			if errors.Is(err, &panicerr) {
				m.opts.println(err.Error())
			}

			return err
		}
	}
}

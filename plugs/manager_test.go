package plugs_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hay-kot/plugs/plugs"
)

type plugResults struct {
	mu          sync.Mutex
	start, stop bool
}

func (p *plugResults) setStart(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.start = v
}

func (p *plugResults) setStop(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stop = v
}

func (p *plugResults) getStart() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.start
}

func (p *plugResults) getStop() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stop
}

func Test_Runner_LifeCycle(t *testing.T) {
	mgr := plugs.New(
		plugs.WithTimeout(3*time.Millisecond),
		plugs.WithPrintln(func(args ...any) {
			t.Log(args)
		}),
	)

	plug1Got := plugResults{}
	plug2Got := plugResults{}
	plug3Got := plugResults{}

	mgr.AddFunc("plug1", func(ctx context.Context) error {
		plug1Got.setStart(true)
		<-ctx.Done()
		plug1Got.setStop(true)
		return nil
	})

	mgr.AddFunc("plug2", func(ctx context.Context) error {
		plug2Got.setStart(true)
		<-ctx.Done()
		plug2Got.setStop(true)
		return nil
	})

	mgr.AddFunc("plug3", func(ctx context.Context) error {
		plug3Got.setStart(true)
		<-ctx.Done()

		// Block forever
		<-make(chan struct{})
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = mgr.Start(ctx)
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	wg.Wait()

	// Plug 1 was started and stopped
	assert(t, plug1Got.getStart(), true)
	assert(t, plug1Got.getStop(), true)

	// Plug 2 was started and stopped
	assert(t, plug2Got.getStart(), true)
	assert(t, plug2Got.getStop(), true)

	// Plug 3 was started and never stopped
	assert(t, plug3Got.getStart(), true)
	assert(t, plug3Got.getStop(), false)
}

func assert[T comparable](t *testing.T, got, expect T) {
	t.Helper()
	if expect != got {
		t.Errorf("expect %v, got %v", expect, got)
	}
}

func Test_Manager_CapturePluginVar(t *testing.T) {
	// This test verifies that each goroutine correctly captures its own plugin variable
	// rather than referencing the loop variable which would lead to all plugins being the same
	mgr := plugs.New()

	// Create a map to track which plugins were started
	startedPlugins := make(map[string]bool)
	mu := sync.Mutex{}

	// Add 5 plugins with different names
	for i := range 5 {
		pluginName := fmt.Sprintf("plugin-%d", i)
		mgr.AddFunc(pluginName, func(ctx context.Context) error {
			mu.Lock()
			startedPlugins[pluginName] = true
			mu.Unlock()
			<-ctx.Done()
			return nil
		})
	}

	// Start the manager with a context that will be canceled soon
	ctx, cancel := context.WithCancel(context.Background())

	// Start the manager in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = mgr.Start(ctx)
	}()

	// Wait for all plugins to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the context to signal shutdown
	cancel()

	// Wait for Start to complete
	wg.Wait()

	// The error might be context.Canceled or nil depending on timing
	// We don't need to assert a specific error value

	// Verify all 5 plugins were started
	if len(startedPlugins) != 5 {
		t.Errorf("expected 5 unique plugins to start, but got %d", len(startedPlugins))
	}

	// Verify each specific plugin was started
	for i := range 5 {
		pluginName := fmt.Sprintf("plugin-%d", i)
		if !startedPlugins[pluginName] {
			t.Errorf("plugin %s was not started", pluginName)
		}
	}
}

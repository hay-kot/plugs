# plugs

Plugin and Startup System for Go Projects

[Go Reference](https://pkg.go.dev/github.com/hay-kot/plugs)

## Install

```bash
go get -u github.com/hay-kot/plugs
```

## Features

- Plugin based architecture for running multiple services in the same binary (ex: run two web servers)
- Graceful shutdowns via context and os signals
- Retry mechinism for restart services that have failed

## Examples

```go
func run() error {
	mgr := plugs.New(
		plugs.WithPrintln(log.Println),
		plugs.WithRestart(3),
		plugs.WithSignals(os.Interrupt, syscall.SIGTERM),
	)

	mgr.Add(
		&server{port: 9091},
		&server{port: 9082},
		&server{port: 9083},
	)

	return mgr.Start(context.Background())
}

type server struct {
	port int
}

func (s *server) Name() string {
	return fmt.Sprintf("server-%d", s.port)
}

func (s *server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		select {
		case <-shutdown:
		case <-ctx.Done():
		}

		_ = server.Shutdown(context.Background())
	}()

	log.Printf("server-%d started", s.port)
	err := server.ListenAndServe()
	log.Printf("server-%d stopped", s.port)
	return err
}
```

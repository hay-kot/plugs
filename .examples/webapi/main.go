package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/hay-kot/plugs/plugs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	mgr := plugs.New(
		plugs.WithPrintln(log.Println),
		plugs.WithRetries(3),
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
	shutdown := make(chan struct{})

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		close(shutdown)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	log.Printf("registering endpoints")
	log.Printf("http://localhost:%d/health", s.port)
	log.Printf("http://localhost:%d/shutdown", s.port)

	// Print endpoints
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

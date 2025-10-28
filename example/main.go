package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/pmatteo/chi_server"
)

func main() {
	// Create logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Configure server
	cfg := chi_server.Config{
		Addr:   ":8080",
		Logger: logger,
	}

	// Create server with routes
	server := chi_server.NewServer(cfg, func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello, World!"))
		})

		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OK"))
		})
	})

	// Run with graceful shutdown on SIGINT/SIGTERM
	ctx := chi_server.WaitForSignal()
	if err := server.Run(ctx); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

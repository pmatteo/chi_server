package chi_server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config holds configuration options for the server.
type Config struct {
	Addr   string
	Logger *slog.Logger
}

// Server defines a reusable HTTP server with slog logging and graceful shutdown.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// RouteConfigurator allows injecting custom routes into the router.
type RouteConfigurator func(r chi.Router)

// NewServer creates a new HTTP server with a configurable heartbeat path and slog logging.
func NewServer(cfg Config, configureRoutes RouteConfigurator) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	r := chi.NewRouter()

	// Common middlewares
	r.Use(middleware.RequestID)
	r.Use(CorrelationID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(RequestLogger(cfg.Logger))

	// Service specific routes
	configureRoutes(r)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: r,
	}
	return &Server{httpServer: srv, logger: cfg.Logger}
}

// Run starts the server and gracefully shuts down on context cancellation.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("server starting", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutdown signal received")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		s.logger.Info("server gracefully stopped")

	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// WaitForSignal returns a context canceled on SIGINT/SIGTERM.
func WaitForSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()
	return ctx
}

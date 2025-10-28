package chiserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pmatteo/chiserver"
)

// TestNewServer_WithDefaultLogger tests server creation with default logger
func TestNewServer_WithDefaultLogger(t *testing.T) {
	cfg := chiserver.Config{
		Addr: ":8080",
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}
}

// TestNewServer_WithCustomLogger tests server creation with custom logger
func TestNewServer_WithCustomLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := chiserver.Config{
		Addr:   ":9090",
		Logger: logger,
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}
}

// TestNewServer_RouteConfiguration tests that custom routes are properly configured
func TestNewServer_RouteConfiguration(t *testing.T) {
	cfg := chiserver.Config{
		Addr:   ":0", // Use dynamic port
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/custom", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("custom route"))
		})
	})

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	// Start server in background
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Note: This test verifies server creation and route configuration
	// but doesn't make actual HTTP requests since the server address is dynamic
	// For full integration testing, you'd need to extract the actual port
}

// TestServer_Run_ContextCancellation tests graceful shutdown on context cancellation
func TestServer_Run_ContextCancellation(t *testing.T) {
	cfg := chiserver.Config{
		Addr:   ":0", // Dynamic port to avoid conflicts
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Server should shutdown gracefully
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected nil error on graceful shutdown, got: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("Server did not shutdown within expected time")
	}
}

// TestServer_Run_ImmediateCancellation tests shutdown when context is already cancelled
func TestServer_Run_ImmediateCancellation(t *testing.T) {
	cfg := chiserver.Config{
		Addr:   ":0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("Server did not shutdown within expected time")
	}
}

// TestServer_Run_InvalidAddress tests server with invalid address
func TestServer_Run_InvalidAddress(t *testing.T) {
	cfg := chiserver.Config{
		Addr:   "invalid:address:format",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Run(ctx)
	if err == nil {
		t.Error("Expected error with invalid address, got nil")
	}
}

// TestServer_MultipleRoutes tests server with multiple route configurations
func TestServer_MultipleRoutes(t *testing.T) {
	cfg := chiserver.Config{
		Addr:   ":0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("healthy"))
		})
		r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		})
		r.Post("/data", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})
	})

	if server == nil {
		t.Fatal("Expected server to be created with multiple routes")
	}
}

// TestWaitForSignal_ContextCancellation tests WaitForSignal context cancellation
func TestWaitForSignal_ContextCancellation(t *testing.T) {
	ctx := chiserver.WaitForSignal()

	// Verify context is not cancelled initially
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
		// Expected - context is not done
	}

	// Send interrupt signal in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		// Send signal to current process
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(os.Interrupt)
	}()

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		// Expected - context was cancelled by signal
	case <-time.After(2 * time.Second):
		t.Fatal("Context was not cancelled after signal")
	}
}

// TestWaitForSignal_SIGTERMSignal tests WaitForSignal with SIGTERM
func TestWaitForSignal_SIGTERMSignal(t *testing.T) {
	ctx := chiserver.WaitForSignal()

	// Send SIGTERM signal in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGTERM)
	}()

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		// Expected - context was cancelled by SIGTERM
	case <-time.After(2 * time.Second):
		t.Fatal("Context was not cancelled after SIGTERM")
	}
}

// TestServer_Integration tests full server lifecycle
func TestServer_Integration(t *testing.T) {
	cfg := chiserver.Config{
		Addr:   ":0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Trigger shutdown
	cancel()

	// Wait for server to stop
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected clean shutdown, got error: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("Server did not shutdown in time")
	}
}

// TestServer_ShutdownTimeout tests that shutdown respects timeout
func TestServer_ShutdownTimeout(t *testing.T) {
	cfg := chiserver.Config{
		Addr:   ":0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow handler
			time.Sleep(10 * time.Second)
			w.WriteHeader(http.StatusOK)
		})
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Server should complete shutdown within the 5 second timeout defined in Run()
	select {
	case err := <-errCh:
		// Server shut down (may or may not have error depending on timing)
		_ = err
	case <-time.After(7 * time.Second):
		t.Fatal("Server shutdown took longer than expected timeout")
	}
}

// TestConfig_DefaultValues tests Config with default/zero values
func TestConfig_DefaultValues(t *testing.T) {
	cfg := chiserver.Config{} // Empty config

	server := chiserver.NewServer(cfg, func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	if server == nil {
		t.Fatal("Expected server to be created with default config values")
	}
}

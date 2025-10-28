# Chi Server

A production-ready HTTP server package built on top of [go-chi/chi](https://github.com/go-chi/chi) with structured logging, request correlation, and graceful shutdown support.

## Features

- 🚀 **Simple Server Setup** - Create production-ready HTTP servers with minimal boilerplate
- 📊 **Structured Logging** - Built-in support for `log/slog` with JSON formatting
- 🔍 **Request Correlation** - Automatic correlation ID generation and propagation
- 🛡️ **Graceful Shutdown** - Context-based shutdown with configurable timeout
- 🔄 **Request Logging** - Automatic logging of all HTTP requests with duration, status, and correlation ID
- 🎯 **Middleware Ready** - Pre-configured with essential middlewares (RequestID, RealIP, Recoverer)

## Installation

```bash
go get your_module_path/chi_server
```

## Quick Start

```go
package main

import (
    "log/slog"
    "os"
    "github.com/go-chi/chi/v5"
    "your_module_path/chi_server"
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
        r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("healthy"))
        })
    })

    // Run with graceful shutdown
    ctx := chi_server.WaitForSignal()
    if err := server.Run(ctx); err != nil {
        logger.Error("server failed", slog.String("error", err.Error()))
        os.Exit(1)
    }
}
```

## Usage

### Creating a Server

The `NewServer` function creates a new HTTP server with pre-configured middlewares:

```go
cfg := chi_server.Config{
    Addr:   ":8080",
    Logger: logger, // Optional: uses slog.Default() if nil
}

server := chi_server.NewServer(cfg, func(r chi.Router) {
    // Define your routes here
    r.Get("/api/users", getUsersHandler)
    r.Post("/api/users", createUserHandler)
})
```

### Middleware Stack

The server comes with the following middlewares pre-configured:

1. **RequestID** - Generates a unique request ID
2. **CorrelationID** - Propagates or generates correlation IDs via `X-Correlation-ID` header
3. **RealIP** - Extracts the real client IP from headers
4. **Recoverer** - Recovers from panics and logs them
5. **RequestLogger** - Logs all HTTP requests with structured logging

### Correlation ID

Correlation IDs are automatically handled:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Get correlation ID from context
    corrID := chi_server.GetCorrID(r.Context())

    // Use it in your logs
    slog.Info("processing request", slog.String("correlation_id", corrID))
}
```

If a client sends an `X-Correlation-ID` header, it will be propagated. Otherwise, a new UUID is generated.

### Custom Header Name

You can customize the correlation ID header name:

```go
chi_server.CorrelationIDHeader = "X-Request-ID"
```

### Graceful Shutdown

The server supports graceful shutdown with a 5-second timeout:

```go
// Option 1: Use WaitForSignal for automatic signal handling
ctx := chi_server.WaitForSignal()
server.Run(ctx)

// Option 2: Use custom context
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Run(ctx)
```

The `WaitForSignal()` function creates a context that cancels on `SIGINT` or `SIGTERM`.

### Request Logging

All requests are automatically logged with the following fields:

- `method` - HTTP method (GET, POST, etc.)
- `path` - Request path
- `status` - Response status code
- `bytes` - Response size in bytes
- `remote` - Client IP address
- `correlation_id` - Request correlation ID
- `duration` - Request processing duration

Example log output:

```json
{
  "time": "2025-10-28T10:30:45Z",
  "level": "INFO",
  "msg": "request",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "bytes": 1234,
  "remote": "192.168.1.1:12345",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "duration": 15000000
}
```

## Configuration

### Config Options

```go
type Config struct {
    Addr   string        // Server address (e.g., ":8080")
    Logger *slog.Logger  // Optional: structured logger
}
```

### Route Configurator

The `RouteConfigurator` function allows you to define your application routes:

```go
type RouteConfigurator func(r chi.Router)
```

You can organize routes using chi's routing features:

```go
server := chi_server.NewServer(cfg, func(r chi.Router) {
    // Group routes with common prefix
    r.Route("/api/v1", func(r chi.Router) {
        r.Get("/users", listUsers)
        r.Post("/users", createUser)
        r.Get("/users/{id}", getUser)
    })

    // Add middleware to specific routes
    r.Group(func(r chi.Router) {
        r.Use(authMiddleware)
        r.Get("/admin", adminHandler)
    })
})
```

## Testing

Run the test suite:

```bash
go test -v ./...
```

Run tests with coverage:

```bash
go test -v -cover ./...
```

## Dependencies

- [go-chi/chi](https://github.com/go-chi/chi) - Lightweight HTTP router
- [google/uuid](https://github.com/google/uuid) - UUID generation
- Standard library `log/slog` - Structured logging

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

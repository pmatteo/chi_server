package chi_server

import (
	"context"
	"net/http"

	"log/slog"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/google/uuid"
)

// Key to use when setting the request ID.
type ctxKeyCorrelationID int

// CorrelationIDKey is the key that holds the unique request ID in a request context.
const CorrelationIDKey ctxKeyCorrelationID = 0

// CorrelationIDHeader is the name of the HTTP Header which contains the request id.
// Exported so that it can be changed by developers
var CorrelationIDHeader = "X-Correlation-ID"

// CorrelationID is a chi middleware that sets or propagates a correlation ID
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Add it to the request context
		ctx := context.WithValue(r.Context(), CorrelationIDKey, correlationID)
		r = r.WithContext(ctx)

		// Also add it to the response header
		w.Header().Set(CorrelationIDHeader, correlationID)

		next.ServeHTTP(w, r)
	})
}

// FromContext extracts correlation ID from context
func GetCorrID(ctx context.Context) string {
	if val, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return val
	}
	return ""
}

// RequestLogger logs each HTTP request using slog.
func RequestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.String("remote", r.RemoteAddr),
				slog.String("correlation_id", GetCorrID(r.Context())),
				slog.Duration("duration", time.Since(start)),
			)
		}
		return http.HandlerFunc(fn)
	}
}

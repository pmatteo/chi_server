package chi_server_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pmatteo/chi_server"
)

// TestCorrelationID_GeneratesNewID tests that a new correlation ID is generated when none is provided
func TestCorrelationID_GeneratesNewID(t *testing.T) {
	handler := chi_server.CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := chi_server.GetCorrID(r.Context())
		if correlationID == "" {
			t.Error("Expected correlation ID to be set in context, got empty string")
		}

		// Validate it's a valid UUID
		if _, err := uuid.Parse(correlationID); err != nil {
			t.Errorf("Expected valid UUID, got: %s, error: %v", correlationID, err)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check response header
	headerID := w.Header().Get(chi_server.CorrelationIDHeader)
	if headerID == "" {
		t.Error("Expected correlation ID to be set in response header")
	}

	if _, err := uuid.Parse(headerID); err != nil {
		t.Errorf("Expected valid UUID in header, got: %s, error: %v", headerID, err)
	}
}

// TestCorrelationID_PropagatesExistingID tests that an existing correlation ID is propagated
func TestCorrelationID_PropagatesExistingID(t *testing.T) {
	expectedID := "test-correlation-id-123"

	handler := chi_server.CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := chi_server.GetCorrID(r.Context())
		if correlationID != expectedID {
			t.Errorf("Expected correlation ID %s, got %s", expectedID, correlationID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(chi_server.CorrelationIDHeader, expectedID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check response header
	headerID := w.Header().Get(chi_server.CorrelationIDHeader)
	if headerID != expectedID {
		t.Errorf("Expected correlation ID %s in header, got %s", expectedID, headerID)
	}
}

// TestCorrelationID_CustomHeader tests that custom header name can be used
func TestCorrelationID_CustomHeader(t *testing.T) {
	// Save original and restore after test
	originalHeader := chi_server.CorrelationIDHeader
	defer func() { chi_server.CorrelationIDHeader = originalHeader }()

	chi_server.CorrelationIDHeader = "X-Custom-Request-ID"
	expectedID := "custom-id-456"

	handler := chi_server.CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := chi_server.GetCorrID(r.Context())
		if correlationID != expectedID {
			t.Errorf("Expected correlation ID %s, got %s", expectedID, correlationID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Custom-Request-ID", expectedID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	headerID := w.Header().Get("X-Custom-Request-ID")
	if headerID != expectedID {
		t.Errorf("Expected custom header to contain %s, got %s", expectedID, headerID)
	}
}

// TestGetCorrID_ReturnsEmptyForMissingID tests that GetCorrID returns empty string when no ID in context
func TestGetCorrID_ReturnsEmptyForMissingID(t *testing.T) {
	ctx := context.Background()
	correlationID := chi_server.GetCorrID(ctx)

	if correlationID != "" {
		t.Errorf("Expected empty string, got %s", correlationID)
	}
}

// TestGetCorrID_ReturnsEmptyForWrongType tests that GetCorrID handles wrong type in context
func TestGetCorrID_ReturnsEmptyForWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), chi_server.CorrelationIDKey, 12345) // wrong type
	correlationID := chi_server.GetCorrID(ctx)

	if correlationID != "" {
		t.Errorf("Expected empty string for wrong type, got %s", correlationID)
	}
}

// TestGetCorrID_ReturnsCorrectID tests that GetCorrID extracts the correct ID
func TestGetCorrID_ReturnsCorrectID(t *testing.T) {
	expectedID := "test-id-789"
	ctx := context.WithValue(context.Background(), chi_server.CorrelationIDKey, expectedID)
	correlationID := chi_server.GetCorrID(ctx)

	if correlationID != expectedID {
		t.Errorf("Expected %s, got %s", expectedID, correlationID)
	}
}

// TestRequestLogger_LogsRequest tests that RequestLogger logs the request with correct fields
func TestRequestLogger_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	middleware := chi_server.RequestLogger(logger)(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/test/path", nil)
	// Add correlation ID to context
	ctx := context.WithValue(req.Context(), chi_server.CorrelationIDKey, "test-corr-id")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	logOutput := buf.String()

	// Verify log contains expected fields
	expectedFields := []string{
		`"method":"POST"`,
		`"path":"/test/path"`,
		`"status":200`,
		`"correlation_id":"test-corr-id"`,
		`"duration"`,
		`"bytes"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(logOutput, field) {
			t.Errorf("Expected log to contain %s, got: %s", field, logOutput)
		}
	}
}

// TestRequestLogger_WithoutCorrelationID tests logging when no correlation ID is present
func TestRequestLogger_WithoutCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	middleware := chi_server.RequestLogger(logger)(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	logOutput := buf.String()

	// Should log empty correlation_id
	if !strings.Contains(logOutput, `"correlation_id":""`) {
		t.Errorf("Expected empty correlation_id in log, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, `"status":404`) {
		t.Errorf("Expected status 404 in log, got: %s", logOutput)
	}
}

// TestRequestLogger_CapturesBytesWritten tests that RequestLogger captures response size
func TestRequestLogger_CapturesBytesWritten(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	responseBody := "Hello, World!"
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	})

	middleware := chi_server.RequestLogger(logger)(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	logOutput := buf.String()

	// Should log bytes written
	if !strings.Contains(logOutput, `"bytes":13`) {
		t.Errorf("Expected bytes:13 in log, got: %s", logOutput)
	}
}

// TestMiddlewareChain_Integration tests both middlewares working together
func TestMiddlewareChain_Integration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := chi_server.GetCorrID(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("correlation_id: " + corrID))
	})

	// Chain middlewares: CorrelationID -> RequestLogger -> handler
	handler := chi_server.CorrelationID(chi_server.RequestLogger(logger)(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/integration", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify response header has correlation ID
	corrID := w.Header().Get(chi_server.CorrelationIDHeader)
	if corrID == "" {
		t.Error("Expected correlation ID in response header")
	}

	// Verify log contains the same correlation ID
	logOutput := buf.String()
	if !strings.Contains(logOutput, `"correlation_id":"`+corrID+`"`) {
		t.Errorf("Expected log to contain correlation ID %s, got: %s", corrID, logOutput)
	}

	// Verify response body contains correlation ID
	if !strings.Contains(w.Body.String(), corrID) {
		t.Errorf("Expected response body to contain correlation ID %s", corrID)
	}
}

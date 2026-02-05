// Package telemetry provides OpenTelemetry initialization and configuration.
package telemetry

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// HTTPMiddleware returns an HTTP middleware that injects/extracts OTEL context.
func HTTPMiddleware(next http.Handler) http.Handler {
	prop := otel.GetTextMapPropagator()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from incoming headers
		carrier := propagation.HeaderCarrier(r.Header)
		ctx := prop.Extract(r.Context(), carrier)

		// Inject trace context into response headers
		prop.Inject(ctx, propagation.HeaderCarrier(w.Header()))

		// Set the context with trace info in the request
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// ResponseWriterWrapper wraps http.ResponseWriter to capture status code.
type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
	Written    int64
}

// NewResponseWriterWrapper creates a new ResponseWriterWrapper.
func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code.
func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the number of bytes written.
func (w *ResponseWriterWrapper) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.Written += int64(n)
	return n, err
}

// Unwrap returns the original ResponseWriter.
func (w *ResponseWriterWrapper) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

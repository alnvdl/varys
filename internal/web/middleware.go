package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

const defaultCSPPolicy = "default-src 'self'; img-src * data:"

// loggingResponseWriter is a wrapper around http.ResponseWriter that stores
// the status code written to the response for logging.
type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// log wraps an HTTP handler and logs the request and response.
func log(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()))
		lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		handler(lw, r)
		slog.Info("response",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("status", fmt.Sprintf("%d", lw.status)),
		)
	}
}

// logpanics wraps an HTTP handler and recovers from panics, logging the error
// and returning a 500 response.
func logpanics(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("panic in handler",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())))
				writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		handler(w, r)
	}
}

// addCSPPolicyHeader adds a Content-Security-Policy header to the response.
func addCSPPolicyHeader(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", defaultCSPPolicy)
		handler(w, r)
	}
}

// verifyCSRFHeaders wraps an HTTP handler and verifies CSRF headers.
func verifyCSRFHeaders(handler http.HandlerFunc) http.HandlerFunc {
	return http.NewCrossOriginProtection().Handler(handler).ServeHTTP
}

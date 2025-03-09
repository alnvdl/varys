package web

import (
	"fmt"
	"log/slog"
	"net/http"
)

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

package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

type errorResponse struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (s *handler) writeErrorResponse(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(errorResponse{
		Code:    fmt.Sprintf("%d", code),
		Name:    http.StatusText(code),
		Message: message,
	})
	if err != nil {
		slog.Error("cannot send error response", slog.String("err", err.Error()))
	}
}

// recover wraps an HTTP handler and recovers from panics, logging the error
// and returning a 500 response.
func (s *handler) recover(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("panic in handler",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())))
				s.writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		handler(w, r)
	}
}

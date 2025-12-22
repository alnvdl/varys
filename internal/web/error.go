package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func writeErrorResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
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

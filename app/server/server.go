package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

func New(mux *http.ServeMux, addr string) *http.Server {
	return &http.Server{
		Handler:      mux,
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func ErrorResponse(w http.ResponseWriter, code int, err error) {
	slog.Error("Error response", "err", err, "code", code)
	w.WriteHeader(code)
	output := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}

	err = json.NewEncoder(w).Encode(output)
	if err != nil {
		slog.Error("Failed writing error response", "err", err)
	}
}

func JSONResponse(w http.ResponseWriter, code int, data any) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		slog.Error("Failed writing response", "err", err, "data", data)
	}
}

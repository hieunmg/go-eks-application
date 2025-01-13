package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

const (
	host = "0.0.0.0"
	port = 8080
)

func main() {

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/app-v1", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"message": "app-v1",
		})
	})

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: allowCORS(withLogger(mux)),
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down the http server")

		if err := s.Shutdown(context.Background()); err != nil {
			slog.Error("failed to shutdown http server", err)
		}
	}()

	slog.Info("start listening...", "address", fmt.Sprintf("%s:%d", host, port))

	if err := s.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to listen and serve", err)
	}
}

func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)

				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	headers := []string{"*"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))

	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))

	slog.Info("preflight request", "http_path", r.URL.Path)
}

func withLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Run request", "http_method", r.Method, "http_url", r.URL)

		h.ServeHTTP(w, r)
	})
}

func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Marshal the data into JSON and write it to the response
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func NotImplemented(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s %s", r.Method, r.URL.String())
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", NotImplemented)
	mux.HandleFunc("POST /register", NotImplemented)
	mux.HandleFunc("POST /login", NotImplemented)

	mux.HandleFunc("GET /posts", NotImplemented)
	mux.HandleFunc("POST /posts", NotImplemented)
	mux.HandleFunc("GET /posts/{id}", NotImplemented)
	mux.HandleFunc("PUT /posts/{id}", NotImplemented)
	mux.HandleFunc("DELETE /posts/{id}", NotImplemented)

	mux.HandleFunc("GET /posts/{id}/comments", NotImplemented)
	mux.HandleFunc("POST /posts/{id}/comments", NotImplemented)

	srv := &http.Server{
		Handler:      mux,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("listen and serve returned err: %v", err)
		}
	}()
	<-ctx.Done()
	log.Println("got interruption signal")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown returned err: %v", err)
	}
}

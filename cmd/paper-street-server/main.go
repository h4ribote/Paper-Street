package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/h4ribote/Paper-Street/internal/api"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	engine := engine.NewEngine(nil)
	handler := api.NewRouter(engine)
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	if err := engine.Shutdown(ctx); err != nil {
		log.Printf("engine shutdown error: %v", err)
	}
}

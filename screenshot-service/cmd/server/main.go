package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const warmupBackoffBase = 500 * time.Millisecond

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "11235"
	}

	// Simple handler with file extension routing
	handler := NewHandler()

	// Start server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		healthURL := "http://127.0.0.1:" + port + "/healthz"
		var lastErr error

		for attempt := 1; attempt <= 5; attempt++ {
			resp, err := http.Get(healthURL)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					log.Printf("Warmup health check OK at %s", healthURL)
					return
				}
				err = fmt.Errorf("unexpected status %s", resp.Status)
			}

			lastErr = err
			time.Sleep(time.Duration(attempt) * warmupBackoffBase)
		}

		if lastErr != nil {
			log.Printf("Warmup health check failed: %v", lastErr)
		}
	}()

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down crawler service...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		if err := handler.Close(); err != nil {
			log.Printf("Crawler shutdown error: %v", err)
		}
	}()

	log.Printf("Crawler service starting on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Crawler service stopped")
}

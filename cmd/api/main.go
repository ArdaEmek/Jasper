package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"s3/internal/server"

	"github.com/quic-go/quic-go/http3"
)

func gracefulShutdown(httpServer *http.Server, http3Server *http3.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("Shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP/1.1 server forced to shutdown with error: %v", err)
	}

	if err := http3Server.Shutdown(ctx); err != nil {
		log.Printf("HTTP/3 server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	httpServer, http3Server := server.NewServer()

	done := make(chan bool, 1)
	go gracefulShutdown(httpServer, http3Server, done)

	// Start HTTP/3 server in a goroutine
	go func() {
		log.Printf("HTTP/3 Server is running on %s\n", http3Server.Addr)
		err := http3Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP/3 Server error: %v\n", err)
		}
	}()

	// Start HTTP/1.1 server
	log.Printf("Server is running on %s\n", httpServer.Addr)
	err := httpServer.ListenAndServeTLS("", "")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP/1.1 Server error: %v\n", err)
	}

	<-done
}

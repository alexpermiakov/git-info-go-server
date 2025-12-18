package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// --- METADATA (Baked in via ldflags) ---
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mux := http.NewServeMux()

	mux.HandleFunc("/", versionHandler)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,   // Max time to read request
		WriteTimeout: 10 * time.Second,  // Max time to write response
		IdleTimeout:  120 * time.Second, // Keep-alive
	}

	// 4. Graceful Shutdown Logic
	// This ensures the pod doesn't kill active requests during a rollout
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("starting production server", "port", "8080", "version", Version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-done // Wait for SIGTERM/SIGINT
	logger.Info("shutting down gracefully...")

	// Give the server 5 seconds to finish current requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown failed", "error", err)
	}
	logger.Info("server stopped")
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := VersionInfo{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}

	if err := json.NewEncoder(w).Encode(info); err != nil {
		slog.Error("failed to encode version", "error", err)
	}
}

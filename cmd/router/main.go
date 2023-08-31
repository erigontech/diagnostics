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

	"github.com/ledgerwatch/diagnostics/api"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Use of system calls SIGINT and SIGTERM signals that cause a gracefully  stop.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	// Initialize Services
	cache := sessions.NewCache(5, 5)
	ErigonNodeClient := erigon_node.NewErigonNodeClient()
	uiSessions := sessions.NewUISession(cache)

	// Passing in the services to REST layer
	handlers := api.NewHandler(
		api.APIServices{
			UISessions:   uiSessions,
			ErigonNode:   ErigonNodeClient,
			StoreSession: &cache,
		})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", clientAddr, clientPort),
		Handler:           handlers,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 1 * time.Minute,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Graceful and eager terminations
	switch s := <-signalCh; s {
	case syscall.SIGTERM:
		log.Println("Terminating gracefully.")
		if err := srv.Shutdown(context.Background()); err != http.ErrServerClosed {
			log.Println("Failed to shutdown server:", err)
		}
	case syscall.SIGINT:
		log.Println("Terminating eagerly.")
		os.Exit(-int(syscall.SIGINT))
	}

}

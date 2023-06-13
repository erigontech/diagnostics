package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ledgerwatch/diagnostics/api"
	"github.com/ledgerwatch/diagnostics/assets"
	"github.com/ledgerwatch/diagnostics/internal/erigon_node"
	"github.com/ledgerwatch/diagnostics/internal/logging"
	"github.com/ledgerwatch/diagnostics/internal/sessions"
)

func main() {
	// Retrieving flags and configurations
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//set up logger to implement log rotation
	logging.SetupLogger(logDirPath, logFileName, logFileSizeMax, logFilesAgeMax, logFilesMax, logCompress)

	// Use of system calls SIGINT and SIGTERM signals that cause a gracefully  stop.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	// Initialize Services
	cache := sessions.NewCache(5, 5)
	ErigonNodeClient := erigon_node.NewErigonNodeClient()
	uiSessions := sessions.NewUISession(cache)
	htmlTemplates, err := template.ParseFS(assets.Templates, "template/*.html")
	if err != nil {
		log.Fatalf("parsing session.html template: %v", err)
	}

	// Initializes and adds the provided certificate to the pool, to be used in TLS config
	certPool := x509.NewCertPool()
	for _, caCertFile := range caCertFiles {
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			log.Fatalf("reading server certificate: %v", err)
		}
		certPool.AppendCertsFromPEM(caCert)
	}

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	// Passing in the services to REST layer
	handlers := api.NewHandler(
		api.APIServices{
			UISessions:    uiSessions,
			ErigonNode:    ErigonNodeClient,
			StoreSession:  &cache,
			HtmlTemplates: htmlTemplates,
		})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", listenAddr, listenPort),
		Handler:           handlers,
		MaxHeaderBytes:    1 << 20,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 1 * time.Minute,
	}
	go func() {
		if err := srv.ListenAndServeTLS(serverCertFile, serverKeyFile); err != http.ErrServerClosed {
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

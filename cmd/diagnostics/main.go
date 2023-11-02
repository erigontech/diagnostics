package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ledgerwatch/diagnostics/api"
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
	cache, err := sessions.NewCache(5, 5)

	if err != nil {
		log.Fatalf("session cache creation  failed: %v", err)
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
			StoreSession: cache,
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

	if routerPort > 0 {
		srv := &http.Server{
			Addr:              fmt.Sprintf("%s:%d", listenAddr, routerPort),
			Handler:           handlers,
			MaxHeaderBytes:    1 << 20,
			ReadHeaderTimeout: 1 * time.Minute,
		}

		go func() {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()

	}

	r := http.NewServeMux()

	r.HandleFunc("/network", index)
	r.HandleFunc("/logs", index)
	r.HandleFunc("/chain", index)
	r.HandleFunc("/data", index)
	r.HandleFunc("/debug", index)
	r.HandleFunc("/testing", index)
	r.HandleFunc("/performance", index)
	r.HandleFunc("/documentation", index)
	r.HandleFunc("/admin", index)
	buildHandler := http.FileServer(http.Dir("./../../web/dist"))
	r.Handle("/", buildHandler)

	server := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	/*http.Handle("/", web.UI)

	server := &http.Server{
		Addr:              ":8000",
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()*/

	open("http://localhost:8000")

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

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./../../web/dist/index.html")
}

// open opens the specified URL in the default browser of the user.
func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

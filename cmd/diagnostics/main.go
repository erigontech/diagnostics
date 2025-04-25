package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/erigontech/diagnostics/api"
	"github.com/erigontech/diagnostics/internal/logging"
	"github.com/erigontech/diagnostics/internal/sessions"
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

	// Passing in the services to REST layer
	handlers := api.NewHandler(
		api.APIServices{
			StoreSession: cache,
		})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", listenAddr, listenPort),
		Handler:           handlers,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 1 * time.Minute,
	}

	go func() {
		err := srv.ListenAndServe()

		if err != nil {
			log.Fatal(err)
		}
	}()

	printUIVersion()

	fmt.Printf("Diagnostics UI is running on http://%s:%d\n", listenAddr, listenPort)
	//open(fmt.Sprintf("http://%s:%d", listenAddr, listenPort))

	// Graceful and eager terminations
	switch s := <-signalCh; s {
	case syscall.SIGTERM:
		log.Println("Terminating gracefully.")
		if err := srv.Shutdown(context.Background()); !errors.Is(err, http.ErrServerClosed) {
			log.Println("Failed to shutdown server:", err)
		}
	case syscall.SIGINT:
		log.Println("Terminating eagerly.")
		os.Exit(-int(syscall.SIGINT))
	}
}

func printUIVersion() {
	packagePath := "github.com/erigontech/erigonwatch"
	version, err := GetPackageVersion(packagePath)
	if err == nil {
		fmt.Printf("Diagnostics version: %s\n", version)
	}
}

// open opens the specified URL in the default browser of the user.
/*func open(url string) error {
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
}*/

// GetPackageVersion returns the version of a package from the go.mod file.
func GetPackageVersion(packagePath string) (string, error) {
	file, err := os.Open("./../../go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t"+packagePath+" ") {
			// Extract the version from the line
			split := strings.Split(line, " ")
			version := strings.TrimSpace(split[1])
			return version, nil
		}
	}

	return "", fmt.Errorf("package not found in go.mod file: %s", packagePath)
}

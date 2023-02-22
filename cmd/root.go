package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile    string
	listenAddr string
	listenPort int

	rootCmd = &cobra.Command{
		Use:   "diagnostics",
		Short: "Diagnostics web server for Erigon support",
		Long:  `Diagnostics web server for Erigon support`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return webServer()
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.Flags().StringVar(&listenAddr, "addr", "localhost", "network interface to listen on")
	rootCmd.Flags().IntVar(&listenPort, "port", 8080, "port to listen on")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cobra")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

var validPath = regexp.MustCompile("^/([a-zA-Z0-9]+)/(giveRequests|takeResponses|show)$")

func (dh *DiagHandler) giveRequestsHandler(w http.ResponseWriter, r *http.Request) {
loop:
	for {
		select {
		case request := <-dh.requestCh:
			fmt.Fprintf(w, "%s\n", request)
		default:
			break loop
		}
	}
}

func takeResponsesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Take responses %s", r.URL.Path)
}

func showHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Show %s", r.URL.Path)
}

type DiagHandler struct {
	// Requests can be of arbitrary types, they get converted to a string and sent to the support subcommand
	requestCh chan interface{}
}

func (dh *DiagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	// User token is the first submatch m[1]
	// TODO: validate the token
	switch m[2] {
	case "giveRequests":
		dh.giveRequestsHandler(w, r)
	case "takeResponses":
		takeResponsesHandler(w, r)
	case "show":
		showHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (dh *DiagHandler) RequestNodeInfo() {
	dh.requestCh <- "nodeInfo"
}

func webServer() error {
	sigs := make(chan os.Signal, 1)
	interruptCh := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		interruptCh <- true
	}()
	dh := &DiagHandler{requestCh: make(chan interface{}, 1024)}
	s := &http.Server{
		Addr:           ":8080",
		Handler:        dh,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	// Initial request for node info
	dh.RequestNodeInfo()
	interrupt := false
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Printf("Running server problem: %v", err)
		}
	}()
	pollInterval := 500 * time.Millisecond
	pollEvery := time.NewTicker(pollInterval)
	defer pollEvery.Stop()
	for !interrupt {
		select {
		case interrupt = <-interruptCh:
			log.Printf("interrupted, please wait for cleanup")
		case <-pollEvery.C:
		}
	}
	s.Shutdown(context.Background())
	return nil
}

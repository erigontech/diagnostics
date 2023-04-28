package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/google/btree"
	"github.com/ledgerwatch/diagnostics/assets"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile        string
	listenAddr     string
	listenPort     int
	serverKeyFile  string
	serverCertFile string
	caCertFiles    []string
	insecure	   bool

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
	rootCmd.Flags().StringVar(&serverKeyFile, "tls.key", "", "path to server TLS key")
	rootCmd.MarkFlagRequired("tls.key")
	rootCmd.Flags().StringVar(&serverCertFile, "tls.cert", "", "paths to server TLS certificates")
	rootCmd.MarkFlagRequired("tls.cert")
	rootCmd.Flags().StringSliceVar(&caCertFiles, "tls.cacerts", []string{}, "comma-separated list of paths to and CAs TLS certificates")
	rootCmd.Flags().BoolVar(&insecure, "insecure", false, "whether to use insecure PIN generation for testing purposes (default is false)")
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

const successLine = "SUCCESS"

// NodeSession corresponds to one Erigon node connected via "erigon support" bridge to an operator
type NodeSession struct {
	lock           sync.Mutex
	sessionPin     uint64
	Connected      bool
	RemoteAddr     string
	SupportVersion uint64            // Version of the erigon support command
	requestCh      chan *NodeRequest // Channel for incoming metrics requests
}

func (ns *NodeSession) connect(remoteAddr string) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	ns.Connected = true
	ns.RemoteAddr = remoteAddr
}

func (ns *NodeSession) disconnect() {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	ns.Connected = false
}

type UiNodeSession struct {
	SessionName string
	SessionPin  uint64
}

type UiSession struct {
	lock               sync.Mutex
	Session            bool
	SessionPin         uint64
	SessionName        string
	Errors             []string // Transient field - only filled for the time of template execution
	currentSessionName string
	NodeS              *NodeSession // Transient field - only filled for the time of template execution
	uiNodeTree         *btree.BTreeG[UiNodeSession]
	UiNodes            []UiNodeSession // Transient field - only filled forthe time of template execution
}

func (uiSession *UiSession) appendError(err string) {
	uiSession.lock.Lock()
	defer uiSession.lock.Unlock()
	uiSession.Errors = append(uiSession.Errors, err)
}

func webServer() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux := http.NewServeMux()
	uiTemplate, err := template.ParseFS(assets.Templates, "template/*.html")
	if err != nil {
		return fmt.Errorf("parsing session.html template: %v", err)
	}
	uih := &UiHandler{
		nodeSessions: map[uint64]*NodeSession{},
		uiSessions:   map[string]*UiSession{},
		uiTemplate:   uiTemplate,
	}
	mux.Handle("/script/", http.FileServer(http.FS(assets.Scripts)))
	mux.Handle("/ui/", uih)
	bh := &BridgeHandler{uih: uih}
	mux.Handle("/support/", bh)
	certPool := x509.NewCertPool()
	for _, caCertFile := range caCertFiles {
		caCert, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			return fmt.Errorf("reading server certificate: %v", err)
		}
		certPool.AppendCertsFromPEM(caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", listenAddr, listenPort),
		Handler:        mux,
		MaxHeaderBytes: 1 << 20,
		ConnContext: func(_ context.Context, _ net.Conn) context.Context {
			return ctx
		},
		TLSConfig: tlsConfig,
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
		s.Shutdown(ctx)
	}()
	if err = s.ListenAndServeTLS(serverCertFile, serverKeyFile); err != nil {
		select {
		case <-ctx.Done():
			return nil
		default:
			return fmt.Errorf("running server: %v", err)
		}
	}
	return nil
}

package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ledgerwatch/diagnostics/assets"
	"github.com/ledgerwatch/diagnostics/pkg/handler"
)

var (
	// Used for flags.
	cfgFile         string
	listenAddr      string
	listenPort      int
	serverKeyFile   string
	serverCertFile  string
	caCertFiles     []string
	insecure        bool
	maxNodeSessions int
	maxUISessions   int

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
	_ = rootCmd.MarkFlagRequired("tls.key")
	rootCmd.Flags().StringVar(&serverCertFile, "tls.cert", "", "paths to server TLS certificates")
	_ = rootCmd.MarkFlagRequired("tls.cert")
	rootCmd.Flags().StringSliceVar(&caCertFiles, "tls.cacerts", []string{}, "comma-separated list of paths to and CAs TLS certificates")
	rootCmd.Flags().BoolVar(&insecure, "insecure", false, "whether to use insecure PIN generation for testing purposes (default is false)")
	rootCmd.Flags().IntVar(&maxNodeSessions, "node.sessions", 5000, "maximum number of node sessions to allow")
	rootCmd.Flags().IntVar(&maxUISessions, "ui.sessions", 5000, "maximum number of UI sessions to allow")
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

func webServer() error {

	uih, err := handler.NewUIHandler(handler.UIHandlerConf{
		MaxNodeSessions: maxNodeSessions,
		MaxUISessions:   maxUISessions,
		UITmplPath:      "template/*.html",
	})
	if err != nil {
		return errors.Wrap(err, "failed providing UIHandler")
	}
	bh := handler.NewBridgeHandler(uih)

	mux := http.NewServeMux()
	mux.Handle("/script/", http.FileServer(http.FS(assets.Scripts)))
	mux.Handle("/ui/", uih)
	mux.Handle("/support/", bh)

	certPool := x509.NewCertPool()
	for _, f := range caCertFiles {
		caCert, err := os.ReadFile(f)
		if err != nil {
			return errors.Wrap(err, "failed reading server certificate")
		}
		certPool.AppendCertsFromPEM(caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &http.Server{
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
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown server due to error:%s", err.Error())
		}
	}()

	log.Printf("Starting diagnostics Server listening at %s:%d", listenAddr, listenPort)
	if err = srv.ListenAndServeTLS(serverCertFile, serverKeyFile); err != nil {
		select {
		case <-ctx.Done():
			return nil
		default:
			return errors.Wrap(err, "failed running server")
		}
	}
	return nil
}

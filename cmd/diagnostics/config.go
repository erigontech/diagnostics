package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
			return nil
		},
	}
)

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

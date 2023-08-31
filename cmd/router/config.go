package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile    string
	clientAddr string
	clientPort int
	nodeAddr   string
	nodePort   int

	rootCmd = &cobra.Command{
		Use:   "router",
		Short: "Diagnostics router for Erigon support",
		Long:  "Diagnostics router for Erigon support",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.Flags().StringVar(&clientAddr, "client-addr", "localhost", "network interface to listen on for client connections")
	rootCmd.Flags().IntVar(&clientPort, "port", 8080, "port to listen on")
	rootCmd.Flags().StringVar(&nodeAddr, "addr", "localhost", "network interface to listen on for node connections")
	rootCmd.Flags().IntVar(&nodePort, "port", 9080, "port to listen on")
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

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
	logDirPath      string //path of directory to save log file
	logFileName     string //name of log file with format
	logFileSizeMax  int    //maximum file size for 1 log file
	logFilesMax     int    //maximum number of backup log files the specified directory can have
	logFilesAgeMax  int    //maximum number of days a log file will persist in file
	logCompress     bool   //whether to compress old log files

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
	rootCmd.Flags().StringVar(&logDirPath, "log.dir.path", "./logs", "directory path to store logs data")
	rootCmd.Flags().StringVar(&logFileName, "log.file.name", "diagnostics.log", "directory path to store logs data")
	rootCmd.Flags().IntVar(&logFileSizeMax, "log.file.size.max", 100, "maximum size of log file in mega bytes to allow")
	rootCmd.Flags().IntVar(&logFilesAgeMax, "log.file.age.max", 28, "maximum age in days a log file can persist in system")
	rootCmd.Flags().IntVar(&logFilesMax, "log.max.backup", 5, "maximum number of log files that can persist")
	rootCmd.Flags().BoolVar(&logCompress, "log.compress", false, "whether to compress historical log files or not")
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

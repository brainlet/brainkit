package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	jsonOutput bool
	verbose    bool
	timeout    time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "brainkit",
	Short: "brainkit — embeddable runtime for AI agents",
	Long:  "A CLI tool for running, deploying, and managing brainkit agent runtimes.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./brainkit.yaml or ~/.brainkit/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output JSON instead of human-readable text")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose logging")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "command timeout")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("brainkit")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home + "/.brainkit")
		}
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Config:", err)
		}
	}
}

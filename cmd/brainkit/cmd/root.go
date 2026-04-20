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

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "brainkit",
		Short: "brainkit — embeddable runtime for AI agents",
		Long:  "A CLI tool for running, deploying, and managing brainkit agent runtimes.",
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./brainkit.yaml or ~/.brainkit/config.yaml)")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output JSON instead of human-readable text")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose logging")
	root.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "command timeout")

	cobra.OnInitialize(initConfig)

	root.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newStartCmd(),
		newDeployCmd(),
		newTeardownCmd(),
		newCallCmd(),
		newSendCmd(),
		newEvalCmd(),
		newInspectCmd(),
		newSecretsCmd(),
		newPluginCmd(),
		newTestCmd(),
		newNewCmd(),
	)

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
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

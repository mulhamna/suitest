package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version = "dev"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "suitest",
	Short: "AI-powered testing agent CLI",
	Long: `suitest is an open-source AI-powered testing agent that automatically
generates, executes, and debugs tests for your project.

Provider-agnostic: works with Claude, OpenAI, OpenRouter, Ollama, or any
OpenAI-compatible provider.`,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.suitest/config.yaml)")
	rootCmd.PersistentFlags().String("provider", "auto", "AI provider: auto, claude, openai, openrouter, ollama")
	rootCmd.PersistentFlags().String("model", "", "Model name/slug (uses provider default if not set)")
	rootCmd.PersistentFlags().String("base-url", "", "Custom OpenAI-compatible base URL")

	viper.BindPFlag("provider", rootCmd.PersistentFlags().Lookup("provider"))
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(serveCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(fmt.Sprintf("%s/.suitest", home))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Error reading config:", err)
		}
	}
}

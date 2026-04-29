package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/report"
	"github.com/mulhamna/suitest/internal/runners"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run [path]",
	Short: "Run AI-powered tests on a project",
	Long: `Run the full agent loop: discover project, generate test plan,
execute tests, fix failures, and produce a report.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTests,
}

func init() {
	runCmd.Flags().String("mode", "auto", "Test mode: auto, browser, api, unit")
	runCmd.Flags().Bool("fix", false, "Auto-apply AI-suggested fixes to source files")
	runCmd.Flags().Bool("dry-run", false, "Show test plan only, do not execute")
	runCmd.Flags().String("output", "terminal", "Output format: terminal, json, markdown")
	runCmd.Flags().Int("max-retries", 3, "Maximum fix retry attempts per test")
	runCmd.Flags().Int("concurrency", 4, "Number of parallel test runners")

	viper.BindPFlag("mode", runCmd.Flags().Lookup("mode"))
	viper.BindPFlag("fix", runCmd.Flags().Lookup("fix"))
	viper.BindPFlag("dry_run", runCmd.Flags().Lookup("dry-run"))
	viper.BindPFlag("output", runCmd.Flags().Lookup("output"))
	viper.BindPFlag("agent.max_retries", runCmd.Flags().Lookup("max-retries"))
	viper.BindPFlag("agent.concurrency", runCmd.Flags().Lookup("concurrency"))
}

func runTests(cmd *cobra.Command, args []string) error {
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config from flags
	if mode, _ := cmd.Flags().GetString("mode"); mode != "auto" {
		cfg.Mode = mode
	}
	if fix, _ := cmd.Flags().GetBool("fix"); fix {
		cfg.Agent.AutoFix = true
	}
	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		cfg.DryRun = true
	}
	if output, _ := cmd.Flags().GetString("output"); output != "terminal" {
		cfg.Output = output
	}
	if maxRetries, _ := cmd.Flags().GetInt("max-retries"); maxRetries != 3 {
		cfg.Agent.MaxRetries = maxRetries
	}
	if concurrency, _ := cmd.Flags().GetInt("concurrency"); concurrency != 4 {
		cfg.Agent.Concurrency = concurrency
	}

	// Override provider from persistent flags
	if p := viper.GetString("provider"); p != "" && p != "auto" {
		cfg.DefaultProvider = p
	}
	if m := viper.GetString("model"); m != "" {
		if cfg.Providers[cfg.DefaultProvider] == nil {
			cfg.Providers[cfg.DefaultProvider] = &config.ProviderConfig{}
		}
		cfg.Providers[cfg.DefaultProvider].Model = m
	}
	if bu := viper.GetString("base_url"); bu != "" {
		if cfg.Providers[cfg.DefaultProvider] == nil {
			cfg.Providers[cfg.DefaultProvider] = &config.ProviderConfig{}
		}
		cfg.Providers[cfg.DefaultProvider].BaseURL = bu
	}

	// Initialize provider
	provider, err := providers.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize provider: %w", err)
	}

	fmt.Printf("Using provider: %s\n", provider.Name())

	// Detect runner
	mode := cfg.Mode
	if mode == "auto" || mode == "" {
		detected, err := runners.Detect(absPath)
		if err != nil {
			return fmt.Errorf("failed to detect runner: %w", err)
		}
		mode = detected
	}
	fmt.Printf("Test mode: %s\n", mode)
	fmt.Printf("Target path: %s\n", absPath)

	// Create agent
	a := agent.New(agent.Config{
		Provider:    provider,
		Path:        absPath,
		Mode:        mode,
		MaxRetries:  cfg.Agent.MaxRetries,
		Concurrency: cfg.Agent.Concurrency,
		AutoFix:     cfg.Agent.AutoFix,
		DryRun:      cfg.DryRun,
	})

	ctx := context.Background()
	result, err := a.Run(ctx)
	if err != nil {
		return fmt.Errorf("agent run failed: %w", err)
	}

	// Output report
	outputFmt := cfg.Output
	if outputFmt == "" {
		outputFmt = "terminal"
	}

	switch outputFmt {
	case "json":
		reporter := report.NewJSONReporter()
		return reporter.Write(os.Stdout, result)
	case "markdown":
		reporter := report.NewMarkdownReporter()
		return reporter.Write(os.Stdout, result)
	default:
		reporter := report.NewTerminalReporter()
		return reporter.Write(os.Stdout, result)
	}
}

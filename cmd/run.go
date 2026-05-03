package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/catalog"
	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/report"
	"github.com/spf13/cobra"
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
	runCmd.Flags().String("target", "", "Saved target name to run")
	runCmd.Flags().String("scenario-set", "", "Scenario set name for saved target runs")
	runCmd.Flags().Bool("yes", false, "Skip saved target confirmation prompt")
}

func runTests(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := applyRunOverrides(cmd, cfg); err != nil {
		return err
	}

	provider, err := initProvider(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Using provider: %s\n", provider.Name())

	targetName, _ := cmd.Flags().GetString("target")
	if targetName == "" && len(args) > 0 {
		targetName = args[0]
	}
	if targetName != "" {
		if target, ok := resolveRunTarget(targetName); ok {
			return runSavedTarget(cmd, cfg, provider, target)
		}
	}

	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}
	absPath, err := resolvePath(targetPath)
	if err != nil {
		return err
	}

	a := agent.New(agent.Config{
		Provider:    provider,
		Path:        absPath,
		Mode:        cfg.Mode,
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

	return writeRunOutput(outputFmt, result)
}

func runSavedTarget(cmd *cobra.Command, cfg *config.Config, provider providers.Provider, target *catalog.Target) error {
	setName, _ := cmd.Flags().GetString("scenario-set")
	if setName == "" {
		setName = target.ScenarioSet
	}
	if setName == "" {
		setName = "default"
	}

	scenarioSet, err := catalog.LoadScenarioSet(target.Name, setName)
	if err != nil {
		return fmt.Errorf("load scenario set %q: %w", setName, err)
	}

	assumeYes, _ := cmd.Flags().GetBool("yes")
	if !scenarioSet.Approved && !assumeYes {
		fmt.Printf("Scenario set %q for target %q is still a draft.\n", scenarioSet.Name, target.Name)
		approved, err := promptConfirm("Continue with this draft scenario set?", false)
		if err != nil {
			return err
		}
		if !approved {
			return fmt.Errorf("run cancelled; approve it first with 'suitest scenario approve %s %s'", target.Name, scenarioSet.Name)
		}
	}
	ok, err := confirmSavedRun(context.Background(), target, scenarioSet, assumeYes)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("Run cancelled.")
		return nil
	}

	runMode := scenarioSet.Mode
	if cfg.Mode != "" && cfg.Mode != "auto" {
		runMode = cfg.Mode
	}

	a := agent.New(agent.Config{
		Provider:    provider,
		Path:        target.Path,
		Mode:        runMode,
		MaxRetries:  cfg.Agent.MaxRetries,
		Concurrency: cfg.Agent.Concurrency,
		AutoFix:     cfg.Agent.AutoFix,
		DryRun:      cfg.DryRun,
		TargetName:  target.Name,
		TargetType:  target.Type,
		EntryURL:    target.URL,
		SeedCurl:    target.Curl,
		Expectation: target.Expectation,
		Plans:       scenarioSet.Plans,
	})

	result, err := a.Run(context.Background())
	if err != nil {
		return fmt.Errorf("agent run failed: %w", err)
	}
	return writeRunOutput(cfg.Output, result)
}

func writeRunOutput(outputFmt string, result *agent.RunResult) error {
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

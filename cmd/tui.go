package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/catalog"
	"github.com/mulhamna/suitest/internal/report"
	"github.com/mulhamna/suitest/internal/storage"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open an interactive terminal view",
	RunE:  runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
	for {
		fmt.Print(suitestBanner)
		fmt.Println("Interactive view")
		fmt.Println("1. Targets")
		fmt.Println("2. Scenario sets")
		fmt.Println("3. Run history")
		fmt.Println("4. Show run report")
		fmt.Println("5. Exit")

		choice, err := promptText("Choose a section", "5")
		if err != nil {
			return err
		}

		switch strings.TrimSpace(choice) {
		case "1":
			if err := showTargetsView(); err != nil {
				return err
			}
		case "2":
			if err := showScenarioSetsView(); err != nil {
				return err
			}
		case "3":
			if err := showRunHistoryView(); err != nil {
				return err
			}
		case "4":
			if err := showRunReportView(); err != nil {
				return err
			}
		case "5":
			return nil
		default:
			fmt.Println("Unknown option.")
		}

		if _, err := promptText("Press enter to continue", ""); err != nil {
			return err
		}
		fmt.Println()
	}
}

func showTargetsView() error {
	targets, err := catalog.ListTargets()
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Println("No saved targets yet.")
		return nil
	}
	for _, target := range targets {
		fmt.Printf("- %s [%s]\n", target.Name, target.Type)
		fmt.Printf("  path: %s\n", target.Path)
		if target.URL != "" {
			fmt.Printf("  url: %s\n", target.URL)
		}
		if target.Expectation != "" {
			fmt.Printf("  expectation: %s\n", target.Expectation)
		}
		fmt.Printf("  default scenario: %s\n", target.ScenarioSet)
	}
	return nil
}

func showScenarioSetsView() error {
	targets, err := catalog.ListTargets()
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Println("No saved targets yet.")
		return nil
	}
	targetName, err := promptText("Target name", targets[0].Name)
	if err != nil {
		return err
	}
	sets, err := catalog.ListScenarioSets(targetName)
	if err != nil {
		return err
	}
	if len(sets) == 0 {
		fmt.Println("No saved scenario sets for that target.")
		return nil
	}
	for _, set := range sets {
		status := "draft"
		if set.Approved {
			status = "approved"
		}
		fmt.Printf("- %s (%s)\n", set.Name, status)
		for index, plan := range set.Plans {
			fmt.Printf("  %d. %s\n", index+1, plan.Name)
		}
	}
	return nil
}

func showRunHistoryView() error {
	summaries, err := storage.ListRunSummaries(20)
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		fmt.Println("No run history yet.")
		return nil
	}
	for _, summary := range summaries {
		status := "PASS"
		if summary.Failed > 0 {
			status = "FAIL"
		}
		if summary.DryRun {
			status = "DRY"
		}
		fmt.Printf("- %s [%s] %s\n", summary.RunID, status, summary.Path)
		fmt.Printf("  mode: %s | provider: %s | passed: %d | failed: %d | total: %d\n",
			summary.Mode, summary.Provider, summary.Passed, summary.Failed, summary.TotalTests)
	}
	return nil
}

func showRunReportView() error {
	runID, err := promptText("Run ID", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" {
		fmt.Println("Run ID is required.")
		return nil
	}
	data, err := storage.LoadRunReportData(runID)
	if err != nil {
		return err
	}
	var result agent.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	reporter := report.NewTerminalReporter()
	return reporter.Write(os.Stdout, &result)
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/report"
	"github.com/mulhamna/suitest/internal/storage"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Print or export the last run report",
	Long:  `Display the results from the last suitest run. Supports terminal, JSON, and Markdown output formats.`,
	RunE:  showReport,
}

var reportHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List recent run history",
	RunE:  showReportHistory,
}

var reportShowCmd = &cobra.Command{
	Use:   "show <run-id>",
	Short: "Show a specific saved run report",
	Args:  cobra.ExactArgs(1),
	RunE:  showReportByID,
}

func init() {
	reportCmd.Flags().String("format", "terminal", "Output format: terminal, json, markdown")
	reportHistoryCmd.Flags().Int("limit", 20, "Maximum number of runs to show")
	reportShowCmd.Flags().String("format", "terminal", "Output format: terminal, json, markdown")
	reportCmd.AddCommand(reportHistoryCmd)
	reportCmd.AddCommand(reportShowCmd)
}

func showReport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")

	data, err := storage.LoadLastReportData()
	if err != nil {
		return fmt.Errorf("no report found. Run 'suitest run' first: %w", err)
	}
	return writeReportData(format, data)
}

func showReportHistory(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	summaries, err := storage.ListRunSummaries(limit)
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		fmt.Println("No run history found yet.")
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
		fmt.Printf("  started: %s\n", summary.StartedAt)
	}
	return nil
}

func showReportByID(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	data, err := storage.LoadRunReportData(args[0])
	if err != nil {
		return fmt.Errorf("load run %s: %w", args[0], err)
	}
	return writeReportData(format, data)
}

func writeReportData(format string, data []byte) error {
	var result agent.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parse report: %w", err)
	}

	switch format {
	case "json":
		reporter := report.NewJSONReporter()
		return reporter.Write(os.Stdout, &result)
	case "markdown":
		reporter := report.NewMarkdownReporter()
		return reporter.Write(os.Stdout, &result)
	default:
		reporter := report.NewTerminalReporter()
		return reporter.Write(os.Stdout, &result)
	}
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/report"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Print or export the last run report",
	Long:  `Display the results from the last suitest run. Supports terminal, JSON, and Markdown output formats.`,
	RunE:  showReport,
}

func init() {
	reportCmd.Flags().String("format", "terminal", "Output format: terminal, json, markdown")
}

func showReport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")

	// Load last report from disk
	result, err := loadLastReport()
	if err != nil {
		return fmt.Errorf("no report found. Run 'suitest run' first: %w", err)
	}

	switch format {
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

func loadLastReport() (*agent.RunResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	reportPath := filepath.Join(home, ".suitest", "last-report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("could not read report at %s: %w", reportPath, err)
	}

	var result agent.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("could not parse report: %w", err)
	}

	return &result, nil
}

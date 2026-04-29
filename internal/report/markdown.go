package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/mulhamna/suitest/internal/agent"
)

// MarkdownReporter writes a Markdown report.
type MarkdownReporter struct{}

// NewMarkdownReporter creates a MarkdownReporter.
func NewMarkdownReporter() *MarkdownReporter {
	return &MarkdownReporter{}
}

func (r *MarkdownReporter) Write(w io.Writer, result *agent.RunResult) error {
	fmt.Fprintln(w, "# suitest Test Report")
	fmt.Fprintln(w)

	// Metadata table
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Field | Value |")
	fmt.Fprintln(w, "|-------|-------|")
	fmt.Fprintf(w, "| Path | `%s` |\n", result.Path)
	fmt.Fprintf(w, "| Mode | %s |\n", result.Mode)
	fmt.Fprintf(w, "| Provider | %s |\n", result.Provider)
	fmt.Fprintf(w, "| Started | %s |\n", result.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "| Duration | %s |\n", result.FinishedAt.Sub(result.StartedAt))
	fmt.Fprintf(w, "| Total Tests | %d |\n", result.TotalTests)
	fmt.Fprintf(w, "| Passed | %d |\n", result.Passed)
	fmt.Fprintf(w, "| Failed | %d |\n", result.Failed)
	fmt.Fprintln(w)

	if result.DryRun {
		fmt.Fprintln(w, "## Test Plan (Dry Run)")
		fmt.Fprintln(w)
		for i, tr := range result.Tests {
			fmt.Fprintf(w, "%d. **%s** — %s\n", i+1, tr.Plan.Name, tr.Plan.Description)
		}
		return nil
	}

	// Overall status badge
	if result.Failed == 0 {
		fmt.Fprintln(w, "**Status: PASSED** ✅")
	} else {
		fmt.Fprintf(w, "**Status: FAILED** ❌ (%d/%d tests failed)\n", result.Failed, result.TotalTests)
	}
	fmt.Fprintln(w)

	// Test results
	fmt.Fprintln(w, "## Test Results")
	fmt.Fprintln(w)

	for _, tr := range result.Tests {
		statusEmoji := "✅"
		if !tr.Passed {
			statusEmoji = "❌"
		}

		fmt.Fprintf(w, "### %s %s\n\n", statusEmoji, tr.Plan.Name)
		fmt.Fprintf(w, "> %s\n\n", tr.Plan.Description)

		if tr.Retries > 0 {
			fmt.Fprintf(w, "_Fixed after %d retries_\n\n", tr.Retries)
		}

		if len(tr.Results) > 0 {
			fmt.Fprintln(w, "| Test | Status | Details |")
			fmt.Fprintln(w, "|------|--------|---------|")
			for _, rr := range tr.Results {
				status := "✅ Pass"
				details := ""
				if !rr.Passed {
					status = "❌ Fail"
					details = strings.ReplaceAll(rr.Error, "\n", " ")
					if len(details) > 80 {
						details = details[:80] + "..."
					}
				}
				fmt.Fprintf(w, "| `%s` | %s | %s |\n", rr.Name, status, details)
			}
			fmt.Fprintln(w)
		}
	}

	return nil
}

package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/mulhamna/suitest/internal/agent"
)

// TerminalReporter writes a colored terminal report.
type TerminalReporter struct{}

// NewTerminalReporter creates a TerminalReporter.
func NewTerminalReporter() *TerminalReporter {
	return &TerminalReporter{}
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
)

func (r *TerminalReporter) Write(w io.Writer, result *agent.RunResult) error {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s%s=== suitest Report ===%s\n", colorBold, colorBlue, colorReset)
	if result.RunID != "" {
		fmt.Fprintf(w, "Run ID:   %s\n", result.RunID)
	}
	fmt.Fprintf(w, "Path:     %s\n", result.Path)
	fmt.Fprintf(w, "Mode:     %s\n", result.Mode)
	fmt.Fprintf(w, "Provider: %s\n", result.Provider)
	fmt.Fprintf(w, "Duration: %s\n", result.FinishedAt.Sub(result.StartedAt))

	if result.DryRun {
		fmt.Fprintf(w, "\n%sDry run — test plan only:%s\n", colorYellow, colorReset)
		for i, tr := range result.Tests {
			fmt.Fprintf(w, "  %d. %s%s%s\n", i+1, colorCyan, tr.Plan.Name, colorReset)
			fmt.Fprintf(w, "     %s\n", tr.Plan.Description)
		}
		return nil
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s%sResults:%s\n", colorBold, colorBlue, colorReset)
	fmt.Fprintf(w, "%s─────────────────────────────────────────%s\n", colorBlue, colorReset)

	for _, tr := range result.Tests {
		if tr.Passed {
			fmt.Fprintf(w, "  %s✓%s %s", colorGreen, colorReset, tr.Plan.Name)
		} else {
			fmt.Fprintf(w, "  %s✗%s %s", colorRed, colorReset, tr.Plan.Name)
		}
		if tr.Retries > 0 {
			fmt.Fprintf(w, " %s(fixed after %d retries)%s", colorYellow, tr.Retries, colorReset)
		}
		fmt.Fprintln(w)

		// Show sub-results
		for _, rr := range tr.Results {
			if rr.Passed {
				fmt.Fprintf(w, "    %s  ✓%s %s\n", colorGreen, colorReset, rr.Name)
			} else {
				fmt.Fprintf(w, "    %s  ✗%s %s\n", colorRed, colorReset, rr.Name)
				if rr.Error != "" {
					// Indent error output
					lines := strings.Split(rr.Error, "\n")
					for _, line := range lines {
						if line != "" {
							fmt.Fprintf(w, "      %s%s%s\n", colorRed, line, colorReset)
						}
					}
				}
			}
		}
	}

	fmt.Fprintf(w, "%s─────────────────────────────────────────%s\n", colorBlue, colorReset)

	// Summary
	passColor := colorGreen
	if result.Failed > 0 {
		passColor = colorRed
	}
	fmt.Fprintf(w, "\n%s%sTotal: %d | Passed: %d | Failed: %d%s\n",
		colorBold, passColor, result.TotalTests, result.Passed, result.Failed, colorReset)

	if result.Failed == 0 {
		fmt.Fprintf(w, "\n%s%sAll tests passed!%s\n", colorBold, colorGreen, colorReset)
	} else {
		fmt.Fprintf(w, "\n%s%s%d test(s) failed%s\n", colorBold, colorRed, result.Failed, colorReset)
	}
	fmt.Fprintln(w)

	return nil
}
